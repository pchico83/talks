package providers

import (
	logger "log"
	"sort"

	"bitbucket.org/okteto/okteto/backend/model"
)

// DevDeploy deploys the dev version of a service
func DevDeploy(s *model.Service, e *model.Environment, log *logger.Logger) error {
	log.Printf("Enabling development mode for service '%s'...", s.Name)
	devContainer := findDevContainer(s)
	swapDevContainerConfiguration(devContainer)
	injectSyncthingContainer(s, devContainer.Development.Persistent)
	s.Replicas = 1

	if s.Labels == nil {
		s.Labels = make(map[string]string)
	}

	s.Labels["okteto-cnd"] = s.Name

	if err := deploy(s, e, log); err != nil {
		return err
	}

	log.Printf("Enabled development mode for service '%s'.", s.Name)
	return nil
}

func findDevContainerKey(s *model.Service) string {
	var devContainerKey string
	names := make([]string, 0, len(s.Containers))
	for name := range s.Containers {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		c := s.Containers[name]
		if devContainerKey == "" {
			devContainerKey = name
		}

		if c.Development != nil {
			devContainerKey = name
		}
	}

	return devContainerKey
}

func findDevContainer(s *model.Service) *model.Container {
	devContainerKey := findDevContainerKey(s)
	devContainer := s.Containers[devContainerKey]

	if devContainer.Development == nil {
		devContainer.Development = &model.Development{}
	}

	devContainer.Development.EnsureValues(devContainer.Image)
	return devContainer
}

func swapDevContainerConfiguration(dev *model.Container) {
	dev.Image = dev.Development.Image
	dev.WorkingDir = dev.Development.Path
	dev.Command = dev.Development.Command
	dev.Arguments = dev.Development.Arguments
	if dev.Mounts == nil {
		dev.Mounts = map[string]*model.Mount{}
	}
	dev.Mounts[model.OktetoSyncVolume] = &model.Mount{Path: dev.Development.Path}
}

func injectSyncthingContainer(s *model.Service, persistent bool) {
	if s.Volumes == nil {
		s.Volumes = map[string]*model.Volume{}
	}
	s.Volumes[model.OktetoSyncVolume] = &model.Volume{Name: model.OktetoSyncVolume}
	if persistent {
		s.Volumes[model.OktetoSyncVolume].Persistent = true
		s.Volumes[model.OktetoSyncVolume].Size = "10Gi"
	}
	s.Containers[model.OktetoSyncContainer] = model.GetSyncthingContainer()
}
