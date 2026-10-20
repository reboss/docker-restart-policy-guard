package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"github.com/docker/go-plugins-helpers/authorization"
)

const (
	pluginSocket = "/run/docker/plugins/restart-policy-guard.sock"
)

type Plugin struct{}

type ContainerCreateRequest struct {
	HostConfig *HostConfig `json:"HostConfig"`
}

type HostConfig struct {
	RestartPolicy RestartPolicy `json:"RestartPolicy"`
}

type RestartPolicy struct {
	Name              string `json:"Name"`
	MaximumRetryCount int    `json:"MaximumRetryCount"`
}

func (p *Plugin) AuthZReq(req authorization.Request) authorization.Response {
	// Only check container creation and updates
	if req.RequestMethod == "POST" && (req.RequestURI == "/containers/create" ||
		req.RequestURI == "/containers/create?" ||
		contains(req.RequestURI, "/containers/") && contains(req.RequestURI, "/update")) {

		body, err := io.ReadAll(req.RequestBody)
		if err != nil {
			log.Printf("Failed to read request body: %v", err)
			return authorization.Response{Allow: true}
		}

		var createReq ContainerCreateRequest
		if err := json.Unmarshal(body, &createReq); err != nil {
			log.Printf("Failed to parse request body: %v", err)
			return authorization.Response{Allow: true}
		}

		if createReq.HostConfig != nil && createReq.HostConfig.RestartPolicy.Name != "" &&
			createReq.HostConfig.RestartPolicy.Name != "no" {
			return authorization.Response{
				Allow: false,
				Msg:   fmt.Sprintf("Container restart policies are not allowed in this cluster (attempted: %s)", createReq.HostConfig.RestartPolicy.Name),
			}
		}
	}

	return authorization.Response{Allow: true}
}

func (p *Plugin) AuthZRes(req authorization.Request) authorization.Response {
	return authorization.Response{Allow: true}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr ||
		   len(s) > len(substr) && s[len(s)-len(substr):] == substr ||
		   len(s) > len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func main() {
	// Remove existing socket
	if err := os.RemoveAll(pluginSocket); err != nil {
		log.Fatalf("Failed to remove existing socket: %v", err)
	}

	// Create plugin socket directory
	socketDir := filepath.Dir(pluginSocket)
	if err := os.MkdirAll(socketDir, 0755); err != nil {
		log.Fatalf("Failed to create socket directory: %v", err)
	}

	// Create Unix listener
	listener, err := net.Listen("unix", pluginSocket)
	if err != nil {
		log.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	// Set socket permissions
	if err := os.Chmod(pluginSocket, 0660); err != nil {
		log.Fatalf("Failed to set socket permissions: %v", err)
	}

	log.Printf("Docker restart policy guard listening on %s", pluginSocket)

	// Start plugin handler
	handler := authorization.NewHandler(&Plugin{})
	if err := http.Serve(listener, handler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
