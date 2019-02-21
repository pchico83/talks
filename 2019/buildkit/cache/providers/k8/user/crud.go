package user

import (
	"fmt"
	"strings"

	logger "log"

	"bitbucket.org/okteto/okteto/backend/model"
	"bitbucket.org/okteto/okteto/backend/providers/k8/client"
	"bitbucket.org/okteto/okteto/backend/providers/k8/secret"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

//DevName returns the name of dev mode related resources
func DevName(e *model.Environment) string {
	return fmt.Sprintf("%s-dev", e.Name)
}

//Create creates a service account for with dev mode priviledges
func Create(e *model.Environment, c *kubernetes.Clientset, log *logger.Logger) error {
	if err := createServiceAccount(e, c, log); err != nil {
		return err
	}
	if err := createRole(e, c, log); err != nil {
		return err
	}
	if err := createRoleBinding(e, c, log); err != nil {
		return err
	}
	return nil
}

//GetServiceAccountCredential returns the credential for accessing the dev mode container
func GetServiceAccountCredential(e *model.Environment) (string, error) {
	c, err := client.Get(e.Provider)
	if err != nil {
		return "", err
	}
	saName := DevName(e)
	sa, err := c.CoreV1().ServiceAccounts(e.Name).Get(saName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("Error getting kubernetes service account: %s", err)
	}
	s, err := secret.Get(sa.Secrets[0].Name, e, c)
	if err != nil {
		return "", err
	}
	return string(s.Data["token"]), nil
}

func createServiceAccount(e *model.Environment, c *kubernetes.Clientset, log *logger.Logger) error {
	saName := DevName(e)
	sa, err := c.CoreV1().ServiceAccounts(e.Name).Get(saName, metav1.GetOptions{})
	if err != nil && !strings.Contains(err.Error(), "not found") {
		return fmt.Errorf("Error getting kubernetes service account: %s", err)
	}
	if sa.Name != "" {
		return nil
	}
	log.Printf("Creating service account '%s'...", saName)
	sa = translateServiceAccount(e)
	_, err = c.CoreV1().ServiceAccounts(e.Name).Create(sa)
	if err != nil {
		return fmt.Errorf("Error creating kubernetes service account: %s", err)
	}
	log.Printf("Created service account '%s'.", e.Name)
	return nil
}

func createRole(e *model.Environment, c *kubernetes.Clientset, log *logger.Logger) error {
	rName := DevName(e)
	r, err := c.RbacV1().Roles(e.Name).Get(rName, metav1.GetOptions{})
	if err != nil && !strings.Contains(err.Error(), "not found") {
		return fmt.Errorf("Error getting kubernetes role: %s", err)
	}
	if r.Name != "" {
		return nil
	}
	log.Printf("Creating role '%s'...", rName)
	r = translateRole(e)
	_, err = c.RbacV1().Roles(e.Name).Create(r)
	if err != nil {
		return fmt.Errorf("Error creating kubernetes role: %s", err)
	}
	log.Printf("Created role '%s'.", rName)
	return nil
}

func createRoleBinding(e *model.Environment, c *kubernetes.Clientset, log *logger.Logger) error {
	rbName := DevName(e)
	rb, err := c.RbacV1().RoleBindings(e.Name).Get(rbName, metav1.GetOptions{})
	if err != nil && !strings.Contains(err.Error(), "not found") {
		return fmt.Errorf("Error getting kubernetes role binding: %s", err)
	}
	if rb.Name != "" {
		return nil
	}
	log.Printf("Creating role binding '%s'...", rbName)
	rb = translateRoleBinding(e)
	_, err = c.RbacV1().RoleBindings(e.Name).Create(rb)
	if err != nil {
		return fmt.Errorf("Error creating kubernetes role binding: %s", err)
	}
	log.Printf("Created role binding '%s'.", rbName)
	return nil
}
