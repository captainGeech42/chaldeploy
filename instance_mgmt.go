package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// InstanceManager stores the necessary data for creating and destroying challenge instances on a k8s cluster
type InstanceManager struct {
	Config    *rest.Config
	Clientset *kubernetes.Clientset
}

func (im *InstanceManager) Init() error {
	// load the cluster config
	k8sConfig, err := getConfigForCluster()
	if err != nil {
		return err
	} else {
		im.Config = k8sConfig
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		return err
	} else {
		im.Clientset = clientset
	}

	// TODO: init memcache
	return nil
}

// Deploy an instance of a challenge for a team
// Returns the connection string and error
// ref:
//   - https://github.com/kubernetes/client-go/blob/master/examples/in-cluster-client-configuration/main.go
//   - https://github.com/kubernetes/client-go/blob/master/examples/create-update-delete-deployment/main.go
func (im *InstanceManager) CreateDeployment(teamName, teamId string) (string, error) {
	appName := strings.ToLower(fmt.Sprintf("chaldeploy-app-%s", teamName))

	deployment := getDeployment(appName)

	deploymentsClient := im.Clientset.AppsV1().Deployments(corev1.NamespaceDefault)
	_, err := deploymentsClient.Create(context.TODO(), &deployment, metav1.CreateOptions{})
	if err != nil {
		panic(err)
	}

	return "", nil
}

/////////////////////////////////

// get a labelselector object that can be used for the deployment and service objects
func getSelector(appName string) *metav1.LabelSelector {
	return &metav1.LabelSelector{
		MatchLabels: map[string]string{"app": appName},
	}
}

// get the deployment struct for the target app
func getDeployment(appName string) appsv1.Deployment {
	selector := getSelector(appName)

	deployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:   appName,
			Labels: map[string]string{"app": appName, "chaldeploy-target": "yes"},
		},
		Spec: appsv1.DeploymentSpec{
			Selector: selector,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": appName},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            strings.Split(config.ChallengeImage, ":")[0],
							Image:           config.ChallengeImage,
							ImagePullPolicy: "Never", // TODO: adjust as needed off of minikube
							Ports:           []corev1.ContainerPort{{ContainerPort: int32(config.ChallengePort)}},
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("500m"),
									corev1.ResourceMemory: resource.MustParse("256Mi"),
								},
							},
						},
					},
				},
			},
		},
	}

	return deployment
}

// Identify the proper source for the cluster config and load it
// Load order:
//   - $CHALDEPLOY_K8SCONFIG
//   - /var/run/secrets/kubernetes.io/serviceaccount
//   - ~/.kube/config current context
func getConfigForCluster() (*rest.Config, error) {
	// check if a path to the k8s config was specified
	if config.K8sConfigPath != "" {
		log.Printf("using k8s config path from env var: %s\n", config.K8sConfigPath)

		// check if it exists
		if _, err := os.Stat(config.K8sConfigPath); os.IsExist(err) {
			// file exists, try to use it
			k8sConfig, err := clientcmd.BuildConfigFromFlags("", config.K8sConfigPath)
			if err != nil {
				return k8sConfig, nil
			} else {
				return nil, err
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
				return k8sConfig, nil
			} else {
				return nil, err
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
				return k8sConfig, nil
			} else {
				return nil, err
			}
		}
	}
}
