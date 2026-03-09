package services

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ServiceEndpoint provides HTTP communication with a managed service.
type ServiceEndpoint struct {
	BaseURL string
	Client  *http.Client
}

// NewEndpoint creates an endpoint for a given service type and port.
func NewEndpoint(stype ServiceType, port int) *ServiceEndpoint {
	return &ServiceEndpoint{
		BaseURL: fmt.Sprintf("http://127.0.0.1:%d", port),
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// BrowserEndpoint returns an endpoint for the browser service.
func (m *Manager) BrowserEndpoint() *ServiceEndpoint {
	return NewEndpoint(ServiceBrowser, m.config.BrowserPort)
}

// SandboxEndpoint returns an endpoint for the sandbox service.
func (m *Manager) SandboxEndpoint() *ServiceEndpoint {
	return NewEndpoint(ServiceSandbox, m.config.SandboxPort)
}

// RepoEndpoint returns an endpoint for the repo service.
func (m *Manager) RepoEndpoint() *ServiceEndpoint {
	return NewEndpoint(ServiceRepo, m.config.RepoPort)
}

// Get sends a GET request to the service.
func (e *ServiceEndpoint) Get(ctx context.Context, path string) ([]byte, error) {
	url := e.BaseURL + path
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := e.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("service request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("service returned %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// Post sends a POST request to the service.
func (e *ServiceEndpoint) Post(ctx context.Context, path string, contentType string, body string) ([]byte, error) {
	url := e.BaseURL + path
	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)

	resp, err := e.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("service request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("service returned %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// Health checks if the service endpoint is responding.
func (e *ServiceEndpoint) Health(ctx context.Context) error {
	_, err := e.Get(ctx, "/health")
	return err
}
