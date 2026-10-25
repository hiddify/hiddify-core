package hcore

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/hiddify/hiddify-core/v2/hcore/buildid"
	hutils "github.com/hiddify/hiddify-core/v2/hutils"
)

const adminBgServicePort uint16 = 18020

// NeedAdminBgService decides whether the elevated admin_bgservice is required for this connect.
func (s *CoreService) NeedAdminBgService(ctx context.Context, req *NeedAdminBgServiceRequest) (*NeedAdminBgServiceResponse, error) {
	if runtime.GOOS != "windows" && runtime.GOOS != "linux" {
		return &NeedAdminBgServiceResponse{Decision: AdminBgServiceDecision_ADMIN_BG_SERVICE_NO}, nil
	}

	features := scanAdminNeededFeatures(req.GetHiddifySettingsJson())
	vpn := isVpnServiceMode(req.GetServiceMode(), req.GetHiddifySettingsJson())

	if hutils.IsAdmin() {
		return &NeedAdminBgServiceResponse{Decision: AdminBgServiceDecision_ADMIN_BG_SERVICE_NO}, nil
	}

	ready := adminBgServiceReady()
	pref := req.GetAdminPreference()

	if vpn {
		if ready {
			return &NeedAdminBgServiceResponse{Decision: AdminBgServiceDecision_ADMIN_BG_SERVICE_ALREADY_CONNECTED}, nil
		}
		return &NeedAdminBgServiceResponse{Decision: AdminBgServiceDecision_ADMIN_BG_SERVICE_NEEDS_ADMIN}, nil
	}

	needsOptional := len(features) > 0 || req.GetPrivilegeErrorRemembered()
	if !needsOptional {
		return &NeedAdminBgServiceResponse{Decision: AdminBgServiceDecision_ADMIN_BG_SERVICE_NO}, nil
	}

	switch pref {
	case AdminPreference_ADMIN_PREFERENCE_PREFER_ADMIN:
		if ready {
			return &NeedAdminBgServiceResponse{
				Decision:           AdminBgServiceDecision_ADMIN_BG_SERVICE_ALREADY_CONNECTED,
				AdminNeededProxies: features,
			}, nil
		}
		return &NeedAdminBgServiceResponse{
			Decision:           AdminBgServiceDecision_ADMIN_BG_SERVICE_NEEDS_ADMIN,
			AdminNeededProxies: features,
		}, nil
	case AdminPreference_ADMIN_PREFERENCE_PREFER_NO_ADMIN:
		return &NeedAdminBgServiceResponse{
			Decision:           AdminBgServiceDecision_ADMIN_BG_SERVICE_NO,
			AdminNeededProxies: features,
		}, nil
	default:
		return &NeedAdminBgServiceResponse{
			Decision:           AdminBgServiceDecision_ADMIN_BG_SERVICE_OPTIONAL_PROXY_NEED_ADMIN,
			AdminNeededProxies: features,
		}, nil
	}
}

// EnsureAdminBgService installs/starts admin_bgservice if needed and connects until STARTED.
// Uses the bundled CLI (cli_path) so this package does not import adminbgservice (import cycle).
func (s *CoreService) EnsureAdminBgService(req *EnsureAdminBgServiceRequest, stream Core_EnsureAdminBgServiceServer) error {
	if runtime.GOOS != "windows" && runtime.GOOS != "linux" {
		return sendEnsureEvent(stream, EnsureAdminBgServiceProgress_ENSURE_ADMIN_BG_FAILED_OTHER, "admin_bgservice is only available on Windows and Linux", 0)
	}
	cliPath := strings.TrimSpace(req.GetCliPath())
	if cliPath == "" {
		return sendEnsureEvent(stream, EnsureAdminBgServiceProgress_ENSURE_ADMIN_BG_FAILED_OTHER, "cli_path is required", 0)
	}
	if req.GetConfigContent() == "" {
		return sendEnsureEvent(stream, EnsureAdminBgServiceProgress_ENSURE_ADMIN_BG_FAILED_OTHER, "missing configuration", 0)
	}

	if !adminBgServiceReady() {
		_ = sendEnsureEvent(stream, EnsureAdminBgServiceProgress_ENSURE_ADMIN_BG_WAITING_FOR_ELEVATION, "Administrator permission is required to install the VPN service", adminBgServicePort)
		if err := ensureAdminBgServiceInstalled(cliPath, req.GetOwnerUser()); err != nil {
			msg := err.Error()
			progress := EnsureAdminBgServiceProgress_ENSURE_ADMIN_BG_FAILED_OTHER
			if isPermissionDenied(err) {
				progress = EnsureAdminBgServiceProgress_ENSURE_ADMIN_BG_FAILED_PERMISSION_DENIED
			}
			_ = sendEnsureEvent(stream, progress, msg, 0)
			return nil
		}
		if err := waitAdminBgServiceReady(25 * time.Second); err != nil {
			_ = sendEnsureEvent(stream, EnsureAdminBgServiceProgress_ENSURE_ADMIN_BG_FAILED_OTHER, err.Error(), 0)
			return nil
		}
	}

	_ = sendEnsureEvent(stream, EnsureAdminBgServiceProgress_ENSURE_ADMIN_BG_SERVICE_STARTING, "Starting the VPN service", adminBgServicePort)

	tmp, err := writeAdminBgConnectRequest(req)
	if err != nil {
		_ = sendEnsureEvent(stream, EnsureAdminBgServiceProgress_ENSURE_ADMIN_BG_FAILED_OTHER, err.Error(), 0)
		return nil
	}
	defer os.Remove(tmp)

	if err := runCli(cliPath, "admin_bgservice", "connect", tmp); err != nil {
		msg := err.Error()
		progress := EnsureAdminBgServiceProgress_ENSURE_ADMIN_BG_FAILED_OTHER
		if isPermissionDenied(err) {
			progress = EnsureAdminBgServiceProgress_ENSURE_ADMIN_BG_FAILED_PERMISSION_DENIED
		}
		_ = sendEnsureEvent(stream, progress, msg, 0)
		return nil
	}

	return sendEnsureEvent(stream, EnsureAdminBgServiceProgress_ENSURE_ADMIN_BG_SERVICE_CONNECTED, "VPN service is ready", adminBgServicePort)
}

func sendEnsureEvent(stream Core_EnsureAdminBgServiceServer, progress EnsureAdminBgServiceProgress, message string, port uint16) error {
	return stream.Send(&EnsureAdminBgServiceEvent{
		Progress:    progress,
		Message:     message,
		ServicePort: uint32(port),
	})
}

func isVpnServiceMode(mode string, settingsJSON string) bool {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "vpn", "tun":
		return true
	case "proxy", "system-proxy", "system_proxy":
		return false
	}
	root, err := parseSettingsMap(settingsJSON)
	if err != nil {
		return false
	}
	if v, ok := root["enable-tun"].(bool); ok {
		return v
	}
	if inbound, ok := root["inbound_options"].(map[string]any); ok {
		if v, ok := inbound["enable_tun"].(bool); ok {
			return v
		}
		if v, ok := inbound["enableTun"].(bool); ok {
			return v
		}
	}
	return false
}

func scanAdminNeededFeatures(settingsJSON string) []*AdminNeededProxy {
	root, err := parseSettingsMap(settingsJSON)
	if err != nil {
		return nil
	}
	var out []*AdminNeededProxy
	if mixedSNIEnabled(root) {
		out = append(out, &AdminNeededProxy{
			Tag:     "tls-tricks",
			Summary: "Mixed SNI case",
			Detail:  "TLS mixed-SNI case needs administrator permission on this platform.",
		})
	}
	return out
}

func mixedSNIEnabled(root map[string]any) bool {
	if tricks, ok := root["tls-tricks"].(map[string]any); ok {
		if v, ok := tricks["mixed-sni-case"].(bool); ok {
			return v
		}
		if v, ok := tricks["mixedSniCase"].(bool); ok {
			return v
		}
	}
	if tricks, ok := root["tls_tricks"].(map[string]any); ok {
		if v, ok := tricks["mixed_sni_case"].(bool); ok {
			return v
		}
	}
	return false
}

func parseSettingsMap(settingsJSON string) (map[string]any, error) {
	if strings.TrimSpace(settingsJSON) == "" {
		return nil, fmt.Errorf("empty settings")
	}
	var root map[string]any
	if err := json.Unmarshal([]byte(settingsJSON), &root); err != nil {
		return nil, err
	}
	return root, nil
}

func adminBgDataDir() string {
	if dir := os.Getenv("HIDDIFY_TUNNEL_DATA"); dir != "" {
		return dir
	}
	switch runtime.GOOS {
	case "windows":
		base := os.Getenv("PROGRAMDATA")
		if base == "" {
			base = `C:\ProgramData`
		}
		return filepath.Join(base, "Hiddify", "AdminBgService")
	default:
		return "/var/lib/hiddify-admin-bgservice"
	}
}

func adminBgSecretPath() string {
	return filepath.Join(adminBgDataDir(), "control.secret")
}

func adminBgBuildStampPath() string {
	return filepath.Join(adminBgDataDir(), "build.id")
}

func adminBgServiceReady() bool {
	secret, err := os.ReadFile(adminBgSecretPath())
	if err != nil || len(strings.TrimSpace(string(secret))) < 32 {
		return false
	}
	stamped, err := os.ReadFile(adminBgBuildStampPath())
	if err != nil {
		return false
	}
	want := strings.TrimSpace(string(stamped))
	have := strings.TrimSpace(buildid.Get())
	if want == "" || want == "unset" || have == "" || have == "unset" || want != have {
		return false
	}
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", adminBgServicePort), 200*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func waitAdminBgServiceReady(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if adminBgServiceReady() {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("the VPN service was installed but did not start with this app build")
}

func ensureAdminBgServiceInstalled(cliPath, ownerUser string) error {
	if adminBgServiceReady() {
		return nil
	}
	if _, err := os.Stat(adminBgSecretPath()); err == nil {
		stamped, _ := os.ReadFile(adminBgBuildStampPath())
		if strings.TrimSpace(string(stamped)) == strings.TrimSpace(buildid.Get()) {
			if err := runCli(cliPath, "admin_bgservice", "start"); err == nil && adminBgServiceReady() {
				return nil
			}
		}
	}
	owner := strings.TrimSpace(ownerUser)
	if owner == "" {
		owner = os.Getenv("USER")
		if owner == "" {
			owner = os.Getenv("USERNAME")
		}
	}
	if owner != "" {
		_ = os.Setenv("HIDDIFY_TUNNEL_OWNER", owner)
	}
	return elevateAdminBgInstall(cliPath, owner)
}

func runCli(cliPath string, args ...string) error {
	cmd := exec.Command(cliPath, args...)
	cmd.Dir = filepath.Dir(cliPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			return err
		}
		return fmt.Errorf("%s", msg)
	}
	return nil
}

func writeAdminBgConnectRequest(req *EnsureAdminBgServiceRequest) (string, error) {
	// Match adminbgservice.ConnectRequest JSON (resources as base64 via encoding/json []byte).
	body := struct {
		ConfigContent      string            `json:"configContent"`
		ConfigName         string            `json:"configName"`
		SettingsJSON       string            `json:"settingsJSON"`
		Resources          map[string][]byte `json:"resources"`
		DisableMemoryLimit bool              `json:"disableMemoryLimit"`
		OwnerPID           int32             `json:"ownerPid"`
	}{
		ConfigContent:      req.GetConfigContent(),
		ConfigName:         req.GetConfigName(),
		SettingsJSON:       req.GetSettingsJson(),
		Resources:          req.GetResources(),
		DisableMemoryLimit: req.GetDisableMemoryLimit(),
		OwnerPID:           req.GetOwnerPid(),
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	f, err := os.CreateTemp("", "hiddify-admin-bgservice-*.json")
	if err != nil {
		return "", err
	}
	path := f.Name()
	if _, err := f.Write(raw); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return "", err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(path)
		return "", err
	}
	return path, nil
}

func isPermissionDenied(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	for _, needle := range []string{
		"permission denied",
		"access is denied",
		"operation not permitted",
		"user canceled",
		"user cancelled",
		"canceled by the user",
		"cancelled by the user",
		"polkit",
		"not authorized",
		"elevation",
	} {
		if strings.Contains(msg, needle) {
			return true
		}
	}
	return false
}
