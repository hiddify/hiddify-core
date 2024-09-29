package extension

import (
	"github.com/hiddify/hiddify-core/config"
	"github.com/hiddify/hiddify-core/extension/ui"
	pb "github.com/hiddify/hiddify-core/hiddifyrpc"
	"github.com/hiddify/hiddify-core/v2/common"
	"github.com/jellydator/validation"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
)

type Extension interface {
	GetUI() ui.Form
	SubmitData(data map[string]string) error
	Cancel() error
	Stop() error
	UpdateUI(form ui.Form) error

	BeforeAppConnect(hiddifySettings *config.HiddifyOptions, singconfig *option.Options) error

	StoreData()

	init(id string)
	getQueue() chan *pb.ExtensionResponse
	getId() string
}

type Base[T any] struct {
	id string
	// responseStream grpc.ServerStreamingServer[pb.ExtensionResponse]
	queue chan *pb.ExtensionResponse
	Data  T
}

// func (b *Base) mustEmbdedBaseExtension() {
// }

func (b *Base[T]) BeforeAppConnect(hiddifySettings *config.HiddifyOptions, singconfig *option.Options) error {
	return nil
}

func (b *Base[T]) StoreData() {
	common.Storage.SaveExtensionData(b.id, &b.Data)
}

func (b *Base[T]) init(id string) {
	b.id = id
	b.queue = make(chan *pb.ExtensionResponse, 1)
	common.Storage.GetExtensionData(b.id, &b.Data)
}

func (b *Base[T]) getQueue() chan *pb.ExtensionResponse {
	return b.queue
}

func (b *Base[T]) getId() string {
	return b.id
}

func (e *Base[T]) ShowMessage(title string, msg string) error {
	return e.ShowDialog(ui.Form{
		Title:       title,
		Description: msg,
		Buttons:     []string{ui.Button_Ok},
	})
}

func (p *Base[T]) UpdateUI(form ui.Form) error {
	p.queue <- &pb.ExtensionResponse{
		ExtensionId: p.id,
		Type:        pb.ExtensionResponseType_UPDATE_UI,
		JsonUi:      form.ToJSON(),
	}
	return nil
}

func (p *Base[T]) ShowDialog(form ui.Form) error {
	p.queue <- &pb.ExtensionResponse{
		ExtensionId: p.id,
		Type:        pb.ExtensionResponseType_SHOW_DIALOG,
		JsonUi:      form.ToJSON(),
	}
	// log.Printf("Updated UI for extension %s: %s", err, p.id)
	return nil
}

func (base *Base[T]) ValName(fieldPtr interface{}) string {
	val, err := validation.ErrorFieldName(&base.Data, fieldPtr)
	if err != nil {
		log.Warn(err)
		return ""
	}
	if val == "" {
		log.Warn("Field not found")
		return ""
	}
	return val
}

type ExtensionFactory struct {
	Id          string
	Title       string
	Description string
	Builder     func() Extension
}
