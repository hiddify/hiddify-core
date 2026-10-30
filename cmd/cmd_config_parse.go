package cmd

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hiddify/hiddify-core/v2/config"
	"github.com/sagernet/sing-box/experimental/libbox"
	"github.com/sagernet/sing-box/log"
	"github.com/spf13/cobra"
	"github.com/xtls/libxray/share"
	"github.com/xtls/xray-core/infra/conf"
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
	cfg, err := convertShareLinksToXray(string(data))
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

// Share links are converted one by one, working around three things libxray gets wrong:
//   - it reads HTML-escaped links ("&amp;" between the parameters, as some subscriptions write them) as
//     parameters named "amp;pbk", and so on. "&amp;" is turned back into "&", as ray2sing does;
//   - it decodes a link's xhttp "extra" into Xray's config structs and encodes it again, which turns REALITY's
//     unset "target" into null: Xray then builds the download leg as a REALITY server and drops the outbound.
//     Such links are converted without "extra", and the link's own "extra" is put back;
//   - it reads no REALITY from vmess:// links ("tls": "reality" with "pbk", "sid"), so the outbound fails with an
//     empty "realitySettings", nor their "extra". Such links are converted without them, and both are set from the
//     link.
func convertShareLinksToXray(text string) (*conf.Config, error) {
	if !strings.Contains(text, "://") {
		if decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(text)); err == nil {
			text = string(decoded)
		}
	}
	if !strings.Contains(text, "://") {
		return share.ConvertShareLinksToXrayJson(text)
	}
	var result *conf.Config
	var lastErr error
	for _, line := range strings.Split(text, "\n") {
		line = strings.ReplaceAll(strings.TrimSpace(line), "&amp;", "&")
		if line == "" {
			continue
		}
		line, extra := splitXHTTPExtra(line)
		line, reality, vmessExtra := splitVMess(line)
		if vmessExtra != "" {
			extra = vmessExtra
		}
		cfg, err := share.ConvertShareLinksToXrayJson(line)
		if err != nil {
			lastErr = err
			continue
		}
		for i := range cfg.OutboundConfigs {
			stream := cfg.OutboundConfigs[i].StreamSetting
			if stream == nil {
				continue
			}
			if extra != "" && stream.XHTTPSettings != nil {
				stream.XHTTPSettings.Extra = json.RawMessage(extra)
			}
			if reality != nil {
				stream.Security = "reality"
				stream.TLSSettings = nil
				stream.REALITYSettings = reality
			}
		}
		if result == nil {
			result = cfg
		} else {
			result.OutboundConfigs = append(result.OutboundConfigs, cfg.OutboundConfigs...)
		}
	}
	if result == nil {
		return nil, lastErr
	}
	return result, nil
}

// splitXHTTPExtra returns the link without its "extra" query parameter, and that parameter.
func splitXHTTPExtra(link string) (string, string) {
	u, err := url.Parse(link)
	if err != nil || !u.Query().Has("extra") {
		return link, ""
	}
	query := u.Query()
	extra := query.Get("extra")
	query.Del("extra")
	u.RawQuery = query.Encode()
	return u.String(), extra
}

// splitVMess returns a vmess:// link without its REALITY and its xhttp "extra", with those.
func splitVMess(link string) (string, *conf.REALITYConfig, string) {
	body, found := strings.CutPrefix(link, "vmess://")
	if !found {
		return link, nil, ""
	}
	raw, err := base64.StdEncoding.DecodeString(body)
	if err != nil {
		raw, err = base64.RawStdEncoding.DecodeString(body)
	}
	var fields map[string]any
	if err != nil || json.Unmarshal(raw, &fields) != nil {
		return link, nil, ""
	}
	text := func(key string) string {
		value, _ := fields[key].(string)
		return value
	}
	extra := text("extra")
	if fields["tls"] != "reality" && extra == "" {
		return link, nil, ""
	}
	var reality *conf.REALITYConfig
	if fields["tls"] == "reality" {
		reality = &conf.REALITYConfig{
			ServerName:  text("sni"),
			Fingerprint: text("fp"),
			PublicKey:   text("pbk"),
			ShortId:     text("sid"),
			SpiderX:     text("spx"),
		}
		if reality.Fingerprint == "" {
			reality.Fingerprint = "chrome"
		}
		fields["tls"] = ""
	}
	delete(fields, "extra")
	raw, err = json.Marshal(fields)
	if err != nil {
		return link, nil, ""
	}
	return "vmess://" + base64.StdEncoding.EncodeToString(raw), reality, extra
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
