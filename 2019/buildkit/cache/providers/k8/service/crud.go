package service

import (
	"fmt"
	"strings"
	"time"

	logger "log"

	"bitbucket.org/okteto/okteto/backend/model"
	"bitbucket.org/okteto/okteto/backend/providers/k8/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

//Deploy deploys a k8 service for a service
func Deploy(s *model.Service, e *model.Environment, c *kubernetes.Clientset, log *logger.Logger) error {
	sClient := c.CoreV1().Services(e.Name)
	for _, this := range translate(s, e) {
		k8Service, err := sClient.Get(this.Name, metav1.GetOptions{})
		if err != nil && !strings.Contains(err.Error(), "not found") {
			return fmt.Errorf("Error getting kubernetes service: %s", err)
		}
		if k8Service.Name == "" {
			log.Printf("Creating service '%s'...", this.Name)
			_, err = sClient.Create(this)
			if err != nil {
				return fmt.Errorf("Error creating kubernetes service: %s", err)
			}
			log.Printf("Created service '%s'.", this.Name)
		} else {
			log.Printf("Updating service '%s'...", this.Name)
			k8Service.Spec.Ports = this.Spec.Ports
			_, err = sClient.Update(k8Service)
			if err != nil {
				return fmt.Errorf("Error updating kubernetes service: %s", err)
			}
			log.Printf("Updated service '%s'.", this.Name)
		}
	}
	return nil
}

//GetEndpoint returns the endpoint of a given k8 service
func GetEndpoint(s *model.Service, e *model.Environment) (string, error) {
	if e.Provider.IsTestProvider() {
		return "", nil
	}
	serviceName := s.Name
	c, err := client.Get(e.Provider)
	if err != nil {
		return "", err
	}
	sClient := c.CoreV1().Services(e.Name)
	tries := 0
	for tries < 30 {
		tries++
		time.Sleep(6 * time.Second)
		k8Service, err := sClient.Get(getLoadBalancerName(serviceName), metav1.GetOptions{})
		if err != nil {
			return "", fmt.Errorf("Error getting kubernetes service: %s", err)
		}
		if k8Service.Name == "" {
			return "", nil
		}
		if len(k8Service.Status.LoadBalancer.Ingress) > 0 {
			return k8Service.Status.LoadBalancer.Ingress[0].IP, nil
		}
	}
	return "", fmt.Errorf("External load balancer not created after 3 minutes")
}

//Destroy destroys the k8 services created by a okteto service
func Destroy(s *model.Service, e *model.Environment, c *kubernetes.Clientset, log *logger.Logger) error {
	for _, this := range []string{s.Name, getLoadBalancerName(s.Name)} {
		if err := destroy(this, e, c, log); err != nil {
			return err
		}
	}
	return nil
}

func destroy(name string, e *model.Environment, c *kubernetes.Clientset, log *logger.Logger) error {
	log.Printf("Deleting service '%s'...", name)
	sClient := c.CoreV1().Services(e.Name)
	err := sClient.Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			log.Printf("Service '%s' was already deleted.", name)
			return nil
		}
		return fmt.Errorf("Error getting kubernetes service: %s", err)
	}
	log.Printf("Waiting for the service '%s' to be deleted...", name)
	tries := 0
	for tries < 30 {
		tries++
		time.Sleep(6 * time.Second)
		_, err := sClient.Get(name, metav1.GetOptions{})
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				log.Printf("Service '%s' successfully deleted.", name)
				return nil
			}
			return fmt.Errorf("Error getting kubernetes service: %s", err)
		}
	}
	return fmt.Errorf("kubernetes service not deleted after 3 minutes")
}
