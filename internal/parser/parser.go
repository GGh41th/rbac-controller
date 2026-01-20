package parser

import (
	"context"
	"fmt"

	rbaccontrollerv1 "github.com/GGh41th/rbac-controller/api/v1alpha1"
	"github.com/GGh41th/rbac-controller/internal/utils"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	RBACApiGroup = "rbac.authorization.k8s.io"
	CRB          = "ClusterRole"
	RB           = "Role"
)

type Parser struct {
	client.Client
	Subjects            []rbacv1.Subject
	RoleBindings        []rbacv1.RoleBinding
	ClusterRoleBindings []rbacv1.ClusterRoleBinding
}

func (p *Parser) Parse(ctx context.Context, binding *rbaccontrollerv1.Binding, RBACLabels map[string]string, ownerRef []metav1.OwnerReference, RBACRuleName string) error {
	//we start by parsing the subjects contained in the binding
	if len(binding.Subjects) > 0 {
		err := p.parseSubjects(ctx, binding.Subjects, RBACLabels, ownerRef)
		if err != nil {
			return err
		}
	}
	// we build clusterrolebindings based on the RoleBindings field and the subjects
	// extracted earlier

	if len(binding.ClusterRoleBindings) > 0 {
		p.parseCRBs(RBACRuleName, binding.Name, binding.ClusterRoleBindings, RBACLabels, ownerRef)
	}
	if len(binding.RoleBindings) > 0 {
		if err := p.parseRBs(ctx, RBACRuleName, binding.Name, binding.RoleBindings, RBACLabels, ownerRef); err != nil {
			return err
		}
	}
	return nil
}

func (p *Parser) parseSubjects(ctx context.Context, subjects []rbaccontrollerv1.Subject, RBACLabels map[string]string, ownerRef []metav1.OwnerReference) error {
	for _, s := range subjects {
		switch s.Kind {
		case rbaccontrollerv1.User:
			{
				p.Subjects = append(p.Subjects, rbacv1.Subject{
					APIGroup:  RBACApiGroup,
					Kind:      string(rbaccontrollerv1.User),
					Name:      s.Name,
					Namespace: "",
				})
			}
		case rbaccontrollerv1.Group:
			{
				p.Subjects = append(p.Subjects, rbacv1.Subject{
					APIGroup:  RBACApiGroup,
					Kind:      string(rbaccontrollerv1.Group),
					Name:      s.Name,
					Namespace: "",
				})
			}
		case rbaccontrollerv1.ServiceAccount:
			{
				ns, err := p.retrieveNamespaces(ctx, &s.NameSpaceSelector)
				ns = append(ns, s.Namespaces...)
				if err != nil {
					return err
				}
				for _, n := range ns {
					p.Subjects = append(p.Subjects, rbacv1.Subject{
						APIGroup:  "",
						Kind:      string(rbaccontrollerv1.ServiceAccount),
						Name:      s.Name,
						Namespace: n,
					})
				}
			}
		}
	}
	return nil
}

func (p *Parser) parseCRBs(RBACRuleName, BindingName string, CRBs []rbaccontrollerv1.ClusterRoleBinding, RBACLabels map[string]string, ownerRef []metav1.OwnerReference) {
	for _, crb := range CRBs {
		p.ClusterRoleBindings = append(p.ClusterRoleBindings, rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:            utils.GenerateName(RBACRuleName, BindingName, CRB, crb.ClusterRole),
				Labels:          RBACLabels,
				OwnerReferences: ownerRef,
			},
			Subjects: p.Subjects,
			RoleRef: rbacv1.RoleRef{
				APIGroup: RBACApiGroup,
				Kind:     CRB,
				Name:     crb.ClusterRole,
			},
		})
	}
}

func (p *Parser) parseRBs(ctx context.Context, RBACRuleName, BindingName string, RBs []rbaccontrollerv1.RoleBinding, RBAClabels map[string]string, ownerRef []metav1.OwnerReference) error {
	for _, rb := range RBs {
		ns, err := p.retrieveNamespaces(ctx, &rb.NameSpaceSelector)
		ns = append(ns, rb.Namespaces...)
		if err != nil {
			return err
		}
		if rb.ClusterRole != "" {
			for _, n := range ns {
				p.RoleBindings = append(p.RoleBindings, rbacv1.RoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name:            utils.GenerateName(RBACRuleName, BindingName, RB, rb.ClusterRole),
						Namespace:       n,
						Labels:          RBAClabels,
						OwnerReferences: ownerRef,
					},
					Subjects: p.Subjects,
					RoleRef: rbacv1.RoleRef{
						APIGroup: RBACApiGroup,
						Kind:     CRB,
						Name:     rb.ClusterRole,
					},
				})
			}
		}
		if rb.Role != "" {
			for _, n := range ns {
				p.RoleBindings = append(p.RoleBindings, rbacv1.RoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name:            utils.GenerateName(RBACRuleName, BindingName, RB, rb.Role),
						Namespace:       n,
						Labels:          RBAClabels,
						OwnerReferences: ownerRef,
					},
					Subjects: p.Subjects,
					RoleRef: rbacv1.RoleRef{
						APIGroup: RBACApiGroup,
						Kind:     RB,
						Name:     rb.Role,
					},
				})
			}

		}
	}
	return nil
}

func (p *Parser) retrieveNamespaces(ctx context.Context, ls *metav1.LabelSelector) ([]string, error) {
	nsMetaData := &metav1.PartialObjectMetadataList{}
	nsMetaData.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "Namespace",
	})
	if len(ls.MatchExpressions) > 0 || ls.MatchLabels != nil {
		selector, err := metav1.LabelSelectorAsSelector(ls)
		if err != nil {
			return nil, fmt.Errorf("failed to extract a selector from the label selector %w", err)
		}
		if err := p.List(ctx, nsMetaData, &client.ListOptions{
			LabelSelector: selector,
		}); err != nil {
			return nil, fmt.Errorf("failed to list namespaces metadata %w", err)
		}
	}
	ns := []string{}
	for _, i := range nsMetaData.Items {
		ns = append(ns, i.Name)
	}
	return ns, nil
}
