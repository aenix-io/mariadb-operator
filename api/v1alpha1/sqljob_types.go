package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SqlJobSpec defines the desired state of SqlJob
type SqlJobSpec struct {
	// MariaDBRef is a reference to a MariaDB object.
	// +kubebuilder:validation:Required
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	MariaDBRef MariaDBRef `json:"mariaDbRef" webhook:"inmutable"`
	// Schedule defines when the SqlJob will be executed.
	// +kubebuilder:validation:Required
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Schedule *Schedule `json:"schedule,omitempty"`
	// Username to be impersonated when executing the SqlJob.
	// +kubebuilder:validation:Required
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Username string `json:"username" webhook:"inmutable"`
	// UserPasswordSecretKeyRef is a reference to the impersonated user's password to be used when executing the SqlJob.
	// +kubebuilder:validation:Required
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	PasswordSecretKeyRef corev1.SecretKeySelector `json:"passwordSecretKeyRef" webhook:"inmutable"`
	// Username to be used when executing the SqlJob.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Database *string `json:"database,omitempty" webhook:"inmutable"`
	// DependsOn defines dependencies with other SqlJob objectecs.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	DependsOn []corev1.LocalObjectReference `json:"dependsOn,omitempty" webhook:"inmutable"`
	// Sql is the script to be executed by the SqlJob.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Sql *string `json:"sql,omitempty" webhook:"inmutable"`
	// SqlConfigMapKeyRef is a reference to a ConfigMap containing the Sql script.
	// It is defaulted to a ConfigMap with the contents of the Sql field.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	SqlConfigMapKeyRef *corev1.ConfigMapKeySelector `json:"sqlConfigMapKeyRef,omitempty" webhook:"inmutableinit"`
	// BackoffLimit defines the maximum number of attempts to successfully execute a SqlJob.
	// +optional
	// +kubebuilder:default=5
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:number"}
	BackoffLimit int32 `json:"backoffLimit,omitempty"`
	// RestartPolicy to be added to the SqlJob Pod.
	// +optional
	// +kubebuilder:default=OnFailure
	// +kubebuilder:validation:Enum=Always;OnFailure;Never
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	RestartPolicy corev1.RestartPolicy `json:"restartPolicy,omitempty" webhook:"inmutable"`
	// Resouces describes the compute resource requirements.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:resourceRequirements"}
	Resources *corev1.ResourceRequirements `json:"resources,omitempty" webhook:"inmutable"`
	// Affinity to be used in the SqlJob Pod.
	// +kubebuilder:validation:Required
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Affinity *corev1.Affinity `json:"affinity,omitempty"`
	// NodeSelector to be used in the SqlJob Pod.
	// +kubebuilder:validation:Required
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	// Tolerations to be used in the SqlJob Pod.
	// +kubebuilder:validation:Required
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
	// SecurityContext holds security configuration that will be applied to a container.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	SecurityContext *corev1.SecurityContext `json:"securityContext,omitempty"`
	// SecurityContext holds pod-level security attributes and common container settings.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	PodSecurityContext *corev1.PodSecurityContext `json:"podSecurityContext,omitempty"`
}

// SqlJobStatus defines the observed state of SqlJob
type SqlJobStatus struct {
	// Conditions for the SqlJob object.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=status,xDescriptors={"urn:alm:descriptor:io.kubernetes.conditions"}
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

func (s *SqlJobStatus) SetCondition(condition metav1.Condition) {
	if s.Conditions == nil {
		s.Conditions = make([]metav1.Condition, 0)
	}
	meta.SetStatusCondition(&s.Conditions, condition)
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=smdb
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Complete",type="string",JSONPath=".status.conditions[?(@.type==\"Complete\")].status"
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.conditions[?(@.type==\"Complete\")].message"
// +kubebuilder:printcolumn:name="MariaDB",type="string",JSONPath=".spec.mariaDbRef.name"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +operator-sdk:csv:customresourcedefinitions:resources={{SqlJob,v1alpha1},{ConfigMap,v1},{CronJob,v1},{Job,v1}}

// SqlJob is the Schema for the sqljobs API. It is used to run sql scripts as jobs.
type SqlJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SqlJobSpec   `json:"spec,omitempty"`
	Status SqlJobStatus `json:"status,omitempty"`
}

func (s *SqlJob) IsComplete() bool {
	return meta.IsStatusConditionTrue(s.Status.Conditions, ConditionTypeComplete)
}

//+kubebuilder:object:root=true

// SqlJobList contains a list of SqlJob
type SqlJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SqlJob `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SqlJob{}, &SqlJobList{})
}
