package user

import (
	"bitbucket.org/okteto/okteto/backend/model"
	apiv1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func translateServiceAccount(e *model.Environment) *apiv1.ServiceAccount {
	return &apiv1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      DevName(e),
			Namespace: e.Name,
		},
	}
}

func translateRole(e *model.Environment) *rbacv1.Role {
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      DevName(e),
			Namespace: e.Name,
		},
		Rules: []rbacv1.PolicyRule{
			rbacv1.PolicyRule{
				APIGroups: []string{""},
				Resources: []string{"pods", "pods/log"},
				Verbs:     []string{"get", "list"},
			},
			rbacv1.PolicyRule{
				APIGroups: []string{""},
				Resources: []string{"pods/exec"},
				Verbs:     []string{"create"},
			},
			rbacv1.PolicyRule{
				APIGroups: []string{""},
				Resources: []string{"pods/portforward"},
				Verbs:     []string{"create"},
			},
		},
	}
}

func translateRoleBinding(e *model.Environment) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      DevName(e),
			Namespace: e.Name,
		},
		Subjects: []rbacv1.Subject{
			rbacv1.Subject{
				Kind:      "ServiceAccount",
				Name:      DevName(e),
				Namespace: e.Name,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     DevName(e),
		},
	}
}
