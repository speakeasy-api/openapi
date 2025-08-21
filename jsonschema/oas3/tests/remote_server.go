package tests

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// RemoteServer manages a testcontainer that serves the JSON Schema Test Suite remote files
type RemoteServer struct {
	container testcontainers.Container
	baseURL   string
}

// startRemoteServer starts a container serving the remote files at localhost:1234
func startRemoteServer() (*RemoteServer, error) {
	ctx := context.Background()

	// Get the absolute path to the remotes directory
	remotesPath, err := filepath.Abs("testsuite/remotes")
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path to remotes directory: %w", err)
	}

	// Check if remotes directory exists
	if _, err := os.Stat(remotesPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("remotes directory does not exist: %s", remotesPath)
	}

	// Create nginx configuration that enables directory indexing and proper file serving
	nginxConfig := `
server {
	   listen 80;
	   server_name localhost;
	   root /remotes;
	   
	   # Enable directory indexing
	   autoindex on;
	   autoindex_exact_size off;
	   autoindex_localtime on;
	   
	   # Serve files with proper MIME types
	   location / {
	       try_files $uri $uri/ =404;
	       add_header Access-Control-Allow-Origin *;
	       add_header Access-Control-Allow-Methods "GET, POST, OPTIONS";
	       add_header Access-Control-Allow-Headers "DNT,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type,Range";
	   }
	   
	   # Specific handling for JSON files
	   location ~* \.json$ {
	       add_header Content-Type application/json;
	       add_header Access-Control-Allow-Origin *;
	   }
}
`

	// Create container request with fixed port binding to 1234
	req := testcontainers.ContainerRequest{
		Image:        "nginx:alpine",
		ExposedPorts: []string{"80/tcp"},
		Files: []testcontainers.ContainerFile{
			{
				HostFilePath:      remotesPath,
				ContainerFilePath: "/remotes",
				FileMode:          0755,
			},
			{
				HostFilePath:      "",
				ContainerFilePath: "/etc/nginx/conf.d/default.conf",
				FileMode:          0644,
				Reader:            strings.NewReader(nginxConfig),
			},
		},
		WaitingFor: wait.ForHTTP("/draft2020-12/integer.json").WithPort("80/tcp").WithStartupTimeout(30 * time.Second),
	}

	// Start the container with port binding
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start remote server container: %w", err)
	}

	// Get the mapped port
	mappedPort, err := container.MappedPort(ctx, "80")
	if err != nil {
		if termErr := container.Terminate(ctx); termErr != nil {
			return nil, fmt.Errorf("failed to get mapped port: %w (cleanup error: %w)", err, termErr)
		}
		return nil, fmt.Errorf("failed to get mapped port: %w", err)
	}

	// Get the host
	host, err := container.Host(ctx)
	if err != nil {
		if termErr := container.Terminate(ctx); termErr != nil {
			return nil, fmt.Errorf("failed to get container host: %w (cleanup error: %w)", err, termErr)
		}
		return nil, fmt.Errorf("failed to get container host: %w", err)
	}

	baseURL := fmt.Sprintf("http://%s:%s", host, mappedPort.Port())

	// Verify the server is working by checking a known file
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(baseURL + "/draft2020-12/integer.json")
	if err != nil {
		if termErr := container.Terminate(ctx); termErr != nil {
			return nil, fmt.Errorf("failed to verify remote server is working: %w (cleanup error: %w)", err, termErr)
		}
		return nil, fmt.Errorf("failed to verify remote server is working: %w", err)
	}
	if resp == nil {
		if termErr := container.Terminate(ctx); termErr != nil {
			return nil, fmt.Errorf("received nil response from remote server (cleanup error: %w)", termErr)
		}
		return nil, errors.New("received nil response from remote server")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if termErr := container.Terminate(ctx); termErr != nil {
			return nil, fmt.Errorf("remote server health check failed with status: %d (cleanup error: %w)", resp.StatusCode, termErr)
		}
		return nil, fmt.Errorf("remote server health check failed with status: %d", resp.StatusCode)
	}

	return &RemoteServer{
		container: container,
		baseURL:   baseURL,
	}, nil
}

// GetBaseURL returns the base URL where the remote files are served
func (rs *RemoteServer) GetBaseURL() string {
	return rs.baseURL
}

// Stop stops and removes the container
func (rs *RemoteServer) Stop() {
	if rs.container != nil {
		ctx := context.Background()
		err := rs.container.Terminate(ctx)
		if err != nil {
			// Use fmt.Printf since we can't access log in this context
			fmt.Printf("Warning: failed to terminate remote server container: %v\n", err)
		}
	}
}

// GetExpectedURL returns the URL that the test suite expects for a given path
// Since we're binding to localhost:1234, this is the same as GetActualURL
func (rs *RemoteServer) GetExpectedURL(path string) string {
	return "http://localhost:1234/" + path
}

// GetActualURL returns the actual URL where the file is served
func (rs *RemoteServer) GetActualURL(path string) string {
	return rs.baseURL + "/" + path
}

// GetHTTPClient returns an HTTP client that redirects localhost:1234 requests to the actual container
func (rs *RemoteServer) GetHTTPClient() *http.Client {
	return &http.Client{
		Transport: &redirectTransport{
			baseURL: rs.baseURL,
			base:    http.DefaultTransport,
		},
		Timeout: 30 * time.Second,
	}
}

// redirectTransport is an HTTP transport that redirects localhost:1234 requests to the actual container URL
type redirectTransport struct {
	baseURL string
	base    http.RoundTripper
}

func (rt *redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Check if this is a localhost:1234 request
	if req.URL.Host == "localhost:1234" {
		// Parse the container base URL
		containerURL, err := url.Parse(rt.baseURL)
		if err != nil {
			return nil, fmt.Errorf("failed to parse container URL: %w", err)
		}

		// Create a new URL with the container host/port but keep the original path
		newURL := &url.URL{
			Scheme:   containerURL.Scheme,
			Host:     containerURL.Host,
			Path:     req.URL.Path,
			RawQuery: req.URL.RawQuery,
			Fragment: req.URL.Fragment,
		}

		// Clone the request with the new URL
		newReq := req.Clone(req.Context())
		newReq.URL = newURL

		return rt.base.RoundTrip(newReq)
	}

	// For all other requests, use the base transport
	return rt.base.RoundTrip(req)
}
