package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/captainGeech42/chaldeploy/internal/generic_map"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// how long an instance will run, or how much time will be added to the expiration
const INSTANCE_RUNTIME = time.Duration(5) * time.Minute

type InstanceState int64

const (
	// a Running instance is live and can be accessed by the team
	Running InstanceState = iota

	// a Destroying instance is something in the process of being torn down.
	// From the perspective of the user, it is destroyed.
	// However, from the perspective of the backend, it isn't in a state where
	// it can be recreated.
	Destroying

	// a Destroyed instance doesn't exist anymore, and can be (re)deployed.
	// This is the first state of a DeploymentInstance
	Destroyed
)

func (s InstanceState) String() string {
	switch s {
	case Running:
		return "running"
	case Destroying:
		return "destroying"
	case Destroyed:
		return "destroyed"
	default:
		return "(unknown enum value)"
	}
}

// DeploymentInstance is a single deployment of a challenge for a team
type DeploymentInstance struct {
	// value for the `app` label
	AppName string

	// k8s namespace used for the instance
	Namespace string

	// expiration time for the instance
	ExpTime *time.Time

	// the current state of the instance
	State InstanceState

	// lock for mutating the state of the instance
	mu *sync.Mutex

	// hostname for connecting to the instance
	Hostname string

	// port for connecting to the instance
	Port int
}

// implement sync.Locker on DeploymentInstance
func (di *DeploymentInstance) Lock() {
	di.mu.Lock()
}

func (di *DeploymentInstance) Unlock() {
	di.mu.Unlock()
}

func (di *DeploymentInstance) GetCxn() string {
	return fmt.Sprintf("%s:%d", di.Hostname, di.Port)
}

// InstanceManager stores the necessary data for creating and destroying challenge instances on a k8s cluster
type InstanceManager struct {
	// k8s config
	Config *rest.Config

	// k8s client
	Clientset *kubernetes.Clientset

	// mutex for controlling access to the instance map
	Lock *sync.RWMutex

	// map of team id -> instance
	Instances *generic_map.MapOf[string, *DeploymentInstance]
}

// Initialize the instance manager object, including authing to the cluster
// TODO: ensure necessary permissions are obtained
func (im *InstanceManager) Init() error {
	// load the cluster config
	k8sConfig, err := getConfigForCluster()
	if err != nil {
		return err
	} else {
		im.Config = k8sConfig
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(im.Config)
	if err != nil {
		return err
	} else {
		im.Clientset = clientset
	}

	// initialize the map
	im.Instances = new(generic_map.MapOf[string, *DeploymentInstance])

	// get the chaldeploy namespaces for this challenge
	namespaceClient := im.Clientset.CoreV1().Namespaces()
	cdNamespaces, err := namespaceClient.List(context.TODO(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("chaldeploy.captaingee.ch/managed-by=yes,chaldeploy.captaingee.ch/chal=%s", HashString(config.ChallengeName)),
	})
	if err != nil {
		return err
	}

	if l := len(cdNamespaces.Items); l > 0 {
		log.Printf("found %d existing deployment(s) while initializing InstanceManager, ingesting them", l)

		// store info for each valid namespace identified
		for _, ns := range cdNamespaces.Items {
			di := &DeploymentInstance{
				AppName:   ns.Name,
				Namespace: ns.Name,
				State:     Running,
				mu:        &sync.Mutex{},
			}

			teamId := ns.Labels["chaldeploy.captaingee.ch/team-id"]

			// get the expiration time for the deployment instance
			if expTimeInt, err := strconv.Atoi(ns.Labels["chaldeploy.captaingee.ch/expiration-time"]); err != nil {
				log.Printf("couldn't parse expiration time for %s as int, setting 1hr expiration: %s", ns.Name, ns.Labels["chaldeploy.captaingee.ch/expiration-time"])
				expTime := time.Now().UTC().Add(INSTANCE_RUNTIME)
				di.ExpTime = &expTime
			} else {
				expTime := time.Unix(int64(expTimeInt), 0).UTC()
				di.ExpTime = &expTime
			}

			// get the connection info
			servicesClient := clientset.CoreV1().Services(di.Namespace)
			if service, err := servicesClient.Get(context.TODO(), di.AppName, metav1.GetOptions{}); err == nil {
				// found a running service, check if gcp assigned an lb to it
				if len(service.Status.LoadBalancer.Ingress) > 0 {
					// it did, save it
					di.Hostname = service.Status.LoadBalancer.Ingress[0].IP
					di.Port = config.ChallengePort
				}
			} else {
				log.Printf("couldn't get service when enumerating existing deployments: %v", err)
			}

			// if we couldn't get info from the running service, fill it out as unknown
			if di.Hostname == "" {
				di.Hostname = "<unknown>"
				di.Port = -1
			}

			// save the deployment
			im.Instances.Store(teamId, di)
		}
	}

	return nil
}

// Deploy an instance of a challenge for a team
// Returns the connection string and error
// ref:
//   - https://github.com/kubernetes/client-go/blob/master/examples/in-cluster-client-configuration/main.go
//   - https://github.com/kubernetes/client-go/blob/master/examples/create-update-delete-deployment/main.go
func (im *InstanceManager) CreateDeployment(teamId string) (string, error) {
	// compute a unique identifer for this deployment
	uniqName := strings.ToLower(fmt.Sprintf("chaldeploy-%s-%s", HashString(config.ChallengeName), strings.ReplaceAll(teamId, "-", "")))

	// initialize the DeploymentInstance
	di := &DeploymentInstance{
		AppName:   uniqName,
		Namespace: uniqName,
		State:     Destroyed,
		mu:        &sync.Mutex{},
	}
	di, _ = im.Instances.LoadOrStore(teamId, di)

	di.mu.Lock()
	defer di.mu.Unlock()
	if di.State == Destroyed {
		// get the k8s objects
		// TODO: create the other necessary resources ref rcds
		namespace := getNamespace(uniqName, teamId)
		deployment := getDeployment(di.AppName, teamId)
		service := getService(di.AppName, teamId)

		// set the expiration time
		now := time.Now().UTC()
		expTime := now.Add(INSTANCE_RUNTIME)
		namespace.ObjectMeta.Labels["chaldeploy.captaingee.ch/expiration-time"] = strconv.Itoa(int(expTime.Unix()))
		di.ExpTime = &expTime

		// create the k8s objects
		namespaceClient := im.Clientset.CoreV1().Namespaces()
		if _, err := namespaceClient.Create(context.TODO(), namespace, metav1.CreateOptions{}); err != nil {
			return "", fmt.Errorf("failed to create the namespace for %s: %v", uniqName, err)
		}
		deploymentsClient := im.Clientset.AppsV1().Deployments(di.Namespace)
		if _, err := deploymentsClient.Create(context.TODO(), deployment, metav1.CreateOptions{}); err != nil {
			return "", fmt.Errorf("failed to create the deployment for %s: %v", uniqName, err)
		}
		servicesClient := im.Clientset.CoreV1().Services(di.Namespace)
		if _, err := servicesClient.Create(context.TODO(), service, metav1.CreateOptions{}); err != nil {
			return "", fmt.Errorf("failed to create the service for %s: %v", uniqName, err)
		}

		// block until deployment is finished
		if !di.BlockUntilDeployed(20, 6) {
			return "", fmt.Errorf("timed out waiting for challenge to finish deploying for %s", uniqName)
		}

		// update the instance state
		createdService, err := servicesClient.Get(context.TODO(), di.AppName, metav1.GetOptions{})
		if err != nil {
			return "", fmt.Errorf("failed to retrieve connection info for %s: %v", uniqName, err)
		} else {
			di.State = Running
			di.Hostname = createdService.Status.LoadBalancer.Ingress[0].IP
			di.Port = config.ChallengePort
		}

	}

	return di.GetCxn(), nil
}

// get the deployment instance for a team, if there is one.
// if the return value is nil, that means there is no deployment
func (im *InstanceManager) GetDeploymentInstance(teamId string) *DeploymentInstance {
	di, _ := im.Instances.Load(teamId)
	return di
}

// Extend the expiration time of a deployment by 1hr
// Returns the new expiration time
func (im *InstanceManager) ExtendDeployment(teamId string) (string, error) {
	// get a ptr to the instance
	di, ok := im.Instances.Load(teamId)
	if !ok || di == nil {
		return "", fmt.Errorf("tried to extend a non-exist deployment for %s", teamId)
	}

	// validate state
	if di.State != Running {
		return "", fmt.Errorf("tried to extend a non-running deployment for %s (current state: %s)", teamId, di.State)
	}

	if di.ExpTime.Before(time.Now().UTC()) {
		return "", fmt.Errorf("tried to extend an already expired deployment for %s (exp time: %s)", teamId, di.GetExpTime())
	}

	// update the di instance
	newExp := di.ExpTime.Add(INSTANCE_RUNTIME)
	di.ExpTime = &newExp

	// update the namespace label
	namespacesClient := im.Clientset.CoreV1().Namespaces()
	ns, err := namespacesClient.Get(context.TODO(), di.Namespace, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("couldn't get namespace object from k8s to extend instance for %s", teamId)
	}

	ns.ObjectMeta.Labels["chaldeploy.captaingee.ch/expiration-time"] = strconv.Itoa(int(newExp.Unix()))
	if _, err := namespacesClient.Update(context.TODO(), ns, metav1.UpdateOptions{}); err != nil {
		return "", fmt.Errorf("couldn't update namespace in k8s to extend instance for %s", teamId)
	}

	return di.GetExpTime(), nil
}

// Destroy a challenge deployment
func (im *InstanceManager) DestroyDeployment(teamId string) error {
	// get a ptr to the instance
	di, ok := im.Instances.Load(teamId)
	if !ok || di == nil {
		return fmt.Errorf("tried to destroy a non-exist deployment for %s", teamId)
	}

	return di.DestroyInstance()
}

func (im *InstanceManager) DestroyExpiredInstances() error {
	var retErr error = nil

	now := time.Now().UTC()

	im.Instances.Range(func(key string, value *DeploymentInstance) bool {
		if value.ExpTime != nil && value.ExpTime.Before(now) {
			if err := value.DestroyInstance(); err != nil {
				retErr = err
				return false
			}
		}

		return true
	})

	return retErr
}

// destroy a deployment
func (di *DeploymentInstance) DestroyInstance() error {
	if di.State != Running {
		// deployment isn't running, probably already being destroyed, don't try to destroy it again
		return nil
	}

	// acquire the lock on the deployment and mark it as being destroyed
	di.mu.Lock()
	di.State = Destroying
	di.mu.Unlock()

	// init client
	client := im.Clientset.CoreV1().Namespaces()

	// check if the namespace exists, return if it doesn't
	if namespace, err := client.Get(context.TODO(), di.AppName, metav1.GetOptions{}); err != nil || namespace == nil {
		// TODO: investigate how err can be set (e.g., failed to lookup vs successfully looked up and confirmed non-existent)
		return nil
	}

	// delete resources
	di.mu.Lock()
	defer di.mu.Unlock()
	deletePolicy := metav1.DeletePropagationForeground

	if err := client.Delete(context.TODO(), di.Namespace, metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	}); err != nil {
		return fmt.Errorf("failed to delete namespace %s: %v", di.Namespace, err)
	}

	if !di.BlockUntilTerminated(20, 6) {
		return fmt.Errorf("failed to delete namespace %s: took too long to delete resource from k8s", di.Namespace)
	}

	di.State = Destroyed

	return nil

}

// Expontential backoff spin until the deployment service has an external IP assigned
// Returns true if blocked until successful deployment, otherwise false.
func (di *DeploymentInstance) BlockUntilDeployed(wait int, maxTries int) bool {
	client := im.Clientset.CoreV1().Services(di.Namespace)
	counter := 0

	if wait > 0 {
		time.Sleep(time.Duration(wait) * time.Second)
	}

	for {
		service, err := client.Get(context.TODO(), di.AppName, metav1.GetOptions{})
		if err == nil {
			if len(service.Status.LoadBalancer.Ingress) > 0 {
				if service.Status.LoadBalancer.Ingress[0].IP != "" {
					return true
				}
			}
		}

		counter += 1
		if counter == maxTries {
			return false
		}

		time.Sleep(time.Duration(math.Pow(2, float64(counter))) * time.Second)
	}
}

// Exponential backoff spin until the deployment is terminated.
// Returns true if blocked until successful deletion, otherwise false.
func (di *DeploymentInstance) BlockUntilTerminated(wait int, maxTries int) bool {
	client := im.Clientset.CoreV1().Namespaces()
	counter := 0

	if wait > 0 {
		time.Sleep(time.Duration(wait) * time.Second)
	}

	for {
		// namespace won't be deleted until all of the resources contained within it are terminated
		// wait for the ns to disappear
		_, err := client.Get(context.TODO(), di.Namespace, metav1.GetOptions{})
		if err != nil && strings.Contains(err.Error(), " not found") {
			return true
		}

		counter += 1
		if counter == maxTries {
			return false
		}

		time.Sleep(time.Duration(math.Pow(2, float64(counter))) * time.Second)
	}
}

// Get a human readable string for the expiration time of a deployment
func (di *DeploymentInstance) GetExpTime() string {
	if di.ExpTime == nil {
		return "<unknown>"
	}

	return di.ExpTime.Format("2006-01-02 15:04:05 UTC")
}

/////////////////////////////////

// An image could be in the form of path/image:tag
// Return just the image name. Matches [a-z0-9]([-a-z0-9]*[a-z0-9])?
func getImageName(image string) string {
	parts := strings.Split(image, "/")

	return strings.Split(parts[len(parts)-1], ":")[0]
}

// get a labelselector object that can be used for the deployment and service objects
func getSelector(appName, teamId string) *metav1.LabelSelector {
	return &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"app":                              appName,
			"chaldeploy.captaingee.ch/chal":    HashString(config.ChallengeName),
			"chaldeploy.captaingee.ch/team-id": teamId,
		},
	}
}

// get the namespace struct for the deployment
func getNamespace(name, teamId string) *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by":        "chaldeploy",
				"chaldeploy.captaingee.ch/chal":       HashString(config.ChallengeName),
				"chaldeploy.captaingee.ch/team-id":    teamId,
				"chaldeploy.captaingee.ch/managed-by": "yes",
			},
		},
	}
}

// get the deployment struct for the target app
func getDeployment(appName, teamId string) *appsv1.Deployment {
	selector := getSelector(appName, teamId)

	b := false

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: appName,
			Labels: map[string]string{
				"app":                              appName,
				"app.kubernetes.io/managed-by":     "chaldeploy",
				"chaldeploy.captaingee.ch/chal":    HashString(config.ChallengeName),
				"chaldeploy.captaingee.ch/team-id": teamId,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Selector: selector,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":                              appName,
						"app.kubernetes.io/managed-by":     "chaldeploy",
						"chaldeploy.captaingee.ch/chal":    HashString(config.ChallengeName),
						"chaldeploy.captaingee.ch/team-id": teamId,
					},
				},
				Spec: corev1.PodSpec{
					AutomountServiceAccountToken: &b,
					Containers: []corev1.Container{
						{
							Name:  getImageName(config.ChallengeImage),
							Image: config.ChallengeImage,
							Ports: []corev1.ContainerPort{{ContainerPort: int32(config.ChallengePort)}},

							// Resources: corev1.ResourceRequirements{
							// 	Limits: corev1.ResourceList{
							// 		corev1.ResourceCPU:    resource.MustParse("500m"), // TODO: configify these
							// 		corev1.ResourceMemory: resource.MustParse("256Mi"),
							// 	},
							// },
						},
					},
				},
			},
		},
	}
}

// get the service struct for the target app
func getService(appName, teamId string) *corev1.Service {
	selector := getSelector(appName, teamId)

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: appName,
			Labels: map[string]string{
				"app":                              appName,
				"app.kubernetes.io/managed-by":     "chaldeploy",
				"chaldeploy.captaingee.ch/chal":    HashString(config.ChallengeName),
				"chaldeploy.captaingee.ch/team-id": teamId,
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{Port: int32(config.ChallengePort), TargetPort: intstr.FromInt(config.ChallengePort), Protocol: corev1.ProtocolTCP},
			},
			Selector: selector.MatchLabels,
			Type:     corev1.ServiceTypeLoadBalancer,
		},
	}
}

// Identify the proper source for the cluster config and load it
// Load order:
//   - $CHALDEPLOY_K8SCONFIG
//   - /var/run/secrets/kubernetes.io/serviceaccount
//   - ~/.kube/config current context
func getConfigForCluster() (*rest.Config, error) {
	// check if a path to the k8s config was specified
	if config.K8sConfigPath != "" {
		log.Printf("using k8s config path from env var: %s", config.K8sConfigPath)

		// check if it exists
		if _, err := os.Stat(config.K8sConfigPath); os.IsExist(err) {
			// file exists, try to use it
			k8sConfig, err := clientcmd.BuildConfigFromFlags("", config.K8sConfigPath)
			if err != nil {
				return nil, err
			} else {
				return k8sConfig, nil
			}
		} else {
			return nil, errors.New("specified filepath for k8s config doesn't exist")
		}
	} else {
		// no path was specified, try an injected service account
		if _, err := os.Stat("/var/run/secrets/kubernetes.io/serviceaccount"); os.IsExist(err) {
			log.Println("found a service account, using k8s config from it")

			// ref: https://github.com/kubernetes/client-go/blob/master/examples/in-cluster-client-configuration/main.go#L41
			k8sConfig, err := rest.InClusterConfig()
			if err != nil {
				return nil, err
			} else {
				return k8sConfig, nil
			}
		} else {
			// no service account, try ~/.kube/config
			log.Println("service account not found, loading current context from k8s config in home dir")

			// ref: https://github.com/kubernetes/client-go/blob/master/examples/out-of-cluster-client-configuration/main.go#L43
			var configPath string
			if home := homedir.HomeDir(); home != "" {
				configPath = filepath.Join(home, ".kube", "config")
			} else {
				return nil, errors.New("couldn't resolve home directory, can't load local k8s config")
			}

			if _, err := os.Stat(configPath); os.IsNotExist(err) {
				return nil, errors.New("couldn't find a k8s config to load")
			}

			// use the current context in kubeconfig
			k8sConfig, err := clientcmd.BuildConfigFromFlags("", configPath)
			if err != nil {
				return nil, err
			} else {
				return k8sConfig, nil
			}
		}
	}
}
