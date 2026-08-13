package hcore

import (
	"encoding/json"
	"fmt"

	"github.com/hiddify/hiddify-core/v2/db"
	hcommon "github.com/hiddify/hiddify-core/v2/hcommon"
)

// SetConnectionMode changes only the daemon-owned inbound mode. It deliberately
// refuses system-proxy mode: that mode must be applied by the logged-in session
// agent rather than by the privileged daemon.
func SetConnectionMode(mode string) error {
	static.lock.Lock()
	defer static.lock.Unlock()
	if static.HiddifyOptions == nil {
		return fmt.Errorf("HiddifyOptions not initialized")
	}
	switch mode {
	case "profile-default":
		return nil
	case "tun":
		static.HiddifyOptions.EnableTun = true
		static.HiddifyOptions.SetSystemProxy = false
	case "local-proxy":
		static.HiddifyOptions.EnableTun = false
		static.HiddifyOptions.SetSystemProxy = false
	case "system-proxy":
		// The daemon owns only the loopback listener. The session agent applies
		// the desktop setting after receiving its authorized control instruction.
		static.HiddifyOptions.EnableTun = false
		static.HiddifyOptions.SetSystemProxy = false
	default:
		return fmt.Errorf("unsupported connection mode %q", mode)
	}
	encoded, err := json.Marshal(static.HiddifyOptions)
	if err != nil {
		return fmt.Errorf("encode connection mode: %w", err)
	}
	return db.GetTable[hcommon.AppSettings]().UpdateInsert(&hcommon.AppSettings{
		Id: "HiddifySettingsJson", Value: string(encoded),
	})
}

func LocalProxyPort() uint32 {
	static.lock.Lock()
	defer static.lock.Unlock()
	if static.HiddifyOptions == nil {
		return 0
	}
	return uint32(static.HiddifyOptions.MixedPort)
}
