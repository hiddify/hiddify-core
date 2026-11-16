package config

import (
	context "context"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"

	"github.com/sagernet/sing-box/option"
)

func SaveCurrentConfig(ctx context.Context, path string, options option.Options) error {
	json, err := options.MarshalJSONContext(ctx)
	if err != nil {
		return err
	}
	p, err := filepath.Abs(path)
	os.MkdirAll(filepath.Dir(p), 0o755)
	fmt.Printf("Saving config to %v %+v\n", p, err)
	if err != nil {
		return err
	}
	return os.WriteFile(p, []byte(json), 0o644)
}

// DeferPanicToError recovers a panic and hands it to err as an error with its stack. It returns at
// once: it used to wait 5 seconds, which in an operation held the core's lock for that long.
func DeferPanicToError(name string, err func(error)) {
	if r := recover(); r != nil {
		err(fmt.Errorf("%s panic: %s\n%s", name, r, string(debug.Stack())))
	}
}
