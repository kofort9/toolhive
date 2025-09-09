package operator_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	mcpv1alpha1 "github.com/stacklok/toolhive/cmd/thv-operator/api/v1alpha1"
)

var _ = Describe("MCPServer Controller E2E Tests", func() {
	const (
		timeout  = time.Second * 30
		interval = time.Millisecond * 250
	)

	Context("When creating an MCPServer", func() {
		var testNamespace string

		BeforeEach(func() {
			testNamespace = "test-" + time.Now().Format("20060102150405")
			createNamespace(testNamespace)
		})

		AfterEach(func() {
			deleteNamespace(testNamespace)
		})

		It("Should create basic MCPServer resources successfully", func() {
			By("Creating a new MCPServer")
			mcpServer := &mcpv1alpha1.MCPServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-mcpserver",
					Namespace: testNamespace,
				},
				Spec: mcpv1alpha1.MCPServerSpec{
					Image:      "test/mcp-server:latest",
					Transport:  "streamable-http",
					Port:       8080,
					TargetPort: 3000,
				},
			}
			Expect(k8sClient.Create(ctx, mcpServer)).Should(Succeed())

			mcpServerKey := types.NamespacedName{
				Name:      mcpServer.Name,
				Namespace: testNamespace,
			}

			By("Checking the MCPServer is created")
			createdMCPServer := &mcpv1alpha1.MCPServer{}
			Eventually(func() error {
				return k8sClient.Get(ctx, mcpServerKey, createdMCPServer)
			}, timeout, interval).Should(Succeed())

			By("Checking the Deployment is created")
			deployment := waitForDeploymentCreation(mcpServerKey, timeout)
			Expect(deployment.Spec.Template.Spec.Containers).To(HaveLen(1))
			
			// Check toolhive container (which runs the proxy runner)
			container := deployment.Spec.Template.Spec.Containers[0]
			Expect(container.Name).To(Equal("toolhive"))
			// The MCP server image is passed as an argument to the proxy runner
			Expect(container.Args).To(ContainElement("test/mcp-server:latest"))

			By("Checking the Service is created")
			service := waitForServiceCreation(mcpServerKey, timeout)
			Expect(service.Spec.Ports).To(HaveLen(1))
			Expect(service.Spec.Ports[0].Port).To(Equal(int32(8080)))
			Expect(service.Spec.Selector["app"]).To(Equal("test-mcpserver"))

			By("Checking the ServiceAccount is created")
			saKey := types.NamespacedName{
				Name:      mcpServer.Name + "-proxy-runner",
				Namespace: testNamespace,
			}
			waitForServiceAccountCreation(saKey, timeout)

			By("Checking the Role is created")
			roleKey := types.NamespacedName{
				Name:      mcpServer.Name + "-proxy-runner",
				Namespace: testNamespace,
			}
			role := waitForRoleCreation(roleKey, timeout)
			Expect(role.Rules).ToNot(BeEmpty())

			By("Checking the RoleBinding is created")
			roleBindingKey := types.NamespacedName{
				Name:      mcpServer.Name + "-proxy-runner",
				Namespace: testNamespace,
			}
			roleBinding := waitForRoleBindingCreation(roleBindingKey, timeout)
			Expect(roleBinding.Subjects).To(HaveLen(1))
			Expect(roleBinding.Subjects[0].Name).To(Equal(mcpServer.Name + "-proxy-runner"))
		})

		It("Should handle MCPServer with custom environment variables", func() {
			By("Creating an MCPServer with environment variables")
			mcpServer := &mcpv1alpha1.MCPServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-mcpserver-env",
					Namespace: testNamespace,
				},
				Spec: mcpv1alpha1.MCPServerSpec{
					Image:     "test/mcp-server:latest",
					Transport: "stdio",
					Env: []mcpv1alpha1.EnvVar{
						{
							Name:  "TEST_VAR",
							Value: "test-value",
						},
						{
							Name:  "ANOTHER_VAR",
							Value: "another-value",
						},
					},
					Secrets: []mcpv1alpha1.SecretRef{
						{
							Name:          "test-secret",
							Key:           "password",
							TargetEnvName: "SECRET_VAR",
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, mcpServer)).Should(Succeed())

			mcpServerKey := types.NamespacedName{
				Name:      mcpServer.Name,
				Namespace: testNamespace,
			}

			By("Checking environment variables in the deployment")
			deployment := waitForDeploymentCreation(mcpServerKey, timeout)
			container := deployment.Spec.Template.Spec.Containers[0]
			
			// Check for TEST_VAR
			testVarFound := false
			for _, env := range container.Env {
				if env.Name == "TEST_VAR" && env.Value == "test-value" {
					testVarFound = true
					break
				}
			}
			Expect(testVarFound).To(BeTrue())

			// Check for ANOTHER_VAR
			anotherVarFound := false
			for _, env := range container.Env {
				if env.Name == "ANOTHER_VAR" && env.Value == "another-value" {
					anotherVarFound = true
					break
				}
			}
			Expect(anotherVarFound).To(BeTrue())

			// Check for SECRET_VAR from SecretRef
			secretVarFound := false
			for _, env := range container.Env {
				if env.Name == "SECRET_VAR" && env.ValueFrom != nil && 
					env.ValueFrom.SecretKeyRef != nil &&
					env.ValueFrom.SecretKeyRef.Name == "test-secret" {
					secretVarFound = true
					break
				}
			}
			Expect(secretVarFound).To(BeTrue())
		})

		It("Should handle MCPServer with resource limits", func() {
			By("Creating an MCPServer with resource requirements")
			mcpServer := &mcpv1alpha1.MCPServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-mcpserver-resources",
					Namespace: testNamespace,
				},
				Spec: mcpv1alpha1.MCPServerSpec{
					Image:     "test/mcp-server:latest",
					Transport: "stdio",
					Resources: mcpv1alpha1.ResourceRequirements{
						Limits: mcpv1alpha1.ResourceList{
							Memory: "512Mi",
							CPU:    "500m",
						},
						Requests: mcpv1alpha1.ResourceList{
							Memory: "256Mi",
							CPU:    "250m",
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, mcpServer)).Should(Succeed())

			mcpServerKey := types.NamespacedName{
				Name:      mcpServer.Name,
				Namespace: testNamespace,
			}

			By("Checking resource requirements in the deployment")
			deployment := waitForDeploymentCreation(mcpServerKey, timeout)
			container := deployment.Spec.Template.Spec.Containers[0]
			
			// Note: Resources are applied to the proxy runner container
			Expect(container.Resources.Limits).ToNot(BeNil())
			Expect(container.Resources.Limits[corev1.ResourceMemory]).To(Equal(resource.MustParse("512Mi")))
			Expect(container.Resources.Limits[corev1.ResourceCPU]).To(Equal(resource.MustParse("500m")))
			
			Expect(container.Resources.Requests).ToNot(BeNil())
			Expect(container.Resources.Requests[corev1.ResourceMemory]).To(Equal(resource.MustParse("256Mi")))
			Expect(container.Resources.Requests[corev1.ResourceCPU]).To(Equal(resource.MustParse("250m")))
		})

		It("Should handle MCPServer with volumes", func() {
			By("Creating an MCPServer with volume mounts")
			mcpServer := &mcpv1alpha1.MCPServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-mcpserver-volumes",
					Namespace: testNamespace,
				},
				Spec: mcpv1alpha1.MCPServerSpec{
					Image:     "test/mcp-server:latest",
					Transport: "stdio",
					Volumes: []mcpv1alpha1.Volume{
						{
							Name:      "config-volume",
							MountPath: "/etc/config",
							HostPath:  "/tmp/config",
							ReadOnly:  true,
						},
						{
							Name:      "data-volume",
							MountPath: "/data",
							HostPath:  "/tmp/data",
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, mcpServer)).Should(Succeed())

			mcpServerKey := types.NamespacedName{
				Name:      mcpServer.Name,
				Namespace: testNamespace,
			}

			By("Checking volumes in the deployment")
			deployment := waitForDeploymentCreation(mcpServerKey, timeout)
			
			// Check volume mounts in container
			container := deployment.Spec.Template.Spec.Containers[0]
			Expect(container.VolumeMounts).To(HaveLen(2))
			
			configMount := container.VolumeMounts[0]
			Expect(configMount.Name).To(Equal("config-volume"))
			Expect(configMount.MountPath).To(Equal("/etc/config"))
			
			dataMount := container.VolumeMounts[1]
			Expect(dataMount.Name).To(Equal("data-volume"))
			Expect(dataMount.MountPath).To(Equal("/data"))

			// Check volumes in pod spec
			podSpec := deployment.Spec.Template.Spec
			Expect(podSpec.Volumes).To(HaveLen(2))
			
			configVolume := podSpec.Volumes[0]
			Expect(configVolume.Name).To(Equal("config-volume"))
			Expect(configVolume.HostPath).ToNot(BeNil())
			Expect(configVolume.HostPath.Path).To(Equal("/tmp/config"))
			
			dataVolume := podSpec.Volumes[1]
			Expect(dataVolume.Name).To(Equal("data-volume"))
			Expect(dataVolume.HostPath).ToNot(BeNil())
			Expect(dataVolume.HostPath.Path).To(Equal("/tmp/data"))
		})

		It("Should handle MCPServer with custom service account", func() {
			By("Creating a custom service account")
			customSA := &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "custom-sa",
					Namespace: testNamespace,
				},
			}
			Expect(k8sClient.Create(ctx, customSA)).Should(Succeed())

			By("Creating an MCPServer with custom service account")
			saName := "custom-sa"
			mcpServer := &mcpv1alpha1.MCPServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-mcpserver-sa",
					Namespace: testNamespace,
				},
				Spec: mcpv1alpha1.MCPServerSpec{
					Image:          "test/mcp-server:latest",
					Transport:      "stdio",
					ServiceAccount: &saName,
				},
			}
			Expect(k8sClient.Create(ctx, mcpServer)).Should(Succeed())

			mcpServerKey := types.NamespacedName{
				Name:      mcpServer.Name,
				Namespace: testNamespace,
			}

			By("Checking the deployment uses custom service account")
			deployment := waitForDeploymentCreation(mcpServerKey, timeout)
			Expect(deployment.Spec.Template.Spec.ServiceAccountName).To(Equal("custom-sa"))

			By("Verifying no auto-created service account")
			autoSA := &corev1.ServiceAccount{}
			err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      mcpServer.Name,
				Namespace: testNamespace,
			}, autoSA)
			Expect(err).To(HaveOccurred())
		})

		It("Should handle MCPServer with different transport types", func() {
			transports := []string{"stdio", "streamable-http", "sse"}
			
			for _, transport := range transports {
				By("Creating MCPServer with transport: " + transport)
				mcpServer := &mcpv1alpha1.MCPServer{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-mcpserver-" + transport,
						Namespace: testNamespace,
					},
					Spec: mcpv1alpha1.MCPServerSpec{
						Image:     "test/mcp-server:latest",
						Transport: transport,
						Port:      8080,
					},
				}
				Expect(k8sClient.Create(ctx, mcpServer)).Should(Succeed())

				mcpServerKey := types.NamespacedName{
					Name:      mcpServer.Name,
					Namespace: testNamespace,
				}

				By("Checking deployment for transport: " + transport)
				deployment := waitForDeploymentCreation(mcpServerKey, timeout)
				
				// Check proxy container environment
				container := deployment.Spec.Template.Spec.Containers[0]
				transportEnvFound := false
				for _, env := range container.Env {
					if env.Name == "TRANSPORT" && env.Value == transport {
						transportEnvFound = true
						break
					}
				}
				Expect(transportEnvFound).To(BeTrue())

				// For non-stdio transports, check service configuration
				if transport != "stdio" {
					service := waitForServiceCreation(mcpServerKey, timeout)
					Expect(service.Spec.Ports[0].Port).To(Equal(int32(8080)))
				}
			}
		})

		It("Should update MCPServer status", func() {
			By("Creating an MCPServer")
			mcpServer := &mcpv1alpha1.MCPServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-mcpserver-status",
					Namespace: testNamespace,
				},
				Spec: mcpv1alpha1.MCPServerSpec{
					Image:     "test/mcp-server:latest",
					Transport: "stdio",
				},
			}
			Expect(k8sClient.Create(ctx, mcpServer)).Should(Succeed())

			mcpServerKey := types.NamespacedName{
				Name:      mcpServer.Name,
				Namespace: testNamespace,
			}

			By("Waiting for status update")
			updatedMCPServer := waitForMCPServerStatusUpdate(mcpServerKey, timeout)
			
			// The status should be set to Pending initially
			Expect(updatedMCPServer.Status.Phase).To(Equal(mcpv1alpha1.MCPServerPhasePending))
		})

		It("Should handle MCPServer deletion with finalizer", func() {
			By("Creating an MCPServer")
			mcpServer := &mcpv1alpha1.MCPServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-mcpserver-delete",
					Namespace: testNamespace,
				},
				Spec: mcpv1alpha1.MCPServerSpec{
					Image:     "test/mcp-server:latest",
					Transport: "stdio",
				},
			}
			Expect(k8sClient.Create(ctx, mcpServer)).Should(Succeed())

			mcpServerKey := types.NamespacedName{
				Name:      mcpServer.Name,
				Namespace: testNamespace,
			}

			By("Waiting for MCPServer to be ready")
			waitForDeploymentCreation(mcpServerKey, timeout)

			By("Checking finalizer is added")
			finalMCPServer := &mcpv1alpha1.MCPServer{}
			Expect(k8sClient.Get(ctx, mcpServerKey, finalMCPServer)).Should(Succeed())
			Expect(finalMCPServer.Finalizers).To(ContainElement("mcpserver.toolhive.stacklok.dev/finalizer"))

			By("Deleting the MCPServer")
			Expect(k8sClient.Delete(ctx, finalMCPServer)).Should(Succeed())

			By("Checking the MCPServer is eventually deleted")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, mcpServerKey, &mcpv1alpha1.MCPServer{})
				return err != nil
			}, timeout, interval).Should(BeTrue())

			By("Checking associated resources are deleted")
			deployment := &appsv1.Deployment{}
			err := k8sClient.Get(ctx, mcpServerKey, deployment)
			Expect(err).To(HaveOccurred())
			
			service := &corev1.Service{}
			err = k8sClient.Get(ctx, mcpServerKey, service)
			Expect(err).To(HaveOccurred())
		})

		It("Should handle MCPServer with target port configuration", func() {
			By("Creating an MCPServer with custom target port")
			mcpServer := &mcpv1alpha1.MCPServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-mcpserver-targetport",
					Namespace: testNamespace,
				},
				Spec: mcpv1alpha1.MCPServerSpec{
					Image:      "test/mcp-server:latest",
					Transport:  "streamable-http",
					Port:       8080,
					TargetPort: 3000,
				},
			}
			Expect(k8sClient.Create(ctx, mcpServer)).Should(Succeed())

			mcpServerKey := types.NamespacedName{
				Name:      mcpServer.Name,
				Namespace: testNamespace,
			}

			By("Checking service target port configuration")
			service := waitForServiceCreation(mcpServerKey, timeout)
			Expect(service.Spec.Ports[0].Port).To(Equal(int32(8080)))
			Expect(service.Spec.Ports[0].TargetPort).To(Equal(intstr.FromInt(3000)))

			By("Checking deployment arguments for target port")
			deployment := waitForDeploymentCreation(mcpServerKey, timeout)
			container := deployment.Spec.Template.Spec.Containers[0]
			
			// Check that the target port is passed in arguments
			Expect(container.Args).To(ContainElement("--proxy-port=3000"))
		})
	})
})