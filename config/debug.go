package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"

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

func DeferPanicToError(name string, err func(error)) {
	if r := recover(); r != nil {
		s := fmt.Errorf("%s panic: %s\n%s", name, r, string(debug.Stack()))
		err(s)
	}
}
