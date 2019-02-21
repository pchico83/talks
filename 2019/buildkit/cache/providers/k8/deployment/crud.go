package deployment

import (
	"fmt"
	"strings"
	"time"

	logger "log"

	"bitbucket.org/okteto/okteto/backend/model"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

//Deploy deploys a service as a k8 deployment
func Deploy(s *model.Service, e *model.Environment, c *kubernetes.Clientset, log *logger.Logger) error {
	deploymentName := s.Name
	dClient := c.AppsV1().Deployments(e.Name)

	d, err := dClient.Get(deploymentName, metav1.GetOptions{})
	if err != nil && !strings.Contains(err.Error(), "not found") {
		return fmt.Errorf("Error getting kubernetes deployment: %s", err)
	}

	if d.Name == "" {
		log.Printf("Creating deployment '%s'...", deploymentName)
		d = translate(s, e)
		_, err = dClient.Create(d)
		if err != nil {
			return fmt.Errorf("Error creating kubernetes deployment: %s", err)
		}
		log.Printf("Created deployment %s.", deploymentName)
	} else {
		log.Printf("Updating deployment '%s'...", deploymentName)
		d = translate(s, e)
		_, err = dClient.Update(d)
		if err != nil {
			return fmt.Errorf("Error updating kubernetes deployment: %s", err)
		}
	}

	log.Printf("Waiting for the deployment '%s' to be ready...", deploymentName)
	tries := 0
	for tries < 50 {
		tries++
		time.Sleep(6 * time.Second)
		d, err = dClient.Get(deploymentName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("Error getting kubernetes deployment: %s", err)
		}
		if d.Status.ReadyReplicas == int32(s.Replicas) && d.Status.Replicas == int32(s.Replicas) && d.Status.UpdatedReplicas == int32(s.Replicas) {
			if d.Status.UnavailableReplicas == 0 {
				log.Printf("kubernetes deployment '%s' is ready.", deploymentName)
				return nil
			}
		}
	}
	return fmt.Errorf("kubernetes deployment not ready after 5 minutes")
}

//Destroy destroys the k8 deployment created by a service
func Destroy(s *model.Service, e *model.Environment, c *kubernetes.Clientset, log *logger.Logger) error {
	deploymentName := s.Name
	log.Printf("Deleting deployment '%s'...", deploymentName)
	dClient := c.AppsV1().Deployments(e.Name)
	deletePolicy := metav1.DeletePropagationForeground
	err := dClient.Delete(deploymentName, &metav1.DeleteOptions{PropagationPolicy: &deletePolicy})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil
		}
		return fmt.Errorf("Error getting kubernetes deployment: %s", err)
	}

	log.Printf("Waiting for the deployment '%s' to be deleted...", deploymentName)
	tries := 0
	for tries < 30 {
		tries++
		time.Sleep(6 * time.Second)
		_, err := dClient.Get(deploymentName, metav1.GetOptions{})
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				log.Printf("Deployment '%s' successfully deleted.", deploymentName)
				return nil
			}
			return fmt.Errorf("Error getting kubernetes deployment: %s", err)
		}
	}
	return fmt.Errorf("kubernetes deployment not deleted after 3 minutes")
}
