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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:validation:Enum=User;Group;ServiceAccount
type SubjectType string

var (
	User           SubjectType = "User"
	Group          SubjectType = "Group"
	ServiceAccount SubjectType = "ServiceAccount"
)

// +kubebuilder:validation:XValidation:rule="(has(self.namespaces) || has(self.nameSpaceSelector) || has(self.namespaceMatchExpression))",message="at least one namespace must be specified"
type Subject struct {
	// +required
	Kind SubjectType `json:"kind"`
	// +required
	Name string `json:"name"`
	// +optional
	Namespaces []string `json:"namespaces,omitempty"`
	// +optional
	NameSpaceSelector metav1.LabelSelector `json:"nameSpaceSelector,omitempty"`
	// +optional
	NamespaceMatchExpression string `json:"namespaceMatchExpression,omitempty"`
}

// +kubebuilder:validation:XValidation:rule="(has(self.namespaces) || has(self.nameSpaceSelector) || has(self.namespaceMatchExpression))",message="at least one namespace must be specified"
// +kubebuilder:validation:XValidation:rule="(has(self.role) || has(self.clusterRole))",message="at least one role must be specified"
type RoleBinding struct {
	// +optional
	Role string `json:"role,omitempty"`
	// +optional
	ClusterRole string `json:"clusterRole,omitempty"`
	// +optional
	Namespaces []string `json:"namespaces,omitempty"`
	// +optional
	NameSpaceSelector metav1.LabelSelector `json:"nameSpaceSelector,omitempty"`
	// +optional
	NamespaceMatchExpression string `json:"namespaceMatchExpression,omitempty"`
}

type ClusterRoleBinding struct {
	// +required
	ClusterRole string `json:"clusterRole"`
}

// +kubebuilder:validation:XValidation:rule="(has(self.roleBindings) || has(self.clusterRoleBindings))",message="RoleBindings or ClusterRoleBindings should be specified"
type Binding struct {
	// +required
	Name string `json:"name"`
	// +required
	Subjects []Subject `json:"subjects"`
	// +optional
	RoleBindings []RoleBinding `json:"roleBindings,omitempty"`
	// +optional
	ClusterRoleBindings []ClusterRoleBinding `json:"clusterRoleBindings,omitempty"`
}

// RBACRuleSpec defines the desired state of RBACRule
type RBACRuleSpec struct {
	// +required
	Bindings []Binding `json:"bindings"`
	// If defined it will apply to all bindings. Specifying it at individual
	// binding will override it.
	// +optional
	// +kubebuilder:validation:Format="date-time"
	StartTime metav1.Time `json:"startTime,omitempty,omitzero"`
	// If defined it will apply to all bindings. Specifying it at individual
	// binding will override it.
	// +optional
	// +kubebuilder:validation:Format="date-time"
	EndTime metav1.Time `json:"endTime,omitempty,omitzero"`
	//If specified it controls wether inexistant SAs will be created or no.
	// +optional
	// +default:value=true
	CreateSA bool `json:"createSA"`
}

// RBACRuleStatus defines the observed state of RBACRule.
type RBACRuleStatus struct {
	// conditions represent the current state of the RBACRule resource.
	// Each condition has a unique type and reflects the status of a specific aspect of the resource.
	//
	// Standard condition types include:
	// - "Available": the resource is fully functional
	// - "Progressing": the resource is being created or updated
	// - "Degraded": the resource failed to reach or maintain its desired state
	//
	// The status of each condition is one of True, False, or Unknown.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// A list of the established role bindings , in the form of Role/Namespace.
	// +listType=atomic
	// +optional
	RoleBindings []string `json:"roleBindings,omitempty"`
	// +listType=atomic
	// +optional
	// A list of the established cluster role bindings.
	ClusterRoleBindings []string `json:"clusterRoleBindings,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster

// RBACRule is the Schema for the rbacrules API
type RBACRule struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of RBACRule
	// +required
	Spec RBACRuleSpec `json:"spec"`

	// status defines the observed state of RBACRule
	// +optional
	Status RBACRuleStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// RBACRuleList contains a list of RBACRule
type RBACRuleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []RBACRule `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RBACRule{}, &RBACRuleList{})
}
