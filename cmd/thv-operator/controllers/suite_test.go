package controllers

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	mcpv1alpha1 "github.com/stacklok/toolhive/cmd/thv-operator/api/v1alpha1"
	"github.com/stacklok/toolhive/pkg/container/kubernetes"
)

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment
var ctx context.Context
var cancel context.CancelFunc

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.TODO())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = mcpv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).ToNot(HaveOccurred())

	// Create a mock platform detector for testing
	mockDetector := &mockPlatformDetector{
		platform: kubernetes.PlatformKubernetes,
	}

	err = (&MCPServerReconciler{
		Client:           k8sManager.GetClient(),
		Scheme:           k8sManager.GetScheme(),
		platformDetector: mockDetector,
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	go func() {
		defer GinkgoRecover()
		err = k8sManager.Start(ctx)
		Expect(err).ToNot(HaveOccurred(), "failed to run manager")
	}()
})

var _ = AfterSuite(func() {
	cancel()
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

// Helper functions for tests

func createNamespace(name string) *corev1.Namespace {
	ns := &corev1.Namespace{}
	ns.Name = name
	Expect(k8sClient.Create(ctx, ns)).To(Succeed())
	return ns
}

func deleteNamespace(name string) {
	ns := &corev1.Namespace{}
	ns.Name = name
	Expect(k8sClient.Delete(ctx, ns)).To(Succeed())
}

func waitForDeploymentCreation(namespacedName client.ObjectKey, timeout time.Duration) *appsv1.Deployment {
	deployment := &appsv1.Deployment{}
	Eventually(func() error {
		return k8sClient.Get(ctx, namespacedName, deployment)
	}, timeout, time.Second).Should(Succeed())
	return deployment
}

func waitForServiceCreation(namespacedName client.ObjectKey, timeout time.Duration) *corev1.Service {
	service := &corev1.Service{}
	Eventually(func() error {
		return k8sClient.Get(ctx, namespacedName, service)
	}, timeout, time.Second).Should(Succeed())
	return service
}

func waitForServiceAccountCreation(namespacedName client.ObjectKey, timeout time.Duration) *corev1.ServiceAccount {
	sa := &corev1.ServiceAccount{}
	Eventually(func() error {
		return k8sClient.Get(ctx, namespacedName, sa)
	}, timeout, time.Second).Should(Succeed())
	return sa
}

func waitForRoleCreation(namespacedName client.ObjectKey, timeout time.Duration) *rbacv1.Role {
	role := &rbacv1.Role{}
	Eventually(func() error {
		return k8sClient.Get(ctx, namespacedName, role)
	}, timeout, time.Second).Should(Succeed())
	return role
}

func waitForRoleBindingCreation(namespacedName client.ObjectKey, timeout time.Duration) *rbacv1.RoleBinding {
	roleBinding := &rbacv1.RoleBinding{}
	Eventually(func() error {
		return k8sClient.Get(ctx, namespacedName, roleBinding)
	}, timeout, time.Second).Should(Succeed())
	return roleBinding
}

func waitForMCPServerStatusUpdate(namespacedName client.ObjectKey, timeout time.Duration) *mcpv1alpha1.MCPServer {
	mcpServer := &mcpv1alpha1.MCPServer{}
	Eventually(func() bool {
		err := k8sClient.Get(ctx, namespacedName, mcpServer)
		if err != nil {
			return false
		}
		return mcpServer.Status.Phase != ""
	}, timeout, time.Second).Should(BeTrue())
	return mcpServer
}