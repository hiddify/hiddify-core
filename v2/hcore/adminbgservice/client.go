package adminbgservice

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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
// It blocks until the background core has fully started (STARTED). A nil error
// means the tunnel is ready for the app to treat as connected; any failure
// (service down, start error, timeout) is returned and must not be shown as connected.
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
		return fmt.Errorf("VPN service is not running: %w", err)
	}
	defer conn.Close()
	client := NewHiddifyAdminBGClient(conn)
	total := time.Now()
	// Only stop if something is running — avoids a long no-op Stop before every connect.
	statusStarted := time.Now()
	pre, preErr := client.Status(ctx, &hcommon.Empty{})
	preMsg := ""
	if pre != nil {
		preMsg = pre.GetMessage()
	}
	log.Printf("tunnel connect: status took %s (msg=%q err=%v)", time.Since(statusStarted), preMsg, preErr)
	if preErr == nil && preMsg == "started" {
		stopStarted := time.Now()
		stopCtx, stopCancel := context.WithTimeout(ctx, 15*time.Second)
		_, stopErr := client.Stop(stopCtx, &hcommon.Empty{})
		stopCancel()
		log.Printf("tunnel connect: pre-stop took %s (err=%v)", time.Since(stopStarted), stopErr)
	}
	startStarted := time.Now()
	res, err := client.Start(ctx, &AdminBgStartRequest{
		ConfigContent:      request.ConfigContent,
		ConfigName:         request.ConfigName,
		SettingsJson:       request.SettingsJSON,
		Resources:          request.Resources,
		DisableMemoryLimit: request.DisableMemoryLimit,
		OwnerPid:           request.OwnerPID,
	})
	log.Printf("tunnel connect: Start took %s (err=%v)", time.Since(startStarted), err)
	if err != nil {
		detail := ""
		if res != nil && res.GetMessage() != "" && res.GetMessage() != "OK" {
			detail = res.GetMessage()
		}
		if detail != "" {
			return fmt.Errorf("VPN service failed to start the background core: %s (%w)", detail, err)
		}
		return fmt.Errorf("VPN service failed to start the background core: %w", err)
	}
	if res == nil || res.GetMessage() != "OK" {
		msg := "unknown error"
		if res != nil && res.GetMessage() != "" {
			msg = res.GetMessage()
		}
		return fmt.Errorf("VPN service failed to start the background core: %s", msg)
	}
	// Start returned OK; confirm the engine is actually running before the app
	// treats this as connected (avoids a Connected UI with no tunnel).
	status, err := client.Status(ctx, &hcommon.Empty{})
	if err != nil {
		return fmt.Errorf("VPN service started but status check failed: %w", err)
	}
	if status.GetMessage() != "started" {
		return fmt.Errorf("VPN service started but engine status is %q", status.GetMessage())
	}
	log.Printf("tunnel connect: total %s", time.Since(total))
	return nil
}

// DisconnectAdminBg stops the tunnel in the root daemon.
func DisconnectAdminBg() error {
	if _, err := LoadControlSecret(); err != nil {
		return fmt.Errorf("read tunnel control secret: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	conn, ctx, err := dialTunnel(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	started := time.Now()
	_, err = NewHiddifyAdminBGClient(conn).Stop(ctx, &hcommon.Empty{})
	log.Printf("tunnel disconnect: Stop took %s (err=%v)", time.Since(started), err)
	return err
}
