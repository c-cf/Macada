package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// DockerClient wraps the Docker Engine API via HTTP.
// Uses the Docker socket directly — no third-party SDK dependency.
type DockerClient struct {
	client  *http.Client
	baseURL string
}

// NewDockerClient creates a client connecting to the Docker daemon.
// socketPath defaults to "unix:///var/run/docker.sock" on Linux.
func NewDockerClient(host string) *DockerClient {
	if host == "" {
		host = "http://localhost:2375"
	}
	return &DockerClient{
		client:  &http.Client{Timeout: 2 * time.Minute},
		baseURL: strings.TrimRight(host, "/"),
	}
}

// CreateOpts configures container creation.
type CreateOpts struct {
	Image      string
	Name       string
	Env        []string
	Network    string
	MemoryMB   int64
	CPUs       float64
	WorkDir    string
}

// ContainerInfo holds info about a created container.
type ContainerInfo struct {
	ID     string
	IP     string
	Status string
}

// CreateAndStart creates and starts a container, returning its info.
func (c *DockerClient) CreateAndStart(ctx context.Context, opts CreateOpts) (*ContainerInfo, error) {
	// Create container
	createBody := map[string]interface{}{
		"Image": opts.Image,
		"Env":   opts.Env,
		"HostConfig": map[string]interface{}{
			"NetworkMode": opts.Network,
		},
	}
	if opts.WorkDir != "" {
		createBody["WorkingDir"] = opts.WorkDir
	}
	if opts.MemoryMB > 0 {
		hc := createBody["HostConfig"].(map[string]interface{})
		hc["Memory"] = opts.MemoryMB * 1024 * 1024
	}
	if opts.CPUs > 0 {
		hc := createBody["HostConfig"].(map[string]interface{})
		hc["NanoCpus"] = int64(opts.CPUs * 1e9)
	}

	data, _ := json.Marshal(createBody)
	url := fmt.Sprintf("%s/containers/create", c.baseURL)
	if opts.Name != "" {
		url += "?name=" + opts.Name
	}

	resp, err := c.doJSON(ctx, http.MethodPost, url, data)
	if err != nil {
		return nil, fmt.Errorf("create container: %w", err)
	}

	var createResp struct {
		ID string `json:"Id"`
	}
	if err := json.Unmarshal(resp, &createResp); err != nil {
		return nil, fmt.Errorf("parse create response: %w", err)
	}

	// Start container
	startURL := fmt.Sprintf("%s/containers/%s/start", c.baseURL, createResp.ID)
	if _, err := c.do(ctx, http.MethodPost, startURL, nil); err != nil {
		return nil, fmt.Errorf("start container: %w", err)
	}

	// Inspect for IP
	info, err := c.Inspect(ctx, createResp.ID)
	if err != nil {
		return nil, err
	}

	return info, nil
}

// Inspect returns info about a container.
func (c *DockerClient) Inspect(ctx context.Context, containerID string) (*ContainerInfo, error) {
	url := fmt.Sprintf("%s/containers/%s/json", c.baseURL, containerID)
	resp, err := c.do(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("inspect container: %w", err)
	}

	var inspectResp struct {
		ID    string `json:"Id"`
		State struct {
			Status string `json:"Status"`
		} `json:"State"`
		NetworkSettings struct {
			Networks map[string]struct {
				IPAddress string `json:"IPAddress"`
			} `json:"Networks"`
		} `json:"NetworkSettings"`
	}
	if err := json.Unmarshal(resp, &inspectResp); err != nil {
		return nil, fmt.Errorf("parse inspect response: %w", err)
	}

	ip := ""
	for _, net := range inspectResp.NetworkSettings.Networks {
		if net.IPAddress != "" {
			ip = net.IPAddress
			break
		}
	}

	return &ContainerInfo{
		ID:     inspectResp.ID,
		IP:     ip,
		Status: inspectResp.State.Status,
	}, nil
}

// Stop stops a running container.
func (c *DockerClient) Stop(ctx context.Context, containerID string) error {
	url := fmt.Sprintf("%s/containers/%s/stop?t=10", c.baseURL, containerID)
	_, err := c.do(ctx, http.MethodPost, url, nil)
	return err
}

// Remove removes a container.
func (c *DockerClient) Remove(ctx context.Context, containerID string) error {
	url := fmt.Sprintf("%s/containers/%s?force=true", c.baseURL, containerID)
	_, err := c.do(ctx, http.MethodDelete, url, nil)
	return err
}

// CopyToContainer copies files into a container as a tar archive.
func (c *DockerClient) CopyToContainer(ctx context.Context, containerID, destPath string, files map[string][]byte) error {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	for name, content := range files {
		hdr := &tar.Header{
			Name: name,
			Size: int64(len(content)),
			Mode: 0o644,
		}
		if strings.HasPrefix(name, "scripts/") || strings.HasSuffix(name, ".sh") {
			hdr.Mode = 0o755
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return fmt.Errorf("write tar header %s: %w", name, err)
		}
		if _, err := tw.Write(content); err != nil {
			return fmt.Errorf("write tar content %s: %w", name, err)
		}
	}
	if err := tw.Close(); err != nil {
		return fmt.Errorf("close tar: %w", err)
	}

	url := fmt.Sprintf("%s/containers/%s/archive?path=%s", c.baseURL, containerID, destPath)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, &buf)
	if err != nil {
		return fmt.Errorf("create copy request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-tar")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("copy to container: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("copy failed %d: %s", resp.StatusCode, body)
	}

	return nil
}

func (c *DockerClient) doJSON(ctx context.Context, method, url string, body []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("docker api %d: %s", resp.StatusCode, respBody)
	}

	return respBody, nil
}

func (c *DockerClient) do(ctx context.Context, method, url string, body []byte) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("docker api %d: %s", resp.StatusCode, respBody)
	}

	return respBody, nil
}
