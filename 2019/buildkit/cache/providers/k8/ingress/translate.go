package ingress

import (
	"strconv"

	"bitbucket.org/okteto/okteto/backend/model"
	v1beta1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func translate(s *model.Service, e *model.Environment) *v1beta1.Ingress {
	ingressName := s.Name
	serviceName := s.Name
	ingress := &v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        ingressName,
			Annotations: e.Provider.Ingress.Annotations,
		},
		Spec: v1beta1.IngressSpec{
			Rules: []v1beta1.IngressRule{},
		},
	}
	if ingress.Annotations == nil {
		ingress.Annotations = map[string]string{
			"kubernetes.io/ingress.class": "nginx",
		}
	}
	if e.Provider.IsTLSIngress() && e.Provider.Ingress.TLS.Type == model.LetsEncrypt {
		ingress.Annotations["kubernetes.io/tls-acme"] = "true"
	}
	for _, hostname := range s.GetIngressHostnames(e) {
		ir := v1beta1.IngressRule{
			Host: hostname,
			IngressRuleValue: v1beta1.IngressRuleValue{
				HTTP: &v1beta1.HTTPIngressRuleValue{
					Paths: []v1beta1.HTTPIngressPath{},
				},
			},
		}
		ingress.Spec.Rules = append(ingress.Spec.Rules, ir)
		for _, i := range s.GetIngressRules(true) {
			if i.GetIngressHostname(e) == hostname {
				portNumber, _ := strconv.Atoi(i.Port)
				ir.IngressRuleValue.HTTP.Paths = append(
					ir.IngressRuleValue.HTTP.Paths,
					v1beta1.HTTPIngressPath{
						Path: i.Path,
						Backend: v1beta1.IngressBackend{
							ServiceName: serviceName,
							ServicePort: intstr.FromInt(portNumber),
						},
					},
				)
			}
		}
	}
	if e.Provider.IsTLSIngress() {
		ingress.Spec.TLS = []v1beta1.IngressTLS{
			v1beta1.IngressTLS{
				SecretName: s.GetIngressCertificateName(e),
				Hosts:      s.GetIngressHostnames(e),
			},
		}
	}
	return ingress
}
