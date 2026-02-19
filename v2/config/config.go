package config

import (
	context "context"

	"github.com/sagernet/sing-box/experimental/libbox"
	"github.com/sagernet/sing-box/option"
)

type ReadOptions struct {
	Path    string
	Content string
	Options *option.Options
}

func ReadSingOptions(ctx context.Context, opt *ReadOptions) (*option.Options, error) {
	if opt.Options != nil {
		return opt.Options, nil
	}
	content, err := ReadContent(ctx, opt)
	if err != nil {
		return nil, err
	}
	var options option.Options
	err = options.UnmarshalJSONContext(ctx, content)
	return &options, err
}
func BuildConfigJson(ctx context.Context, configOpt *HiddifyOptions, input *ReadOptions) ([]byte, error) {
	options, err := BuildConfig(ctx, configOpt, input)
	if err != nil {
		return nil, err
	}
	if err := libbox.CheckConfigOptions(options); err != nil {
		return nil, err
	}

	return options.MarshalJSONContext(ctx)

}
func ParseBuildConfigBytes(ctx context.Context, hopts *HiddifyOptions, input *ReadOptions) ([]byte, error) {

	options, err := ParseBuildConfig(ctx, hopts, input)
	if err != nil {
		return nil, err
	}
	return options.MarshalJSONContext(ctx)
}
func ParseBuildConfig(ctx context.Context, hopts *HiddifyOptions, input *ReadOptions) (*option.Options, error) {
	options := input.Options
	if options == nil {
		var err error
		options, err = ParseConfig(ctx, input, false, hopts, false)
		if err != nil {
			return nil, err
		}

	}
	return BuildConfig(ctx, hopts, &ReadOptions{Options: options})
}
