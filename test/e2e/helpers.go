// Package e2e provides end-to-end testing utilities for ToolHive.
package e2e

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck // Standard practice for Ginkgo
	. "github.com/onsi/gomega"    //nolint:staticcheck // Standard practice for Gomega

	"github.com/stacklok/toolhive/pkg/groups"
	"github.com/stacklok/toolhive/pkg/workloads"
)

// TestConfig holds configuration for e2e tests
type TestConfig struct {
	THVBinary    string
	TestTimeout  time.Duration
	CleanupAfter bool
}

// NewTestConfig creates a new test configuration with defaults
func NewTestConfig() *TestConfig {
	// Look for thv binary in PATH or use a configurable path
	thvBinary := os.Getenv("THV_BINARY")
	if thvBinary == "" {
		thvBinary = "thv" // Assume it's in PATH
	}

	return &TestConfig{
		THVBinary:    thvBinary,
		TestTimeout:  10 * time.Minute,
		CleanupAfter: true,
	}
}

// THVCommand represents a ToolHive CLI command execution
type THVCommand struct {
	config *TestConfig
	args   []string
	env    []string
	dir    string
}

// NewTHVCommand creates a new ToolHive command
func NewTHVCommand(config *TestConfig, args ...string) *THVCommand {
	return &THVCommand{
		config: config,
		args:   args,
		env:    os.Environ(),
		dir:    "",
	}
}

// WithEnv adds environment variables to the command
func (c *THVCommand) WithEnv(env ...string) *THVCommand {
	c.env = append(c.env, env...)
	return c
}

// WithDir sets the working directory for the command
func (c *THVCommand) WithDir(dir string) *THVCommand {
	c.dir = dir
	return c
}

// Run executes the ToolHive command and returns stdout, stderr, and error
func (c *THVCommand) Run() (string, string, error) {
	return c.RunWithTimeout(c.config.TestTimeout)
}

// RunWithTimeout executes the ToolHive command with a specific timeout
func (c *THVCommand) RunWithTimeout(timeout time.Duration) (string, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, c.config.THVBinary, c.args...) //nolint:gosec // Intentional for e2e testing
	cmd.Env = c.env
	if c.dir != "" {
		cmd.Dir = c.dir
	}

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	return stdout.String(), stderr.String(), err
}

// ExpectSuccess runs the command and expects it to succeed
func (c *THVCommand) ExpectSuccess() (string, string) {
	stdout, stderr, err := c.Run()
	if err != nil {
		// Log the command that failed for debugging
		GinkgoWriter.Printf("Command failed: %s %v\nError: %v\nStdout: %s\nStderr: %s\n",
			c.config.THVBinary, c.args, err, stdout, stderr)
	}
	ExpectWithOffset(1, err).ToNot(HaveOccurred(),
		fmt.Sprintf("Command failed: %v\nStdout: %s\nStderr: %s", err, stdout, stderr))
	return stdout, stderr
}

// ExpectFailure runs the command and expects it to fail
func (c *THVCommand) ExpectFailure() (string, string, error) {
	stdout, stderr, err := c.Run()
	ExpectWithOffset(1, err).To(HaveOccurred(),
		fmt.Sprintf("Command should have failed but succeeded\nStdout: %s\nStderr: %s", stdout, stderr))
	return stdout, stderr, err
}

// WaitForMCPServer waits for an MCP server to be running
func WaitForMCPServer(config *TestConfig, serverName string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for MCP server %s to be running", serverName)
		case <-ticker.C:
			stdout, _, err := NewTHVCommand(config, "list").Run()
			if err != nil {
				continue
			}

			// Check if the server is listed and running
			if strings.Contains(stdout, serverName) && strings.Contains(stdout, "running") {
				return nil
			}
		}
	}
}

// StopAndRemoveMCPServer stops and removes an MCP server
// This function is designed for cleanup and tolerates servers that don't exist
func StopAndRemoveMCPServer(config *TestConfig, serverName string) error {
	// Try to stop the server first (ignore errors as server might not exist)
	_, _, _ = NewTHVCommand(config, "stop", serverName).Run()

	// Then remove it
	_, stderr, err := NewTHVCommand(config, "rm", serverName).Run()
	if err != nil {
		// In cleanup scenarios, it's okay if the container doesn't exist
		if strings.Contains(stderr, "not found") {
			return nil
		}
		return err
	}
	return nil
}

// GetMCPServerURL gets the URL for an MCP server
func GetMCPServerURL(config *TestConfig, serverName string) (string, error) {
	stdout, stderr, err := NewTHVCommand(config, "list").Run()
	if err != nil {
		GinkgoWriter.Printf("Failed to list servers: %v\nStdout: %s\nStderr: %s\n", err, stdout, stderr)
		return "", fmt.Errorf("failed to list servers: %w", err)
	}

	GinkgoWriter.Printf("thv list output:\n%s\n", stdout)

	lines := strings.Split(stdout, "\n")
	for _, line := range lines {
		if strings.Contains(line, serverName) {
			GinkgoWriter.Printf("Found server line: %s\n", line)
			// Parse the URL from the list output
			// This is a simplified parser - you might need to adjust based on actual output format
			parts := strings.Fields(line)
			for _, part := range parts {
				if strings.HasPrefix(part, "http://") || strings.HasPrefix(part, "https://") {
					GinkgoWriter.Printf("Found URL: %s\n", part)
					return part, nil
				}
			}
		}
	}

	return "", fmt.Errorf("could not find URL for server %s in output: %s", serverName, stdout)
}

// GetServerLogs gets the logs for a server to help with debugging
func GetServerLogs(config *TestConfig, serverName string) (string, error) {
	stdout, stderr, err := NewTHVCommand(config, "logs", serverName).Run()
	if err != nil {
		return "", fmt.Errorf("failed to get logs for %s: %w (stderr: %s)", serverName, err, stderr)
	}
	return stdout, nil
}

// DebugServerState prints debugging information about a server
func DebugServerState(config *TestConfig, serverName string) {
	GinkgoWriter.Printf("=== Debugging server state for %s ===\n", serverName)

	// Get list output
	stdout, stderr, err := NewTHVCommand(config, "list").Run()
	GinkgoWriter.Printf("thv list output:\nStdout: %s\nStderr: %s\nError: %v\n", stdout, stderr, err)

	// Get logs
	logs, err := GetServerLogs(config, serverName)
	if err != nil {
		GinkgoWriter.Printf("Failed to get logs: %v\n", err)
	} else {
		GinkgoWriter.Printf("Server logs:\n%s\n", logs)
	}

	GinkgoWriter.Printf("=== End debugging for %s ===\n", serverName)
}

// CheckTHVBinaryAvailable checks if the thv binary is available
func CheckTHVBinaryAvailable(config *TestConfig) error {
	_, _, err := NewTHVCommand(config, "--help").Run()
	if err != nil {
		return fmt.Errorf("thv binary not available at %s: %w", config.THVBinary, err)
	}
	return nil
}

// StartLongRunningTHVCommand starts a long-running ToolHive command and returns the process
func StartLongRunningTHVCommand(config *TestConfig, args ...string) *exec.Cmd {
	cmd := exec.Command(config.THVBinary, args...) //nolint:gosec // Intentional for e2e testing
	cmd.Env = os.Environ()

	// Capture stdout and stderr for debugging
	cmd.Stdout = GinkgoWriter
	cmd.Stderr = GinkgoWriter

	err := cmd.Start()
	ExpectWithOffset(1, err).ToNot(HaveOccurred(),
		fmt.Sprintf("Failed to start long-running command: %s %v", config.THVBinary, args))

	return cmd
}

// StartDockerCommand starts a docker command with proper environment setup and returns the command
func StartDockerCommand(args ...string) *exec.Cmd {
	cmd := exec.Command("docker", args...) //nolint:gosec // Intentional for e2e testing
	cmd.Env = os.Environ()
	return cmd
}

// Helper function to clean up a specific group and its workloads
func cleanupSpecificGroup(groupName string) {
	// Use the groups manager to delete the specific group and its workloads
	groupManager, err := groups.NewManager()
	if err != nil {
		// If we can't create the group manager, we can't clean up, so just return
		return
	}

	ctx := context.Background()

	// Check if the group exists before trying to delete it
	exists, err := groupManager.Exists(ctx, groupName)
	if err != nil || !exists {
		// Group doesn't exist or we can't check, so nothing to clean up
		return
	}

	// Get all workloads in the group
	groupWorkloads, err := groupManager.ListWorkloadsInGroup(ctx, groupName)
	if err != nil {
		// If we can't list workloads, just try to delete the group
		_ = groupManager.Delete(ctx, groupName)
		return
	}

	// If there are workloads in the group, delete them first
	if len(groupWorkloads) > 0 {
		workloadManager, err := workloads.NewManager(ctx)
		if err != nil {
			// If we can't create the workload manager, just try to delete the group
			_ = groupManager.Delete(ctx, groupName)
			return
		}

		// Delete all workloads in the group
		group, err := workloadManager.DeleteWorkloads(ctx, groupWorkloads)
		if err != nil {
			// If we can't delete workloads, just try to delete the group
			_ = groupManager.Delete(ctx, groupName)
			return
		}

		// Wait for all workload deletions to complete
		if err := group.Wait(); err != nil {
			// If workload deletion failed, just try to delete the group
			_ = groupManager.Delete(ctx, groupName)
			return
		}
	}

	// Delete the group itself
	_ = groupManager.Delete(ctx, groupName) // Ignore errors during cleanup
}

// Helper functions for group and workload management

func createGroup(config *TestConfig, groupName string) {
	NewTHVCommand(config, "group", "create", groupName).ExpectSuccess()
}

func createWorkloadInGroup(config *TestConfig, workloadName, groupName string) {
	NewTHVCommand(config, "run", "fetch", "--group", groupName, "--name", workloadName).ExpectSuccess()
}

func createWorkload(config *TestConfig, workloadName string) {
	NewTHVCommand(config, "run", "fetch", "--name", workloadName).ExpectSuccess()
}

func removeWorkload(config *TestConfig, workloadName string) {
	NewTHVCommand(config, "rm", workloadName).ExpectSuccess()
}

func isWorkloadRunning(config *TestConfig, workloadName string) bool {
	stdout, _ := NewTHVCommand(config, "list", "--all").ExpectSuccess()
	return strings.Contains(stdout, workloadName)
}

func waitForWorkload(config *TestConfig, workloadName string) bool {
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if isWorkloadRunning(config, workloadName) {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}
