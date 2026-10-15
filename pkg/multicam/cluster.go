package multicam

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/hitraa/siyi-sdk-go"
	"github.com/hitraa/siyi-sdk-go/pkg/models"
	"github.com/hitraa/siyi-sdk-go/pkg/protocol"
)

// Role designates the camera's purpose in a multi-camera system.
type Role string

const (
	RolePrimary   Role = "primary"   // Default gimbal camera (192.168.144.25)
	RoleSecondary Role = "secondary" // Secondary optical pod (192.168.144.26)
	RoleAI        Role = "ai"        // AI tracking module (192.168.144.60)
)

// NodeConfig configures a single camera endpoint in the cluster.
type NodeConfig struct {
	Name string
	Role Role
	IP   string
	Port int
}

// DefaultTopology returns the standard SIYI dual-camera and AI module layout.
func DefaultTopology() []NodeConfig {
	return []NodeConfig{
		{Name: "MainCamera", Role: RolePrimary, IP: "192.168.144.25", Port: protocol.DefaultUDPPort},
		{Name: "CameraB", Role: RoleSecondary, IP: "192.168.144.26", Port: protocol.DefaultUDPPort},
		{Name: "AIModule", Role: RoleAI, IP: "192.168.144.60", Port: protocol.DefaultUDPPort},
	}
}

// Cluster manages simultaneous communication and synchronized commands across multiple SIYI cameras.
type Cluster struct {
	mu      sync.RWMutex
	clients map[string]*siyi.Client
	roles   map[Role]*siyi.Client
	configs []NodeConfig
}

// NewCluster instantiates a cluster with specified camera configurations.
func NewCluster(configs ...NodeConfig) *Cluster {
	if len(configs) == 0 {
		configs = DefaultTopology()
	}
	return &Cluster{
		clients: make(map[string]*siyi.Client),
		roles:   make(map[Role]*siyi.Client),
		configs: configs,
	}
}

// Connect initializes and connects all camera nodes concurrently.
func (c *Cluster) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var wg sync.WaitGroup
	errCh := make(chan error, len(c.configs))

	for _, cfg := range c.configs {
		wg.Add(1)
		go func(conf NodeConfig) {
			defer wg.Done()
			client, err := siyi.ConnectUDP(ctx, conf.IP, conf.Port)
			if err != nil {
				errCh <- fmt.Errorf("node %s (%s): %w", conf.Name, conf.IP, err)
				return
			}
			c.mu.Lock()
			c.clients[conf.Name] = client
			c.roles[conf.Role] = client
			c.mu.Unlock()
		}(cfg)
	}

	wg.Wait()
	close(errCh)

	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// Primary returns the primary gimbal camera client.
func (c *Cluster) Primary() *siyi.Client {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.roles[RolePrimary]
}

// Secondary returns the secondary camera client (e.g. Camera B).
func (c *Cluster) Secondary() *siyi.Client {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.roles[RoleSecondary]
}

// AI returns the AI tracking module client.
func (c *Cluster) AI() *siyi.Client {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.roles[RoleAI]
}

// Node returns the client for a specific node by name.
func (c *Cluster) Node(name string) (*siyi.Client, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	client, ok := c.clients[name]
	return client, ok
}

// AddClient registers an already created client to the cluster under the given name and role.
func (c *Cluster) AddClient(name string, role Role, client *siyi.Client) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.clients[name] = client
	c.roles[role] = client
}

// RecenterAll dispatches recentering commands concurrently across all connected cameras.
func (c *Cluster) RecenterAll(ctx context.Context) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var wg sync.WaitGroup
	for _, client := range c.clients {
		wg.Add(1)
		go func(cl *siyi.Client) {
			defer wg.Done()
			_ = cl.OneKeyCentering(ctx, models.CenteringActionOneKeyCenter)
		}(client)
	}
	wg.Wait()
	return nil
}

// SyncCapture triggers synchronized photo capture across all connected cameras.
func (c *Cluster) SyncCapture(ctx context.Context) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var wg sync.WaitGroup
	for _, client := range c.clients {
		wg.Add(1)
		go func(cl *siyi.Client) {
			defer wg.Done()
			_ = cl.TakePhoto(ctx)
		}(client)
	}
	wg.Wait()
	return nil
}

// Close gracefully closes all connected camera clients.
func (c *Cluster) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, client := range c.clients {
		_ = client.Close()
	}
	c.clients = make(map[string]*siyi.Client)
	c.roles = make(map[Role]*siyi.Client)
	return nil
}
