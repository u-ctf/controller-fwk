/*
Copyright 2025.

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
	testv1 "operator/api/v1"
	"operator/internal/testlabels"

	ctrlfwk "github.com/u-ctf/controller-fwk"
	corev1 "k8s.io/api/core/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("UntypedTest Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"
		const secretName = "test-resource-secret"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}
		secretNamespacedName := types.NamespacedName{
			Name:      secretName,
			Namespace: "default",
		}
		untypedtest := &testv1.UntypedTest{}

		BeforeEach(func() {
			By("creating the ready secret dependency")
			secret := &corev1.Secret{}
			err := k8sClient.Get(ctx, secretNamespacedName, secret)
			if err != nil && errors.IsNotFound(err) {
				secret = &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      secretName,
						Namespace: "default",
					},
					Data: map[string][]byte{
						"ready": []byte("true"),
					},
				}
				Expect(k8sClient.Create(ctx, secret)).To(Succeed())
			}

			By("creating the custom resource for the Kind UntypedTest")
			err = k8sClient.Get(ctx, typeNamespacedName, untypedtest)
			if err != nil && errors.IsNotFound(err) {
				resource := &testv1.UntypedTest{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: testv1.UntypedTestSpec{
						Dependencies: testv1.TestDependencies{
							Secret: testv1.SecretDependency{
								Name:      secretName,
								Namespace: "default",
							},
						},
						ConfigMap: testv1.ConfigMapSpec{
							Enabled: false,
						},
					},
				}
				testlabels.ApplyToObject(resource)
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			resource := &testv1.UntypedTest{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance UntypedTest")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())

			secret := &corev1.Secret{}
			err = k8sClient.Get(ctx, secretNamespacedName, secret)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the dependency secret")
			Expect(k8sClient.Delete(ctx, secret)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			watchCache := ctrlfwk.NewWatchCache(nil)
			watchCache.AddWatchSource(ctrlfwk.NewWatchKey(schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Secret"}, ctrlfwk.CacheTypeEnqueueForOwner))
			controllerReconciler := &UntypedTestReconciler{
				Client:        k8sClient,
				WatchCache:    watchCache,
				RuntimeScheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(k8sClient.Get(ctx, typeNamespacedName, untypedtest)).To(Succeed())
			readyCondition := meta.FindStatusCondition(untypedtest.Status.Conditions, "Ready")
			Expect(readyCondition).NotTo(BeNil())
			Expect(readyCondition.Status).To(Equal(metav1.ConditionTrue))
			Expect(untypedtest.Finalizers).To(ContainElement(ctrlfwk.FinalizerDependenciesManagedBy))
		})
	})
})
