package v1alpha1

import (
	"reflect"

	"github.com/mariadb-operator/mariadb-operator/pkg/environment"
	"github.com/mariadb-operator/mariadb-operator/pkg/statefulset"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

// InheritMetadata defines the metadata to be inherited by children resources.
type InheritMetadata struct {
	// Labels to be added to children resources.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Labels map[string]string `json:"labels,omitempty"`
	// Annotations to be added to children resources.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Annotations map[string]string `json:"annotations,omitempty"`
}

// Exporter defines a metrics exporter container.
type Exporter struct {
	// ContainerTemplate defines a template to configure Container objects.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	ContainerTemplate `json:",inline"`
	// Image name to be used as metrics exporter. The supported format is `<image>:<tag>`.
	// Only mysqld-exporter >= v0.15.0 is supported: https://github.com/prometheus/mysqld_exporter
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Image string `json:"image,omitempty"`
	// ImagePullPolicy is the image pull policy. One of `Always`, `Never` or `IfNotPresent`. If not defined, it defaults to `IfNotPresent`.
	// +optional
	// +kubebuilder:validation:Enum=Always;Never;IfNotPresent
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:imagePullPolicy"}
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`
	// Port where the exporter will be listening for connections.
	// +optional
	// +kubebuilder:default=9104
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:number"}
	Port int32 `json:"port,omitempty"`
}

// ServiceMonitor defines a prometheus ServiceMonitor object.
type ServiceMonitor struct {
	// PrometheusRelease is the release label to add to the ServiceMonitor object.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	PrometheusRelease string `json:"prometheusRelease,omitempty"`
	// JobLabel to add to the ServiceMonitor object.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	JobLabel string `json:"jobLabel,omitempty"`
	// Interval for scraping metrics.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Interval string `json:"interval,omitempty"`
	// ScrapeTimeout defines the timeout for scraping metrics.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	ScrapeTimeout string `json:"scrapeTimeout,omitempty"`
}

// Metrics defines the metrics for a MariaDB.
type Metrics struct {
	// Enabled is a flag to enable Metrics
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
	Enabled bool `json:"enabled,omitempty"`
	// Exporter defines the metrics exporter container.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Exporter Exporter `json:"exporter"`
	// ServiceMonitor defines the ServiceMonior object.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	ServiceMonitor ServiceMonitor `json:"serviceMonitor"`
	// Username is the username of the monitoring user used by the exporter.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Username string `json:"username,omitempty" webhook:"inmutableinit"`
	// PasswordSecretKeyRef is a reference to the password of the monitoring user used by the exporter.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	PasswordSecretKeyRef corev1.SecretKeySelector `json:"passwordSecretKeyRef,omitempty" webhook:"inmutableinit"`
}

// MariaDBMaxScaleSpec defines a MaxScale resources to be used with the current MariaDB.
type MariaDBMaxScaleSpec struct {
	// Enabled is a flag to enable a MaxScale instance to be used with the current MariaDB.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
	Enabled bool `json:"enabled,omitempty"`
	// ContainerTemplate defines templates to configure Container objects.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	ContainerTemplate `json:",inline"`
	// PodTemplate defines templates to configure Pod objects.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	PodTemplate `json:",inline"`
	// Image name to be used by the MaxScale instances. The supported format is `<image>:<tag>`.
	// Only MaxScale official images are supported.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	Image string `json:"image,omitempty"`
	// ImagePullPolicy is the image pull policy. One of `Always`, `Never` or `IfNotPresent`. If not defined, it defaults to `IfNotPresent`.
	// +optional
	// +kubebuilder:validation:Enum=Always;Never;IfNotPresent
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:imagePullPolicy","urn:alm:descriptor:com.tectonic.ui:advanced"}
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`
	// ImagePullSecrets is the list of pull Secrets to be used to pull the image.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty" webhook:"inmutable"`
	// Services define how the traffic is forwarded to the MariaDB servers.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Services []MaxScaleService `json:"services,omitempty"`
	// Monitor monitors MariaDB server instances.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Monitor *MaxScaleMonitor `json:"monitor,omitempty"`
	// Admin configures the admin REST API and GUI.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Admin *MaxScaleAdmin `json:"admin,omitempty"`
	// Config defines the MaxScale configuration.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Config *MaxScaleConfig `json:"config,omitempty"`
	// Auth defines the credentials required for MaxScale to connect to MariaDB.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Auth *MaxScaleAuth `json:"auth,omitempty"`
	// Connection provides a template to define the Connection for MaxScale.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Connection *ConnectionTemplate `json:"connection,omitempty"`
	// Replicas indicates the number of desired instances.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:podCount"}
	Replicas *int32 `json:"replicas,omitempty"`
	// PodDisruptionBudget defines the budget for replica availability.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	PodDisruptionBudget *PodDisruptionBudget `json:"podDisruptionBudget,omitempty"`
	// UpdateStrategy defines the update strategy for the StatefulSet object.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:updateStrategy"}
	UpdateStrategy *appsv1.StatefulSetUpdateStrategy `json:"updateStrategy,omitempty"`
	// Service defines templates to configure the Kubernetes Service object.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	KubernetesService *ServiceTemplate `json:"kubernetesService,omitempty"`
	// RequeueInterval is used to perform requeue reconcilizations.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	RequeueInterval *metav1.Duration `json:"requeueInterval,omitempty"`
}

// MariaDBSpec defines the desired state of MariaDB
type MariaDBSpec struct {
	// ContainerTemplate defines templates to configure Container objects.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	ContainerTemplate `json:",inline"`
	// PodTemplate defines templates to configure Pod objects.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	PodTemplate `json:",inline"`
	// Image name to be used by the MariaDB instances. The supported format is `<image>:<tag>`.
	// Only MariaDB official images are supported.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	Image string `json:"image,omitempty"`
	// ImagePullPolicy is the image pull policy. One of `Always`, `Never` or `IfNotPresent`. If not defined, it defaults to `IfNotPresent`.
	// +optional
	// +kubebuilder:validation:Enum=Always;Never;IfNotPresent
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:imagePullPolicy","urn:alm:descriptor:com.tectonic.ui:advanced"}
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`
	// ImagePullSecrets is the list of pull Secrets to be used to pull the image.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty" webhook:"inmutable"`
	// InheritMetadata defines the metadata to be inherited by children resources.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	InheritMetadata *InheritMetadata `json:"inheritMetadata,omitempty"`
	// PodAnnotations to add to the Pods metadata.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	PodAnnotations map[string]string `json:"podAnnotations,omitempty"`
	// RootPasswordSecretKeyRef is a reference to a Secret key containing the root password.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	RootPasswordSecretKeyRef corev1.SecretKeySelector `json:"rootPasswordSecretKeyRef,omitempty" webhook:"inmutableinit"`
	// RootEmptyPassword indicates if the root password should be empty.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	RootEmptyPassword *bool `json:"rootEmptyPassword,omitempty" webhook:"inmutableinit"`
	// Database is the database to be created on bootstrap.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Database *string `json:"database,omitempty" webhook:"inmutable"`
	// Username is the username of the user to be created on bootstrap.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Username *string `json:"username,omitempty" webhook:"inmutable"`
	// PasswordSecretKeyRef is a reference to the password of the initial user provided via a Secret.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	PasswordSecretKeyRef *corev1.SecretKeySelector `json:"passwordSecretKeyRef,omitempty" webhook:"inmutableinit"`
	// MyCnf allows to specify the my.cnf file mounted by Mariadb.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	MyCnf *string `json:"myCnf,omitempty" webhook:"inmutable"`
	// MyCnfConfigMapKeyRef is a reference to the my.cnf config file provided via a ConfigMap.
	// If not provided, it will be defaulted with reference to a ConfigMap with the contents of the MyCnf field.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	MyCnfConfigMapKeyRef *corev1.ConfigMapKeySelector `json:"myCnfConfigMapKeyRef,omitempty" webhook:"inmutableinit"`
	// BootstrapFrom defines a source to bootstrap from.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	BootstrapFrom *RestoreSource `json:"bootstrapFrom,omitempty"`
	// Metrics configures metrics and how to scrape them.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Metrics *Metrics `json:"metrics,omitempty"`
	// Replication configures high availability via replication.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Replication *Replication `json:"replication,omitempty"`
	// Replication configures high availability via Galera.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Galera *Galera `json:"galera,omitempty"`
	// MaxScaleRef is a reference to a MaxScale resource to be used with the current MariaDB.
	// Providing this field implies delegating high availability tasks such as primary failover to MaxScale.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	MaxScaleRef *corev1.ObjectReference `json:"maxScaleRef,omitempty"`
	// MaxScale is the MaxScale specification that defines the MaxScale resource to be used with the current MariaDB.
	// When enabling this field, MaxScaleRef is automatically set.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	MaxScale *MariaDBMaxScaleSpec `json:"maxScale,omitempty"`
	// Replicas indicates the number of desired instances.
	// +kubebuilder:default=1
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:podCount"}
	Replicas int32 `json:"replicas,omitempty"`
	// Port where the instances will be listening for connections.
	// +optional
	// +kubebuilder:default=3306
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:number","urn:alm:descriptor:com.tectonic.ui:advanced"}
	Port int32 `json:"port,omitempty"`
	// EphemeralStorage indicates whether to use ephemeral storage for the instances.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	EphemeralStorage *bool `json:"ephemeralStorage,omitempty" webhook:"inmutableinit"`
	// VolumeClaimTemplate provides a template to define the Pod PVCs.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	VolumeClaimTemplate VolumeClaimTemplate `json:"volumeClaimTemplate" webhook:"inmutable"`
	// PodDisruptionBudget defines the budget for replica availability.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	PodDisruptionBudget *PodDisruptionBudget `json:"podDisruptionBudget,omitempty"`
	// PodDisruptionBudget defines the update strategy for the StatefulSet object.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:updateStrategy"}
	UpdateStrategy *appsv1.StatefulSetUpdateStrategy `json:"updateStrategy,omitempty"`
	// Service defines templates to configure the general Service object.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Service *ServiceTemplate `json:"service,omitempty"`
	// Connection defines templates to configure the general Connection object.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Connection *ConnectionTemplate `json:"connection,omitempty" webhook:"inmutable"`
	// PrimaryService defines templates to configure the primary Service object.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	PrimaryService *ServiceTemplate `json:"primaryService,omitempty"`
	// PrimaryConnection defines templates to configure the primary Connection object.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	PrimaryConnection *ConnectionTemplate `json:"primaryConnection,omitempty" webhook:"inmutable"`
	// SecondaryService defines templates to configure the secondary Service object.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	SecondaryService *ServiceTemplate `json:"secondaryService,omitempty"`
	// SecondaryConnection defines templates to configure the secondary Connection object.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	SecondaryConnection *ConnectionTemplate `json:"secondaryConnection,omitempty" webhook:"inmutable"`
}

// MariaDBStatus defines the observed state of MariaDB
type MariaDBStatus struct {
	// Conditions for the Mariadb object.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=status,xDescriptors={"urn:alm:descriptor:io.kubernetes.conditions"}
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// Replicas indicates the number of current instances.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:podCount"}
	Replicas int32 `json:"replicas,omitempty"`
	// CurrentPrimaryPodIndex is the primary Pod index.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=status
	CurrentPrimaryPodIndex *int `json:"currentPrimaryPodIndex,omitempty"`
	// CurrentPrimary is the primary Pod.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=status,xDescriptors={"urn:alm:descriptor:io.kubernetes:Pod"}
	CurrentPrimary *string `json:"currentPrimary,omitempty"`
	// GaleraRecovery is the Galera recovery current state.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=status
	GaleraRecovery *GaleraRecoveryStatus `json:"galeraRecovery,omitempty"`
	// ReplicationStatus is the replication current state for each Pod.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=status
	ReplicationStatus ReplicationStatus `json:"replicationStatus,omitempty"`
}

// SetCondition sets a status condition to MariaDB
func (s *MariaDBStatus) SetCondition(condition metav1.Condition) {
	if s.Conditions == nil {
		s.Conditions = make([]metav1.Condition, 0)
	}
	meta.SetStatusCondition(&s.Conditions, condition)
}

// UpdateCurrentPrimary updates the current primary status.
func (s *MariaDBStatus) UpdateCurrentPrimary(mariadb *MariaDB, index int) {
	s.CurrentPrimaryPodIndex = &index
	currentPrimary := statefulset.PodName(mariadb.ObjectMeta, index)
	s.CurrentPrimary = &currentPrimary
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=mdb
// +kubebuilder:subresource:status
// +kubebuilder:subresource:scale:specpath=.spec.replicas,statuspath=.status.replicas
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type==\"Ready\")].status"
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.conditions[?(@.type==\"Ready\")].message"
// +kubebuilder:printcolumn:name="Primary Pod",type="string",JSONPath=".status.currentPrimary"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +operator-sdk:csv:customresourcedefinitions:resources={{MariaDB,v1alpha1},{MaxScale,v1alpha1},{Connection,v1alpha1},{Restore,v1alpha1},{User,v1alpha1},{Grant,v1alpha1},{ConfigMap,v1},{Service,v1},{Secret,v1},{Event,v1},{ServiceAccount,v1},{StatefulSet,v1},{Deployment,v1},{PodDisruptionBudget,v1},{Role,v1},{RoleBinding,v1},{ClusterRoleBinding,v1}}

// MariaDB is the Schema for the mariadbs API. It is used to define MariaDB clusters.
type MariaDB struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MariaDBSpec   `json:"spec"`
	Status MariaDBStatus `json:"status,omitempty"`
}

// SetDefaults sets default values.
func (m *MariaDB) SetDefaults(env *environment.Environment) {
	if m.Spec.Image == "" {
		m.Spec.Image = env.RelatedMariadbImage
	}

	if m.Spec.EphemeralStorage == nil {
		m.Spec.EphemeralStorage = ptr.To(false)
	}

	if m.Spec.RootEmptyPassword == nil {
		m.Spec.RootEmptyPassword = ptr.To(false)
	}
	if m.Spec.RootPasswordSecretKeyRef == (corev1.SecretKeySelector{}) && !m.IsRootPasswordEmpty() {
		m.Spec.RootPasswordSecretKeyRef = m.RootPasswordSecretKeyRef()
	}

	if m.Spec.Port == 0 {
		m.Spec.Port = 3306
	}
	if m.Spec.MyCnf != nil && m.Spec.MyCnfConfigMapKeyRef == nil {
		myCnfKeyRef := m.MyCnfConfigMapKeyRef()
		m.Spec.MyCnfConfigMapKeyRef = &myCnfKeyRef
	}
	if m.IsInitialDataEnabled() && m.Spec.PasswordSecretKeyRef == nil {
		secretKeyRef := m.PasswordSecretKeyRef()
		m.Spec.PasswordSecretKeyRef = &secretKeyRef
	}
	if m.AreMetricsEnabled() {
		if m.Spec.Metrics.Exporter.Image == "" {
			m.Spec.Metrics.Exporter.Image = env.RelatedExporterImage
		}
		if m.Spec.Metrics.Exporter.Port == 0 {
			m.Spec.Metrics.Exporter.Port = 9104
		}
		if m.Spec.Metrics.Username == "" {
			m.Spec.Metrics.Username = m.MetricsKey().Name
		}
		if m.Spec.Metrics.PasswordSecretKeyRef == (corev1.SecretKeySelector{}) {
			m.Spec.Metrics.PasswordSecretKeyRef = m.MetricsPasswordSecretKeyRef()
		}
	}
	if ptr.Deref(m.Spec.MaxScale, MariaDBMaxScaleSpec{}).Enabled && m.Spec.MaxScaleRef == nil {
		m.Spec.MaxScaleRef = &corev1.ObjectReference{
			Name:      m.MaxScaleKey().Name,
			Namespace: m.Namespace,
		}
	}
}

// Replication with defaulting accessor
func (m *MariaDB) Replication() Replication {
	if m.Spec.Replication == nil {
		m.Spec.Replication = &Replication{}
	}
	m.Spec.Replication.FillWithDefaults()
	return *m.Spec.Replication
}

// Galera with defaulting accessor
func (m *MariaDB) Galera() Galera {
	if m.Spec.Galera == nil {
		m.Spec.Galera = &Galera{}
	}
	m.Spec.Galera.FillWithDefaults()
	return *m.Spec.Galera
}

// IsHAEnabled indicates whether the MariaDB instance has HA enabled
func (m *MariaDB) IsHAEnabled() bool {
	return m.Replication().Enabled || m.Galera().Enabled
}

// IsMaxScaleEnabled indicates that a MaxScale instance is forwarding traffic to this MariaDB instance
func (m *MariaDB) IsMaxScaleEnabled() bool {
	return m.Spec.MaxScaleRef != nil
}

// AreMetricsEnabled indicates whether the MariaDB instance has metrics enabled
func (m *MariaDB) AreMetricsEnabled() bool {
	return m.Spec.Metrics != nil && m.Spec.Metrics.Enabled
}

// IsInitialDataEnabled indicates whether the MariaDB instance has initial data enabled
func (m *MariaDB) IsInitialDataEnabled() bool {
	return m.Spec.Username != nil
}

// IsRootPasswordEmpty indicates whether the MariaDB instance has an empty root password
func (m *MariaDB) IsRootPasswordEmpty() bool {
	return m.Spec.RootEmptyPassword != nil && *m.Spec.RootEmptyPassword
}

// IsRootPasswordDefined indicates whether the MariaDB instance has a root password defined
func (m *MariaDB) IsRootPasswordDefined() bool {
	return m.Spec.RootPasswordSecretKeyRef != (corev1.SecretKeySelector{})
}

// IsEphemeralStorageEnabled indicates whether the MariaDB instance has ephemeral storage enabled
func (m *MariaDB) IsEphemeralStorageEnabled() bool {
	return m.Spec.EphemeralStorage != nil && *m.Spec.EphemeralStorage
}

// IsVolumeClaimTemplateDefined indicates whether the MariaDB instance has a VolumeClaimTemplate defined
func (m *MariaDB) IsVolumeClaimTemplateDefined() bool {
	return !reflect.ValueOf(m.Spec.VolumeClaimTemplate).IsZero()
}

// IsReady indicates whether the MariaDB instance is ready
func (m *MariaDB) IsReady() bool {
	return meta.IsStatusConditionTrue(m.Status.Conditions, ConditionTypeReady)
}

// IsRestoringBackup indicates whether the MariaDB instance is restoring backup
func (m *MariaDB) IsRestoringBackup() bool {
	return meta.IsStatusConditionFalse(m.Status.Conditions, ConditionTypeBackupRestored)
}

// HasRestoredBackup indicates whether the MariaDB instance has restored a Backup
func (m *MariaDB) HasRestoredBackup() bool {
	return meta.IsStatusConditionTrue(m.Status.Conditions, ConditionTypeBackupRestored)
}

// +kubebuilder:object:root=true

// MariaDBList contains a list of MariaDB
type MariaDBList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MariaDB `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MariaDB{}, &MariaDBList{})
}
