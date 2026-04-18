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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	testv1 "operator/api/v1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DependencyLifecycleTests verifies managed-by bookkeeping for dependency adoption and cleanup.
func DependencyLifecycleTests(getClient func() client.Client, ctx context.Context, getTestNamespace func() corev1.Namespace, resourceFactory ResourceFactory, resourceTypeName string) {
	Context(fmt.Sprintf("Dependency Lifecycle (%s)", resourceTypeName), func() {
		var testResource TestableResource
		var secret *corev1.Secret

		BeforeEach(func() {
			secret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-managed-secret-" + uuid.NewString()[:8],
					Namespace: getTestNamespace().Name,
				},
				Data: map[string][]byte{
					"ready": []byte("true"),
					"data":  []byte("managed"),
				},
			}
			Expect(getClient().Create(ctx, secret)).To(Succeed(), "Create dependency secret")
		})

		AfterEach(func() {
			if testResource != nil && testResource.GetName() != "" {
				err := getClient().Delete(ctx, testResource)
				Expect(client.IgnoreNotFound(err)).To(Succeed(), "Cleanup test resource")
			}

			if secret != nil {
				err := getClient().Delete(ctx, secret)
				Expect(client.IgnoreNotFound(err)).To(Succeed(), "Cleanup secret")
			}
		})

		It("should add and remove managed-by references when the custom resource is deleted", func() {
			testResource = resourceFactory("test-managed-by-"+uuid.NewString()[:8], getTestNamespace().Name)
			testResource.SetSpec(GenericTestSpec{
				Dependencies: testv1.TestDependencies{
					Secret: testv1.SecretDependency{
						Name:      secret.Name,
						Namespace: secret.Namespace,
					},
				},
				ConfigMap: testv1.ConfigMapSpec{
					Enabled: false,
				},
			})

			Expect(getClient().Create(ctx, testResource)).To(Succeed(), "Create the test resource")

			By("verifying the dependency cleanup finalizer is present")
			Eventually(func(g Gomega) {
				err := getClient().Get(ctx, client.ObjectKeyFromObject(testResource), testResource)
				g.Expect(err).NotTo(HaveOccurred(), "Get test resource")
				g.Expect(testResource.GetFinalizers()).To(ContainElement(ctrlfwk.FinalizerDependenciesManagedBy))
			}, 20*time.Second, 500*time.Millisecond).Should(Succeed())

			By("verifying the dependency secret is annotated with the managing custom resource")
			Eventually(func(g Gomega) {
				err := getClient().Get(ctx, client.ObjectKeyFromObject(secret), secret)
				g.Expect(err).NotTo(HaveOccurred(), "Get dependency secret")

				managedBy, err := ctrlfwk.GetManagedBy(secret)
				g.Expect(err).NotTo(HaveOccurred(), "Parse managed-by annotation")
				g.Expect(managedBy).To(HaveLen(1))
				g.Expect(managedBy[0].Name).To(Equal(testResource.GetName()))
				g.Expect(managedBy[0].Namespace).To(Equal(testResource.GetNamespace()))
				g.Expect(managedBy[0].GVK.Group).To(Equal(testv1.GroupVersion.Group))
				g.Expect(managedBy[0].GVK.Version).To(Equal(testv1.GroupVersion.Version))
				g.Expect(managedBy[0].GVK.Kind).To(Equal(resourceTypeName))
			}, 20*time.Second, 500*time.Millisecond).Should(Succeed())

			By("deleting the custom resource")
			Expect(getClient().Delete(ctx, testResource)).To(Succeed(), "Delete the test resource")

			By("waiting for the custom resource to be fully finalized and removed")
			Eventually(func() bool {
				err := getClient().Get(ctx, client.ObjectKeyFromObject(testResource), testResource)
				return apierrors.IsNotFound(err)
			}, 35*time.Second, 500*time.Millisecond).Should(BeTrue())

			By("verifying the managed-by annotation is removed from the dependency secret")
			Eventually(func(g Gomega) {
				err := getClient().Get(ctx, client.ObjectKeyFromObject(secret), secret)
				g.Expect(err).NotTo(HaveOccurred(), "Get dependency secret after finalization")

				managedBy, err := ctrlfwk.GetManagedBy(secret)
				g.Expect(err).NotTo(HaveOccurred(), "Parse managed-by annotation after finalization")
				g.Expect(managedBy).To(BeEmpty())
				g.Expect(secret.GetAnnotations()).NotTo(HaveKey(ctrlfwk.AnnotationRef))
			}, 35*time.Second, 500*time.Millisecond).Should(Succeed())
		})
	})
}
