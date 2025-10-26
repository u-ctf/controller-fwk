//go:build e2e
// +build e2e

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

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	testv1 "operator/api/v1"
	"operator/test/utils"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
)

// namespace where the project is deployed in
const namespace = "operator-system"

// serviceAccountName created for the project
const serviceAccountName = "operator-controller-manager"

// metricsServiceName is the name of the metrics service of the project
const metricsServiceName = "operator-controller-manager-metrics-service"

// metricsRoleBindingName is the name of the RBAC that will be created to allow get the metrics data
const metricsRoleBindingName = "operator-metrics-binding"

var _ = Describe("Manager", Ordered, func() {
	var controllerPodName string

	// Before running the tests, set up the environment by creating the namespace,
	// enforce the restricted security policy to the namespace, installing CRDs,
	// and deploying the controller.
	BeforeAll(func() {
		By("creating manager namespace")
		cmd := exec.Command("kubectl", "create", "ns", namespace)
		_, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to create namespace")

		By("labeling the namespace to enforce the restricted security policy")
		cmd = exec.Command("kubectl", "label", "--overwrite", "ns", namespace,
			"pod-security.kubernetes.io/enforce=restricted")
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to label namespace with restricted policy")

		By("installing CRDs")
		cmd = exec.Command("make", "install")
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to install CRDs")

		By("deploying the controller-manager")
		cmd = exec.Command("make", "deploy", fmt.Sprintf("IMG=%s", projectImage))
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to deploy the controller-manager")
	})

	// After all tests have been executed, clean up by undeploying the controller, uninstalling CRDs,
	// and deleting the namespace.
	AfterAll(func() {
		By("cleaning up the curl pod for metrics")
		cmd := exec.Command("kubectl", "delete", "pod", "curl-metrics", "-n", namespace)
		_, _ = utils.Run(cmd)

		By("undeploying the controller-manager")
		cmd = exec.Command("make", "undeploy")
		_, _ = utils.Run(cmd)

		By("uninstalling CRDs")
		cmd = exec.Command("make", "uninstall")
		_, _ = utils.Run(cmd)

		By("removing manager namespace")
		cmd = exec.Command("kubectl", "delete", "ns", namespace)
		_, _ = utils.Run(cmd)
	})

	// After each test, check for failures and collect logs, events,
	// and pod descriptions for debugging.
	AfterEach(func() {
		specReport := CurrentSpecReport()
		if specReport.Failed() {
			By("Fetching controller manager pod logs")
			cmd := exec.Command("kubectl", "logs", controllerPodName, "-n", namespace)
			controllerLogs, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Controller logs:\n %s", controllerLogs)
			} else {
				_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get Controller logs: %s", err)
			}

			By("Fetching Kubernetes events")
			cmd = exec.Command("kubectl", "get", "events", "-n", namespace, "--sort-by=.lastTimestamp")
			eventsOutput, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Kubernetes events:\n%s", eventsOutput)
			} else {
				_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get Kubernetes events: %s", err)
			}

			By("Fetching curl-metrics logs")
			cmd = exec.Command("kubectl", "logs", "curl-metrics", "-n", namespace)
			metricsOutput, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Metrics logs:\n %s", metricsOutput)
			} else {
				_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get curl-metrics logs: %s", err)
			}

			By("Fetching controller manager pod description")
			cmd = exec.Command("kubectl", "describe", "pod", controllerPodName, "-n", namespace)
			podDescription, err := utils.Run(cmd)
			if err == nil {
				fmt.Println("Pod description:\n", podDescription)
			} else {
				fmt.Println("Failed to describe controller pod")
			}
		}
	})

	SetDefaultEventuallyTimeout(1 * time.Minute)
	SetDefaultEventuallyPollingInterval(500 * time.Millisecond)

	Context("Manager", func() {
		It("should run successfully", func() {
			By("validating that the controller-manager pod is running as expected")
			verifyControllerUp := func(g Gomega) {
				// Get the name of the controller-manager pod
				cmd := exec.Command("kubectl", "get",
					"pods", "-l", "control-plane=controller-manager",
					"-o", "go-template={{ range .items }}"+
						"{{ if not .metadata.deletionTimestamp }}"+
						"{{ .metadata.name }}"+
						"{{ \"\\n\" }}{{ end }}{{ end }}",
					"-n", namespace,
				)

				podOutput, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred(), "Failed to retrieve controller-manager pod information")
				podNames := utils.GetNonEmptyLines(podOutput)
				g.Expect(podNames).To(HaveLen(1), "expected 1 controller pod running")
				controllerPodName = podNames[0]
				g.Expect(controllerPodName).To(ContainSubstring("controller-manager"))

				// Validate the pod's status
				cmd = exec.Command("kubectl", "get",
					"pods", controllerPodName, "-o", "jsonpath={.status.phase}",
					"-n", namespace,
				)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("Running"), "Incorrect controller-manager pod status")
			}
			Eventually(verifyControllerUp).Should(Succeed())
		})

		It("should ensure the metrics endpoint is serving metrics", func() {
			By("creating a ClusterRoleBinding for the service account to allow access to metrics")
			cmd := exec.Command("kubectl", "create", "clusterrolebinding", metricsRoleBindingName,
				"--clusterrole=operator-metrics-reader",
				fmt.Sprintf("--serviceaccount=%s:%s", namespace, serviceAccountName),
			)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create ClusterRoleBinding")

			By("validating that the metrics service is available")
			cmd = exec.Command("kubectl", "get", "service", metricsServiceName, "-n", namespace)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Metrics service should exist")

			By("getting the service account token")
			token, err := serviceAccountToken()
			Expect(err).NotTo(HaveOccurred())
			Expect(token).NotTo(BeEmpty())

			By("waiting for the metrics endpoint to be ready")
			verifyMetricsEndpointReady := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "endpoints", metricsServiceName, "-n", namespace)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(ContainSubstring("8443"), "Metrics endpoint is not ready")
			}
			Eventually(verifyMetricsEndpointReady).Should(Succeed())

			By("verifying that the controller manager is serving the metrics server")
			verifyMetricsServerStarted := func(g Gomega) {
				cmd := exec.Command("kubectl", "logs", controllerPodName, "-n", namespace)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(ContainSubstring("controller-runtime.metrics\tServing metrics server"),
					"Metrics server not yet started")
			}
			Eventually(verifyMetricsServerStarted).Should(Succeed())

			By("creating the curl-metrics pod to access the metrics endpoint")
			cmd = exec.Command("kubectl", "run", "curl-metrics", "--restart=Never",
				"--namespace", namespace,
				"--image=curlimages/curl:latest",
				"--overrides",
				fmt.Sprintf(`{
					"spec": {
						"containers": [{
							"name": "curl",
							"image": "curlimages/curl:latest",
							"command": ["/bin/sh", "-c"],
							"args": ["curl -v -k -H 'Authorization: Bearer %s' https://%s.%s.svc.cluster.local:8443/metrics"],
							"securityContext": {
								"readOnlyRootFilesystem": true,
								"allowPrivilegeEscalation": false,
								"capabilities": {
									"drop": ["ALL"]
								},
								"runAsNonRoot": true,
								"runAsUser": 1000,
								"seccompProfile": {
									"type": "RuntimeDefault"
								}
							}
						}],
						"serviceAccountName": "%s"
					}
				}`, token, metricsServiceName, namespace, serviceAccountName))
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create curl-metrics pod")

			By("waiting for the curl-metrics pod to complete.")
			verifyCurlUp := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "pods", "curl-metrics",
					"-o", "jsonpath={.status.phase}",
					"-n", namespace)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("Succeeded"), "curl pod in wrong status")
			}
			Eventually(verifyCurlUp, 5*time.Minute).Should(Succeed())

			By("getting the metrics by checking curl-metrics logs")
			verifyMetricsAvailable := func(g Gomega) {
				metricsOutput, err := getMetricsOutput()
				g.Expect(err).NotTo(HaveOccurred(), "Failed to retrieve logs from curl pod")
				g.Expect(metricsOutput).NotTo(BeEmpty())
				g.Expect(metricsOutput).To(ContainSubstring("< HTTP/1.1 200 OK"))
			}
			Eventually(verifyMetricsAvailable, 2*time.Minute).Should(Succeed())
		})

		// +kubebuilder:scaffold:e2e-webhooks-checks

		// TODO: Customize the e2e test suite with scenarios specific to your project.
		// Consider applying sample/CR(s) and check their status and/or verifying
		// the reconciliation by using the metrics, i.e.:
		// metricsOutput, err := getMetricsOutput()
		// Expect(err).NotTo(HaveOccurred(), "Failed to retrieve logs from curl pod")
		// Expect(metricsOutput).To(ContainSubstring(
		//    fmt.Sprintf(`controller_runtime_reconcile_total{controller="%s",result="success"} 1`,
		//    strings.ToLower(<Kind>),
		// ))
	})

	Context("Resource", func() {
		var testNamespace corev1.Namespace
		var c client.Client
		ctx := context.Background()
		scheme := runtime.NewScheme()

		utilruntime.Must(clientgoscheme.AddToScheme(scheme))
		utilruntime.Must(testv1.AddToScheme(scheme))

		BeforeEach(func() {
			var err error
			c, err = client.New(ctrl.GetConfigOrDie(), client.Options{
				Scheme: scheme,
			})
			Expect(err).NotTo(HaveOccurred(), "Create the runtime client")

			testNamespace = corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: uuid.NewString(),
				},
			}
			err = c.Create(ctx, &testNamespace)
			Expect(err).NotTo(HaveOccurred(), "Create the namespace")
		})

		AfterEach(func() {
			err := c.Delete(ctx, &testNamespace)
			Expect(err).NotTo(HaveOccurred(), "Cleanup the namespace")
		})

		Context("ConfigMap Management", func() {
			var testResource testv1.Test
			var originalConfigMapName string
			var originalConfigMapData map[string]string

			BeforeEach(func() {
				originalConfigMapName = "test-cm-" + uuid.NewString()[:8]
				originalConfigMapData = map[string]string{
					"key1": "value1",
					"key2": "value2",
				}
			})

			AfterEach(func() {
				// Cleanup test resource if it exists
				if testResource.Name != "" {
					err := c.Delete(ctx, &testResource)
					Expect(client.IgnoreNotFound(err)).To(Succeed(), "Cleanup test resource")
				}
			})

			It("should not create ConfigMap when enabled is false", func() {
				By("creating Test resource with ConfigMap disabled")
				testResource = testv1.Test{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-disabled-" + uuid.NewString()[:8],
						Namespace: testNamespace.Name,
					},
					Spec: testv1.TestSpec{
						ConfigMap: testv1.ConfigMapSpec{
							Enabled: false,
							Name:    originalConfigMapName,
							Data:    originalConfigMapData,
						},
					},
				}

				err := c.Create(ctx, &testResource)
				Expect(err).NotTo(HaveOccurred(), "Create the Test resource")

				By("verifying ConfigMap is not created")
				cm := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      originalConfigMapName,
						Namespace: testResource.Namespace,
					},
				}

				Consistently(func(g Gomega) {
					err := c.Get(ctx, client.ObjectKeyFromObject(cm), cm)
					g.Expect(err).To(HaveOccurred(), "ConfigMap should not exist")
					g.Expect(client.IgnoreNotFound(err)).To(Succeed(), "ConfigMap should not exist")
				}, 10*time.Second, time.Second).Should(Succeed())

				By("verifying Test status reflects no ConfigMap and no condition")
				Eventually(func(g Gomega) {
					err := c.Get(ctx, client.ObjectKeyFromObject(&testResource), &testResource)
					g.Expect(err).NotTo(HaveOccurred(), "Get Test resource")
					g.Expect(testResource.Status.ConfigMapStatus).To(BeNil(), "ConfigMapStatus should be nil when disabled")

					// Verify ConfigMap condition is not present
					var configMapCondition *metav1.Condition
					for _, cond := range testResource.Status.Conditions {
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
				testResource = testv1.Test{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-enabled-" + uuid.NewString()[:8],
						Namespace: testNamespace.Name,
					},
					Spec: testv1.TestSpec{
						ConfigMap: testv1.ConfigMapSpec{
							Enabled: true,
							Name:    originalConfigMapName,
							Data:    originalConfigMapData,
						},
					},
				}

				err := c.Create(ctx, &testResource)
				Expect(err).NotTo(HaveOccurred(), "Create the Test resource")

				By("verifying ConfigMap is created with correct data")
				cm := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      originalConfigMapName,
						Namespace: testResource.Namespace,
					},
				}

				Eventually(func(g Gomega) {
					err := c.Get(ctx, client.ObjectKeyFromObject(cm), cm)
					g.Expect(err).NotTo(HaveOccurred(), "Get the ConfigMap")
					g.Expect(cm.Data).To(Equal(originalConfigMapData), "ConfigMap data should match")
				}, 10*time.Second, 500*time.Millisecond).Should(Succeed())

				By("verifying Test status is properly filled with correct generation")
				Eventually(func(g Gomega) {
					err := c.Get(ctx, client.ObjectKeyFromObject(&testResource), &testResource)
					g.Expect(err).NotTo(HaveOccurred(), "Get Test resource")
					g.Expect(testResource.Status.ConfigMapStatus).NotTo(BeNil(), "ConfigMapStatus should not be nil")
					g.Expect(testResource.Status.ConfigMapStatus.Name).To(Equal(originalConfigMapName), "Status should reflect ConfigMap name")

					// Check for ConfigMap condition with proper generation
					var configMapCondition *metav1.Condition
					for _, cond := range testResource.Status.Conditions {
						if cond.Type == "ConfigMap" {
							configMapCondition = &cond
							break
						}
					}
					g.Expect(configMapCondition).NotTo(BeNil(), "ConfigMap condition should exist")
					g.Expect(configMapCondition.Status).To(Equal(metav1.ConditionTrue), "ConfigMap condition should be True")
					g.Expect(configMapCondition.Reason).To(Equal("UpToDate"), "ConfigMap condition reason should be UpToDate")
					g.Expect(configMapCondition.ObservedGeneration).To(Equal(testResource.Generation), "ConfigMap condition should have correct generation")
				}, 10*time.Second, 500*time.Millisecond).Should(Succeed())
			})

			It("should update ConfigMap data when fields are changed", func() {
				By("creating Test resource with initial ConfigMap")
				testResource = testv1.Test{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-update-data-" + uuid.NewString()[:8],
						Namespace: testNamespace.Name,
					},
					Spec: testv1.TestSpec{
						ConfigMap: testv1.ConfigMapSpec{
							Enabled: true,
							Name:    originalConfigMapName,
							Data:    originalConfigMapData,
						},
					},
				}

				err := c.Create(ctx, &testResource)
				Expect(err).NotTo(HaveOccurred(), "Create the Test resource")

				By("waiting for initial ConfigMap to be created")
				cm := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      originalConfigMapName,
						Namespace: testResource.Namespace,
					},
				}

				Eventually(func(g Gomega) {
					err := c.Get(ctx, client.ObjectKeyFromObject(cm), cm)
					g.Expect(err).NotTo(HaveOccurred(), "Get the ConfigMap")
					g.Expect(cm.Data).To(Equal(originalConfigMapData), "Initial ConfigMap data should match")
				}, 10*time.Second, 500*time.Millisecond).Should(Succeed())

				By("updating ConfigMap data")
				updatedData := map[string]string{
					"key1": "updated-value1",
					"key3": "new-value3",
				}

				err = c.Get(ctx, client.ObjectKeyFromObject(&testResource), &testResource)
				Expect(err).NotTo(HaveOccurred(), "Get current Test resource")

				testResource.Spec.ConfigMap.Data = updatedData
				err = c.Update(ctx, &testResource)
				Expect(err).NotTo(HaveOccurred(), "Update Test resource")

				By("verifying ConfigMap data is updated")
				Eventually(func(g Gomega) {
					err := c.Get(ctx, client.ObjectKeyFromObject(cm), cm)
					g.Expect(err).NotTo(HaveOccurred(), "Get the updated ConfigMap")
					g.Expect(cm.Data).To(Equal(updatedData), "ConfigMap data should be updated")
				}, 10*time.Second, 500*time.Millisecond).Should(Succeed())

				By("verifying Test status remains consistent with updated generation")
				Eventually(func(g Gomega) {
					err := c.Get(ctx, client.ObjectKeyFromObject(&testResource), &testResource)
					g.Expect(err).NotTo(HaveOccurred(), "Get Test resource")
					g.Expect(testResource.Status.ConfigMapStatus).NotTo(BeNil(), "ConfigMapStatus should not be nil")
					g.Expect(testResource.Status.ConfigMapStatus.Name).To(Equal(originalConfigMapName), "Status should still reflect original ConfigMap name")

					// Check for ConfigMap condition with updated generation
					var configMapCondition *metav1.Condition
					for _, cond := range testResource.Status.Conditions {
						if cond.Type == "ConfigMap" {
							configMapCondition = &cond
							break
						}
					}
					g.Expect(configMapCondition).NotTo(BeNil(), "ConfigMap condition should exist")
					g.Expect(configMapCondition.Status).To(Equal(metav1.ConditionTrue), "ConfigMap condition should be True")
					g.Expect(configMapCondition.Reason).To(Equal("UpToDate"), "ConfigMap condition reason should be UpToDate")
					g.Expect(configMapCondition.ObservedGeneration).To(Equal(testResource.Generation), "ConfigMap condition should have updated generation")
				}, 10*time.Second, 500*time.Millisecond).Should(Succeed())
			})

			It("should create new ConfigMap and delete old one when name is changed", func() {
				By("creating Test resource with initial ConfigMap")
				testResource = testv1.Test{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-rename-" + uuid.NewString()[:8],
						Namespace: testNamespace.Name,
					},
					Spec: testv1.TestSpec{
						ConfigMap: testv1.ConfigMapSpec{
							Enabled: true,
							Name:    originalConfigMapName,
							Data:    originalConfigMapData,
						},
					},
				}

				err := c.Create(ctx, &testResource)
				Expect(err).NotTo(HaveOccurred(), "Create the Test resource")

				By("waiting for initial ConfigMap to be created")
				originalCm := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      originalConfigMapName,
						Namespace: testResource.Namespace,
					},
				}

				Eventually(func(g Gomega) {
					err := c.Get(ctx, client.ObjectKeyFromObject(originalCm), originalCm)
					g.Expect(err).NotTo(HaveOccurred(), "Get the original ConfigMap")
				}, 10*time.Second, 500*time.Millisecond).Should(Succeed())

				By("updating ConfigMap name")
				newConfigMapName := "test-cm-new-" + uuid.NewString()[:8]

				err = c.Get(ctx, client.ObjectKeyFromObject(&testResource), &testResource)
				Expect(err).NotTo(HaveOccurred(), "Get current Test resource")

				testResource.Spec.ConfigMap.Name = newConfigMapName
				err = c.Update(ctx, &testResource)
				Expect(err).NotTo(HaveOccurred(), "Update Test resource")

				By("verifying new ConfigMap is created")
				newCm := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      newConfigMapName,
						Namespace: testResource.Namespace,
					},
				}

				Eventually(func(g Gomega) {
					err := c.Get(ctx, client.ObjectKeyFromObject(newCm), newCm)
					g.Expect(err).NotTo(HaveOccurred(), "Get the new ConfigMap")
					g.Expect(newCm.Data).To(Equal(originalConfigMapData), "New ConfigMap should have same data")
				}, 10*time.Second, 500*time.Millisecond).Should(Succeed())

				By("verifying old ConfigMap is deleted")
				Eventually(func(g Gomega) {
					err := c.Get(ctx, client.ObjectKeyFromObject(originalCm), originalCm)
					g.Expect(err).To(HaveOccurred(), "Original ConfigMap should be deleted")
					g.Expect(client.IgnoreNotFound(err)).To(Succeed(), "Original ConfigMap should not exist")
				}, 10*time.Second, 500*time.Millisecond).Should(Succeed())

				By("verifying Test status reflects new ConfigMap name with correct generation")
				Eventually(func(g Gomega) {
					err := c.Get(ctx, client.ObjectKeyFromObject(&testResource), &testResource)
					g.Expect(err).NotTo(HaveOccurred(), "Get Test resource")
					g.Expect(testResource.Status.ConfigMapStatus).NotTo(BeNil(), "ConfigMapStatus should not be nil")
					g.Expect(testResource.Status.ConfigMapStatus.Name).To(Equal(newConfigMapName), "Status should reflect new ConfigMap name")

					// Check for ConfigMap condition with updated generation
					var configMapCondition *metav1.Condition
					for _, cond := range testResource.Status.Conditions {
						if cond.Type == "ConfigMap" {
							configMapCondition = &cond
							break
						}
					}
					g.Expect(configMapCondition).NotTo(BeNil(), "ConfigMap condition should exist")
					g.Expect(configMapCondition.Status).To(Equal(metav1.ConditionTrue), "ConfigMap condition should be True")
					g.Expect(configMapCondition.Reason).To(Equal("UpToDate"), "ConfigMap condition reason should be UpToDate")
					g.Expect(configMapCondition.ObservedGeneration).To(Equal(testResource.Generation), "ConfigMap condition should have updated generation")
				}, 10*time.Second, 500*time.Millisecond).Should(Succeed())
			})

			It("should create new ConfigMap with updated data when both name and fields are changed", func() {
				By("creating Test resource with initial ConfigMap")
				testResource = testv1.Test{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-rename-data-" + uuid.NewString()[:8],
						Namespace: testNamespace.Name,
					},
					Spec: testv1.TestSpec{
						ConfigMap: testv1.ConfigMapSpec{
							Enabled: true,
							Name:    originalConfigMapName,
							Data:    originalConfigMapData,
						},
					},
				}

				err := c.Create(ctx, &testResource)
				Expect(err).NotTo(HaveOccurred(), "Create the Test resource")

				By("waiting for initial ConfigMap to be created")
				originalCm := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      originalConfigMapName,
						Namespace: testResource.Namespace,
					},
				}

				Eventually(func(g Gomega) {
					err := c.Get(ctx, client.ObjectKeyFromObject(originalCm), originalCm)
					g.Expect(err).NotTo(HaveOccurred(), "Get the original ConfigMap")
				}, 10*time.Second, 500*time.Millisecond).Should(Succeed())

				By("updating both ConfigMap name and data")
				newConfigMapName := "test-cm-renamed-" + uuid.NewString()[:8]
				updatedData := map[string]string{
					"new-key1": "new-value1",
					"new-key2": "new-value2",
				}

				err = c.Get(ctx, client.ObjectKeyFromObject(&testResource), &testResource)
				Expect(err).NotTo(HaveOccurred(), "Get current Test resource")

				testResource.Spec.ConfigMap.Name = newConfigMapName
				testResource.Spec.ConfigMap.Data = updatedData
				err = c.Update(ctx, &testResource)
				Expect(err).NotTo(HaveOccurred(), "Update Test resource")

				By("verifying new ConfigMap is created with updated data")
				newCm := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      newConfigMapName,
						Namespace: testResource.Namespace,
					},
				}

				Eventually(func(g Gomega) {
					err := c.Get(ctx, client.ObjectKeyFromObject(newCm), newCm)
					g.Expect(err).NotTo(HaveOccurred(), "Get the new ConfigMap")
					g.Expect(newCm.Data).To(Equal(updatedData), "New ConfigMap should have updated data")
				}, 10*time.Second, 500*time.Millisecond).Should(Succeed())

				By("verifying old ConfigMap is deleted")
				Eventually(func(g Gomega) {
					err := c.Get(ctx, client.ObjectKeyFromObject(originalCm), originalCm)
					g.Expect(err).To(HaveOccurred(), "Original ConfigMap should be deleted")
					g.Expect(client.IgnoreNotFound(err)).To(Succeed(), "Original ConfigMap should not exist")
				}, 10*time.Second, 500*time.Millisecond).Should(Succeed())

				By("verifying Test status reflects new ConfigMap with updated data and correct generation")
				Eventually(func(g Gomega) {
					err := c.Get(ctx, client.ObjectKeyFromObject(&testResource), &testResource)
					g.Expect(err).NotTo(HaveOccurred(), "Get Test resource")
					g.Expect(testResource.Status.ConfigMapStatus).NotTo(BeNil(), "ConfigMapStatus should not be nil")
					g.Expect(testResource.Status.ConfigMapStatus.Name).To(Equal(newConfigMapName), "Status should reflect new ConfigMap name")

					// Check for ConfigMap condition with updated generation
					var configMapCondition *metav1.Condition
					for _, cond := range testResource.Status.Conditions {
						if cond.Type == "ConfigMap" {
							configMapCondition = &cond
							break
						}
					}
					g.Expect(configMapCondition).NotTo(BeNil(), "ConfigMap condition should exist")
					g.Expect(configMapCondition.Status).To(Equal(metav1.ConditionTrue), "ConfigMap condition should be True")
					g.Expect(configMapCondition.Reason).To(Equal("UpToDate"), "ConfigMap condition reason should be UpToDate")
					g.Expect(configMapCondition.ObservedGeneration).To(Equal(testResource.Generation), "ConfigMap condition should have updated generation")
				}, 10*time.Second, 500*time.Millisecond).Should(Succeed())
			})

			It("should delete ConfigMap when disabled", func() {
				By("creating Test resource with ConfigMap enabled")
				testResource = testv1.Test{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-disable-" + uuid.NewString()[:8],
						Namespace: testNamespace.Name,
					},
					Spec: testv1.TestSpec{
						ConfigMap: testv1.ConfigMapSpec{
							Enabled: true,
							Name:    originalConfigMapName,
							Data:    originalConfigMapData,
						},
					},
				}

				err := c.Create(ctx, &testResource)
				Expect(err).NotTo(HaveOccurred(), "Create the Test resource")

				By("waiting for ConfigMap to be created")
				cm := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      originalConfigMapName,
						Namespace: testResource.Namespace,
					},
				}

				Eventually(func(g Gomega) {
					err := c.Get(ctx, client.ObjectKeyFromObject(cm), cm)
					g.Expect(err).NotTo(HaveOccurred(), "Get the ConfigMap")
				}, 10*time.Second, 500*time.Millisecond).Should(Succeed())

				By("disabling ConfigMap")
				err = c.Get(ctx, client.ObjectKeyFromObject(&testResource), &testResource)
				Expect(err).NotTo(HaveOccurred(), "Get current Test resource")

				testResource.Spec.ConfigMap.Enabled = false
				err = c.Update(ctx, &testResource)
				Expect(err).NotTo(HaveOccurred(), "Update Test resource")

				By("verifying ConfigMap is deleted")
				Eventually(func(g Gomega) {
					err := c.Get(ctx, client.ObjectKeyFromObject(cm), cm)
					g.Expect(err).To(HaveOccurred(), "ConfigMap should be deleted")
					g.Expect(client.IgnoreNotFound(err)).To(Succeed(), "ConfigMap should not exist")
				}, 10*time.Second, 500*time.Millisecond).Should(Succeed())

				By("verifying Test status reflects no ConfigMap and condition is removed")
				Eventually(func(g Gomega) {
					err := c.Get(ctx, client.ObjectKeyFromObject(&testResource), &testResource)
					g.Expect(err).NotTo(HaveOccurred(), "Get Test resource")
					g.Expect(testResource.Status.ConfigMapStatus).To(BeNil(), "ConfigMapStatus should be nil when disabled")

					// Verify ConfigMap condition is removed
					var configMapCondition *metav1.Condition
					for _, cond := range testResource.Status.Conditions {
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
				testResource = testv1.Test{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-disable-rename-" + uuid.NewString()[:8],
						Namespace: testNamespace.Name,
					},
					Spec: testv1.TestSpec{
						ConfigMap: testv1.ConfigMapSpec{
							Enabled: true,
							Name:    originalConfigMapName,
							Data:    originalConfigMapData,
						},
					},
				}

				err := c.Create(ctx, &testResource)
				Expect(err).NotTo(HaveOccurred(), "Create the Test resource")

				By("waiting for ConfigMap to be created")
				originalCm := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      originalConfigMapName,
						Namespace: testResource.Namespace,
					},
				}

				Eventually(func(g Gomega) {
					err := c.Get(ctx, client.ObjectKeyFromObject(originalCm), originalCm)
					g.Expect(err).NotTo(HaveOccurred(), "Get the original ConfigMap")
				}, 10*time.Second, 500*time.Millisecond).Should(Succeed())

				By("disabling ConfigMap and changing name simultaneously")
				newConfigMapName := "test-cm-disabled-" + uuid.NewString()[:8]

				err = c.Get(ctx, client.ObjectKeyFromObject(&testResource), &testResource)
				Expect(err).NotTo(HaveOccurred(), "Get current Test resource")

				testResource.Spec.ConfigMap.Enabled = false
				testResource.Spec.ConfigMap.Name = newConfigMapName
				err = c.Update(ctx, &testResource)
				Expect(err).NotTo(HaveOccurred(), "Update Test resource")

				By("verifying original ConfigMap is deleted")
				Eventually(func(g Gomega) {
					err := c.Get(ctx, client.ObjectKeyFromObject(originalCm), originalCm)
					g.Expect(err).To(HaveOccurred(), "Original ConfigMap should be deleted")
					g.Expect(client.IgnoreNotFound(err)).To(Succeed(), "Original ConfigMap should not exist")
				}, 10*time.Second, 500*time.Millisecond).Should(Succeed())

				By("verifying new ConfigMap is not created")
				newCm := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      newConfigMapName,
						Namespace: testResource.Namespace,
					},
				}

				Consistently(func(g Gomega) {
					err := c.Get(ctx, client.ObjectKeyFromObject(newCm), newCm)
					g.Expect(err).To(HaveOccurred(), "New ConfigMap should not be created")
					g.Expect(client.IgnoreNotFound(err)).To(Succeed(), "New ConfigMap should not exist")
				}, 10*time.Second, time.Second).Should(Succeed())

				By("verifying Test status reflects no ConfigMap and proper cleanup")
				Eventually(func(g Gomega) {
					err := c.Get(ctx, client.ObjectKeyFromObject(&testResource), &testResource)
					g.Expect(err).NotTo(HaveOccurred(), "Get Test resource")
					g.Expect(testResource.Status.ConfigMapStatus).To(BeNil(), "ConfigMapStatus should be nil when disabled")

					// Verify ConfigMap condition is removed
					var configMapCondition *metav1.Condition
					for _, cond := range testResource.Status.Conditions {
						if cond.Type == "ConfigMap" {
							configMapCondition = &cond
							break
						}
					}
					g.Expect(configMapCondition).To(BeNil(), "ConfigMap condition should be removed when disabled")
				}, 10*time.Second, 500*time.Millisecond).Should(Succeed())
			})
		})
	})
})

// serviceAccountToken returns a token for the specified service account in the given namespace.
// It uses the Kubernetes TokenRequest API to generate a token by directly sending a request
// and parsing the resulting token from the API response.
func serviceAccountToken() (string, error) {
	const tokenRequestRawString = `{
		"apiVersion": "authentication.k8s.io/v1",
		"kind": "TokenRequest"
	}`

	// Temporary file to store the token request
	secretName := fmt.Sprintf("%s-token-request", serviceAccountName)
	tokenRequestFile := filepath.Join("/tmp", secretName)
	err := os.WriteFile(tokenRequestFile, []byte(tokenRequestRawString), os.FileMode(0o644))
	if err != nil {
		return "", err
	}

	var out string
	verifyTokenCreation := func(g Gomega) {
		// Execute kubectl command to create the token
		cmd := exec.Command("kubectl", "create", "--raw", fmt.Sprintf(
			"/api/v1/namespaces/%s/serviceaccounts/%s/token",
			namespace,
			serviceAccountName,
		), "-f", tokenRequestFile)

		output, err := cmd.CombinedOutput()
		g.Expect(err).NotTo(HaveOccurred())

		// Parse the JSON output to extract the token
		var token tokenRequest
		err = json.Unmarshal(output, &token)
		g.Expect(err).NotTo(HaveOccurred())

		out = token.Status.Token
	}
	Eventually(verifyTokenCreation).Should(Succeed())

	return out, err
}

// getMetricsOutput retrieves and returns the logs from the curl pod used to access the metrics endpoint.
func getMetricsOutput() (string, error) {
	By("getting the curl-metrics logs")
	cmd := exec.Command("kubectl", "logs", "curl-metrics", "-n", namespace)
	return utils.Run(cmd)
}

// tokenRequest is a simplified representation of the Kubernetes TokenRequest API response,
// containing only the token field that we need to extract.
type tokenRequest struct {
	Status struct {
		Token string `json:"token"`
	} `json:"status"`
}
