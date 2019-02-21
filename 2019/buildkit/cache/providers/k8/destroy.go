package k8

import (
	logger "log"

	"bitbucket.org/okteto/okteto/backend/model"
	"bitbucket.org/okteto/okteto/backend/providers/k8/client"
	k8Deployment "bitbucket.org/okteto/okteto/backend/providers/k8/deployment"
	k8Ingress "bitbucket.org/okteto/okteto/backend/providers/k8/ingress"
	k8Service "bitbucket.org/okteto/okteto/backend/providers/k8/service"
	k8Volume "bitbucket.org/okteto/okteto/backend/providers/k8/volume"
)

//Destroy destroys a k8 deployment
func Destroy(s *model.Service, e *model.Environment, log *logger.Logger) error {
	if e.Provider.IsTestProvider() {
		return nil
	}
	c, err := client.Get(e.Provider)
	if err != nil {
		return err
	}
	if err = k8Ingress.Destroy(s, e, c, log); err != nil {
		return err
	}
	if err = k8Service.Destroy(s, e, c, log); err != nil {
		return err
	}
	if err := k8Deployment.Destroy(s, e, c, log); err != nil {
		return err
	}
	if s.Volumes == nil {
		s.Volumes = map[string]*model.Volume{}
	}
	s.Volumes[model.OktetoSyncVolume] = &model.Volume{
		Name:       model.OktetoSyncVolume,
		Persistent: true,
		Size:       "10Gi",
	}
	for _, v := range s.Volumes {
		if v.Persistent {
			if err := k8Volume.Destroy(s, v, e, c, log); err != nil {
				return err
			}
		}
	}
	return nil
}
