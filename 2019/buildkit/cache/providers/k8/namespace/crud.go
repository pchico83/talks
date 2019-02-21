package namespace

import (
	"fmt"
	"strings"

	logger "log"

	"bitbucket.org/okteto/okteto/backend/model"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

//Create creates a namespace for a given project
func Create(e *model.Environment, c *kubernetes.Clientset, log *logger.Logger) error {
	log.Printf("Creating namespace '%s'...", e.Name)
	n, err := c.Core().Namespaces().Get(e.Name, metav1.GetOptions{})
	if err != nil && !strings.Contains(err.Error(), "not found") {
		return fmt.Errorf("Error getting kubernetes namespace: %s", err)
	}
	if n.Name == "" {
		n := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: e.Name}}
		_, err := c.Core().Namespaces().Create(n)
		if err != nil {
			return fmt.Errorf("Error creating kubernetes namespace: %s", err)
		}
		log.Printf("Created namespace '%s'.", e.Name)
	} else {
		log.Printf("Namespace '%s' was already created.", e.Name)
	}
	return nil
}
