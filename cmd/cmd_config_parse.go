package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/hiddify/hiddify-core/v2/config"
	"github.com/sagernet/sing-box/experimental/libbox"
	"github.com/sagernet/sing-box/log"
	"github.com/spf13/cobra"
	"github.com/xtls/libxray/share"
)

var (
	commandParseOutputPath string
	commandParseXray       bool
)

var commandParse = &cobra.Command{
	Use:   "parse",
	Short: "Parse configuration",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		err := parse(args[0])
		if err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	commandParse.Flags().StringVarP(&commandParseOutputPath, "output", "o", "", "write result to file path instead of stdout")
	commandParse.Flags().BoolVar(&commandParseXray, "xray", false, "parse share links to Xray JSON via libxray")

	mainCommand.AddCommand(commandParse)
}

func parse(path string) error {
	var configStr []byte
	var err error
	if commandParseXray {
		configStr, err = parseToXray(path)
	} else {
		ctx := libbox.BaseContext(nil)
		configStr, err = config.ParseConfigBytes(ctx, &config.ReadOptions{Path: path}, true, nil, false)
	}
	if err != nil {
		return err
	}

	if commandParseOutputPath != "" {
		outputPath, _ := filepath.Abs(filepath.Join(workingDir, commandParseOutputPath))
		err = os.WriteFile(outputPath, configStr, 0o644)
		if err != nil {
			return err
		}
		fmt.Println("result successfully written to ", outputPath)
	} else {
		os.Stdout.Write(configStr)
	}
	if !commandParseXray {
		if err := libbox.CheckConfig(string(configStr)); err != nil {
			return fmt.Errorf("config check failed: %w", err)
		}
	}
	return nil
}

func parseToXray(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg, err := share.ConvertShareLinksToXrayJson(string(data))
	if err != nil {
		return nil, fmt.Errorf("convert share links to xray json: %w", err)
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	cleaned, err := omitEmptyJSONBytes(raw)
	if err != nil {
		return nil, err
	}
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, cleaned, "", "  "); err != nil {
		return nil, err
	}
	pretty.WriteByte('\n')
	return pretty.Bytes(), nil
}

func omitEmptyJSONBytes(raw []byte) ([]byte, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	cleaned := omitEmptyJSON(value)
	if cleaned == nil {
		return []byte("{}"), nil
	}
	fixXraySendThrough(cleaned)
	return json.Marshal(cleaned)
}

// fixXraySendThrough moves libxray's outbound name out of sendThrough into tag
// (when tag is empty) and always drops sendThrough from the output.
func fixXraySendThrough(value any) {
	root, ok := value.(map[string]any)
	if !ok {
		return
	}
	outbounds, ok := root["outbounds"].([]any)
	if !ok {
		return
	}
	for _, item := range outbounds {
		outbound, ok := item.(map[string]any)
		if !ok {
			continue
		}
		sendThrough, _ := outbound["sendThrough"].(string)
		tag, _ := outbound["tag"].(string)
		if tag == "" && sendThrough != "" {
			outbound["tag"] = sendThrough
		}
		delete(outbound, "sendThrough")
	}
}

func omitEmptyJSON(value any) any {
	switch value := value.(type) {
	case nil:
		return nil
	case string:
		if value == "" {
			return nil
		}
		return value
	case bool:
		if !value {
			return nil
		}
		return value
	case json.Number:
		n, err := strconv.ParseFloat(value.String(), 64)
		if err == nil && n == 0 {
			return nil
		}
		return value
	case float64:
		if value == 0 {
			return nil
		}
		return value
	case []any:
		out := make([]any, 0, len(value))
		for _, item := range value {
			cleaned := omitEmptyJSON(item)
			if cleaned == nil {
				continue
			}
			if m, ok := cleaned.(map[string]any); ok && len(m) == 0 {
				continue
			}
			if a, ok := cleaned.([]any); ok && len(a) == 0 {
				continue
			}
			out = append(out, cleaned)
		}
		if len(out) == 0 {
			return nil
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(value))
		for key, item := range value {
			cleaned := omitEmptyJSON(item)
			if cleaned == nil {
				continue
			}
			if m, ok := cleaned.(map[string]any); ok && len(m) == 0 {
				continue
			}
			if a, ok := cleaned.([]any); ok && len(a) == 0 {
				continue
			}
			out[key] = cleaned
		}
		if len(out) == 0 {
			return nil
		}
		return out
	default:
		return value
	}
}
