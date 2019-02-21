package ingress

import (
	"fmt"
	"strings"
	"time"

	logger "log"

	"bitbucket.org/okteto/okteto/backend/model"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

//Deploy deploys a k8 ingress for a service
func Deploy(s *model.Service, e *model.Environment, c *kubernetes.Clientset, log *logger.Logger) error {
	ingressName := s.Name
	iClient := c.ExtensionsV1beta1().Ingresses(e.Name)
	k8Ingress, err := iClient.Get(ingressName, metav1.GetOptions{})
	if err != nil && !strings.Contains(err.Error(), "not found") {
		return fmt.Errorf("Error getting kubernetes ingress: %s", err)
	}
	if k8Ingress.Name == "" {
		log.Printf("Creating ingress '%s'...", ingressName)
		k8Ingress = translate(s, e)
		_, err = iClient.Create(k8Ingress)
		if err != nil {
			return fmt.Errorf("Error creating kubernetes ingress: %s", err)
		}
		log.Printf("Created ingress '%s'.", ingressName)
	} else {
		log.Printf("Updating ingress '%s'...", ingressName)
		i := translate(s, e)
		_, err = iClient.Update(i)
		if err != nil {
			return fmt.Errorf("Error updating kubernetes ingress: %s", err)
		}
		log.Printf("Updated ingress '%s'.", ingressName)
	}
	return nil
}

//Destroy destroys the k8 ingress created by a service
func Destroy(s *model.Service, e *model.Environment, c *kubernetes.Clientset, log *logger.Logger) error {
	ingressName := s.Name
	log.Printf("Deleting ingress '%s'...", ingressName)
	iClient := c.ExtensionsV1beta1().Ingresses(e.Name)
	err := iClient.Delete(ingressName, &metav1.DeleteOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			log.Printf("Ingress '%s' was already deleted.", ingressName)
			return nil
		}
		return fmt.Errorf("Error getting kubernetes ingress: %s", err)
	}

	log.Printf("Waiting for the ingress '%s' to be deleted...", ingressName)
	tries := 0
	for tries < 30 {
		tries++
		time.Sleep(3 * time.Second)
		_, err := iClient.Get(ingressName, metav1.GetOptions{})
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				log.Printf("Ingress '%s' successfully deleted.", ingressName)
				return nil
			}
			return fmt.Errorf("Error getting kubernetes ingress: %s", err)
		}
	}
	return fmt.Errorf("kubernetes ingress not deleted after 3 minutes")
}
