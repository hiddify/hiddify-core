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
	json, err := ToJson(options)
	if err != nil {
		return err
	}
	p, err := filepath.Abs(path)
	fmt.Printf("Saving config to %v %+v\n", p, err)
	if err != nil {
		return err
	}
	return os.WriteFile(p, []byte(json), 0644)
}

func ToJson(options option.Options) (string, error) {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetIndent("", "  ")
	// fmt.Printf("%+v\n", options)
	err := encoder.Encode(options)
	if err != nil {
		fmt.Printf("ERROR in coding:%+v\n", err)
		return "", err
	}
	return buffer.String(), nil
}

func DeferPanicToError(name string, err func(error)) {
	if r := recover(); r != nil {
		s := fmt.Errorf("%s panic: %s\n%s", name, r, string(debug.Stack()))
		err(s)
	}
}
