package ingress

import (
	"reflect"
	"testing"

	"bitbucket.org/okteto/okteto/backend/model"
	v1beta1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestTranslate(t *testing.T) {
	service := &model.Service{
		Name: "test",
		Containers: map[string]*model.Container{
			"api": &model.Container{
				Ingress: []*model.Ingress{
					&model.Ingress{
						Host: "host",
						Path: "/api/v1",
						Port: "8080",
					},
					&model.Ingress{
						Host: "host",
						Path: "/api/v2",
						Port: "8081",
					},
					&model.Ingress{
						Host: model.ProjectName,
						Path: "/",
						Port: "5000",
					},
				},
			},
		},
	}
	tests := []struct {
		name        string
		environment model.Environment
		expected    *v1beta1.Ingress
	}{
		{
			name: "minimum-config",
			environment: model.Environment{
				Name:        "environment-1",
				ProjectName: "environment",
				Provider: &model.Provider{
					Ingress: &model.IngressController{
						Domain: "test.okteto.net",
					},
				},
			},
			expected: &v1beta1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
					Annotations: map[string]string{
						"kubernetes.io/ingress.class": "nginx",
					},
				},
				Spec: v1beta1.IngressSpec{
					Rules: []v1beta1.IngressRule{
						v1beta1.IngressRule{
							Host: "host.test.okteto.net",
							IngressRuleValue: v1beta1.IngressRuleValue{
								HTTP: &v1beta1.HTTPIngressRuleValue{
									Paths: []v1beta1.HTTPIngressPath{
										v1beta1.HTTPIngressPath{
											Path: "/api/v1",
											Backend: v1beta1.IngressBackend{
												ServiceName: "test",
												ServicePort: intstr.FromInt(8080),
											},
										},
										v1beta1.HTTPIngressPath{
											Path: "/api/v2",
											Backend: v1beta1.IngressBackend{
												ServiceName: "test",
												ServicePort: intstr.FromInt(8081),
											},
										},
									},
								},
							},
						},
						v1beta1.IngressRule{
							Host: "environment.test.okteto.net",
							IngressRuleValue: v1beta1.IngressRuleValue{
								HTTP: &v1beta1.HTTPIngressRuleValue{
									Paths: []v1beta1.HTTPIngressPath{
										v1beta1.HTTPIngressPath{
											Path: "/",
											Backend: v1beta1.IngressBackend{
												ServiceName: "test",
												ServicePort: intstr.FromInt(5000),
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "all-config-fixed-certificate",
			environment: model.Environment{
				Name:        "environment-1",
				ProjectName: "environment",
				Provider: &model.Provider{
					Ingress: &model.IngressController{
						Domain: "test.okteto.net",
						Annotations: map[string]string{
							"kubernetes.io/ingress.class": "haproxy",
						},
						TLS: &model.TLS{
							Type: model.FixCertificate,
							Certificate: &model.Certificate{
								Secret:    "tls-secret",
								Namespace: "tls-namespace",
							},
						}},
				},
			},
			expected: &v1beta1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
					Annotations: map[string]string{
						"kubernetes.io/ingress.class": "haproxy",
					},
				},
				Spec: v1beta1.IngressSpec{
					Rules: []v1beta1.IngressRule{
						v1beta1.IngressRule{
							Host: "host.test.okteto.net",
							IngressRuleValue: v1beta1.IngressRuleValue{
								HTTP: &v1beta1.HTTPIngressRuleValue{
									Paths: []v1beta1.HTTPIngressPath{
										v1beta1.HTTPIngressPath{
											Path: "/api/v1",
											Backend: v1beta1.IngressBackend{
												ServiceName: "test",
												ServicePort: intstr.FromInt(8080),
											},
										},
										v1beta1.HTTPIngressPath{
											Path: "/api/v2",
											Backend: v1beta1.IngressBackend{
												ServiceName: "test",
												ServicePort: intstr.FromInt(8081),
											},
										},
									},
								},
							},
						},
						v1beta1.IngressRule{
							Host: "environment.test.okteto.net",
							IngressRuleValue: v1beta1.IngressRuleValue{
								HTTP: &v1beta1.HTTPIngressRuleValue{
									Paths: []v1beta1.HTTPIngressPath{
										v1beta1.HTTPIngressPath{
											Path: "/",
											Backend: v1beta1.IngressBackend{
												ServiceName: "test",
												ServicePort: intstr.FromInt(5000),
											},
										},
									},
								},
							},
						},
					},
					TLS: []v1beta1.IngressTLS{
						v1beta1.IngressTLS{
							SecretName: "tls-secret",
							Hosts:      []string{"host.test.okteto.net", "environment.test.okteto.net"},
						},
					},
				},
			},
		},
		{
			name: "all-config-letsencrypt",
			environment: model.Environment{
				Name:        "environment-1",
				ProjectName: "environment",
				Provider: &model.Provider{
					Ingress: &model.IngressController{
						AppendProject: true,
						Domain:        "test.okteto.net",
						Annotations: map[string]string{
							"kubernetes.io/ingress.class": "haproxy",
						},
						TLS: &model.TLS{
							Type: model.LetsEncrypt,
						}},
				},
			},
			expected: &v1beta1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
					Annotations: map[string]string{
						"kubernetes.io/ingress.class": "haproxy",
						"kubernetes.io/tls-acme":      "true",
					},
				},
				Spec: v1beta1.IngressSpec{
					Rules: []v1beta1.IngressRule{
						v1beta1.IngressRule{
							Host: "host-dot-environment-1.test.okteto.net",
							IngressRuleValue: v1beta1.IngressRuleValue{
								HTTP: &v1beta1.HTTPIngressRuleValue{
									Paths: []v1beta1.HTTPIngressPath{
										v1beta1.HTTPIngressPath{
											Path: "/api/v1",
											Backend: v1beta1.IngressBackend{
												ServiceName: "test",
												ServicePort: intstr.FromInt(8080),
											},
										},
										v1beta1.HTTPIngressPath{
											Path: "/api/v2",
											Backend: v1beta1.IngressBackend{
												ServiceName: "test",
												ServicePort: intstr.FromInt(8081),
											},
										},
									},
								},
							},
						},
						v1beta1.IngressRule{
							Host: "environment-dot-environment-1.test.okteto.net",
							IngressRuleValue: v1beta1.IngressRuleValue{
								HTTP: &v1beta1.HTTPIngressRuleValue{
									Paths: []v1beta1.HTTPIngressPath{
										v1beta1.HTTPIngressPath{
											Path: "/",
											Backend: v1beta1.IngressBackend{
												ServiceName: "test",
												ServicePort: intstr.FromInt(5000),
											},
										},
									},
								},
							},
						},
					},
					TLS: []v1beta1.IngressTLS{
						v1beta1.IngressTLS{
							SecretName: "test-letsencrypt",
							Hosts:      []string{"host-dot-environment-1.test.okteto.net", "environment-dot-environment-1.test.okteto.net"},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := translate(service, &tt.environment)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("ingressTranslate(): %+v, expected: %+v", result, tt.expected)
			}
		})
	}
}
