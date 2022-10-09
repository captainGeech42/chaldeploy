package main

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func getDeployment() {
	deployment := appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "test-nc", Image: "test-nc:v2", ImagePullPolicy: "Never"},
					},
				},
			},
		},
	}

	fmt.Printf("%#v\n", deployment)
}
