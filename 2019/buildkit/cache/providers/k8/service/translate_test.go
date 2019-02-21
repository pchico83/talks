package service

import (
	"reflect"
	"testing"

	"bitbucket.org/okteto/okteto/backend/model"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestTranslate(t *testing.T) {
	tests := []struct {
		name        string
		service     model.Service
		environment model.Environment
		expected    []*apiv1.Service
	}{
		{
			name: "ingress",
			service: model.Service{
				Name: "test",
				Containers: map[string]*model.Container{
					"api": &model.Container{
						Ports: []string{"443"},
						Ingress: []*model.Ingress{
							&model.Ingress{
								Host: "host",
								Path: "/api/v1",
								Port: "8080",
							},
						},
						Expose: []string{"80"},
					},
				},
			},
			environment: model.Environment{
				Provider: &model.Provider{},
			},
			expected: []*apiv1.Service{
				&apiv1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					Spec: apiv1.ServiceSpec{
						Selector: map[string]string{"app": "test"},
						Type:     apiv1.ServiceTypeClusterIP,
						Ports: []apiv1.ServicePort{
							apiv1.ServicePort{
								Name:       "p443",
								Port:       int32(443),
								TargetPort: intstr.IntOrString{StrVal: "443"},
							},
							apiv1.ServicePort{
								Name:       "p8080",
								Port:       int32(8080),
								TargetPort: intstr.IntOrString{StrVal: "8080"},
							},
							apiv1.ServicePort{
								Name:       "p80",
								Port:       int32(80),
								TargetPort: intstr.IntOrString{StrVal: "80"},
							},
						},
					},
				},
				&apiv1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-load-balancer",
					},
					Spec: apiv1.ServiceSpec{
						Selector: map[string]string{"app": "test"},
						Type:     apiv1.ServiceTypeLoadBalancer,
						Ports: []apiv1.ServicePort{
							apiv1.ServicePort{
								Name:       "p443",
								Port:       int32(443),
								TargetPort: intstr.IntOrString{StrVal: "443"},
							},
						},
					},
				},
			},
		},
		{
			name: "ingress",
			service: model.Service{
				Name: "test",
				Containers: map[string]*model.Container{
					"api": &model.Container{
						Ports: []string{"443"},
						Ingress: []*model.Ingress{
							&model.Ingress{
								Host: "host",
								Path: "/api/v1",
								Port: "8080",
							},
						},
						Expose: []string{"80"},
					},
				},
			},
			environment: model.Environment{
				Provider: &model.Provider{
					Ingress: &model.IngressController{
						Domain: "example.com",
					},
				},
			},
			expected: []*apiv1.Service{
				&apiv1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					Spec: apiv1.ServiceSpec{
						Selector: map[string]string{"app": "test"},
						Type:     apiv1.ServiceTypeClusterIP,
						Ports: []apiv1.ServicePort{
							apiv1.ServicePort{
								Name:       "p443",
								Port:       int32(443),
								TargetPort: intstr.IntOrString{StrVal: "443"},
							},
							apiv1.ServicePort{
								Name:       "p8080",
								Port:       int32(8080),
								TargetPort: intstr.IntOrString{StrVal: "8080"},
							},
							apiv1.ServicePort{
								Name:       "p80",
								Port:       int32(80),
								TargetPort: intstr.IntOrString{StrVal: "80"},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := translate(&tt.service, &tt.environment)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("serviceTranslate(): %+v, expected: %+v", result, tt.expected)
			}
		})
	}
}
