package controller

import (
	"os"
	"time"

	mariadbv1alpha1 "github.com/mariadb-operator/mariadb-operator/api/v1alpha1"
	labels "github.com/mariadb-operator/mariadb-operator/pkg/builder/labels"
	"github.com/mariadb-operator/mariadb-operator/pkg/statefulset"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("MariaDB controller", func() {
	Context("When creating a MariaDB", func() {
		It("Should default", func() {
			By("Creating MariaDB")
			testDefaultKey := types.NamespacedName{
				Name:      "test-mariadb-default",
				Namespace: testNamespace,
			}
			testDefaultMariaDb := mariadbv1alpha1.MariaDB{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testDefaultKey.Name,
					Namespace: testDefaultKey.Namespace,
				},
				Spec: mariadbv1alpha1.MariaDBSpec{
					VolumeClaimTemplate: mariadbv1alpha1.VolumeClaimTemplate{
						PersistentVolumeClaimSpec: corev1.PersistentVolumeClaimSpec{
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									"storage": resource.MustParse("100Mi"),
								},
							},
							AccessModes: []corev1.PersistentVolumeAccessMode{
								corev1.ReadWriteOnce,
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(testCtx, &testDefaultMariaDb)).To(Succeed())
			DeferCleanup(func() {
				deleteMariaDB(&testDefaultMariaDb)
			})

			By("Expecting to eventually default")
			Eventually(func() bool {
				if err := k8sClient.Get(testCtx, testDefaultKey, &testDefaultMariaDb); err != nil {
					return false
				}
				return testDefaultMariaDb.Spec.Image != ""
			}, testTimeout, testInterval).Should(BeTrue())
		})
		It("Should reconcile", func() {
			var testMariaDb mariadbv1alpha1.MariaDB
			By("Getting MariaDB")
			Expect(k8sClient.Get(testCtx, testMdbkey, &testMariaDb)).To(Succeed())

			By("Expecting to create a ConfigMap eventually")
			Eventually(func() bool {
				var cm corev1.ConfigMap
				key := types.NamespacedName{
					Name:      testMariaDb.MyCnfConfigMapKeyRef().Name,
					Namespace: testMariaDb.Namespace,
				}
				if err := k8sClient.Get(testCtx, key, &cm); err != nil {
					return false
				}
				Expect(cm.ObjectMeta.Labels).NotTo(BeNil())
				Expect(cm.ObjectMeta.Annotations).NotTo(BeNil())
				return true
			}, testTimeout, testInterval).Should(BeTrue())

			By("Expecting to create a StatefulSet eventually")
			Eventually(func() bool {
				var sts appsv1.StatefulSet
				if err := k8sClient.Get(testCtx, testMdbkey, &sts); err != nil {
					return false
				}
				Expect(sts.ObjectMeta.Labels).NotTo(BeNil())
				Expect(sts.ObjectMeta.Annotations).NotTo(BeNil())
				return true
			}, testTimeout, testInterval).Should(BeTrue())

			By("Expecting to create a Service eventually")
			Eventually(func() bool {
				var svc corev1.Service
				if err := k8sClient.Get(testCtx, testMdbkey, &svc); err != nil {
					return false
				}
				Expect(svc.ObjectMeta.Labels).NotTo(BeNil())
				Expect(svc.ObjectMeta.Labels).To(HaveKeyWithValue("mariadb.mmontes.io/test", "test"))
				Expect(svc.ObjectMeta.Annotations).NotTo(BeNil())
				Expect(svc.ObjectMeta.Annotations).To(HaveKeyWithValue("mariadb.mmontes.io/test", "test"))
				return true
			}, testTimeout, testInterval).Should(BeTrue())

			By("Expecting Connection to be ready eventually")
			Eventually(func() bool {
				var conn mariadbv1alpha1.Connection
				if err := k8sClient.Get(testCtx, client.ObjectKeyFromObject(&testMariaDb), &conn); err != nil {
					return false
				}
				Expect(conn.ObjectMeta.Labels).NotTo(BeNil())
				Expect(conn.ObjectMeta.Annotations).NotTo(BeNil())
				return conn.IsReady()
			}, testTimeout, testInterval).Should(BeTrue())

			By("Expecting metrics User to be ready eventually")
			Eventually(func() bool {
				var user mariadbv1alpha1.User
				if err := k8sClient.Get(testCtx, testMariaDb.MetricsKey(), &user); err != nil {
					return false
				}
				return user.IsReady()
			}, testTimeout, testInterval).Should(BeTrue())

			By("Expecting metrics Grant to be ready eventually")
			Eventually(func() bool {
				var grant mariadbv1alpha1.Grant
				if err := k8sClient.Get(testCtx, testMariaDb.MetricsKey(), &grant); err != nil {
					return false
				}
				return grant.IsReady()
			}, testTimeout, testInterval).Should(BeTrue())

			By("Expecting to create a exporter Deployment eventually")
			Eventually(func(g Gomega) bool {
				var deploy appsv1.Deployment
				if err := k8sClient.Get(testCtx, testMariaDb.MetricsKey(), &deploy); err != nil {
					return false
				}
				expectedImage := os.Getenv("RELATED_IMAGE_EXPORTER")
				g.Expect(expectedImage).ToNot(BeEmpty())
				By("Expecting Deployment to have exporter image")
				g.Expect(deploy.Spec.Template.Spec.Containers).To(ContainElement(MatchFields(IgnoreExtras,
					Fields{
						"Image": Equal(expectedImage),
					})))
				return deploymentReady(&deploy)
			}).WithTimeout(testTimeout).WithPolling(testInterval).Should(BeTrue())

			By("Expecting to create a ServiceMonitor eventually")
			Eventually(func(g Gomega) bool {
				var svcMonitor monitoringv1.ServiceMonitor
				if err := k8sClient.Get(testCtx, testMariaDb.MetricsKey(), &svcMonitor); err != nil {
					return false
				}
				g.Expect(svcMonitor.Spec.Selector.MatchLabels).NotTo(BeEmpty())
				g.Expect(svcMonitor.Spec.Selector.MatchLabels).To(HaveKeyWithValue("app.kubernetes.io/name", "exporter"))
				g.Expect(svcMonitor.Spec.Selector.MatchLabels).To(HaveKeyWithValue("app.kubernetes.io/instance", testMdbkey.Name))
				g.Expect(svcMonitor.Spec.Endpoints).To(HaveLen(1))
				return true
			}).WithTimeout(testTimeout).WithPolling(testInterval).Should(BeTrue())
		})
		It("Should bootstrap from Backup", func() {
			By("Creating Backup")
			backupKey := types.NamespacedName{
				Name:      "backup-mdb-test",
				Namespace: testNamespace,
			}
			backup := mariadbv1alpha1.Backup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      backupKey.Name,
					Namespace: backupKey.Namespace,
				},
				Spec: mariadbv1alpha1.BackupSpec{
					MariaDBRef: mariadbv1alpha1.MariaDBRef{
						ObjectReference: corev1.ObjectReference{
							Name: testMdbkey.Name,
						},
						WaitForIt: true,
					},
					Storage: mariadbv1alpha1.BackupStorage{
						S3: testS3WithBucket("test-mariadb"),
					},
				},
			}
			Expect(k8sClient.Create(testCtx, &backup)).To(Succeed())
			DeferCleanup(func() {
				Expect(k8sClient.Delete(testCtx, &backup)).To(Succeed())
			})

			By("Expecting Backup to complete eventually")
			Eventually(func() bool {
				if err := k8sClient.Get(testCtx, backupKey, &backup); err != nil {
					return false
				}
				return backup.IsComplete()
			}, testTimeout, testInterval).Should(BeTrue())

			By("Creating a MariaDB bootstrapping from backup")
			bootstrapMariaDBKey := types.NamespacedName{
				Name:      "mariadb-from-backup",
				Namespace: testNamespace,
			}
			bootstrapMariaDB := mariadbv1alpha1.MariaDB{
				ObjectMeta: metav1.ObjectMeta{
					Name:      bootstrapMariaDBKey.Name,
					Namespace: bootstrapMariaDBKey.Namespace,
				},
				Spec: mariadbv1alpha1.MariaDBSpec{
					BootstrapFrom: &mariadbv1alpha1.RestoreSource{
						BackupRef: &corev1.LocalObjectReference{
							Name: backupKey.Name,
						},
						TargetRecoveryTime: &metav1.Time{Time: time.Now()},
					},
					VolumeClaimTemplate: mariadbv1alpha1.VolumeClaimTemplate{
						PersistentVolumeClaimSpec: corev1.PersistentVolumeClaimSpec{
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									"storage": resource.MustParse("100Mi"),
								},
							},
							AccessModes: []corev1.PersistentVolumeAccessMode{
								corev1.ReadWriteOnce,
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(testCtx, &bootstrapMariaDB)).To(Succeed())
			DeferCleanup(func() {
				deleteMariaDB(&bootstrapMariaDB)
			})

			By("Expecting MariaDB to be ready eventually")
			Eventually(func() bool {
				if err := k8sClient.Get(testCtx, bootstrapMariaDBKey, &bootstrapMariaDB); err != nil {
					return false
				}
				return bootstrapMariaDB.IsReady()
			}, testHighTimeout, testInterval).Should(BeTrue())

			By("Expecting MariaDB to have restored backup")
			Expect(bootstrapMariaDB.HasRestoredBackup()).To(BeTrue())
		})
	})

	Context("When updating a MariaDB", func() {
		It("Should reconcile", func() {
			By("Creating MariaDB")
			updateMariaDBKey := types.NamespacedName{
				Name:      "test-update-mariadb",
				Namespace: testNamespace,
			}
			updateMariaDB := mariadbv1alpha1.MariaDB{
				ObjectMeta: metav1.ObjectMeta{
					Name:      updateMariaDBKey.Name,
					Namespace: updateMariaDBKey.Namespace,
				},
				Spec: mariadbv1alpha1.MariaDBSpec{
					VolumeClaimTemplate: mariadbv1alpha1.VolumeClaimTemplate{
						PersistentVolumeClaimSpec: corev1.PersistentVolumeClaimSpec{
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									"storage": resource.MustParse("100Mi"),
								},
							},
							AccessModes: []corev1.PersistentVolumeAccessMode{
								corev1.ReadWriteOnce,
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(testCtx, &updateMariaDB)).To(Succeed())
			DeferCleanup(func() {
				deleteMariaDB(&updateMariaDB)
			})

			By("Expecting MariaDB to be ready eventually")
			Eventually(func() bool {
				if err := k8sClient.Get(testCtx, updateMariaDBKey, &updateMariaDB); err != nil {
					return false
				}
				return updateMariaDB.IsReady()
			}, testTimeout, testInterval).Should(BeTrue())

			By("Updating MariaDB image")
			Eventually(func() bool {
				if err := k8sClient.Get(testCtx, updateMariaDBKey, &updateMariaDB); err != nil {
					return false
				}
				updateMariaDB.Spec.Image = "mariadb:lts"
				return k8sClient.Update(testCtx, &updateMariaDB) == nil
			}, testTimeout, testInterval).Should(BeTrue())

			By("Expecting image to be updated in StatefulSet eventually")
			Eventually(func() bool {
				var sts appsv1.StatefulSet
				if err := k8sClient.Get(testCtx, updateMariaDBKey, &sts); err != nil {
					return false
				}
				return sts.Spec.Template.Spec.Containers[0].Image == "mariadb:lts"
			}, testTimeout, testInterval).Should(BeTrue())

			By("Expecting MariaDB to be ready eventually")
			Eventually(func() bool {
				if err := k8sClient.Get(testCtx, updateMariaDBKey, &updateMariaDB); err != nil {
					return false
				}
				return updateMariaDB.IsReady()
			}, testTimeout, testInterval).Should(BeTrue())
		})
	})
})

var _ = Describe("MariaDB replication", func() {
	Context("When creating a MariaDB with replication", func() {
		It("Should reconcile", func() {
			mdb := mariadbv1alpha1.MariaDB{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mariadb-repl",
					Namespace: testNamespace,
				},
				Spec: mariadbv1alpha1.MariaDBSpec{
					Username: &testUser,
					PasswordSecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: testPwdKey.Name,
						},
						Key: testPwdSecretKey,
					},
					Database: &testDatabase,
					VolumeClaimTemplate: mariadbv1alpha1.VolumeClaimTemplate{
						PersistentVolumeClaimSpec: corev1.PersistentVolumeClaimSpec{
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									"storage": resource.MustParse("100Mi"),
								},
							},
							AccessModes: []corev1.PersistentVolumeAccessMode{
								corev1.ReadWriteOnce,
							},
						},
					},
					MyCnf: func() *string {
						cfg := `[mariadb]
						bind-address=*
						default_storage_engine=InnoDB
						binlog_format=row
						innodb_autoinc_lock_mode=2
						max_allowed_packet=256M`
						return &cfg
					}(),
					Replication: &mariadbv1alpha1.Replication{
						ReplicationSpec: mariadbv1alpha1.ReplicationSpec{
							Primary: &mariadbv1alpha1.PrimaryReplication{
								PodIndex:          func() *int { i := 0; return &i }(),
								AutomaticFailover: func() *bool { f := true; return &f }(),
							},
							Replica: &mariadbv1alpha1.ReplicaReplication{
								WaitPoint: func() *mariadbv1alpha1.WaitPoint { w := mariadbv1alpha1.WaitPointAfterSync; return &w }(),
								Gtid:      func() *mariadbv1alpha1.Gtid { g := mariadbv1alpha1.GtidCurrentPos; return &g }(),
							},
							SyncBinlog: func() *bool { s := true; return &s }(),
						},
						Enabled: true,
					},
					Replicas: 3,
					Service: &mariadbv1alpha1.ServiceTemplate{
						Type: corev1.ServiceTypeLoadBalancer,
						Annotations: map[string]string{
							"metallb.universe.tf/loadBalancerIPs": testCidrPrefix + ".0.120",
						},
					},
					Connection: &mariadbv1alpha1.ConnectionTemplate{
						SecretName: func() *string {
							s := "mdb-repl-conn"
							return &s
						}(),
						SecretTemplate: &mariadbv1alpha1.SecretTemplate{
							Key: &testConnSecretKey,
						},
					},
					PrimaryService: &mariadbv1alpha1.ServiceTemplate{
						Type: corev1.ServiceTypeLoadBalancer,
						Annotations: map[string]string{
							"metallb.universe.tf/loadBalancerIPs": testCidrPrefix + ".0.130",
						},
					},
					PrimaryConnection: &mariadbv1alpha1.ConnectionTemplate{
						SecretName: func() *string {
							s := "mdb-repl-conn-primary"
							return &s
						}(),
						SecretTemplate: &mariadbv1alpha1.SecretTemplate{
							Key: &testConnSecretKey,
						},
					},
					SecondaryService: &mariadbv1alpha1.ServiceTemplate{
						Type: corev1.ServiceTypeLoadBalancer,
						Annotations: map[string]string{
							"metallb.universe.tf/loadBalancerIPs": testCidrPrefix + ".0.131",
						},
					},
					SecondaryConnection: &mariadbv1alpha1.ConnectionTemplate{
						SecretName: func() *string {
							s := "mdb-repl-conn-secondary"
							return &s
						}(),
						SecretTemplate: &mariadbv1alpha1.SecretTemplate{
							Key: &testConnSecretKey,
						},
					},
				},
			}

			By("Creating MariaDB with replication")
			Expect(k8sClient.Create(testCtx, &mdb)).To(Succeed())
			DeferCleanup(func() {
				deleteMariaDB(&mdb)
			})

			By("Expecting MariaDB to be ready eventually")
			Eventually(func() bool {
				if err := k8sClient.Get(testCtx, client.ObjectKeyFromObject(&mdb), &mdb); err != nil {
					return false
				}
				return mdb.IsReady()
			}, testHighTimeout, testInterval).Should(BeTrue())

			By("Expecting to create a Service")
			var svc corev1.Service
			Expect(k8sClient.Get(testCtx, client.ObjectKeyFromObject(&mdb), &svc)).To(Succeed())

			By("Expecting to create a primary Service")
			Expect(k8sClient.Get(testCtx, mdb.PrimaryServiceKey(), &svc)).To(Succeed())
			Expect(svc.Spec.Selector["statefulset.kubernetes.io/pod-name"]).To(Equal(statefulset.PodName(mdb.ObjectMeta, 0)))

			By("Expecting to create a secondary Service")
			Expect(k8sClient.Get(testCtx, mdb.SecondaryServiceKey(), &svc)).To(Succeed())

			By("Expecting Connection to be ready eventually")
			Eventually(func() bool {
				var conn mariadbv1alpha1.Connection
				if err := k8sClient.Get(testCtx, client.ObjectKeyFromObject(&mdb), &conn); err != nil {
					return false
				}
				return conn.IsReady()
			}, testTimeout, testInterval).Should(BeTrue())

			By("Expecting primary Connection to be ready eventually")
			Eventually(func() bool {
				var conn mariadbv1alpha1.Connection
				if err := k8sClient.Get(testCtx, mdb.PrimaryConnectioneKey(), &conn); err != nil {
					return false
				}
				return conn.IsReady()
			}, testTimeout, testInterval).Should(BeTrue())

			By("Expecting secondary Connection to be ready eventually")
			Eventually(func() bool {
				var conn mariadbv1alpha1.Connection
				if err := k8sClient.Get(testCtx, mdb.SecondaryConnectioneKey(), &conn); err != nil {
					return false
				}
				return conn.IsReady()
			}, testTimeout, testInterval).Should(BeTrue())

			By("Expecting to create secondary Endpoints")
			var endpoints corev1.Endpoints
			Expect(k8sClient.Get(testCtx, mdb.SecondaryServiceKey(), &endpoints)).To(Succeed())
			Expect(endpoints.Subsets).To(HaveLen(1))
			Expect(endpoints.Subsets[0].Addresses).To(HaveLen(int(mdb.Spec.Replicas) - 1))

			By("Expecting to create a PodDisruptionBudget")
			var pdb policyv1.PodDisruptionBudget
			Expect(k8sClient.Get(testCtx, client.ObjectKeyFromObject(&mdb), &pdb)).To(Succeed())

			By("Expecting MariaDB to eventually update primary")
			podIndex := 1
			Eventually(func(g Gomega) bool {
				g.Expect(k8sClient.Get(testCtx, client.ObjectKeyFromObject(&mdb), &mdb)).To(Succeed())
				mdb.Replication().Primary.PodIndex = &podIndex
				g.Expect(k8sClient.Update(testCtx, &mdb)).To(Succeed())
				return true
			}, testTimeout, testInterval).Should(BeTrue())

			By("Expecting MariaDB to eventually change primary")
			Eventually(func() bool {
				if err := k8sClient.Get(testCtx, client.ObjectKeyFromObject(&mdb), &mdb); err != nil {
					return false
				}
				if !mdb.IsReady() || mdb.Status.CurrentPrimaryPodIndex == nil {
					return false
				}
				return *mdb.Status.CurrentPrimaryPodIndex == podIndex
			}, testTimeout, testInterval).Should(BeTrue())

			By("Expecting primary Service to eventually change primary")
			Eventually(func() bool {
				var svc corev1.Service
				if err := k8sClient.Get(testCtx, mdb.PrimaryServiceKey(), &svc); err != nil {
					return false
				}
				return svc.Spec.Selector["statefulset.kubernetes.io/pod-name"] == statefulset.PodName(mdb.ObjectMeta, podIndex)
			}, testTimeout, testInterval).Should(BeTrue())

			By("Tearing down primary Pod consistently")
			Consistently(func() bool {
				primaryPodKey := types.NamespacedName{
					Name:      statefulset.PodName(mdb.ObjectMeta, 1),
					Namespace: mdb.Namespace,
				}
				var primaryPod corev1.Pod
				if err := k8sClient.Get(testCtx, primaryPodKey, &primaryPod); err != nil {
					return apierrors.IsNotFound(err)
				}
				return k8sClient.Delete(testCtx, &primaryPod) == nil
			}, 10*time.Second, testInterval).Should(BeTrue())

			By("Expecting MariaDB to be ready eventually")
			Eventually(func() bool {
				if err := k8sClient.Get(testCtx, client.ObjectKeyFromObject(&mdb), &mdb); err != nil {
					return false
				}
				return mdb.IsReady()
			}, testHighTimeout, testInterval).Should(BeTrue())

			By("Expecting MariaDB to eventually change primary")
			Eventually(func() bool {
				if err := k8sClient.Get(testCtx, client.ObjectKeyFromObject(&mdb), &mdb); err != nil {
					return false
				}
				if !mdb.IsReady() || mdb.Status.CurrentPrimaryPodIndex == nil {
					return false
				}
				return *mdb.Status.CurrentPrimaryPodIndex == 0 || *mdb.Status.CurrentPrimaryPodIndex == 2
			}, testHighTimeout, testInterval).Should(BeTrue())

			By("Expecting Connection to be ready eventually")
			Eventually(func() bool {
				var conn mariadbv1alpha1.Connection
				if err := k8sClient.Get(testCtx, client.ObjectKeyFromObject(&mdb), &conn); err != nil {
					return false
				}
				return conn.IsReady()
			}, testTimeout, testInterval).Should(BeTrue())

			By("Expecting primary Connection to be ready eventually")
			Eventually(func() bool {
				var conn mariadbv1alpha1.Connection
				if err := k8sClient.Get(testCtx, mdb.PrimaryConnectioneKey(), &conn); err != nil {
					return false
				}
				return conn.IsReady()
			}, testTimeout, testInterval).Should(BeTrue())

			mxsKey := types.NamespacedName{
				Name:      "maxscale-repl",
				Namespace: testNamespace,
			}
			expectMariadbMaxScaleReady(&mdb, mxsKey)
		})
	})
})

var _ = Describe("MariaDB Galera", func() {
	Context("When creating a MariaDB Galera", func() {
		It("Should reconcile", func() {
			clusterHealthyTimeout := metav1.Duration{Duration: 30 * time.Second}
			recoveryTimeout := metav1.Duration{Duration: 5 * time.Minute}
			mdb := mariadbv1alpha1.MariaDB{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mariadb-galera",
					Namespace: testNamespace,
				},
				Spec: mariadbv1alpha1.MariaDBSpec{
					Username: &testUser,
					PasswordSecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: testPwdKey.Name,
						},
						Key: testPwdSecretKey,
					},
					Database: &testDatabase,
					VolumeClaimTemplate: mariadbv1alpha1.VolumeClaimTemplate{
						PersistentVolumeClaimSpec: corev1.PersistentVolumeClaimSpec{
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									"storage": resource.MustParse("100Mi"),
								},
							},
							AccessModes: []corev1.PersistentVolumeAccessMode{
								corev1.ReadWriteOnce,
							},
						},
					},
					MyCnf: func() *string {
						cfg := `[mariadb]
						bind-address=*
						default_storage_engine=InnoDB
						binlog_format=row
						innodb_autoinc_lock_mode=2
						max_allowed_packet=256M`
						return &cfg
					}(),
					Galera: &mariadbv1alpha1.Galera{
						Enabled: true,
						GaleraSpec: mariadbv1alpha1.GaleraSpec{
							Primary: &mariadbv1alpha1.PrimaryGalera{
								PodIndex:          func() *int { i := 0; return &i }(),
								AutomaticFailover: func() *bool { af := true; return &af }(),
							},
							Recovery: &mariadbv1alpha1.GaleraRecovery{
								Enabled:                 true,
								ClusterHealthyTimeout:   &clusterHealthyTimeout,
								ClusterBootstrapTimeout: &recoveryTimeout,
								PodRecoveryTimeout:      &recoveryTimeout,
								PodSyncTimeout:          &recoveryTimeout,
							},
							VolumeClaimTemplate: &mariadbv1alpha1.VolumeClaimTemplate{
								PersistentVolumeClaimSpec: corev1.PersistentVolumeClaimSpec{
									Resources: corev1.ResourceRequirements{
										Requests: corev1.ResourceList{
											"storage": resource.MustParse("100Mi"),
										},
									},
									AccessModes: []corev1.PersistentVolumeAccessMode{
										corev1.ReadWriteOnce,
									},
								},
							},
						},
					},
					Replicas: 3,
					Service: &mariadbv1alpha1.ServiceTemplate{
						Type: corev1.ServiceTypeLoadBalancer,
						Annotations: map[string]string{
							"metallb.universe.tf/loadBalancerIPs": testCidrPrefix + ".0.150",
						},
					},
					Connection: &mariadbv1alpha1.ConnectionTemplate{
						SecretName: func() *string {
							s := "mdb-galera-conn"
							return &s
						}(),
						SecretTemplate: &mariadbv1alpha1.SecretTemplate{
							Key: &testConnSecretKey,
						},
					},
					PrimaryService: &mariadbv1alpha1.ServiceTemplate{
						Type: corev1.ServiceTypeLoadBalancer,
						Annotations: map[string]string{
							"metallb.universe.tf/loadBalancerIPs": testCidrPrefix + ".0.160",
						},
					},
					PrimaryConnection: &mariadbv1alpha1.ConnectionTemplate{
						SecretName: func() *string {
							s := "mdb-galera-conn-primary"
							return &s
						}(),
						SecretTemplate: &mariadbv1alpha1.SecretTemplate{
							Key: &testConnSecretKey,
						},
					},
					SecondaryService: &mariadbv1alpha1.ServiceTemplate{
						Type: corev1.ServiceTypeLoadBalancer,
						Annotations: map[string]string{
							"metallb.universe.tf/loadBalancerIPs": testCidrPrefix + ".0.161",
						},
					},
					SecondaryConnection: &mariadbv1alpha1.ConnectionTemplate{
						SecretName: func() *string {
							s := "mdb-galera-conn-secondary"
							return &s
						}(),
						SecretTemplate: &mariadbv1alpha1.SecretTemplate{
							Key: &testConnSecretKey,
						},
					},
				},
			}

			By("Creating MariaDB Galera")
			Expect(k8sClient.Create(testCtx, &mdb)).To(Succeed())
			DeferCleanup(func() {
				deleteMariaDB(&mdb)
			})

			By("Expecting MariaDB to be ready eventually")
			Eventually(func() bool {
				if err := k8sClient.Get(testCtx, client.ObjectKeyFromObject(&mdb), &mdb); err != nil {
					return false
				}
				return mdb.IsReady()
			}, testVeryHighTimeout, testInterval).Should(BeTrue())

			By("Expecting Galera to be configured eventually")
			Eventually(func() bool {
				if err := k8sClient.Get(testCtx, client.ObjectKeyFromObject(&mdb), &mdb); err != nil {
					return false
				}
				return mdb.HasGaleraConfiguredCondition()
			}, testTimeout, testInterval).Should(BeTrue())

			By("Expecting Galera to be ready eventually")
			Eventually(func() bool {
				if err := k8sClient.Get(testCtx, client.ObjectKeyFromObject(&mdb), &mdb); err != nil {
					return false
				}
				return mdb.HasGaleraReadyCondition()
			}, testTimeout, testInterval).Should(BeTrue())

			By("Expecting to create a Service")
			var svc corev1.Service
			Expect(k8sClient.Get(testCtx, client.ObjectKeyFromObject(&mdb), &svc)).To(Succeed())

			By("Expecting to create a primary Service")
			Expect(k8sClient.Get(testCtx, mdb.PrimaryServiceKey(), &svc)).To(Succeed())
			Expect(svc.Spec.Selector["statefulset.kubernetes.io/pod-name"]).To(Equal(statefulset.PodName(mdb.ObjectMeta, 0)))

			By("Expecting to create a secondary Service")
			Expect(k8sClient.Get(testCtx, mdb.SecondaryServiceKey(), &svc)).To(Succeed())

			By("Expecting Connection to be ready eventually")
			Eventually(func() bool {
				var conn mariadbv1alpha1.Connection
				if err := k8sClient.Get(testCtx, client.ObjectKeyFromObject(&mdb), &conn); err != nil {
					return false
				}
				return conn.IsReady()
			}, testTimeout, testInterval).Should(BeTrue())

			By("Expecting primary Connection to be ready eventually")
			Eventually(func() bool {
				var conn mariadbv1alpha1.Connection
				if err := k8sClient.Get(testCtx, mdb.PrimaryConnectioneKey(), &conn); err != nil {
					return false
				}
				return conn.IsReady()
			}, testTimeout, testInterval).Should(BeTrue())

			By("Expecting secondary Connection to be ready eventually")
			Eventually(func() bool {
				var conn mariadbv1alpha1.Connection
				if err := k8sClient.Get(testCtx, mdb.SecondaryConnectioneKey(), &conn); err != nil {
					return false
				}
				return conn.IsReady()
			}, testTimeout, testInterval).Should(BeTrue())

			By("Expecting to create secondary Endpoints")
			var endpoints corev1.Endpoints
			Expect(k8sClient.Get(testCtx, mdb.SecondaryServiceKey(), &endpoints)).To(Succeed())
			Expect(endpoints.Subsets).To(HaveLen(1))
			Expect(endpoints.Subsets[0].Addresses).To(HaveLen(int(mdb.Spec.Replicas) - 1))

			By("Expecting to create a PodDisruptionBudget")
			var pdb policyv1.PodDisruptionBudget
			Expect(k8sClient.Get(testCtx, client.ObjectKeyFromObject(&mdb), &pdb)).To(Succeed())

			By("Updating MariaDB primary")
			podIndex := 1
			mdb.Galera().Primary.PodIndex = &podIndex
			Expect(k8sClient.Update(testCtx, &mdb)).To(Succeed())

			By("Expecting MariaDB to eventually change primary")
			Eventually(func() bool {
				if err := k8sClient.Get(testCtx, client.ObjectKeyFromObject(&mdb), &mdb); err != nil {
					return false
				}
				if !mdb.IsReady() || mdb.Status.CurrentPrimaryPodIndex == nil {
					return false
				}
				return *mdb.Status.CurrentPrimaryPodIndex == podIndex
			}, testTimeout, testInterval).Should(BeTrue())

			By("Expecting primary Service to eventually change primary")
			Eventually(func() bool {
				var svc corev1.Service
				if err := k8sClient.Get(testCtx, mdb.PrimaryServiceKey(), &svc); err != nil {
					return false
				}
				return svc.Spec.Selector["statefulset.kubernetes.io/pod-name"] == statefulset.PodName(mdb.ObjectMeta, podIndex)
			}, testTimeout, testInterval).Should(BeTrue())

			By("Tearing down all Pods consistently")
			opts := []client.DeleteAllOfOption{
				client.MatchingLabels{
					"app.kubernetes.io/instance": mdb.Name,
				},
				client.InNamespace(mdb.Namespace),
			}
			Expect(k8sClient.DeleteAllOf(testCtx, &corev1.Pod{}, opts...)).To(Succeed())

			By("Expecting MariaDB NOT to be ready eventually")
			Eventually(func() bool {
				if err := k8sClient.Get(testCtx, client.ObjectKeyFromObject(&mdb), &mdb); err != nil {
					return false
				}
				return mdb.IsReady()
			}, testVeryHighTimeout, testInterval).Should(BeTrue())

			By("Expecting Galera NOT to be ready eventually")
			Eventually(func() bool {
				if err := k8sClient.Get(testCtx, client.ObjectKeyFromObject(&mdb), &mdb); err != nil {
					return false
				}
				return mdb.HasGaleraNotReadyCondition()
			}, testVeryHighTimeout, testInterval).Should(BeTrue())

			By("Expecting MariaDB to be ready eventually")
			Eventually(func() bool {
				if err := k8sClient.Get(testCtx, client.ObjectKeyFromObject(&mdb), &mdb); err != nil {
					return false
				}
				return mdb.IsReady()
			}, testVeryHighTimeout, testInterval).Should(BeTrue())

			By("Expecting Galera to be ready eventually")
			Eventually(func() bool {
				if err := k8sClient.Get(testCtx, client.ObjectKeyFromObject(&mdb), &mdb); err != nil {
					return false
				}
				return mdb.HasGaleraReadyCondition()
			}, testVeryHighTimeout, testInterval).Should(BeTrue())

			By("Expecting Connection to be ready eventually")
			Eventually(func() bool {
				var conn mariadbv1alpha1.Connection
				if err := k8sClient.Get(testCtx, client.ObjectKeyFromObject(&mdb), &conn); err != nil {
					return false
				}
				return conn.IsReady()
			}, testTimeout, testInterval).Should(BeTrue())

			By("Expecting primary Connection to be ready eventually")
			Eventually(func() bool {
				var conn mariadbv1alpha1.Connection
				if err := k8sClient.Get(testCtx, mdb.PrimaryConnectioneKey(), &conn); err != nil {
					return false
				}
				return conn.IsReady()
			}, testTimeout, testInterval).Should(BeTrue())

			mxsKey := types.NamespacedName{
				Name:      "maxscale-galera",
				Namespace: testNamespace,
			}
			expectMariadbMaxScaleReady(&mdb, mxsKey)
		})
	})
})

func expectMariadbMaxScaleReady(mdb *mariadbv1alpha1.MariaDB, mxsKey types.NamespacedName) {
	mxs := mariadbv1alpha1.MaxScale{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mxsKey.Name,
			Namespace: mxsKey.Namespace,
		},
		Spec: mariadbv1alpha1.MaxScaleSpec{
			MariaDBRef: &mariadbv1alpha1.MariaDBRef{
				ObjectReference: corev1.ObjectReference{
					Name: client.ObjectKeyFromObject(mdb).Name,
				},
			},
		},
	}
	By("Creating MaxScale")
	Expect(k8sClient.Create(testCtx, &mxs)).To(Succeed())
	DeferCleanup(func() {
		deleteMaxScale(mxsKey)
	})

	By("Point MariaDB to MaxScale")
	mdb.Spec.MaxScaleRef = &corev1.ObjectReference{
		Name:      mxsKey.Name,
		Namespace: mxsKey.Namespace,
	}
	Expect(k8sClient.Update(testCtx, mdb)).To(Succeed())

	By("Expecting MariaDB to be ready eventually")
	Eventually(func() bool {
		if err := k8sClient.Get(testCtx, client.ObjectKeyFromObject(mdb), mdb); err != nil {
			return false
		}
		return mdb.IsReady()
	}, testHighTimeout, testInterval).Should(BeTrue())

	By("Expecting MaxScale to be ready eventually")
	Eventually(func() bool {
		if err := k8sClient.Get(testCtx, mxsKey, &mxs); err != nil {
			return false
		}
		return mxs.IsReady()
	}, testHighTimeout, testInterval).Should(BeTrue())
}

func deploymentReady(deploy *appsv1.Deployment) bool {
	for _, c := range deploy.Status.Conditions {
		if c.Type == appsv1.DeploymentAvailable && c.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func deleteMariaDB(mdb *mariadbv1alpha1.MariaDB) {
	Expect(k8sClient.Delete(testCtx, mdb)).To(Succeed())

	Eventually(func(g Gomega) bool {
		listOpts := &client.ListOptions{
			LabelSelector: klabels.SelectorFromSet(
				labels.NewLabelsBuilder().
					WithMariaDB(mdb).
					Build(),
			),
			Namespace: mdb.GetNamespace(),
		}
		pvcList := &corev1.PersistentVolumeClaimList{}
		g.Expect(k8sClient.List(testCtx, pvcList, listOpts)).To(Succeed())

		for _, pvc := range pvcList.Items {
			g.Expect(k8sClient.Delete(testCtx, &pvc)).To(Succeed())
		}
		return true
	}, 30*time.Second, 1*time.Second).Should(BeTrue())
}
