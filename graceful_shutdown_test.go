package main

import (
	"os/exec"
	"syscall"
	"testing"
	"time"
)

func TestGracefulShutdown(t *testing.T) {
	// Test that the binary handles SIGTERM gracefully
	cmd := exec.Command("./librecash")

	// Start the process
	err := cmd.Start()
	if err != nil {
		t.Fatalf("Failed to start librecash: %v", err)
	}

	// Give it a moment to start
	time.Sleep(2 * time.Second)

	// Send SIGTERM
	err = cmd.Process.Signal(syscall.SIGTERM)
	if err != nil {
		t.Fatalf("Failed to send SIGTERM: %v", err)
	}

	// Wait for process to exit with timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Process exited with error: %v", err)
		} else {
			t.Log("Process exited gracefully")
		}
	case <-time.After(35 * time.Second):
		cmd.Process.Kill()
		t.Error("Process did not exit within 35 seconds")
	}
}

func TestGracefulShutdownSIGINT(t *testing.T) {
	// Test that the binary handles SIGINT (Ctrl+C) gracefully
	cmd := exec.Command("./librecash")

	// Start the process
	err := cmd.Start()
	if err != nil {
		t.Fatalf("Failed to start librecash: %v", err)
	}

	// Give it a moment to start
	time.Sleep(2 * time.Second)

	// Send SIGINT (Ctrl+C)
	err = cmd.Process.Signal(syscall.SIGINT)
	if err != nil {
		t.Fatalf("Failed to send SIGINT: %v", err)
	}

	// Wait for process to exit with timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Process exited with error: %v", err)
		} else {
			t.Log("Process exited gracefully on SIGINT")
		}
	case <-time.After(35 * time.Second):
		cmd.Process.Kill()
		t.Error("Process did not exit within 35 seconds on SIGINT")
	}
}

func TestGracefulShutdownTimeout(t *testing.T) {
	// Test that the binary exits within 30 seconds even if operations are still running
	cmd := exec.Command("./librecash")

	// Start the process
	err := cmd.Start()
	if err != nil {
		t.Fatalf("Failed to start librecash: %v", err)
	}

	// Give it a moment to start
	time.Sleep(2 * time.Second)

	// Send SIGTERM
	err = cmd.Process.Signal(syscall.SIGTERM)
	if err != nil {
		t.Fatalf("Failed to send SIGTERM: %v", err)
	}

	// Wait for process to exit - should be within 30 seconds
	start := time.Now()
	err = cmd.Wait()
	duration := time.Since(start)

	if err != nil {
		t.Errorf("Process exited with error: %v", err)
	}

	if duration > 35*time.Second {
		t.Errorf("Process took too long to exit: %v (should be <= 30s)", duration)
	} else {
		t.Logf("Process exited in %v", duration)
	}
}
