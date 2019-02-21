package volume

import (
	"bitbucket.org/okteto/okteto/backend/model"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func translate(s *model.Service, v *model.Volume, e *model.Environment) *apiv1.PersistentVolumeClaim {
	quantDisk, _ := resource.ParseQuantity(v.Size)
	return &apiv1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: v.FullName(s, e),
		},
		Spec: apiv1.PersistentVolumeClaimSpec{
			AccessModes: []apiv1.PersistentVolumeAccessMode{apiv1.ReadWriteOnce},
			Resources: apiv1.ResourceRequirements{
				Requests: apiv1.ResourceList{
					"storage": quantDisk,
				},
			},
		},
	}
}
