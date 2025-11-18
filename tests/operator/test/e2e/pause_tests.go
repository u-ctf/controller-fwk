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
	ctrlfwk "github.com/u-ctf/controller-fwk"
	"sigs.k8s.io/controller-runtime/pkg/client"

	testv1 "operator/api/v1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// PauseLabelKey is the label used to pause reconciliation
	PauseLabelKey = ctrlfwk.LabelReconciliationPaused
)

// PauseTests adds tests for pausing reconciliation on resources that support it.
func PauseTests(getClient func() client.Client, ctx context.Context, getTestNamespace func() corev1.Namespace, resourceFactory ResourceFactory, resourceTypeName string) {
	Context(fmt.Sprintf("Pause Management (%s)", resourceTypeName), func() {
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
			// Cleanup labels to avoid interference with deletion
			if testResource != nil {
				labels := testResource.GetLabels()
				if labels != nil {
					delete(labels, PauseLabelKey)
					testResource.SetLabels(labels)
					err := getClient().Update(ctx, testResource)
					Expect(err).NotTo(HaveOccurred(), "Cleanup pause label from test resource")
				}
			}

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

		It("should not reconcile when resource has pause label", func() {
			By("creating Test resource with pause label")
			testResource = resourceFactory("test-paused-"+uuid.NewString()[:8], getTestNamespace().Name)

			// Add pause label to prevent reconciliation
			labels := map[string]string{
				PauseLabelKey: "test-pause",
			}
			testResource.SetLabels(labels)

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

			By("verifying ConfigMap is not created due to pause")
			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      originalConfigMapName,
					Namespace: testResource.GetNamespace(),
				},
			}

			Consistently(func(g Gomega) {
				err := getClient().Get(ctx, client.ObjectKeyFromObject(cm), cm)
				g.Expect(err).To(HaveOccurred(), "ConfigMap should not exist while paused")
				g.Expect(client.IgnoreNotFound(err)).To(Succeed(), "ConfigMap should not exist while paused")
			}, 10*time.Second, time.Second).Should(Succeed())

			By("verifying Test resource conditions are not updated while paused")
			Consistently(func(g Gomega) {
				err := getClient().Get(ctx, client.ObjectKeyFromObject(testResource), testResource)
				g.Expect(err).NotTo(HaveOccurred(), "Get Test resource")

				status := testResource.GetStatus()
				// Should have no conditions since reconciliation is paused
				g.Expect(status.Conditions).To(BeEmpty(), "No conditions should be set while paused")
			}, 10*time.Second, time.Second).Should(Succeed())
		})

		It("should resume reconciliation when pause label is removed", func() {
			By("creating Test resource with pause label")
			testResource = resourceFactory("test-resume-"+uuid.NewString()[:8], getTestNamespace().Name)

			// Add pause label to prevent initial reconciliation
			labels := map[string]string{
				PauseLabelKey: "test-pause",
			}
			testResource.SetLabels(labels)

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

			By("verifying resource remains paused initially")
			time.Sleep(5 * time.Second) // Allow some time for potential reconciliation

			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      originalConfigMapName,
					Namespace: testResource.GetNamespace(),
				},
			}

			err = getClient().Get(ctx, client.ObjectKeyFromObject(cm), cm)
			Expect(client.IgnoreNotFound(err)).To(Succeed(), "ConfigMap should not exist while paused")

			By("removing pause label to resume reconciliation")
			err = getClient().Get(ctx, client.ObjectKeyFromObject(testResource), testResource)
			Expect(err).NotTo(HaveOccurred(), "Get Test resource")

			// Remove pause label
			labels = testResource.GetLabels()
			delete(labels, PauseLabelKey)
			testResource.SetLabels(labels)

			// Also update the spec to trigger generation change (required by GenerationChangedPredicate)
			spec = testResource.GetSpec()
			if spec.ConfigMap.Data == nil {
				spec.ConfigMap.Data = make(map[string]string)
			}
			spec.ConfigMap.Data["resume-trigger"] = "true"
			testResource.SetSpec(spec)

			err = getClient().Update(ctx, testResource)
			Expect(err).NotTo(HaveOccurred(), "Update Test resource to remove pause label")

			By("verifying ConfigMap is created after unpausing")
			Eventually(func(g Gomega) {
				err := getClient().Get(ctx, client.ObjectKeyFromObject(cm), cm)
				g.Expect(err).NotTo(HaveOccurred(), "ConfigMap should be created after unpausing")
			}, 30*time.Second, 2*time.Second).Should(Succeed())

			By("verifying ConfigMap has correct data")
			expectedData := make(map[string]string)
			for k, v := range originalConfigMapData {
				expectedData[k] = v
			}
			expectedData["resume-trigger"] = "true"
			Expect(cm.Data).To(Equal(expectedData), "ConfigMap should have correct data including resume trigger")

			By("verifying Test resource has Ready condition after unpausing")
			Eventually(func(g Gomega) {
				err := getClient().Get(ctx, client.ObjectKeyFromObject(testResource), testResource)
				g.Expect(err).NotTo(HaveOccurred(), "Get Test resource")

				status := testResource.GetStatus()
				g.Expect(status.Conditions).NotTo(BeEmpty(), "Should have conditions after unpausing")

				// Check for Ready condition
				var readyCondition *metav1.Condition
				for i := range status.Conditions {
					if status.Conditions[i].Type == "Ready" {
						readyCondition = &status.Conditions[i]
						break
					}
				}
				g.Expect(readyCondition).NotTo(BeNil(), "Should have Ready condition")
				g.Expect(readyCondition.Status).To(Equal(metav1.ConditionTrue), "Ready condition should be True")
			}, 30*time.Second, 2*time.Second).Should(Succeed())
		})

		It("should pause reconciliation when pause label is added to running resource", func() {
			By("creating Test resource without pause label")
			testResource = resourceFactory("test-pause-later-"+uuid.NewString()[:8], getTestNamespace().Name)

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

			By("verifying ConfigMap is created normally")
			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      originalConfigMapName,
					Namespace: testResource.GetNamespace(),
				},
			}

			Eventually(func(g Gomega) {
				err := getClient().Get(ctx, client.ObjectKeyFromObject(cm), cm)
				g.Expect(err).NotTo(HaveOccurred(), "ConfigMap should be created")
			}, 30*time.Second, 2*time.Second).Should(Succeed())

			By("adding pause label to running resource")
			err = getClient().Get(ctx, client.ObjectKeyFromObject(testResource), testResource)
			Expect(err).NotTo(HaveOccurred(), "Get Test resource")

			labels := testResource.GetLabels()
			if labels == nil {
				labels = make(map[string]string)
			}
			labels[PauseLabelKey] = "manual-pause"
			testResource.SetLabels(labels)

			err = getClient().Update(ctx, testResource)
			Expect(err).NotTo(HaveOccurred(), "Update Test resource to add pause label")

			By("modifying ConfigMap spec to test if changes are applied while paused")
			newConfigMapData := map[string]string{
				"key1": "updated-value1",
				"key2": "updated-value2",
				"key3": "new-value3",
			}

			// Wait a moment to ensure the pause is fully in effect
			time.Sleep(2 * time.Second)

			err = getClient().Get(ctx, client.ObjectKeyFromObject(testResource), testResource)
			Expect(err).NotTo(HaveOccurred(), "Get Test resource")

			spec = testResource.GetSpec()
			spec.ConfigMap.Data = newConfigMapData
			testResource.SetSpec(spec)

			err = getClient().Update(ctx, testResource)
			Expect(err).NotTo(HaveOccurred(), "Update Test resource spec")

			By("verifying ConfigMap is not updated while paused")
			// The spec update above should not trigger reconciliation due to NotPausedPredicate
			// So the ConfigMap should remain unchanged with original data
			Consistently(func(g Gomega) {
				err := getClient().Get(ctx, client.ObjectKeyFromObject(cm), cm)
				g.Expect(err).NotTo(HaveOccurred(), "ConfigMap should still exist")
				g.Expect(cm.Data).To(Equal(originalConfigMapData), "ConfigMap should not be updated while paused")
				g.Expect(cm.Data).NotTo(Equal(newConfigMapData), "ConfigMap should not have new data while paused")
			}, 10*time.Second, time.Second).Should(Succeed())
		})

		It("should handle pause with different label values", func() {
			By("creating Test resource with custom pause value")
			testResource = resourceFactory("test-custom-pause-"+uuid.NewString()[:8], getTestNamespace().Name)

			// Add pause label with custom value
			labels := map[string]string{
				PauseLabelKey: "maintenance-window-2024-11-14",
			}
			testResource.SetLabels(labels)

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

			By("verifying reconciliation is paused regardless of label value")
			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      originalConfigMapName,
					Namespace: testResource.GetNamespace(),
				},
			}

			Consistently(func(g Gomega) {
				err := getClient().Get(ctx, client.ObjectKeyFromObject(cm), cm)
				g.Expect(err).To(HaveOccurred(), "ConfigMap should not exist with custom pause value")
				g.Expect(client.IgnoreNotFound(err)).To(Succeed(), "ConfigMap should not exist with custom pause value")
			}, 10*time.Second, time.Second).Should(Succeed())

			By("updating pause label value without removing it")
			err = getClient().Get(ctx, client.ObjectKeyFromObject(testResource), testResource)
			Expect(err).NotTo(HaveOccurred(), "Get Test resource")

			labels = testResource.GetLabels()
			labels[PauseLabelKey] = "updated-maintenance-window"
			testResource.SetLabels(labels)

			err = getClient().Update(ctx, testResource)
			Expect(err).NotTo(HaveOccurred(), "Update Test resource pause label value")

			By("verifying reconciliation remains paused with updated label value")
			Consistently(func(g Gomega) {
				err := getClient().Get(ctx, client.ObjectKeyFromObject(cm), cm)
				g.Expect(err).To(HaveOccurred(), "ConfigMap should still not exist with updated pause value")
				g.Expect(client.IgnoreNotFound(err)).To(Succeed(), "ConfigMap should still not exist with updated pause value")
			}, 5*time.Second, time.Second).Should(Succeed())
		})

		It("should pause reconciliation when label is set to empty string", func() {
			By("creating Test resource with empty pause label value")
			testResource = resourceFactory("test-empty-pause-"+uuid.NewString()[:8], getTestNamespace().Name)

			// Add pause label with empty value - this should still pause reconciliation
			labels := map[string]string{
				PauseLabelKey: "",
			}
			testResource.SetLabels(labels)

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

			By("verifying reconciliation is paused even with empty label value")
			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      originalConfigMapName,
					Namespace: testResource.GetNamespace(),
				},
			}

			Consistently(func(g Gomega) {
				err := getClient().Get(ctx, client.ObjectKeyFromObject(cm), cm)
				g.Expect(err).To(HaveOccurred(), "ConfigMap should not exist with empty pause value")
				g.Expect(client.IgnoreNotFound(err)).To(Succeed(), "ConfigMap should not exist with empty pause value")
			}, 10*time.Second, time.Second).Should(Succeed())
		})

		It("should not update ConfigMap when it has pause label", func() {
			By("creating Test resource without pause label")
			testResource = resourceFactory("test-cm-pause-"+uuid.NewString()[:8], getTestNamespace().Name)

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

			By("verifying ConfigMap is created initially")
			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      originalConfigMapName,
					Namespace: testResource.GetNamespace(),
				},
			}

			Eventually(func(g Gomega) {
				err := getClient().Get(ctx, client.ObjectKeyFromObject(cm), cm)
				g.Expect(err).NotTo(HaveOccurred(), "ConfigMap should be created")
				g.Expect(cm.Data).To(Equal(originalConfigMapData), "ConfigMap should have original data")
			}, 30*time.Second, 2*time.Second).Should(Succeed())

			By("adding pause label to the ConfigMap")
			err = getClient().Get(ctx, client.ObjectKeyFromObject(cm), cm)
			Expect(err).NotTo(HaveOccurred(), "Get ConfigMap")

			labels := cm.GetLabels()
			if labels == nil {
				labels = make(map[string]string)
			}
			labels[PauseLabelKey] = "configmap-pause"
			cm.SetLabels(labels)

			err = getClient().Update(ctx, cm)
			Expect(err).NotTo(HaveOccurred(), "Update ConfigMap to add pause label")

			By("updating the CR spec to trigger ConfigMap changes")
			newConfigMapData := map[string]string{
				"key1":     "updated-value1",
				"key2":     "updated-value2",
				"key3":     "new-value3",
				"modified": "after-pause",
			}

			err = getClient().Get(ctx, client.ObjectKeyFromObject(testResource), testResource)
			Expect(err).NotTo(HaveOccurred(), "Get Test resource")

			spec = testResource.GetSpec()
			spec.ConfigMap.Data = newConfigMapData
			testResource.SetSpec(spec)

			err = getClient().Update(ctx, testResource)
			Expect(err).NotTo(HaveOccurred(), "Update Test resource spec")

			By("verifying ConfigMap data remains unchanged despite CR updates")
			Consistently(func(g Gomega) {
				err := getClient().Get(ctx, client.ObjectKeyFromObject(cm), cm)
				g.Expect(err).NotTo(HaveOccurred(), "ConfigMap should still exist")
				g.Expect(cm.Data).To(Equal(originalConfigMapData), "ConfigMap data should remain unchanged due to pause annotation")
				g.Expect(cm.Data).NotTo(Equal(newConfigMapData), "ConfigMap should not have new data while paused")

				// Verify the pause label is still present
				labels := cm.GetLabels()
				g.Expect(labels).NotTo(BeNil(), "ConfigMap should have labels")
				g.Expect(labels[PauseLabelKey]).To(Equal("configmap-pause"), "Pause label should be preserved")
			}, 15*time.Second, 2*time.Second).Should(Succeed())

			By("removing pause label from ConfigMap to allow updates")
			err = getClient().Get(ctx, client.ObjectKeyFromObject(cm), cm)
			Expect(err).NotTo(HaveOccurred(), "Get ConfigMap")

			labels = cm.GetLabels()
			delete(labels, PauseLabelKey)
			cm.SetLabels(labels)

			err = getClient().Update(ctx, cm)
			Expect(err).NotTo(HaveOccurred(), "Remove pause label from ConfigMap")

			By("triggering reconciliation by updating CR generation")
			err = getClient().Get(ctx, client.ObjectKeyFromObject(testResource), testResource)
			Expect(err).NotTo(HaveOccurred(), "Get Test resource")

			spec = testResource.GetSpec()
			if spec.ConfigMap.Data == nil {
				spec.ConfigMap.Data = make(map[string]string)
			}
			spec.ConfigMap.Data["resume-trigger"] = "configmap-unpaused"
			testResource.SetSpec(spec)

			err = getClient().Update(ctx, testResource)
			Expect(err).NotTo(HaveOccurred(), "Update Test resource to trigger reconciliation")

			By("verifying ConfigMap is updated after removing pause label")
			expectedData := make(map[string]string)
			for k, v := range newConfigMapData {
				expectedData[k] = v
			}
			expectedData["resume-trigger"] = "configmap-unpaused"

			Eventually(func(g Gomega) {
				err := getClient().Get(ctx, client.ObjectKeyFromObject(cm), cm)
				g.Expect(err).NotTo(HaveOccurred(), "ConfigMap should exist")
				g.Expect(cm.Data).To(Equal(expectedData), "ConfigMap should be updated after removing pause label")

				// Verify pause label is no longer present
				labels := cm.GetLabels()
				if labels != nil {
					g.Expect(labels[PauseLabelKey]).To(BeEmpty(), "Pause label should be removed")
				}
			}, 30*time.Second, 2*time.Second).Should(Succeed())
		})
	})
}
