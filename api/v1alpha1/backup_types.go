package v1alpha1

import (
	"errors"
	"fmt"
	"reflect"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BackupStorage defines the storage for a Backup.
type BackupStorage struct {
	// S3 defines the configuration to store backups in a S3 compatible storage.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	S3 *S3 `json:"s3,omitempty"`
	// PersistentVolumeClaim is a Kubernetes PVC specification.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	PersistentVolumeClaim *corev1.PersistentVolumeClaimSpec `json:"persistentVolumeClaim,omitempty"`
	// Volume is a Kubernetes volume specification.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Volume *corev1.VolumeSource `json:"volume,omitempty"`
}

func (b *BackupStorage) Validate() error {
	storageTypes := 0
	fields := reflect.ValueOf(b).Elem()
	for i := 0; i < fields.NumField(); i++ {
		field := fields.Field(i)
		if !field.IsNil() {
			storageTypes++
		}
	}
	if storageTypes != 1 {
		return errors.New("exactly one storage type should be provided")
	}
	return nil
}

// BackupSpec defines the desired state of Backup
type BackupSpec struct {
	// MariaDBRef is a reference to a MariaDB object.
	// +kubebuilder:validation:Required
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	MariaDBRef MariaDBRef `json:"mariaDbRef" webhook:"inmutable"`
	// Storage to be used in the Backup.
	// +kubebuilder:validation:Required
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Storage BackupStorage `json:"storage" webhook:"inmutable"`
	// Args to be used in the Backup container.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Args []string `json:"args,omitempty"`
	// Schedule defines when the Backup will be taken.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Schedule *Schedule `json:"schedule,omitempty"`
	// MaxRetention defines the retention policy for backups. Old backups will be cleaned up by the Backup Job.
	// It defaults to 30 days.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	MaxRetention metav1.Duration `json:"maxRetention,omitempty" webhook:"inmutableinit"`
	// LogLevel to be used n the Backup Job. It defaults to 'info'.
	// +optional
	// +kubebuilder:default=info
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	LogLevel string `json:"logLevel,omitempty"`
	// BackoffLimit defines the maximum number of attempts to successfully take a Backup.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:number"}
	BackoffLimit int32 `json:"backoffLimit,omitempty"`
	// RestartPolicy to be added to the Backup Pod.
	// +optional
	// +kubebuilder:default=OnFailure
	// +kubebuilder:validation:Enum=Always;OnFailure;Never
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	RestartPolicy corev1.RestartPolicy `json:"restartPolicy,omitempty" webhook:"inmutable"`
	// Resouces describes the compute resource requirements.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:resourceRequirements"}
	Resources *corev1.ResourceRequirements `json:"resources,omitempty" webhook:"inmutable"`
	// Affinity to be used in the Backup Pod.
	// +kubebuilder:validation:Required
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Affinity *corev1.Affinity `json:"affinity,omitempty"`
	// NodeSelector to be used in the Backup Pod.
	// +kubebuilder:validation:Required
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	// Tolerations to be used in the Backup Pod.
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

// BackupStatus defines the observed state of Backup
type BackupStatus struct {
	// Conditions for the Backup object.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=status,xDescriptors={"urn:alm:descriptor:io.kubernetes.conditions"}
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

func (b *BackupStatus) SetCondition(condition metav1.Condition) {
	if b.Conditions == nil {
		b.Conditions = make([]metav1.Condition, 0)
	}
	meta.SetStatusCondition(&b.Conditions, condition)
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=bmdb
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Complete",type="string",JSONPath=".status.conditions[?(@.type==\"Complete\")].status"
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.conditions[?(@.type==\"Complete\")].message"
// +kubebuilder:printcolumn:name="MariaDB",type="string",JSONPath=".spec.mariaDbRef.name"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +operator-sdk:csv:customresourcedefinitions:resources={{Backup,v1alpha1},{CronJob,v1},{Job,v1},{PersistentVolumeClaim,v1}}

// Backup is the Schema for the backups API. It is used to define backup jobs and its storage.
type Backup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BackupSpec   `json:"spec,omitempty"`
	Status BackupStatus `json:"status,omitempty"`
}

func (b *Backup) IsComplete() bool {
	return meta.IsStatusConditionTrue(b.Status.Conditions, ConditionTypeComplete)
}

func (b *Backup) Validate() error {
	if b.Spec.Schedule != nil {
		if err := b.Spec.Schedule.Validate(); err != nil {
			return fmt.Errorf("invalid Schedule: %v", err)
		}
	}
	if err := b.Spec.Storage.Validate(); err != nil {
		return fmt.Errorf("invalid Storage: %v", err)
	}
	return nil
}

func (b *Backup) SetDefaults() {
	if b.Spec.MaxRetention == (metav1.Duration{}) {
		b.Spec.MaxRetention = metav1.Duration{Duration: 30 * 24 * time.Hour}
	}
	if b.Spec.BackoffLimit == 0 {
		b.Spec.BackoffLimit = 5
	}
}

func (b *Backup) Volume() (*corev1.VolumeSource, error) {
	if b.Spec.Storage.S3 != nil {
		return &corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		}, nil
	}
	if b.Spec.Storage.PersistentVolumeClaim != nil {
		return &corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: b.Name,
			},
		}, nil
	}
	if b.Spec.Storage.Volume != nil {
		return b.Spec.Storage.Volume, nil
	}
	return nil, errors.New("unable to get volume for Backup")
}

// +kubebuilder:object:root=true

// BackupList contains a list of Backup
type BackupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Backup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Backup{}, &BackupList{})
}
