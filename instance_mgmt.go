package main

import (
	"context"
	"flag"
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

type InstanceManager struct {
	// Pointer to the app config
	appConfig *Config
}

func (im *InstanceManager) Init(config *Config) {
	im.appConfig = config

	// TODO: init memcache
}

// Deploy an instance of a challenge for a team
// Returns the connection string and error
func (im *InstanceManager) CreateDeployment(teamName, teamId string) (string, error) {

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
							Name:            "test-nc",
							Image:           "test-nc:v2",
							ImagePullPolicy: "Never",
							Ports:           []corev1.ContainerPort{{ContainerPort: 31337}},
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

func getConfigForCluster() *rest.Config {
	// check if we have an injected service account token available
	if _, err := os.Stat("/var/run/secrets/kubernetes.io/serviceaccount"); os.IsNotExist(err) {
		log.Println("service account not found, loading current context from k8s config in home dir")

		// ref: https://github.com/kubernetes/client-go/blob/master/examples/out-of-cluster-client-configuration/main.go#L43
		var kubeconfig *string
		if home := homedir.HomeDir(); home != "" {
			kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
		} else {
			kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
		}
		flag.Parse()

		// use the current context in kubeconfig
		config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
		if err != nil {
			panic(err.Error())
		}

		return config
	} else {
		log.Println("found a service account, using k8s config from it")

		// ref: https://github.com/kubernetes/client-go/blob/master/examples/in-cluster-client-configuration/main.go#L41
		config, err := rest.InClusterConfig()
		if err != nil {
			panic(err.Error())
		}

		return config
	}
}

func deployApp(teamName string) {
	// ref:
	//   - https://github.com/kubernetes/client-go/blob/master/examples/in-cluster-client-configuration/main.go
	//   - https://github.com/kubernetes/client-go/blob/master/examples/create-update-delete-deployment/main.go

	// get the proper config
	config := getConfigForCluster()

	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	appName := strings.ToLower(fmt.Sprintf("chaldeploy-app-%s", teamName))

	deployment := getDeployment(appName)

	deploymentsClient := clientset.AppsV1().Deployments(corev1.NamespaceDefault)
	_, err = deploymentsClient.Create(context.TODO(), &deployment, metav1.CreateOptions{})
	if err != nil {
		panic(err)
	}
}
