package extension

import (
	"encoding/json"

	"github.com/hiddify/hiddify-core/config"
	"github.com/hiddify/hiddify-core/extension/ui"
	pb "github.com/hiddify/hiddify-core/hiddifyrpc"
	"github.com/hiddify/hiddify-core/v2/db"
	"github.com/jellydator/validation"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
)

type Extension interface {
	GetUI() ui.Form
	SubmitData(button string, data map[string]string) error
	Close() error
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
	table := db.GetTable[extensionData]()
	ed, err := table.Get(b.id)
	if err != nil {
		log.Warn("error: ", err)
		return
	}
	res, err := json.Marshal(b.Data)
	if err != nil {
		log.Warn("error: ", err)
		return
	}
	ed.JsonData = (res)
	table.UpdateInsert(ed)
}

func (b *Base[T]) init(id string) {
	b.id = id
	b.queue = make(chan *pb.ExtensionResponse, 1)
	table := db.GetTable[extensionData]()
	extdata, err := table.Get(b.id)
	if err != nil {
		log.Warn("error: ", err)
		return
	}
	if extdata == nil {
		log.Warn("extension data not found ", id)
		return
	}
	if extdata.JsonData != nil {
		var t T
		if err := json.Unmarshal(extdata.JsonData, &t); err != nil {
			log.Warn("error loading data of ", id, " : ", err)
		} else {
			b.Data = t
		}
	}
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
		Fields: [][]ui.FormField{
			{{
				Type:  ui.FieldButton,
				Key:   ui.ButtonDialogOk,
				Label: "Ok",
			}},
		},
		// Buttons:     []string{ui.Button_Ok},
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
