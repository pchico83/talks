package k8

import (
	logger "log"

	"bitbucket.org/okteto/okteto/backend/model"
	"bitbucket.org/okteto/okteto/backend/providers/k8/client"
	k8Deployment "bitbucket.org/okteto/okteto/backend/providers/k8/deployment"
	k8Ingress "bitbucket.org/okteto/okteto/backend/providers/k8/ingress"
	"bitbucket.org/okteto/okteto/backend/providers/k8/namespace"
	"bitbucket.org/okteto/okteto/backend/providers/k8/secret"
	k8service "bitbucket.org/okteto/okteto/backend/providers/k8/service"
	"bitbucket.org/okteto/okteto/backend/providers/k8/user"
	k8Volume "bitbucket.org/okteto/okteto/backend/providers/k8/volume"
)

//Deploy deploys a k8 deployment
func Deploy(s *model.Service, e *model.Environment, log *logger.Logger) error {
	if e.Provider.IsTestProvider() {
		return nil
	}
	c, err := client.Get(e.Provider)
	if err != nil {
		return err
	}
	if err := namespace.Create(e, c, log); err != nil {
		return err
	}
	if err := user.Create(e, c, log); err != nil {
		return err
	}
	if err := secret.Create(e, c, log); err != nil {
		return err
	}
	for _, v := range s.Volumes {
		if v.Persistent {
			if err := k8Volume.Deploy(s, v, e, c, log); err != nil {
				return err
			}
		}
	}
	if err := k8Deployment.Deploy(s, e, c, log); err != nil {
		return err
	}
	if len(s.GetPrivatePorts()) == 0 {
		return nil
	}
	if err := k8service.Deploy(s, e, c, log); err != nil {
		return err
	}
	if !e.Provider.IsIngress() || len(s.GetIngressRules(true)) == 0 {
		return nil
	}
	if err := k8Ingress.Deploy(s, e, c, log); err != nil {
		return err
	}
	return nil
}
