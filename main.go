package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

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
		strings.HasPrefix(req.RequestURI, "/containers/create?") ||
		(strings.Contains(req.RequestURI, "/containers/") && strings.Contains(req.RequestURI, "/update"))) {

		var createReq ContainerCreateRequest
		if err := json.Unmarshal(req.RequestBody, &createReq); err != nil {
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

func main() {
	// Create plugin socket directory
	socketDir := filepath.Dir(pluginSocket)
	if err := os.MkdirAll(socketDir, 0755); err != nil {
		log.Fatalf("Failed to create socket directory: %v", err)
	}

	log.Printf("Docker restart policy guard starting on %s", pluginSocket)

	// Start plugin handler
	handler := authorization.NewHandler(&Plugin{})
	if err := handler.ServeUnix(pluginSocket, 0); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
