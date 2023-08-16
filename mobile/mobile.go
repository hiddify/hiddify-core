package mobile

import (
	"github.com/hiddify/libcore/shared"
	_ "github.com/sagernet/gomobile/event/key"
)

func ConvertToSingbox(path string) (string, error) {
	options := shared.ConfigTemplateOptions{IncludeTunInbound: true, IncludeMixedInbound: false, IncludeLogOutput: false}
	config, err := shared.ConvertToSingbox(path, options)
	return string(config), err
}
