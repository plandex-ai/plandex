package model

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"
)

var (
	liteLLMOnce sync.Once
	liteLLMCmd  *exec.Cmd
)

func EnsureLiteLLM(numWorkers int) error {
	var finalErr error
	liteLLMOnce.Do(func() {
		if isLiteLLMHealthy() {
			log.Println("LiteLLM proxy is already healthy")
			return
		}

		log.Println("LiteLLM proxy is not running. Starting...")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := startLiteLLMServer(numWorkers)
		if err != nil {
			log.Println("LiteLLM proxy launch failed:", err)
			finalErr = fmt.Errorf("LiteLLM proxy launch failed: %w", err)
			return
		}

		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Println("LiteLLM proxy launch timed out")
				finalErr = fmt.Errorf("LiteLLM proxy launch timed out")
				return
			case <-ticker.C:
				if isLiteLLMHealthy() {
					log.Println("LiteLLM proxy is healthy")
					return
				} else {
					log.Println("LiteLLM proxy is not healthy yet, retrying after 500ms...")
				}
			}
		}
	})

	return finalErr
}

func ShutdownLiteLLMServer() error {
	if liteLLMCmd != nil && liteLLMCmd.Process != nil {
		log.Println("Shutting down LiteLLM proxy gracefully...")
		if err := liteLLMCmd.Process.Signal(os.Interrupt); err != nil {
			return fmt.Errorf("failed to signal LiteLLM for shutdown: %w", err)
		}

		done := make(chan error, 1)
		go func() {
			done <- liteLLMCmd.Wait()
		}()

		select {
		case <-time.After(5 * time.Second):
			log.Println("LiteLLM proxy shutdown timed out, forcing kill")
			return liteLLMCmd.Process.Kill()
		case err := <-done:
			return err
		}
	}
	return nil
}

func isLiteLLMHealthy() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "http://localhost:4000/health", nil)
	if err != nil {
		log.Println("LiteLLM health check request failed:", err)
		return false
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println("LiteLLM health check failed:", err)
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == 200
}

func startLiteLLMServer(numWorkers int) error {
	liteLLMCmd = exec.Command("python3",
		"-m", "uvicorn",
		"litellm_proxy:app",
		"--host", "0.0.0.0",
		"--port", "4000",
		"--workers", strconv.Itoa(numWorkers),
	)

	if os.Getenv("LITELLM_PROXY_DIR") != "" {
		liteLLMCmd.Dir = os.Getenv("LITELLM_PROXY_DIR")
	}

	// clean env
	liteLLMCmd.Env = []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + os.Getenv("HOME"),
	}

	liteLLMCmd.Stdout = os.Stdout
	liteLLMCmd.Stderr = os.Stderr

	err := liteLLMCmd.Start()
	if err != nil {
		return err
	}

	log.Println("LiteLLM proxy launched")
	return nil
}
