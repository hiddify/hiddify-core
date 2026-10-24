package tunnelservice

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	hcommon "github.com/hiddify/hiddify-core/v2/hcommon"
)

// ConnectRequest is the JSON file the macOS app writes before `Hiddify --tunnel connect`.
type ConnectRequest struct {
	ConfigContent      string            `json:"configContent"`
	ConfigName         string            `json:"configName"`
	SettingsJSON       string            `json:"settingsJSON"`
	Resources          map[string][]byte `json:"resources"`
	DisableMemoryLimit bool              `json:"disableMemoryLimit"`
	OwnerPID           int32             `json:"ownerPid"`
}

// ConnectFromFile starts the full core in the root daemon from a JSON request file.
func ConnectFromFile(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var request ConnectRequest
	if err := json.Unmarshal(content, &request); err != nil {
		return err
	}
	if request.ConfigContent == "" {
		return fmt.Errorf("missing tunnel configuration")
	}
	if _, err := LoadControlSecret(); err != nil {
		return fmt.Errorf("read tunnel control secret: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	conn, ctx, err := dialTunnel(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	client := NewTunnelServiceClient(conn)
	_, _ = client.Stop(ctx, &hcommon.Empty{})
	_, err = client.Start(ctx, &TunnelStartRequest{
		ConfigContent:      request.ConfigContent,
		ConfigName:         request.ConfigName,
		SettingsJson:       request.SettingsJSON,
		Resources:          request.Resources,
		DisableMemoryLimit: request.DisableMemoryLimit,
		OwnerPid:           request.OwnerPID,
	})
	return err
}

// DisconnectTunnel stops the tunnel in the root daemon.
func DisconnectTunnel() error {
	if _, err := LoadControlSecret(); err != nil {
		return fmt.Errorf("read tunnel control secret: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	conn, ctx, err := dialTunnel(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	_, err = NewTunnelServiceClient(conn).Stop(ctx, &hcommon.Empty{})
	return err
}
