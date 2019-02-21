package service

import (
	"fmt"
	"strconv"

	"bitbucket.org/okteto/okteto/backend/model"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func translate(s *model.Service, e *model.Environment) []*apiv1.Service {
	result := []*apiv1.Service{}
	result = append(
		result,
		&apiv1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name: s.Name,
			},
			Spec: apiv1.ServiceSpec{
				Selector: map[string]string{"app": s.Name},
				Type:     apiv1.ServiceTypeClusterIP,
				Ports:    getK8Ports(s.GetPrivatePorts()),
			},
		},
	)
	if e.Provider.IsIngress() || len(s.GetLoadBalancerPorts()) == 0 {
		return result
	}
	result = append(
		result,
		&apiv1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name: getLoadBalancerName(s.Name),
			},
			Spec: apiv1.ServiceSpec{
				Selector: map[string]string{"app": s.Name},
				Type:     apiv1.ServiceTypeLoadBalancer,
				Ports:    getK8Ports(s.GetLoadBalancerPorts()),
			},
		},
	)
	return result
}

func getLoadBalancerName(name string) string {
	return fmt.Sprintf("%s-load-balancer", name)
}

func getK8Ports(ports []string) []apiv1.ServicePort {
	result := []apiv1.ServicePort{}
	for _, port := range ports {
		portInt64, _ := strconv.ParseInt(port, 10, 32)
		result = append(
			result,
			apiv1.ServicePort{
				Name:       fmt.Sprintf("p%s", port),
				Port:       int32(portInt64),
				TargetPort: intstr.IntOrString{StrVal: port},
			})
	}
	return result
}
