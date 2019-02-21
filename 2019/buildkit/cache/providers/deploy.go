package providers

import (
	"fmt"
	logger "log"

	"bitbucket.org/okteto/okteto/backend/config"
	"bitbucket.org/okteto/okteto/backend/providers/aws"
	"bitbucket.org/okteto/okteto/backend/providers/k8"
	K8Service "bitbucket.org/okteto/okteto/backend/providers/k8/service"

	"bitbucket.org/okteto/okteto/backend/model"
)

//Deploy deploys a given service in a given environment
func Deploy(s *model.Service, e *model.Environment, log *logger.Logger) error {
	log.Printf("Deploying the service '%s'...", s.Name)

	if len(s.GetIngressRules(false)) > 0 && !e.Provider.IsIngress() {
		return fmt.Errorf("Support for ingress ports requires ingress configuration in your project")
	}
	if err := deploy(s, e, log); err != nil {
		return err
	}
	log.Printf("Service '%s' successfully deployed.", s.Name)
	return nil
}

func deploy(s *model.Service, e *model.Environment, log *logger.Logger) error {
	if err := k8.Deploy(s, e, log); err != nil {
		return err
	}
	if e.Provider.IsIngress() || len(s.GetLoadBalancerPorts()) == 0 {
		return nil
	}

	log.Print("Waiting for load balancer to be created...")
	target, err := K8Service.GetEndpoint(s, e)
	if err != nil {
		return err
	}
	if target == "" {
		return nil
	}
	if config.IsDNSConfigured() {
		if err := aws.Create(s, e, target, "A", log); err != nil {
			return err
		}
	}
	return nil
}
