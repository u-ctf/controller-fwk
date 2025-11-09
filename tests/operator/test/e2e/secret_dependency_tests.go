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
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"

	testv1 "operator/api/v1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SecretDependencyManagementTests contains all the Secret Dependency-related tests
func SecretDependencyManagementTests(getClient func() client.Client, ctx context.Context, getTestNamespace func() corev1.Namespace, resourceFactory ResourceFactory, resourceTypeName string) {
	Context(fmt.Sprintf("Secret Dependency Management (%s)", resourceTypeName), func() {
		var testResource TestableResource
		var secretName string
		var secretNamespace string

		BeforeEach(func() {
			secretName = "test-secret-" + uuid.NewString()[:8]
			secretNamespace = getTestNamespace().Name
		})

		AfterEach(func() {
			// Cleanup test resource if it exists
			if testResource != nil && testResource.GetName() != "" {
				err := getClient().Delete(ctx, testResource)
				Expect(client.IgnoreNotFound(err)).To(Succeed(), "Cleanup test resource")
			}

			// Cleanup secret if it exists
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: secretNamespace,
				},
			}
			err := getClient().Delete(ctx, secret)
			Expect(client.IgnoreNotFound(err)).To(Succeed(), "Cleanup secret")
		})

		It("should set SecretFound condition to False when secret is missing", func() {
			By("creating Test resource with secret dependency when secret doesn't exist")
			testResource = resourceFactory("test-secret-missing-"+uuid.NewString()[:8], getTestNamespace().Name)

			spec := GenericTestSpec{
				Dependencies: testv1.TestDependencies{
					Secret: testv1.SecretDependency{
						Name:      secretName,
						Namespace: secretNamespace,
					},
				},
				ConfigMap: testv1.ConfigMapSpec{
					Enabled: false,
				},
			}
			testResource.SetSpec(spec)

			err := getClient().Create(ctx, testResource)
			Expect(err).NotTo(HaveOccurred(), "Create the Test resource")

			By("verifying SecretFound condition is set to False")
			Eventually(func(g Gomega) {
				err := getClient().Get(ctx, client.ObjectKeyFromObject(testResource), testResource)
				g.Expect(err).NotTo(HaveOccurred(), "Get Test resource")

				status := testResource.GetStatus()
				secretFoundCondition := meta.FindStatusCondition(status.Conditions, "SecretFound")
				g.Expect(secretFoundCondition).NotTo(BeNil(), "SecretFound condition should exist")
				g.Expect(secretFoundCondition.Status).To(Equal(metav1.ConditionFalse), "SecretFound condition should be False")
				g.Expect(secretFoundCondition.Reason).To(Equal("SecretNotFound"), "SecretFound condition reason should be SecretNotFound")
				g.Expect(secretFoundCondition.ObservedGeneration).To(Equal(testResource.GetGeneration()), "SecretFound condition should have correct generation")
			}, 10*time.Second, 500*time.Millisecond).Should(Succeed())
		})

		It("should set SecretFound condition to False when secret exists but is not ready", func() {
			By("creating a secret that is not ready")
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: secretNamespace,
				},
				Data: map[string][]byte{
					"data": []byte("test-data"),
				},
			}
			err := getClient().Create(ctx, secret)
			Expect(err).NotTo(HaveOccurred(), "Create the not-ready secret")

			By("creating Test resource with secret dependency")
			testResource = resourceFactory("test-secret-not-ready-"+uuid.NewString()[:8], getTestNamespace().Name)

			spec := GenericTestSpec{
				Dependencies: testv1.TestDependencies{
					Secret: testv1.SecretDependency{
						Name:      secretName,
						Namespace: secretNamespace,
					},
				},
				ConfigMap: testv1.ConfigMapSpec{
					Enabled: false,
				},
			}
			testResource.SetSpec(spec)

			err = getClient().Create(ctx, testResource)
			Expect(err).NotTo(HaveOccurred(), "Create the Test resource")

			By("verifying SecretFound condition is set to False")
			Eventually(func(g Gomega) {
				err := getClient().Get(ctx, client.ObjectKeyFromObject(testResource), testResource)
				g.Expect(err).NotTo(HaveOccurred(), "Get Test resource")

				status := testResource.GetStatus()
				secretFoundCondition := meta.FindStatusCondition(status.Conditions, "SecretFound")
				g.Expect(secretFoundCondition).NotTo(BeNil(), "SecretFound condition should exist")
				g.Expect(secretFoundCondition.Status).To(Equal(metav1.ConditionFalse), "SecretFound condition should be False")
				g.Expect(secretFoundCondition.Reason).To(Equal("SecretNotReady"), "SecretFound condition reason should be SecretNotReady")
				g.Expect(secretFoundCondition.ObservedGeneration).To(Equal(testResource.GetGeneration()), "SecretFound condition should have correct generation")
			}, 10*time.Second, 500*time.Millisecond).Should(Succeed())
		})

		It("should not set SecretFound condition when secret exists and is ready", func() {
			By("creating a ready secret")
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: secretNamespace,
				},
				Data: map[string][]byte{
					"ready": []byte("true"),
					"data":  []byte("test-data"),
				},
			}
			err := getClient().Create(ctx, secret)
			Expect(err).NotTo(HaveOccurred(), "Create the ready secret")

			By("creating Test resource with secret dependency")
			testResource = resourceFactory("test-secret-ready-"+uuid.NewString()[:8], getTestNamespace().Name)

			spec := GenericTestSpec{
				Dependencies: testv1.TestDependencies{
					Secret: testv1.SecretDependency{
						Name:      secretName,
						Namespace: secretNamespace,
					},
				},
				ConfigMap: testv1.ConfigMapSpec{
					Enabled: false,
				},
			}
			testResource.SetSpec(spec)

			err = getClient().Create(ctx, testResource)
			Expect(err).NotTo(HaveOccurred(), "Create the Test resource")

			By("checking that the resource is Ready")
			Eventually(func(g Gomega) {
				err := getClient().Get(ctx, client.ObjectKeyFromObject(testResource), testResource)
				g.Expect(err).NotTo(HaveOccurred(), "Get Test resource")

				status := testResource.GetStatus()
				readyCondition := meta.FindStatusCondition(status.Conditions, "Ready")
				g.Expect(readyCondition).NotTo(BeNil(), "Ready condition should exist")
				g.Expect(readyCondition.Status).To(Equal(metav1.ConditionTrue), "Ready condition should be True")
				g.Expect(readyCondition.ObservedGeneration).To(Equal(testResource.GetGeneration()), "Ready condition should have correct generation")
			}, 10*time.Second, 500*time.Millisecond).Should(Succeed())
		})

		It("should update SecretFound condition from False to not existent when secret becomes ready", func() {
			By("creating Test resource with secret dependency when secret doesn't exist")
			testResource = resourceFactory("test-secret-transition-"+uuid.NewString()[:8], getTestNamespace().Name)

			spec := GenericTestSpec{
				Dependencies: testv1.TestDependencies{
					Secret: testv1.SecretDependency{
						Name:      secretName,
						Namespace: secretNamespace,
					},
				},
				ConfigMap: testv1.ConfigMapSpec{
					Enabled: false,
				},
			}
			testResource.SetSpec(spec)

			err := getClient().Create(ctx, testResource)
			Expect(err).NotTo(HaveOccurred(), "Create the Test resource")

			By("verifying SecretFound condition is initially False")
			Eventually(func(g Gomega) {
				err := getClient().Get(ctx, client.ObjectKeyFromObject(testResource), testResource)
				g.Expect(err).NotTo(HaveOccurred(), "Get Test resource")

				status := testResource.GetStatus()
				secretFoundCondition := meta.FindStatusCondition(status.Conditions, "SecretFound")
				g.Expect(secretFoundCondition).NotTo(BeNil(), "SecretFound condition should exist")
				g.Expect(secretFoundCondition.Status).To(Equal(metav1.ConditionFalse), "SecretFound condition should be False")
				g.Expect(secretFoundCondition.Reason).To(Equal("SecretNotFound"), "SecretFound condition reason should be SecretNotFound")
			}, 10*time.Second, 500*time.Millisecond).Should(Succeed())

			By("creating a ready secret")
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: secretNamespace,
				},
				Data: map[string][]byte{
					"ready": []byte("true"),
					"data":  []byte("test-data"),
				},
			}
			err = getClient().Create(ctx, secret)
			Expect(err).NotTo(HaveOccurred(), "Create the ready secret")

			By("verifying SecretFound condition goes away")
			Eventually(func(g Gomega) {
				err := getClient().Get(ctx, client.ObjectKeyFromObject(testResource), testResource)
				g.Expect(err).NotTo(HaveOccurred(), "Get Test resource")

				status := testResource.GetStatus()
				secretFoundCondition := meta.FindStatusCondition(status.Conditions, "SecretFound")
				g.Expect(secretFoundCondition).To(BeNil(), "SecretFound condition should not exist")
			}, 35*time.Second, 500*time.Millisecond).Should(Succeed())

			By("checking that the resource is Ready")
			Eventually(func(g Gomega) {
				err := getClient().Get(ctx, client.ObjectKeyFromObject(testResource), testResource)
				g.Expect(err).NotTo(HaveOccurred(), "Get Test resource")

				status := testResource.GetStatus()
				readyCondition := meta.FindStatusCondition(status.Conditions, "Ready")
				g.Expect(readyCondition).NotTo(BeNil(), "Ready condition should exist")
				g.Expect(readyCondition.Status).To(Equal(metav1.ConditionTrue), "Ready condition should be True")
				g.Expect(readyCondition.ObservedGeneration).To(Equal(testResource.GetGeneration()), "Ready condition should have correct generation")
			}, 10*time.Second, 500*time.Millisecond).Should(Succeed())
		})

		It("should update SecretFound condition from True to False when secret becomes not ready", func() {
			By("creating a ready secret")
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: secretNamespace,
				},
				Data: map[string][]byte{
					"ready": []byte("true"),
					"data":  []byte("test-data"),
				},
			}
			err := getClient().Create(ctx, secret)
			Expect(err).NotTo(HaveOccurred(), "Create the ready secret")

			By("creating Test resource with secret dependency")
			testResource = resourceFactory("test-secret-ready-to-notready-"+uuid.NewString()[:8], getTestNamespace().Name)

			spec := GenericTestSpec{
				Dependencies: testv1.TestDependencies{
					Secret: testv1.SecretDependency{
						Name:      secretName,
						Namespace: secretNamespace,
					},
				},
				ConfigMap: testv1.ConfigMapSpec{
					Enabled: false,
				},
			}
			testResource.SetSpec(spec)

			err = getClient().Create(ctx, testResource)
			Expect(err).NotTo(HaveOccurred(), "Create the Test resource")

			By("verifying SecretFound condition is initially not here")
			Eventually(func(g Gomega) {
				err := getClient().Get(ctx, client.ObjectKeyFromObject(testResource), testResource)
				g.Expect(err).NotTo(HaveOccurred(), "Get Test resource")

				status := testResource.GetStatus()
				secretFoundCondition := meta.FindStatusCondition(status.Conditions, "SecretFound")
				g.Expect(secretFoundCondition).To(BeNil(), "SecretFound condition should not exist")
			}, 10*time.Second, 500*time.Millisecond).Should(Succeed())

			By("checking that the resource is Ready")
			Eventually(func(g Gomega) {
				err := getClient().Get(ctx, client.ObjectKeyFromObject(testResource), testResource)
				g.Expect(err).NotTo(HaveOccurred(), "Get Test resource")

				status := testResource.GetStatus()
				readyCondition := meta.FindStatusCondition(status.Conditions, "Ready")
				g.Expect(readyCondition).NotTo(BeNil(), "Ready condition should exist")
				g.Expect(readyCondition.Status).To(Equal(metav1.ConditionTrue), "Ready condition should be True")
				g.Expect(readyCondition.ObservedGeneration).To(Equal(testResource.GetGeneration()), "Ready condition should have correct generation")
			}, 10*time.Second, 500*time.Millisecond).Should(Succeed())

			By("updating secret to not ready")
			err = getClient().Get(ctx, client.ObjectKeyFromObject(secret), secret)
			Expect(err).NotTo(HaveOccurred(), "Get current secret")

			originalSecret := secret.DeepCopy()
			delete(secret.Data, "ready")
			err = getClient().Patch(ctx, secret, client.MergeFrom(originalSecret))
			Expect(err).NotTo(HaveOccurred(), "Update secret to not ready")

			By("verifying SecretFound condition transitions to False")
			Eventually(func(g Gomega) {
				err := getClient().Get(ctx, client.ObjectKeyFromObject(testResource), testResource)
				g.Expect(err).NotTo(HaveOccurred(), "Get Test resource")

				status := testResource.GetStatus()
				secretFoundCondition := meta.FindStatusCondition(status.Conditions, "SecretFound")
				g.Expect(secretFoundCondition).NotTo(BeNil(), "SecretFound condition should exist")
				g.Expect(secretFoundCondition.Status).To(Equal(metav1.ConditionFalse), "SecretFound condition should be False")
				g.Expect(secretFoundCondition.Reason).To(Equal("SecretNotReady"), "SecretFound condition reason should be SecretNotReady")
				g.Expect(secretFoundCondition.ObservedGeneration).To(Equal(testResource.GetGeneration()), "SecretFound condition should have correct generation")
			}, 10*time.Second, 500*time.Millisecond).Should(Succeed())
		})

		It("should update SecretFound condition from True to False when secret is deleted", func() {
			By("creating a ready secret")
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: secretNamespace,
				},
				Data: map[string][]byte{
					"ready": []byte("true"),
					"data":  []byte("test-data"),
				},
			}
			err := getClient().Create(ctx, secret)
			Expect(err).NotTo(HaveOccurred(), "Create the ready secret")

			By("creating Test resource with secret dependency")
			testResource = resourceFactory("test-secret-deleted-"+uuid.NewString()[:8], getTestNamespace().Name)

			spec := GenericTestSpec{
				Dependencies: testv1.TestDependencies{
					Secret: testv1.SecretDependency{
						Name:      secretName,
						Namespace: secretNamespace,
					},
				},
				ConfigMap: testv1.ConfigMapSpec{
					Enabled: false,
				},
			}
			testResource.SetSpec(spec)

			err = getClient().Create(ctx, testResource)
			Expect(err).NotTo(HaveOccurred(), "Create the Test resource")

			By("verifying resource is Ready initially")
			Eventually(func(g Gomega) {
				err := getClient().Get(ctx, client.ObjectKeyFromObject(testResource), testResource)
				g.Expect(err).NotTo(HaveOccurred(), "Get Test resource")

				status := testResource.GetStatus()
				readyCondition := meta.FindStatusCondition(status.Conditions, "Ready")
				g.Expect(readyCondition).NotTo(BeNil(), "Ready condition should exist")
				g.Expect(readyCondition.Status).To(Equal(metav1.ConditionTrue), "Ready condition should be True")
			}, 10*time.Second, 500*time.Millisecond).Should(Succeed())

			By("deleting the secret")
			err = getClient().Delete(ctx, secret)
			Expect(err).NotTo(HaveOccurred(), "Delete the secret")

			By("verifying SecretFound condition transitions to False")
			Eventually(func(g Gomega) {
				err := getClient().Get(ctx, client.ObjectKeyFromObject(testResource), testResource)
				g.Expect(err).NotTo(HaveOccurred(), "Get Test resource")

				status := testResource.GetStatus()
				secretFoundCondition := meta.FindStatusCondition(status.Conditions, "SecretFound")
				g.Expect(secretFoundCondition).NotTo(BeNil(), "SecretFound condition should exist")
				g.Expect(secretFoundCondition.Status).To(Equal(metav1.ConditionFalse), "SecretFound condition should be False")
				g.Expect(secretFoundCondition.Reason).To(Equal("SecretNotFound"), "SecretFound condition reason should be SecretNotFound")
				g.Expect(secretFoundCondition.ObservedGeneration).To(Equal(testResource.GetGeneration()), "SecretFound condition should have correct generation")
			}, 30*time.Second, 500*time.Millisecond).Should(Succeed())

			// Clear secret reference to prevent cleanup from trying to delete it again
			secret = nil
		})

		It("should handle secret dependency changes in spec", func() {
			originalSecretName := secretName + "-original"
			newSecretName := secretName + "-new"

			By("creating original ready secret")
			originalSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      originalSecretName,
					Namespace: secretNamespace,
				},
				Data: map[string][]byte{
					"ready": []byte("true"),
					"data":  []byte("original-data"),
				},
			}
			err := getClient().Create(ctx, originalSecret)
			Expect(err).NotTo(HaveOccurred(), "Create the original ready secret")

			By("creating Test resource with original secret dependency")
			testResource = resourceFactory("test-secret-dependency-change-"+uuid.NewString()[:8], getTestNamespace().Name)

			spec := GenericTestSpec{
				Dependencies: testv1.TestDependencies{
					Secret: testv1.SecretDependency{
						Name:      originalSecretName,
						Namespace: secretNamespace,
					},
				},
				ConfigMap: testv1.ConfigMapSpec{
					Enabled: false,
				},
			}
			testResource.SetSpec(spec)

			err = getClient().Create(ctx, testResource)
			Expect(err).NotTo(HaveOccurred(), "Create the Test resource")

			By("verifying resource is Ready initially")
			Eventually(func(g Gomega) {
				err := getClient().Get(ctx, client.ObjectKeyFromObject(testResource), testResource)
				g.Expect(err).NotTo(HaveOccurred(), "Get Test resource")

				status := testResource.GetStatus()
				readyCondition := meta.FindStatusCondition(status.Conditions, "Ready")
				g.Expect(readyCondition).NotTo(BeNil(), "Ready condition should exist")
				g.Expect(readyCondition.Status).To(Equal(metav1.ConditionTrue), "Ready condition should be True")
				g.Expect(readyCondition.ObservedGeneration).To(Equal(testResource.GetGeneration()), "Ready condition should have correct generation")
			}, 10*time.Second, 500*time.Millisecond).Should(Succeed())

			By("updating Test resource to depend on new secret (that doesn't exist yet)")
			err = getClient().Get(ctx, client.ObjectKeyFromObject(testResource), testResource)
			Expect(err).NotTo(HaveOccurred(), "Get current Test resource")

			originalResource := testResource.DeepCopyObject().(TestableResource)
			currentSpec := testResource.GetSpec()
			currentSpec.Dependencies.Secret.Name = newSecretName
			testResource.SetSpec(currentSpec)
			err = getClient().Patch(ctx, testResource, client.MergeFrom(originalResource))
			Expect(err).NotTo(HaveOccurred(), "Update Test resource")

			By("verifying SecretFound condition transitions to False for new secret")
			Eventually(func(g Gomega) {
				err := getClient().Get(ctx, client.ObjectKeyFromObject(testResource), testResource)
				g.Expect(err).NotTo(HaveOccurred(), "Get Test resource")

				status := testResource.GetStatus()
				secretFoundCondition := meta.FindStatusCondition(status.Conditions, "SecretFound")
				g.Expect(secretFoundCondition).NotTo(BeNil(), "SecretFound condition should exist")
				g.Expect(secretFoundCondition.Status).To(Equal(metav1.ConditionFalse), "SecretFound condition should be False")
				g.Expect(secretFoundCondition.Reason).To(Equal("SecretNotFound"), "SecretFound condition reason should be SecretNotFound")
				g.Expect(secretFoundCondition.ObservedGeneration).To(Equal(testResource.GetGeneration()), "SecretFound condition should have updated generation")
			}, 10*time.Second, 500*time.Millisecond).Should(Succeed())

			By("creating new ready secret")
			newSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      newSecretName,
					Namespace: secretNamespace,
				},
				Data: map[string][]byte{
					"ready": []byte("true"),
					"data":  []byte("new-data"),
				},
			}
			err = getClient().Create(ctx, newSecret)
			Expect(err).NotTo(HaveOccurred(), "Create the new ready secret")

			By("verifying SecretFound condition goes away for new secret")
			Eventually(func(g Gomega) {
				err := getClient().Get(ctx, client.ObjectKeyFromObject(testResource), testResource)
				g.Expect(err).NotTo(HaveOccurred(), "Get Test resource")

				status := testResource.GetStatus()
				secretFoundCondition := meta.FindStatusCondition(status.Conditions, "SecretFound")
				g.Expect(secretFoundCondition).To(BeNil(), "SecretFound condition should not exist")
			}, 35*time.Second, 500*time.Millisecond).Should(Succeed())

			By("checking that the resource is Ready")
			Eventually(func(g Gomega) {
				err := getClient().Get(ctx, client.ObjectKeyFromObject(testResource), testResource)
				g.Expect(err).NotTo(HaveOccurred(), "Get Test resource")

				status := testResource.GetStatus()
				readyCondition := meta.FindStatusCondition(status.Conditions, "Ready")
				g.Expect(readyCondition).NotTo(BeNil(), "Ready condition should exist")
				g.Expect(readyCondition.Status).To(Equal(metav1.ConditionTrue), "Ready condition should be True")
				g.Expect(readyCondition.ObservedGeneration).To(Equal(testResource.GetGeneration()), "Ready condition should have correct generation")
			}, 10*time.Second, 500*time.Millisecond).Should(Succeed())

			// Cleanup additional secrets
			err = getClient().Delete(ctx, originalSecret)
			Expect(client.IgnoreNotFound(err)).To(Succeed(), "Cleanup original secret")

			err = getClient().Delete(ctx, newSecret)
			Expect(client.IgnoreNotFound(err)).To(Succeed(), "Cleanup new secret")
		})

		It("should handle cross-namespace secret dependencies", func() {
			otherNamespaceName := "test-other-ns-" + uuid.NewString()[:8]

			By("creating another namespace")
			otherNamespace := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: otherNamespaceName,
				},
			}
			err := getClient().Create(ctx, otherNamespace)
			Expect(err).NotTo(HaveOccurred(), "Create other namespace")

			By("creating a ready secret in other namespace")
			crossNsSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: otherNamespaceName,
				},
				Data: map[string][]byte{
					"ready": []byte("true"),
					"data":  []byte("cross-ns-data"),
				},
			}
			err = getClient().Create(ctx, crossNsSecret)
			Expect(err).NotTo(HaveOccurred(), "Create the cross-namespace ready secret")

			By("creating Test resource with cross-namespace secret dependency")
			testResource = resourceFactory("test-secret-cross-ns-"+uuid.NewString()[:8], getTestNamespace().Name)

			spec := GenericTestSpec{
				Dependencies: testv1.TestDependencies{
					Secret: testv1.SecretDependency{
						Name:      secretName,
						Namespace: otherNamespaceName,
					},
				},
				ConfigMap: testv1.ConfigMapSpec{
					Enabled: false,
				},
			}
			testResource.SetSpec(spec)

			err = getClient().Create(ctx, testResource)
			Expect(err).NotTo(HaveOccurred(), "Create the Test resource")

			By("checking that the resource is Ready with cross-namespace secret")
			Eventually(func(g Gomega) {
				err := getClient().Get(ctx, client.ObjectKeyFromObject(testResource), testResource)
				g.Expect(err).NotTo(HaveOccurred(), "Get Test resource")

				status := testResource.GetStatus()
				readyCondition := meta.FindStatusCondition(status.Conditions, "Ready")
				g.Expect(readyCondition).NotTo(BeNil(), "Ready condition should exist")
				g.Expect(readyCondition.Status).To(Equal(metav1.ConditionTrue), "Ready condition should be True")
				g.Expect(readyCondition.ObservedGeneration).To(Equal(testResource.GetGeneration()), "Ready condition should have correct generation")
			}, 10*time.Second, 500*time.Millisecond).Should(Succeed())

			// Cleanup cross-namespace resources
			err = getClient().Delete(ctx, crossNsSecret)
			Expect(client.IgnoreNotFound(err)).To(Succeed(), "Cleanup cross-namespace secret")

			err = getClient().Delete(ctx, otherNamespace)
			Expect(client.IgnoreNotFound(err)).To(Succeed(), "Cleanup other namespace")
		})

	})
}
