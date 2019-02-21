package secret

import (
	"encoding/base64"
	"fmt"
	"strings"

	logger "log"

	"bitbucket.org/okteto/okteto/backend/model"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

//Create creates a docker registry secret for a given project
func Create(e *model.Environment, c *kubernetes.Clientset, log *logger.Logger) error {
	if err := createDockerCredentialsSecret(e, c, log); err != nil {
		return err
	}
	if !e.Provider.IsTLSIngress() || e.Provider.Ingress.TLS.Type != model.FixCertificate {
		return nil
	}
	if err := copyCertificateSecret(e, c, log); err != nil {
		return err
	}
	return nil
}

//Get returns a secret given its name
func Get(name string, e *model.Environment, c *kubernetes.Clientset) (*v1.Secret, error) {
	s, err := c.Core().Secrets(e.Name).Get(name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("Error getting kubernetes secret: %s", err)
	}
	return s, nil
}

func createDockerCredentialsSecret(e *model.Environment, c *kubernetes.Clientset, log *logger.Logger) error {
	if e.Registry == nil || e.Registry.Username == "" || e.Registry.Password == "" {
		return nil
	}
	log.Printf("Creating docker registry secret '%s'...", e.Name)
	s, err := c.Core().Secrets(e.Name).Get(e.Name, metav1.GetOptions{})
	if err != nil && !strings.Contains(err.Error(), "not found") {
		return fmt.Errorf("Error getting kubernetes secret: %s", err)
	}
	dockerConfig, _ := base64.StdEncoding.DecodeString(e.B64DockerConfig())
	data := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: e.Name},
		Type:       v1.SecretTypeDockerConfigJson,
		Data:       map[string][]byte{".dockerconfigjson": dockerConfig},
	}
	if s.Name == "" {
		_, err := c.Core().Secrets(e.Name).Create(data)
		if err != nil {
			return fmt.Errorf("Error creating kubernetes secret: %s", err)
		}
		log.Printf("Created docker registry secret '%s'.", e.Name)
	} else {
		_, err := c.Core().Secrets(e.Name).Update(data)
		if err != nil {
			return fmt.Errorf("Error updating kubernetes secret: %s", err)
		}
		log.Printf("Docker registry secret '%s' was updated.", e.Name)
	}
	return nil
}

func copyCertificateSecret(e *model.Environment, c *kubernetes.Clientset, log *logger.Logger) error {
	log.Print("Configuring TLS certificates...")
	sCertificate, err := c.Core().Secrets(e.Provider.Ingress.TLS.Certificate.Namespace).Get(
		e.Provider.Ingress.TLS.Certificate.Secret,
		metav1.GetOptions{},
	)
	if err != nil {
		return fmt.Errorf("Error getting kubernetes secret: %s", err)
	}

	//creating new certificate in user namespace
	sClient := c.Core().Secrets(e.Name)
	sNamespace, err := sClient.Get(e.Provider.Ingress.TLS.Certificate.Secret, metav1.GetOptions{})
	if err != nil && !strings.Contains(err.Error(), "not found") {
		return fmt.Errorf("Error getting kubernetes secret: %s", err)
	}
	data := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: e.Provider.Ingress.TLS.Certificate.Secret},
		Type:       v1.SecretTypeTLS,
		Data:       sCertificate.Data,
	}
	if sNamespace.Name == "" {
		_, err := sClient.Create(data)
		if err != nil {
			return fmt.Errorf("Error creating kubernetes secret: %s", err)
		}
		log.Print("Configured TLS certificates.")
	} else {
		_, err := sClient.Update(data)
		if err != nil {
			return fmt.Errorf("Error updating kubernetes secret: %s", err)
		}
		log.Print("TLS certificates were already configured.")
	}
	return nil
}
