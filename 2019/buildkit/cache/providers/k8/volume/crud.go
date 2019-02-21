package volume

import (
	"fmt"
	"strings"
	"time"

	logger "log"

	"bitbucket.org/okteto/okteto/backend/model"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

//Deploy deploys a volume claim
func Deploy(s *model.Service, v *model.Volume, e *model.Environment, c *kubernetes.Clientset, log *logger.Logger) error {
	volumeName := v.FullName(s, e)
	vClient := c.CoreV1().PersistentVolumeClaims(e.Name)

	k8Volume, err := vClient.Get(volumeName, metav1.GetOptions{})
	if err != nil && !strings.Contains(err.Error(), "not found") {
		return fmt.Errorf("Error getting kubernetes volume claim: %s", err)
	}

	if k8Volume.Name != "" {
		return nil
	}

	log.Printf("Creating volume claim '%s'...", volumeName)
	k8Volume = translate(s, v, e)
	_, err = vClient.Create(k8Volume)
	if err != nil {
		return fmt.Errorf("Error creating kubernetes volume claim: %s", err)
	}
	log.Printf("Created volume claim %s.", volumeName)

	log.Printf("Waiting for the volme claim '%s' to be ready...", volumeName)
	tries := 0
	for tries < 50 {
		tries++
		time.Sleep(6 * time.Second)
		k8Volume, err = vClient.Get(volumeName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("Error getting kubernetes volume claim: %s", err)
		}
		if k8Volume.Status.Phase == apiv1.ClaimBound {
			log.Printf("kubernetes volume claim '%s' is bound.", volumeName)
			return nil
		}
	}
	return fmt.Errorf("kubernetes volume claim not ready after 5 minutes")
}

//Destroy destroys a volume claim
func Destroy(s *model.Service, v *model.Volume, e *model.Environment, c *kubernetes.Clientset, log *logger.Logger) error {
	volumeName := v.FullName(s, e)
	log.Printf("Deleting volume claim '%s'...", volumeName)
	vClient := c.CoreV1().PersistentVolumeClaims(e.Name)
	err := vClient.Delete(volumeName, &metav1.DeleteOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil
		}
		return fmt.Errorf("Error getting kubernetes volume claim: %s", err)
	}

	log.Printf("Waiting for the volume claim '%s' to be deleted...", volumeName)
	tries := 0
	for tries < 30 {
		tries++
		time.Sleep(6 * time.Second)
		_, err := vClient.Get(volumeName, metav1.GetOptions{})
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				log.Printf("Volume Claim '%s' successfully deleted.", volumeName)
				return nil
			}
			return fmt.Errorf("Error getting kubernetes volume claim: %s", err)
		}
	}
	return fmt.Errorf("kubernetes volume claim not deleted after 3 minutes")
}
