package deployment

import (
	"fmt"
	"strconv"

	"bitbucket.org/okteto/okteto/backend/model"
	uuid "github.com/satori/go.uuid"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func translate(s *model.Service, e *model.Environment) *appsv1.Deployment {
	deploymentName := s.Name
	replicas := int32(s.Replicas)
	gracePeriod := int64(s.GracePeriod)
	volumes := []apiv1.Volume{}
	for _, v := range s.Volumes {
		if v.Persistent {
			volumes = append(
				volumes,
				apiv1.Volume{
					Name: v.Name,
					VolumeSource: apiv1.VolumeSource{
						PersistentVolumeClaim: &apiv1.PersistentVolumeClaimVolumeSource{
							ClaimName: v.FullName(s, e),
							ReadOnly:  false,
						},
					},
				},
			)
		} else {
			volumes = append(volumes, apiv1.Volume{Name: v.Name})
		}
	}
	containers := []apiv1.Container{}
	for name, c := range s.Containers {
		ports := []apiv1.ContainerPort{}
		for _, p := range c.Ports {
			portInt64, _ := strconv.ParseInt(p, 10, 32)
			ports = append(ports, apiv1.ContainerPort{
				Protocol:      apiv1.ProtocolTCP,
				ContainerPort: int32(portInt64),
				Name:          fmt.Sprintf("p%s", p),
			})
		}
		envs := []apiv1.EnvVar{}
		for _, e := range c.Environment {
			envs = append(envs, apiv1.EnvVar{
				Name:  e.Name,
				Value: e.Value,
			})
		}
		volumeMounts := []apiv1.VolumeMount{}
		for name, mount := range c.Mounts {
			volumeMounts = append(volumeMounts, apiv1.VolumeMount{
				Name:      name,
				MountPath: mount.Path,
			})
		}
		command := []string{}
		if c.Command != "" {
			command = append(command, c.Command)
		}

		container := apiv1.Container{
			Name:         name,
			Image:        c.Image,
			WorkingDir:   c.WorkingDir,
			Ports:        ports,
			VolumeMounts: volumeMounts,
			Env:          envs,
			Command:      command,
			Args:         c.Arguments,
		}
		if c.Resources != nil {
			container.Resources = apiv1.ResourceRequirements{}
			if c.Resources.Limits != nil {
				quantMemory, _ := resource.ParseQuantity(c.Resources.Limits.Memory)
				quantCPU, _ := resource.ParseQuantity(c.Resources.Limits.CPU)
				container.Resources.Limits = apiv1.ResourceList{
					apiv1.ResourceMemory: quantMemory,
					apiv1.ResourceCPU:    quantCPU,
				}
			}
			if c.Resources.Requests != nil {
				quantMemory, _ := resource.ParseQuantity(c.Resources.Requests.Memory)
				quantCPU, _ := resource.ParseQuantity(c.Resources.Requests.CPU)
				container.Resources.Requests = apiv1.ResourceList{
					apiv1.ResourceMemory: quantMemory,
					apiv1.ResourceCPU:    quantCPU,
				}
			}
		}
		containers = append(containers, container)
	}

	deploymentLabels := map[string]string{
		"app":         deploymentName,
		"okteto-uuid": uuid.NewV4().String(),
	}

	for k, v := range s.Labels {
		deploymentLabels[k] = v
	}

	var revisionHistoryLimit int32
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: deploymentName,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": deploymentName,
				},
			},
			RevisionHistoryLimit: &revisionHistoryLimit,
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: deploymentLabels,
				},
				Spec: apiv1.PodSpec{
					TerminationGracePeriodSeconds: &gracePeriod,
					Containers:                    containers,
					Volumes:                       volumes,
				},
			},
		},
	}
	if s.IsPersistent() {
		deployment.Spec.Strategy = appsv1.DeploymentStrategy{
			Type: appsv1.RecreateDeploymentStrategyType,
		}
	}
	if e.Registry == nil || e.Registry.Username == "" || e.Registry.Password == "" {
		return deployment
	}

	deployment.Spec.Template.Spec.ImagePullSecrets = []apiv1.LocalObjectReference{
		apiv1.LocalObjectReference{Name: e.Name},
	}
	return deployment
}
