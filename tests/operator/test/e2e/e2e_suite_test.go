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
	"fmt"
	"operator/test/utils"
	"os"
	"os/exec"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var (
	// Optional Environment Variables:
	// - CERT_MANAGER_INSTALL_SKIP=true: Skips CertManager installation during test setup.
	// - COVERAGE_ENABLED=true: Enables coverage collection for github.com/u-ctf/controller-fwk package.
	// These variables are useful if CertManager is already installed, avoiding
	// re-installation and conflicts.
	skipCertManagerInstall = os.Getenv("CERT_MANAGER_INSTALL_SKIP") == "true"
	// isCertManagerAlreadyInstalled will be set true when CertManager CRDs be found on the cluster
	isCertManagerAlreadyInstalled = false

	// coverageEnabled determines if coverage collection is enabled
	coverageEnabled = os.Getenv("COVERAGE_ENABLED") == "true"

	// projectImage is the name of the image which will be build and loaded
	// with the code source changes to be tested.
	projectImage = "example.com/operator:v0.0.1"

	controllerPodName string
)

// collectCoverageData extracts coverage data from the controller pod and saves it locally
func collectCoverageData() {
	if controllerPodName == "" {
		_, _ = fmt.Fprintf(GinkgoWriter, "WARNING: Controller pod name not available, skipping coverage collection\n")
		return
	}

	// Kill pod
	By("terminating the controller pod to flush coverage data")
	cmd := exec.Command("kubectl", "delete", "pod", controllerPodName, "-n", namespace, "--grace-period=30")
	_, err := utils.Run(cmd)
	if err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "WARNING: Failed to delete controller pod: %v\n", err)
		return
	}

	// Wait a bit for coverage data to be flushed
	By("waiting for coverage data to be flushed")
	time.Sleep(30 * time.Second)

	// Check if coverage data is now available
	cmd = exec.Command("kubectl", "exec", "deploy/operator-controller-manager", "-n", namespace, "--", "ls", "-la", "/tmp/coverage")
	if output, err := utils.Run(cmd); err == nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "Coverage directory after flush:\n%s\n", output)
	}

	// Get new pod name (new pod should be starting)
	By("getting new controller pod name")
	cmd = exec.Command("kubectl", "get", "pods", "-l", "control-plane=controller-manager", "-o", "jsonpath={.items[0].metadata.name}", "-n", namespace)
	newControllerPodName, err := utils.Run(cmd)
	if err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "WARNING: Failed to get new controller pod name: %v\n", err)
		return
	}
	if newControllerPodName == controllerPodName {
		_, _ = fmt.Fprintf(GinkgoWriter, "WARNING: New controller pod name is the same as the old one, something went wrong\n")
		return
	}
	controllerPodName = newControllerPodName

	// Copy coverage data (the pod should still be running briefly after SIGTERM)
	By("copying coverage data from controller pod")
	cmd = exec.Command("kubectl", "cp", fmt.Sprintf("%s/%s:/tmp/coverage", namespace, controllerPodName), "./coverage-data")
	_, err = utils.Run(cmd)
	if err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "WARNING: Failed to copy coverage data: %v\n", err)
		return
	}

	_, _ = fmt.Fprintf(GinkgoWriter, "Coverage data saved to ./coverage-data\n")

	// List what we actually copied
	cmd = exec.Command("ls", "-la", "./coverage-data")
	if localOutput, err := utils.Run(cmd); err == nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "Local coverage data contents:\n%s\n", localOutput)
	}

	// Convert coverage data to standard format
	cmd = exec.Command("go", "tool", "covdata", "textfmt", "-i=./coverage-data", "-o", "coverage.out")
	_, err = utils.Run(cmd)
	if err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "WARNING: Failed to convert coverage data: %v\n", err)
		// Try to get more information about what's in the coverage directory
		cmd = exec.Command("find", "./coverage-data", "-type", "f")
		if findOutput, findErr := utils.Run(cmd); findErr == nil {
			_, _ = fmt.Fprintf(GinkgoWriter, "Files in coverage-data:\n%s\n", findOutput)
		}
	} else {
		_, _ = fmt.Fprintf(GinkgoWriter, "Coverage report saved to coverage.out\n")

		// Rename /workspace/ to . in coverage.out to fix paths
		cmd = exec.Command("sed", "-i", "s|/workspace/|./|g", "coverage.out")
		_, err = utils.Run(cmd)
		if err != nil {
			_, _ = fmt.Fprintf(GinkgoWriter, "WARNING: Failed to fix paths in coverage.out: %v\n", err)
		}

		// Show coverage summary
		cmd = exec.Command("go", "tool", "cover", "-func=coverage.out")
		if summaryOutput, summaryErr := utils.Run(cmd); summaryErr == nil {
			_, _ = fmt.Fprintf(GinkgoWriter, "Coverage summary:\n%s\n", summaryOutput)
		}

		// Generate HTML coverage report
		cmd = exec.Command("go", "tool", "cover", "-html=coverage.out", "-o", "coverage.html")
		_, err = utils.Run(cmd)
		if err != nil {
			_, _ = fmt.Fprintf(GinkgoWriter, "WARNING: Failed to generate HTML coverage report: %v\n", err)
		} else {
			_, _ = fmt.Fprintf(GinkgoWriter, "HTML coverage report saved to coverage.html\n")
		}
	}
}

// TestE2E runs the end-to-end (e2e) test suite for the project. These tests execute in an isolated,
// temporary environment to validate project changes with the purpose of being used in CI jobs.
// The default setup requires Kind, builds/loads the Manager Docker image locally, and installs
// CertManager.
func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	_, _ = fmt.Fprintf(GinkgoWriter, "Starting operator integration test suite\n")
	RunSpecs(t, "e2e suite")
}

var _ = SynchronizedBeforeSuite(
	func() {
		var cmd *exec.Cmd
		var err error

		if coverageEnabled {
			By("building the manager(Operator) image with coverage instrumentation")
			cmd = exec.Command("make", "docker-build-coverage", fmt.Sprintf("IMG=%s", projectImage))
			_, err = utils.Run(cmd)
			ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to build the manager(Operator) image with coverage")
		} else {
			By("building the manager(Operator) image")
			cmd = exec.Command("make", "docker-build", fmt.Sprintf("IMG=%s", projectImage))
			_, err = utils.Run(cmd)
			ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to build the manager(Operator) image")
		}

		// TODO(user): If you want to change the e2e test vendor from Kind, ensure the image is
		// built and available before running the tests. Also, remove the following block.
		By("loading the manager(Operator) image on Kind")
		err = utils.LoadImageToKindClusterWithName(projectImage)
		ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to load the manager(Operator) image into Kind")

		// The tests-e2e are intended to run on a temporary cluster that is created and destroyed for testing.
		// To prevent errors when tests run in environments with CertManager already installed,
		// we check for its presence before execution.
		// Setup CertManager before the suite if not skipped and if not already installed
		if !skipCertManagerInstall {
			By("checking if cert manager is installed already")
			isCertManagerAlreadyInstalled = utils.IsCertManagerCRDsInstalled()
			if !isCertManagerAlreadyInstalled {
				_, _ = fmt.Fprintf(GinkgoWriter, "Installing CertManager...\n")
				Expect(utils.InstallCertManager()).To(Succeed(), "Failed to install CertManager")
			} else {
				_, _ = fmt.Fprintf(GinkgoWriter, "WARNING: CertManager is already installed. Skipping installation...\n")
			}
		}

		By("creating manager namespace")
		cmd = exec.Command("kubectl", "create", "ns", namespace)
		_, err = utils.Run(cmd)
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
	},
	func() {
		By("Waiting for the controller-manager to be ready")
		Eventually(func(g Gomega) {
			cmd := exec.Command("kubectl", "get", "pods", "-l", "control-plane=controller-manager", "-n", namespace, "-o", "jsonpath={.items[0].status.conditions[?(@.type=='Ready')].status}")
			output, err := utils.Run(cmd)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(output).To(Equal("True"))
		}).Should(Succeed())

		By("Getting the controller's pod name")
		Eventually(func(g Gomega) {
			cmd := exec.Command("kubectl", "get", "pods", "-l", "control-plane=controller-manager", "-o", "jsonpath={.items[0].metadata.name}", "-n", namespace)
			output, err := utils.Run(cmd)
			g.Expect(err).NotTo(HaveOccurred())
			controllerPodName = output
		}).Should(Succeed())
	},
)

var _ = SynchronizedAfterSuite(func() {}, func() {
	if coverageEnabled {
		By("collecting coverage data from controller pod")
		collectCoverageData()
	}

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

	// Teardown CertManager after the suite if not skipped and if it was not already installed
	if !skipCertManagerInstall && !isCertManagerAlreadyInstalled {
		_, _ = fmt.Fprintf(GinkgoWriter, "Uninstalling CertManager...\n")
		utils.UninstallCertManager()
	}
})
