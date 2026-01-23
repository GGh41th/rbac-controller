/*
Copyright 2025 Ghaith Gtari.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	rbaccontrollerv1alpha1 "github.com/GGh41th/rbac-controller/api/v1alpha1"
)

const (
	DEFAULT_NAMESPACE = "default"
)

// nolint:unused
// log is for logging in this package.
var rbacrulelog = logf.Log.WithName("rbacrule-resource")

// SetupRBACRuleWebhookWithManager registers the webhook for RBACRule in the manager.
func SetupRBACRuleWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&rbaccontrollerv1alpha1.RBACRule{}).
		WithValidator(&RBACRuleCustomValidator{}).
		WithDefaulter(&RBACRuleCustomDefaulter{}).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-rbac-controller-ggh41th-io-v1alpha1-rbacrule,mutating=true,failurePolicy=fail,sideEffects=None,groups=rbac-controller.ggh41th.io,resources=rbacrules,verbs=create;update,versions=v1alpha1,name=mrbacrule-v1alpha1.kb.io,admissionReviewVersions=v1

type RBACRuleCustomDefaulter struct {
}

var _ webhook.CustomDefaulter = &RBACRuleCustomDefaulter{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind RBACRule.
func (d *RBACRuleCustomDefaulter) Default(_ context.Context, obj runtime.Object) error {
	rbacrule, ok := obj.(*rbaccontrollerv1alpha1.RBACRule)

	if !ok {
		return fmt.Errorf("expected an RBACRule object but got %T", obj)
	}
	rbacrulelog.Info("Defaulting for RBACRule", "name", rbacrule.GetName())

	if rbacrule.Spec.Bindings != nil {
		// we need to change the actual Bindings struct , we should do it this
		// way , ignore the linter.
		for i, _ := range rbacrule.Spec.Bindings {
			defaultSubjectsNs(rbacrule.Spec.Bindings[i].Subjects)
			defaultRolesNS(rbacrule.Spec.Bindings[i].RoleBindings)
		}
	}

	return nil
}
func defaultSubjectsNs(subjs []rbaccontrollerv1alpha1.Subject) {
	for i, _ := range subjs {
		if subjs[i].Kind == rbaccontrollerv1alpha1.ServiceAccount && len(subjs[i].Namespaces) == 0 && len(subjs[i].NamespaceMatchExpression) == 0 && reflect.ValueOf(subjs[i].NameSpaceSelector).IsZero() {
			subjs[i].Namespaces = []string{DEFAULT_NAMESPACE}
		}
	}
}

func defaultRolesNS(rbs []rbaccontrollerv1alpha1.RoleBinding) {
	for i, _ := range rbs {
		if rbs[i].Role != "" && len(rbs[i].Namespaces) == 0 && len(rbs[i].NamespaceMatchExpression) == 0 && reflect.ValueOf(rbs[i].NameSpaceSelector).IsZero() {
			rbs[i].Namespaces = []string{DEFAULT_NAMESPACE}
		}
	}
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: If you want to customise the 'path', use the flags '--defaulting-path' or '--validation-path'.
// +kubebuilder:webhook:path=/validate-rbac-controller-ggh41th-io-v1alpha1-rbacrule,mutating=false,failurePolicy=fail,sideEffects=None,groups=rbac-controller.ggh41th.io,resources=rbacrules,verbs=create;update,versions=v1alpha1,name=vrbacrule-v1alpha1.kb.io,admissionReviewVersions=v1

// RBACRuleCustomValidator struct is responsible for validating the RBACRule resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type RBACRuleCustomValidator struct {
	// TODO(user): Add more fields as needed for validation
}

var _ webhook.CustomValidator = &RBACRuleCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type RBACRule.
func (v *RBACRuleCustomValidator) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	rbacrule, ok := obj.(*rbaccontrollerv1alpha1.RBACRule)
	if !ok {
		return nil, fmt.Errorf("expected a RBACRule object but got %T", obj)
	}
	rbacrulelog.Info("Validation for RBACRule upon creation", "name", rbacrule.GetName())

	if time.Now().After(rbacrule.Spec.StartTime.Time) {
		return nil, fmt.Errorf("start time should not be earlier than now")
	}

	if rbacrule.Spec.StartTime.Time.After(rbacrule.Spec.EndTime.Time) {
		return nil, fmt.Errorf("start time should not be higher than end time")
	}

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type RBACRule.
func (v *RBACRuleCustomValidator) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	rbacrule, ok := newObj.(*rbaccontrollerv1alpha1.RBACRule)
	if !ok {
		return nil, fmt.Errorf("expected a RBACRule object for the newObj but got %T", newObj)
	}
	rbacrulelog.Info("Validation for RBACRule upon update", "name", rbacrule.GetName())

	// TODO(user): fill in your validation logic upon object update.
	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type RBACRule.
func (v *RBACRuleCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	rbacrule, ok := obj.(*rbaccontrollerv1alpha1.RBACRule)
	if !ok {
		return nil, fmt.Errorf("expected a RBACRule object but got %T", obj)
	}
	rbacrulelog.Info("Validation for RBACRule upon deletion", "name", rbacrule.GetName())

	// TODO(user): fill in your validation logic upon object deletion.

	return nil, nil
}
