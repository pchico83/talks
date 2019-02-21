package providers

import (
	logger "log"
	"strings"

	"bitbucket.org/okteto/okteto/backend/config"
	"bitbucket.org/okteto/okteto/backend/model"
	"bitbucket.org/okteto/okteto/backend/providers/aws"
	"bitbucket.org/okteto/okteto/backend/providers/k8"
	K8Service "bitbucket.org/okteto/okteto/backend/providers/k8/service"
)

//Destroy destroys a given service in a given environment
func Destroy(s *model.Service, e *model.Environment, log *logger.Logger) error {
	log.Printf("Destroying the service '%s'...", s.Name)

	if !e.Provider.IsIngress() && len(s.GetLoadBalancerPorts()) > 0 && config.IsDNSConfigured() {
		target, err := K8Service.GetEndpoint(s, e)
		if err != nil && !strings.Contains(err.Error(), "not found") {
			return err
		}
		if target != "" {
			aws.Destroy(s, e, target, "A")
		}
	}
	if err := k8.Destroy(s, e, log); err != nil {
		return err
	}
	log.Printf("Service '%s' successfully destroyed.", s.Name)
	return nil
}
