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

package controller

import (
	"context"
	"slices"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	log "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	rbaccontrollerv1 "github.com/GGh41th/rbac-controller/api/v1alpha1"
	"github.com/GGh41th/rbac-controller/internal/constants"
	"github.com/GGh41th/rbac-controller/internal/parser"
	"github.com/go-logr/logr"
)

const (
	RBACRuleFinalizer = "rbac-controller.io/cleanup-rbac-rule"
	ControllerName    = "RBACRule-controller"
)

// RBACRuleReconciler reconciles a RBACRule object
type RBACRuleReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

// +kubebuilder:rbac:groups=rbac-controller.ggh41th.io,resources=rbacrules,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac-controller.ggh41th.io,resources=rbacrules/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=rbac-controller.ggh41th.io,resources=rbacrules/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles,verbs=bind
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles,verbs=bind
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterrolebindings,verbs=get;list;watch;create;update;patch;delete

func (r *RBACRuleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	RBACRule := &rbaccontrollerv1.RBACRule{}
	err := r.Get(ctx, req.NamespacedName, RBACRule)
	if err != nil {
		if apierrors.IsNotFound(err) {
			r.Log.Info("Rule might been deleted")
			return ctrl.Result{}, nil
		}
		// error trying to get the rule , requeue the request
		return ctrl.Result{}, err
	}

	if RBACRule.GetDeletionTimestamp() == nil && !controllerutil.ContainsFinalizer(RBACRule, RBACRuleFinalizer) {
		controllerutil.AddFinalizer(RBACRule, RBACRuleFinalizer)
		if err := r.Update(ctx, RBACRule); err != nil {
			r.Log.Error(err, "failed to add finalizer")
			return ctrl.Result{}, err
		}
	}

	// Handle deletion: If Rule is marked for deletion , delete all assoicated ressources
	if RBACRule.GetDeletionTimestamp() != nil {
		return ctrl.Result{}, r.reconcileDelete(ctx, RBACRule)
	}

	start := RBACRule.Spec.StartTime.Time
	if start != (time.Time{}) && start.After(time.Now()) {
		period := time.Until(start)
		r.Log.Info("Rule shouldn't be active yet , waiting for start time", "Wait Period", period)
		return ctrl.Result{RequeueAfter: period}, nil
	}

	if RBACRule.Spec.Bindings != nil {
		RBAClabels := map[string]string{constants.RBACRuleLabel: RBACRule.Name}
		ownerRef := []metav1.OwnerReference{
			*metav1.NewControllerRef(RBACRule, rbaccontrollerv1.GroupVersion.WithKind("RBACRule")),
		}
		for _, b := range RBACRule.Spec.Bindings {
			p := &parser.Parser{
				Client: r.Client,
			}
			if err := p.Parse(ctx, &b, RBAClabels, ownerRef, RBACRule.Name); err != nil {
				r.Log.Error(err, "failed to parse RBACBinding")
			}
			for _, s := range p.Subjects {
				if s.Kind == string(rbaccontrollerv1.ServiceAccount) {
					if err := r.checkNamespace(ctx, s.Namespace, ownerRef); err != nil {
						r.Log.Error(err, "Failed to create namespace", "namespace", s.Namespace)
						return reconcile.Result{RequeueAfter: 500 * time.Millisecond}, nil
					}
					err = r.createSA(ctx, s.Name, s.Namespace, RBAClabels, ownerRef)
					if err != nil {
						r.Log.Error(err, "Failed to create SA", "name", s.Name, "namespace", s.Namespace)
						return reconcile.Result{RequeueAfter: 500 * time.Millisecond}, nil
					}
				}
			}

			for _, crb := range p.ClusterRoleBindings {
				if err := r.createCRB(ctx, &crb); err != nil {
					r.Log.Error(err, "Failed to create CRB", "name", crb.Name)
					return reconcile.Result{RequeueAfter: 500 * time.Millisecond}, nil
				}
				if slices.Index(RBACRule.Status.ClusterRoleBindings, crb.Name) == -1 {
					RBACRule.Status.ClusterRoleBindings = append(RBACRule.Status.ClusterRoleBindings, crb.Name)
					if err := r.Status().Update(ctx, RBACRule); err != nil {
						r.Log.Error(err, "Failed to update RBACRule status", "CRB", crb.Name)
						return ctrl.Result{}, err
					}
				}

			}

			for _, rb := range p.RoleBindings {
				if err := r.createCR(ctx, &rb); err != nil {
					r.Log.Error(err, "Failed to create RB", "name", rb.Name)
					return reconcile.Result{RequeueAfter: 500 * time.Millisecond}, err
				}
				if slices.Index(RBACRule.Status.RoleBindings, rb.Namespace+"/"+rb.Name) == -1 {
					RBACRule.Status.RoleBindings = append(RBACRule.Status.RoleBindings, rb.Namespace+"/"+rb.Name)
					if err := r.Status().Update(ctx, RBACRule); err != nil {
						r.Log.Error(err, "Failed to update RBACRule status", "CR", rb.Name)
						return ctrl.Result{}, err
					}
				}
			}
		}
	}
	end := RBACRule.Spec.EndTime.Time
	if end != (time.Time{}) && end.After(time.Now()) {
		period := time.Until(end)
		r.Log.Info("Rule will be scheduled for deletion", "Time until deletion", period)
		return ctrl.Result{RequeueAfter: period}, nil
	} else if end.Before(time.Now()) {
		err := r.Delete(ctx, RBACRule)
		if err != nil {
			r.Log.Error(err, "error deleting resource")
			return ctrl.Result{}, nil
		}
	}
	return ctrl.Result{}, nil
}

func (r *RBACRuleReconciler) checkNamespace(ctx context.Context, name string, ownerRef []metav1.OwnerReference) error {
	nsName := types.NamespacedName{Namespace: "", Name: name}
	ns := &corev1.Namespace{}
	// we check if the ns exist , if not we create it
	if err := r.Get(ctx, nsName, ns); err != nil {
		if apierrors.IsNotFound(err) {
			ns.ObjectMeta = metav1.ObjectMeta{
				Name:            name,
				OwnerReferences: ownerRef,
			}
			if err := r.Create(ctx, ns); err != nil {
				return err
			}
			return nil
		}
		return err
	}
	return nil
}

func (r *RBACRuleReconciler) createSA(ctx context.Context, name string, ns string, RBACLAbel map[string]string, ownerRef []metav1.OwnerReference) error {
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       ns,
			Labels:          RBACLAbel,
			OwnerReferences: ownerRef,
		},
	}
	if err := r.Create(ctx, sa); err != nil {
		if apierrors.IsAlreadyExists(err) {
			if err := r.Update(ctx, sa); err != nil {
				return err
			}
			return nil
		}
		return err
	}
	return nil
}

func (r *RBACRuleReconciler) createCRB(ctx context.Context, crb *rbacv1.ClusterRoleBinding) error {
	// TODO: I really hate how this looks , change it.
	if err := r.Create(ctx, crb); err != nil {
		if apierrors.IsAlreadyExists(err) {
			if err = r.Update(ctx, crb); err != nil {
				return err
			}
			return nil
		}
		return err
	}
	return nil
}

func (r *RBACRuleReconciler) createCR(ctx context.Context, cr *rbacv1.RoleBinding) error {
	if err := r.Create(ctx, cr); err != nil {
		if apierrors.IsAlreadyExists(err) {
			if err = r.Update(ctx, cr); err != nil {
				return err
			}
			return nil
		}
		return err
	}
	return nil
}

func (r *RBACRuleReconciler) reconcileDelete(ctx context.Context, RBACRule *rbaccontrollerv1.RBACRule) error {
	r.Log.Info("Deleting RBACRule", "Name", RBACRule.Name, "Namespace", RBACRule.Namespace)
	if controllerutil.ContainsFinalizer(RBACRule, RBACRuleFinalizer) {
		ls := labels.SelectorFromSet(map[string]string{constants.RBACRuleLabel: strings.Join([]string{RBACRule.Name, RBACRule.Namespace}, "-")})
		if err := r.deleteBindings(ctx, RBACRule, ls); err != nil {
			r.Log.Error(err, "failed to delete bindings")
			return err
		}
		if err := r.deleteServiceAccounts(ctx, ls); err != nil {
			r.Log.Error(err, "failed to delete ServiceAccounts")
			return err
		}
	}
	controllerutil.RemoveFinalizer(RBACRule, RBACRuleFinalizer)
	if err := r.Update(ctx, RBACRule); err != nil {
		r.Log.Error(err, "failed to remove finalizer from RBACRule")
		return err
	}
	return nil

}

func (r *RBACRuleReconciler) deleteBindings(ctx context.Context, RBACRule *rbaccontrollerv1.RBACRule, ls labels.Selector) error {
	if len(RBACRule.Status.RoleBindings) > 0 {
		rbs := rbacv1.RoleBindingList{}
		if err := r.List(ctx, &rbs, &client.ListOptions{
			LabelSelector: ls,
		}); err != nil {
			r.Log.Error(err, "failed to list role bindings")
			return err
		}
		for _, rb := range rbs.Items {
			if err := r.Delete(ctx, &rb); err != nil {
				r.Log.Error(err, "failed to delete roleBinding", "name", rb.Name, "namespace", rb.Namespace)
				return err
			}
			i := slices.Index(RBACRule.Status.RoleBindings, rb.Name)
			RBACRule.Status.RoleBindings = slices.Delete(RBACRule.Status.RoleBindings, i, i)
			if err := r.Update(ctx, RBACRule); err != nil {
				r.Log.Error(err, "failed to remove role binding from status", "name", rb.Name, "namepsace", rb.Namespace)
				return err
			}
		}
	}
	if len(RBACRule.Status.ClusterRoleBindings) > 0 {
		crbs := rbacv1.ClusterRoleBindingList{}

		if err := r.List(ctx, &crbs, &client.ListOptions{
			LabelSelector: ls,
		}); err != nil {
			r.Log.Error(err, "failed to list role bindings")
			return err
		}
		for _, crb := range crbs.Items {
			if err := r.Delete(ctx, &crb); err != nil {
				r.Log.Error(err, "failed to delete clusterRoleBinding", "name", crb.Name)
				return err
			}
			i := slices.Index(RBACRule.Status.ClusterRoleBindings, crb.Name)
			RBACRule.Status.ClusterRoleBindings = slices.Delete(RBACRule.Status.ClusterRoleBindings, i, i)
			if err := r.Update(ctx, RBACRule); err != nil {
				r.Log.Error(err, "failed to remove cluster role binding from status", "name", crb.Name)
				return err
			}
		}
	}

	return nil
}

func (r *RBACRuleReconciler) deleteServiceAccounts(ctx context.Context, ls labels.Selector) error {
	log := log.FromContext(ctx)

	sas := corev1.ServiceAccountList{}
	if err := r.List(ctx, &sas, &client.ListOptions{
		LabelSelector: ls,
	}); err != nil {
		log.Error(err, "error listing Rule's serviceaccounts")
		return err
	}

	for _, sa := range sas.Items {
		if err := r.Delete(ctx, &sa); err != nil {
			if !apierrors.IsNotFound(err) {
				r.Log.Error(err, "failed to delete service account", "name", sa.Name, "namespace", sa.Namespace)
				return err
			}
		}
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RBACRuleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&rbaccontrollerv1.RBACRule{}).
		Owns(&corev1.ServiceAccount{}).     //Watches SAs owned by the rbac-rule controller
		Owns(&rbacv1.RoleBinding{}).        //Watches RBs owned by the rbac-rule controller
		Owns(&rbacv1.ClusterRoleBinding{}). //Watches CRBs owned by the rbac-rule controller
		Owns(&corev1.Namespace{}).          //Watches NSs owned by the rbac-rule controller
		Named(ControllerName).
		Complete(r)
}
