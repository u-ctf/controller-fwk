//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	testv1 "operator/api/v1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ConfigMapManagementTests contains all the ConfigMap-related tests
func ConfigMapManagementTests(getClient func() client.Client, ctx context.Context, getTestNamespace func() corev1.Namespace, resourceFactory ResourceFactory, resourceTypeName string) {
	Context(fmt.Sprintf("ConfigMap Management (%s)", resourceTypeName), func() {
		var testResource TestableResource
		var originalConfigMapName string
		var originalConfigMapData map[string]string
		var testSecret *corev1.Secret

		BeforeEach(func() {
			originalConfigMapName = "test-cm-" + uuid.NewString()[:8]
			originalConfigMapData = map[string]string{
				"key1": "value1",
				"key2": "value2",
			}

			// Create a ready secret for ConfigMap tests
			testSecret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cm-secret-" + uuid.NewString()[:8],
					Namespace: getTestNamespace().Name,
				},
				Data: map[string][]byte{
					"ready": []byte("true"),
					"data":  []byte("test-data"),
				},
			}
			err := getClient().Create(ctx, testSecret)
			Expect(err).NotTo(HaveOccurred(), "Create the test secret for ConfigMap tests")
		})

		AfterEach(func() {
			// Cleanup test resource if it exists
			if testResource != nil && testResource.GetName() != "" {
				err := getClient().Delete(ctx, testResource)
				Expect(client.IgnoreNotFound(err)).To(Succeed(), "Cleanup test resource")
			}

			// Cleanup test secret
			if testSecret != nil {
				err := getClient().Delete(ctx, testSecret)
				Expect(client.IgnoreNotFound(err)).To(Succeed(), "Cleanup test secret")
			}
		})

		It("should not create ConfigMap when enabled is false", func() {
			By("creating Test resource with ConfigMap disabled")
			testResource = resourceFactory("test-disabled-"+uuid.NewString()[:8], getTestNamespace().Name)

			spec := GenericTestSpec{
				Dependencies: testv1.TestDependencies{
					Secret: testv1.SecretDependency{
						Name:      testSecret.Name,
						Namespace: testSecret.Namespace,
					},
				},
				ConfigMap: testv1.ConfigMapSpec{
					Enabled: false,
					Name:    originalConfigMapName,
					Data:    originalConfigMapData,
				},
			}
			testResource.SetSpec(spec)

			err := getClient().Create(ctx, testResource)
			Expect(err).NotTo(HaveOccurred(), "Create the Test resource")

			By("verifying ConfigMap is not created")
			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      originalConfigMapName,
					Namespace: testResource.GetNamespace(),
				},
			}

			Consistently(func(g Gomega) {
				err := getClient().Get(ctx, client.ObjectKeyFromObject(cm), cm)
				g.Expect(err).To(HaveOccurred(), "ConfigMap should not exist")
				g.Expect(client.IgnoreNotFound(err)).To(Succeed(), "ConfigMap should not exist")
			}, 10*time.Second, time.Second).Should(Succeed())

			By("verifying Test status reflects no ConfigMap and no condition")
			Eventually(func(g Gomega) {
				err := getClient().Get(ctx, client.ObjectKeyFromObject(testResource), testResource)
				g.Expect(err).NotTo(HaveOccurred(), "Get Test resource")
				status := testResource.GetStatus()
				g.Expect(status.ConfigMapStatus).To(
					Or(BeNil(), BeEquivalentTo(&testv1.ConfigMapStatus{})),
					"ConfigMapStatus should be nil or empty when disabled",
				)

				// Verify ConfigMap condition is not present
				var configMapCondition *metav1.Condition
				for _, cond := range status.Conditions {
					if cond.Type == "ConfigMap" {
						configMapCondition = &cond
						break
					}
				}
				g.Expect(configMapCondition).To(BeNil(), "ConfigMap condition should not exist when disabled")
			}, 10*time.Second, 500*time.Millisecond).Should(Succeed())
		})

		It("should create ConfigMap when enabled is true with proper fields and status", func() {
			By("creating Test resource with ConfigMap enabled")
			testResource = resourceFactory("test-enabled-"+uuid.NewString()[:8], getTestNamespace().Name)

			spec := GenericTestSpec{
				Dependencies: testv1.TestDependencies{
					Secret: testv1.SecretDependency{
						Name:      testSecret.Name,
						Namespace: testSecret.Namespace,
					},
				},
				ConfigMap: testv1.ConfigMapSpec{
					Enabled: true,
					Name:    originalConfigMapName,
					Data:    originalConfigMapData,
				},
			}
			testResource.SetSpec(spec)

			err := getClient().Create(ctx, testResource)
			Expect(err).NotTo(HaveOccurred(), "Create the Test resource")

			By("verifying ConfigMap is created with correct data")
			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      originalConfigMapName,
					Namespace: testResource.GetNamespace(),
				},
			}

			Eventually(func(g Gomega) {
				err := getClient().Get(ctx, client.ObjectKeyFromObject(cm), cm)
				g.Expect(err).NotTo(HaveOccurred(), "Get the ConfigMap")
				g.Expect(cm.Data).To(Equal(originalConfigMapData), "ConfigMap data should match")
			}, 10*time.Second, 500*time.Millisecond).Should(Succeed())

			By("verifying Test status is properly filled with correct generation")
			Eventually(func(g Gomega) {
				err := getClient().Get(ctx, client.ObjectKeyFromObject(testResource), testResource)
				g.Expect(err).NotTo(HaveOccurred(), "Get Test resource")
				status := testResource.GetStatus()
				g.Expect(status.ConfigMapStatus).NotTo(BeNil(), "ConfigMapStatus should not be nil")
				g.Expect(status.ConfigMapStatus.Name).To(Equal(originalConfigMapName), "Status should reflect ConfigMap name")

				// Check for ConfigMap condition with proper generation
				var configMapCondition *metav1.Condition
				for _, cond := range status.Conditions {
					if cond.Type == "ConfigMap" {
						configMapCondition = &cond
						break
					}
				}
				g.Expect(configMapCondition).NotTo(BeNil(), "ConfigMap condition should exist")
				g.Expect(configMapCondition.Status).To(Equal(metav1.ConditionTrue), "ConfigMap condition should be True")
				g.Expect(configMapCondition.Reason).To(Equal("UpToDate"), "ConfigMap condition reason should be UpToDate")
				g.Expect(configMapCondition.ObservedGeneration).To(Equal(testResource.GetGeneration()), "ConfigMap condition should have correct generation")
			}, 10*time.Second, 500*time.Millisecond).Should(Succeed())
		})

		It("should update ConfigMap data when fields are changed", func() {
			By("creating Test resource with initial ConfigMap")
			testResource = resourceFactory("test-update-data-"+uuid.NewString()[:8], getTestNamespace().Name)

			spec := GenericTestSpec{
				Dependencies: testv1.TestDependencies{
					Secret: testv1.SecretDependency{
						Name:      testSecret.Name,
						Namespace: testSecret.Namespace,
					},
				},
				ConfigMap: testv1.ConfigMapSpec{
					Enabled: true,
					Name:    originalConfigMapName,
					Data:    originalConfigMapData,
				},
			}
			testResource.SetSpec(spec)

			err := getClient().Create(ctx, testResource)
			Expect(err).NotTo(HaveOccurred(), "Create the Test resource")

			By("waiting for initial ConfigMap to be created")
			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      originalConfigMapName,
					Namespace: testResource.GetNamespace(),
				},
			}

			Eventually(func(g Gomega) {
				err := getClient().Get(ctx, client.ObjectKeyFromObject(cm), cm)
				g.Expect(err).NotTo(HaveOccurred(), "Get the ConfigMap")
				g.Expect(cm.Data).To(Equal(originalConfigMapData), "Initial ConfigMap data should match")
			}, 10*time.Second, 500*time.Millisecond).Should(Succeed())

			By("updating ConfigMap data")
			updatedData := map[string]string{
				"key1": "updated-value1",
				"key3": "new-value3",
			}

			originalResource := testResource.DeepCopyObject().(TestableResource)
			currentSpec := testResource.GetSpec()
			currentSpec.ConfigMap.Data = updatedData
			testResource.SetSpec(currentSpec)
			err = getClient().Patch(ctx, testResource, client.MergeFrom(originalResource))
			Expect(err).NotTo(HaveOccurred(), "Update Test resource")

			By("verifying ConfigMap data is updated")
			Eventually(func(g Gomega) {
				err := getClient().Get(ctx, client.ObjectKeyFromObject(cm), cm)
				g.Expect(err).NotTo(HaveOccurred(), "Get the updated ConfigMap")
				g.Expect(cm.Data).To(Equal(updatedData), "ConfigMap data should be updated")
			}, 10*time.Second, 500*time.Millisecond).Should(Succeed())

			By("verifying Test status remains consistent with updated generation")
			Eventually(func(g Gomega) {
				err := getClient().Get(ctx, client.ObjectKeyFromObject(testResource), testResource)
				g.Expect(err).NotTo(HaveOccurred(), "Get Test resource")
				status := testResource.GetStatus()
				g.Expect(status.ConfigMapStatus).NotTo(BeNil(), "ConfigMapStatus should not be nil")
				g.Expect(status.ConfigMapStatus.Name).To(Equal(originalConfigMapName), "Status should still reflect original ConfigMap name")

				// Check for ConfigMap condition with updated generation
				var configMapCondition *metav1.Condition
				for _, cond := range status.Conditions {
					if cond.Type == "ConfigMap" {
						configMapCondition = &cond
						break
					}
				}
				g.Expect(configMapCondition).NotTo(BeNil(), "ConfigMap condition should exist")
				g.Expect(configMapCondition.Status).To(Equal(metav1.ConditionTrue), "ConfigMap condition should be True")
				g.Expect(configMapCondition.Reason).To(Equal("UpToDate"), "ConfigMap condition reason should be UpToDate")
				g.Expect(configMapCondition.ObservedGeneration).To(Equal(testResource.GetGeneration()), "ConfigMap condition should have updated generation")
			}, 10*time.Second, 500*time.Millisecond).Should(Succeed())
		})

		It("should create new ConfigMap and delete old one when name is changed", func() {
			By("creating Test resource with initial ConfigMap")
			testResource = resourceFactory("test-rename-"+uuid.NewString()[:8], getTestNamespace().Name)

			spec := GenericTestSpec{
				Dependencies: testv1.TestDependencies{
					Secret: testv1.SecretDependency{
						Name:      testSecret.Name,
						Namespace: testSecret.Namespace,
					},
				},
				ConfigMap: testv1.ConfigMapSpec{
					Enabled: true,
					Name:    originalConfigMapName,
					Data:    originalConfigMapData,
				},
			}
			testResource.SetSpec(spec)

			err := getClient().Create(ctx, testResource)
			Expect(err).NotTo(HaveOccurred(), "Create the Test resource")

			By("waiting for initial ConfigMap to be created")
			originalCm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      originalConfigMapName,
					Namespace: testResource.GetNamespace(),
				},
			}

			Eventually(func(g Gomega) {
				err := getClient().Get(ctx, client.ObjectKeyFromObject(originalCm), originalCm)
				g.Expect(err).NotTo(HaveOccurred(), "Get the original ConfigMap")
			}, 10*time.Second, 500*time.Millisecond).Should(Succeed())

			By("updating ConfigMap name")
			newConfigMapName := "test-cm-new-" + uuid.NewString()[:8]

			err = getClient().Get(ctx, client.ObjectKeyFromObject(testResource), testResource)
			Expect(err).NotTo(HaveOccurred(), "Get current Test resource")

			originalResource := testResource.DeepCopyObject().(TestableResource)
			currentSpec := testResource.GetSpec()
			currentSpec.ConfigMap.Name = newConfigMapName
			testResource.SetSpec(currentSpec)
			err = getClient().Patch(ctx, testResource, client.MergeFrom(originalResource))
			Expect(err).NotTo(HaveOccurred(), "Update Test resource")

			By("verifying new ConfigMap is created")
			newCm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      newConfigMapName,
					Namespace: testResource.GetNamespace(),
				},
			}

			Eventually(func(g Gomega) {
				err := getClient().Get(ctx, client.ObjectKeyFromObject(newCm), newCm)
				g.Expect(err).NotTo(HaveOccurred(), "Get the new ConfigMap")
				g.Expect(newCm.Data).To(Equal(originalConfigMapData), "New ConfigMap should have same data")
			}, 10*time.Second, 500*time.Millisecond).Should(Succeed())

			By("verifying old ConfigMap is deleted")
			Eventually(func(g Gomega) {
				err := getClient().Get(ctx, client.ObjectKeyFromObject(originalCm), originalCm)
				g.Expect(err).To(HaveOccurred(), "Original ConfigMap should be deleted")
				g.Expect(client.IgnoreNotFound(err)).To(Succeed(), "Original ConfigMap should not exist")
			}, 10*time.Second, 500*time.Millisecond).Should(Succeed())

			By("verifying Test status reflects new ConfigMap name with correct generation")
			Eventually(func(g Gomega) {
				err := getClient().Get(ctx, client.ObjectKeyFromObject(testResource), testResource)
				g.Expect(err).NotTo(HaveOccurred(), "Get Test resource")
				status := testResource.GetStatus()
				g.Expect(status.ConfigMapStatus).NotTo(BeNil(), "ConfigMapStatus should not be nil")
				g.Expect(status.ConfigMapStatus.Name).To(Equal(newConfigMapName), "Status should reflect new ConfigMap name")

				// Check for ConfigMap condition with updated generation
				var configMapCondition *metav1.Condition
				for _, cond := range status.Conditions {
					if cond.Type == "ConfigMap" {
						configMapCondition = &cond
						break
					}
				}
				g.Expect(configMapCondition).NotTo(BeNil(), "ConfigMap condition should exist")
				g.Expect(configMapCondition.Status).To(Equal(metav1.ConditionTrue), "ConfigMap condition should be True")
				g.Expect(configMapCondition.Reason).To(Equal("UpToDate"), "ConfigMap condition reason should be UpToDate")
				g.Expect(configMapCondition.ObservedGeneration).To(Equal(testResource.GetGeneration()), "ConfigMap condition should have updated generation")
			}, 10*time.Second, 500*time.Millisecond).Should(Succeed())
		})

		It("should create new ConfigMap with updated data when both name and fields are changed", func() {
			By("creating Test resource with initial ConfigMap")
			testResource = resourceFactory("test-rename-data-"+uuid.NewString()[:8], getTestNamespace().Name)

			spec := GenericTestSpec{
				Dependencies: testv1.TestDependencies{
					Secret: testv1.SecretDependency{
						Name:      testSecret.Name,
						Namespace: testSecret.Namespace,
					},
				},
				ConfigMap: testv1.ConfigMapSpec{
					Enabled: true,
					Name:    originalConfigMapName,
					Data:    originalConfigMapData,
				},
			}
			testResource.SetSpec(spec)

			err := getClient().Create(ctx, testResource)
			Expect(err).NotTo(HaveOccurred(), "Create the Test resource")

			By("waiting for initial ConfigMap to be created")
			originalCm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      originalConfigMapName,
					Namespace: testResource.GetNamespace(),
				},
			}

			Eventually(func(g Gomega) {
				err := getClient().Get(ctx, client.ObjectKeyFromObject(originalCm), originalCm)
				g.Expect(err).NotTo(HaveOccurred(), "Get the original ConfigMap")
			}, 10*time.Second, 500*time.Millisecond).Should(Succeed())

			By("updating both ConfigMap name and data")
			newConfigMapName := "test-cm-renamed-" + uuid.NewString()[:8]
			updatedData := map[string]string{
				"new-key1": "new-value1",
				"new-key2": "new-value2",
			}

			err = getClient().Get(ctx, client.ObjectKeyFromObject(testResource), testResource)
			Expect(err).NotTo(HaveOccurred(), "Get current Test resource")

			originalResource := testResource.DeepCopyObject().(TestableResource)
			currentSpec := testResource.GetSpec()
			currentSpec.ConfigMap.Name = newConfigMapName
			currentSpec.ConfigMap.Data = updatedData
			testResource.SetSpec(currentSpec)
			err = getClient().Patch(ctx, testResource, client.MergeFrom(originalResource))
			Expect(err).NotTo(HaveOccurred(), "Update Test resource")

			By("verifying new ConfigMap is created with updated data")
			newCm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      newConfigMapName,
					Namespace: testResource.GetNamespace(),
				},
			}

			Eventually(func(g Gomega) {
				err := getClient().Get(ctx, client.ObjectKeyFromObject(newCm), newCm)
				g.Expect(err).NotTo(HaveOccurred(), "Get the new ConfigMap")
				g.Expect(newCm.Data).To(Equal(updatedData), "New ConfigMap should have updated data")
			}, 10*time.Second, 500*time.Millisecond).Should(Succeed())

			By("verifying old ConfigMap is deleted")
			Eventually(func(g Gomega) {
				err := getClient().Get(ctx, client.ObjectKeyFromObject(originalCm), originalCm)
				g.Expect(err).To(HaveOccurred(), "Original ConfigMap should be deleted")
				g.Expect(client.IgnoreNotFound(err)).To(Succeed(), "Original ConfigMap should not exist")
			}, 10*time.Second, 500*time.Millisecond).Should(Succeed())

			By("verifying Test status reflects new ConfigMap with updated data and correct generation")
			Eventually(func(g Gomega) {
				err := getClient().Get(ctx, client.ObjectKeyFromObject(testResource), testResource)
				g.Expect(err).NotTo(HaveOccurred(), "Get Test resource")
				status := testResource.GetStatus()
				g.Expect(status.ConfigMapStatus).NotTo(BeNil(), "ConfigMapStatus should not be nil")
				g.Expect(status.ConfigMapStatus.Name).To(Equal(newConfigMapName), "Status should reflect new ConfigMap name")

				// Check for ConfigMap condition with updated generation
				var configMapCondition *metav1.Condition
				for _, cond := range status.Conditions {
					if cond.Type == "ConfigMap" {
						configMapCondition = &cond
						break
					}
				}
				g.Expect(configMapCondition).NotTo(BeNil(), "ConfigMap condition should exist")
				g.Expect(configMapCondition.Status).To(Equal(metav1.ConditionTrue), "ConfigMap condition should be True")
				g.Expect(configMapCondition.Reason).To(Equal("UpToDate"), "ConfigMap condition reason should be UpToDate")
				g.Expect(configMapCondition.ObservedGeneration).To(Equal(testResource.GetGeneration()), "ConfigMap condition should have updated generation")
			}, 10*time.Second, 500*time.Millisecond).Should(Succeed())
		})

		It("should delete ConfigMap when disabled", func() {
			By("creating Test resource with ConfigMap enabled")
			testResource = resourceFactory("test-disable-"+uuid.NewString()[:8], getTestNamespace().Name)

			spec := GenericTestSpec{
				Dependencies: testv1.TestDependencies{
					Secret: testv1.SecretDependency{
						Name:      testSecret.Name,
						Namespace: testSecret.Namespace,
					},
				},
				ConfigMap: testv1.ConfigMapSpec{
					Enabled: true,
					Name:    originalConfigMapName,
					Data:    originalConfigMapData,
				},
			}
			testResource.SetSpec(spec)

			err := getClient().Create(ctx, testResource)
			Expect(err).NotTo(HaveOccurred(), "Create the Test resource")

			By("waiting for ConfigMap to be created")
			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      originalConfigMapName,
					Namespace: testResource.GetNamespace(),
				},
			}

			Eventually(func(g Gomega) {
				err := getClient().Get(ctx, client.ObjectKeyFromObject(cm), cm)
				g.Expect(err).NotTo(HaveOccurred(), "Get the ConfigMap")
			}, 10*time.Second, 500*time.Millisecond).Should(Succeed())

			By("disabling ConfigMap")
			err = getClient().Get(ctx, client.ObjectKeyFromObject(testResource), testResource)
			Expect(err).NotTo(HaveOccurred(), "Get current Test resource")

			originalResource := testResource.DeepCopyObject().(TestableResource)
			currentSpec := testResource.GetSpec()
			currentSpec.ConfigMap.Enabled = false
			testResource.SetSpec(currentSpec)
			err = getClient().Patch(ctx, testResource, client.MergeFrom(originalResource))
			Expect(err).NotTo(HaveOccurred(), "Update Test resource")

			By("verifying ConfigMap is deleted")
			Eventually(func(g Gomega) {
				err := getClient().Get(ctx, client.ObjectKeyFromObject(cm), cm)
				g.Expect(err).To(HaveOccurred(), "ConfigMap should be deleted")
				g.Expect(client.IgnoreNotFound(err)).To(Succeed(), "ConfigMap should not exist")
			}, 10*time.Second, 500*time.Millisecond).Should(Succeed())

			By("verifying Test status reflects no ConfigMap and condition is removed")
			Eventually(func(g Gomega) {
				err := getClient().Get(ctx, client.ObjectKeyFromObject(testResource), testResource)
				g.Expect(err).NotTo(HaveOccurred(), "Get Test resource")
				status := testResource.GetStatus()
				g.Expect(status.ConfigMapStatus).To(
					Or(BeNil(), BeEquivalentTo(&testv1.ConfigMapStatus{})),
					"ConfigMapStatus should be nil or empty when disabled",
				)

				// Verify ConfigMap condition is removed
				var configMapCondition *metav1.Condition
				for _, cond := range status.Conditions {
					if cond.Type == "ConfigMap" {
						configMapCondition = &cond
						break
					}
				}
				g.Expect(configMapCondition).To(BeNil(), "ConfigMap condition should be removed when disabled")
			}, 10*time.Second, 500*time.Millisecond).Should(Succeed())
		})

		It("should delete ConfigMap when disabled and name is changed simultaneously", func() {
			By("creating Test resource with ConfigMap enabled")
			testResource = resourceFactory("test-disable-rename-"+uuid.NewString()[:8], getTestNamespace().Name)

			spec := GenericTestSpec{
				Dependencies: testv1.TestDependencies{
					Secret: testv1.SecretDependency{
						Name:      testSecret.Name,
						Namespace: testSecret.Namespace,
					},
				},
				ConfigMap: testv1.ConfigMapSpec{
					Enabled: true,
					Name:    originalConfigMapName,
					Data:    originalConfigMapData,
				},
			}
			testResource.SetSpec(spec)

			err := getClient().Create(ctx, testResource)
			Expect(err).NotTo(HaveOccurred(), "Create the Test resource")

			By("waiting for ConfigMap to be created")
			originalCm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      originalConfigMapName,
					Namespace: testResource.GetNamespace(),
				},
			}

			Eventually(func(g Gomega) {
				err := getClient().Get(ctx, client.ObjectKeyFromObject(originalCm), originalCm)
				g.Expect(err).NotTo(HaveOccurred(), "Get the original ConfigMap")
			}, 10*time.Second, 500*time.Millisecond).Should(Succeed())

			By("disabling ConfigMap and changing name simultaneously")
			newConfigMapName := "test-cm-disabled-" + uuid.NewString()[:8]

			err = getClient().Get(ctx, client.ObjectKeyFromObject(testResource), testResource)
			Expect(err).NotTo(HaveOccurred(), "Get current Test resource")

			originalResource := testResource.DeepCopyObject().(TestableResource)
			currentSpec := testResource.GetSpec()
			currentSpec.ConfigMap.Enabled = false
			currentSpec.ConfigMap.Name = newConfigMapName
			testResource.SetSpec(currentSpec)
			err = getClient().Patch(ctx, testResource, client.MergeFrom(originalResource))
			Expect(err).NotTo(HaveOccurred(), "Update Test resource")

			By("verifying original ConfigMap is deleted")
			Eventually(func(g Gomega) {
				err := getClient().Get(ctx, client.ObjectKeyFromObject(originalCm), originalCm)
				g.Expect(err).To(HaveOccurred(), "Original ConfigMap should be deleted")
				g.Expect(client.IgnoreNotFound(err)).To(Succeed(), "Original ConfigMap should not exist")
			}, 10*time.Second, 500*time.Millisecond).Should(Succeed())

			By("verifying new ConfigMap is not created")
			newCm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      newConfigMapName,
					Namespace: testResource.GetNamespace(),
				},
			}

			Consistently(func(g Gomega) {
				err := getClient().Get(ctx, client.ObjectKeyFromObject(newCm), newCm)
				g.Expect(err).To(HaveOccurred(), "New ConfigMap should not be created when disabled")
				g.Expect(client.IgnoreNotFound(err)).To(Succeed(), "New ConfigMap should not exist")
			}, 10*time.Second, time.Second).Should(Succeed())

			By("verifying Test status reflects no ConfigMap and proper cleanup")
			Eventually(func(g Gomega) {
				err := getClient().Get(ctx, client.ObjectKeyFromObject(testResource), testResource)
				g.Expect(err).NotTo(HaveOccurred(), "Get Test resource")
				status := testResource.GetStatus()
				g.Expect(status.ConfigMapStatus).To(
					Or(BeNil(), BeEquivalentTo(&testv1.ConfigMapStatus{})),
					"ConfigMapStatus should be nil or empty when disabled",
				)

				// Verify ConfigMap condition is removed
				var configMapCondition *metav1.Condition
				for _, cond := range status.Conditions {
					if cond.Type == "ConfigMap" {
						configMapCondition = &cond
						break
					}
				}
				g.Expect(configMapCondition).To(BeNil(), "ConfigMap condition should be removed when disabled")
			}, 10*time.Second, 500*time.Millisecond).Should(Succeed())
		})
	})
}
