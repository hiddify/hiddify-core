package shared

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/sagernet/sing-box/option"
)

func SaveCurrentConfig(path string, options option.Options) error {
	var buffer bytes.Buffer
	json.NewEncoder(&buffer)
	encoder := json.NewEncoder(&buffer)
	encoder.SetIndent("", "  ")
	err := encoder.Encode(options)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(path, "current-config.json"), buffer.Bytes(), 0777)
}
