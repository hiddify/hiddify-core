package extension

import (
	"encoding/json"

	"github.com/hiddify/hiddify-core/extension/ui"
	"github.com/hiddify/hiddify-core/v2/db"
	"github.com/jellydator/validation"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
)

type Extension interface {
	OnUIOpen() *ui.Form
	OnUIClose() error
	OnDataSubmit(button string, data map[string]string) error

	DoUpdateUI(form *ui.Form) error
	DoStoreData()

	OnMainServicePreStart(singconfig *option.Options) error
	OnMainServiceStart() error
	OnMainServiceClose() error

	init(id string)
	getQueue() chan *ExtensionResponse
	getId() string
}

var _ Extension = (*Base[any])(nil)

type Base[T any] struct {
	id    string
	queue chan *ExtensionResponse
	Data  T
}

func (b *Base[T]) OnUIClose() error {
	return nil
}

func (b *Base[T]) OnMainServicePreStart(singconfig *option.Options) error {
	return nil
}

func (b *Base[T]) OnMainServiceStart() error {
	return nil
}

func (b *Base[T]) OnMainServiceClose() error {
	return nil
}

func (b *Base[T]) OnUIOpen() *ui.Form {
	return nil
}

func (b *Base[T]) OnDataSubmit(button string, data map[string]string) error {
	return nil
}

func (b *Base[T]) DoStoreData() {
	b.doStoreData()
}

func (b *Base[T]) doStoreData() {
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
	b.queue = make(chan *ExtensionResponse, 1)
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

func (b *Base[T]) getQueue() chan *ExtensionResponse {
	return b.queue
}

func (b *Base[T]) getId() string {
	return b.id
}

func (e *Base[T]) ShowMessage(title string, msg string) error {
	return e.ShowDialog(&ui.Form{
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

func (p *Base[T]) DoUpdateUI(form *ui.Form) error {
	p.queue <- &ExtensionResponse{
		ExtensionId: p.id,
		Type:        ExtensionResponseType_UPDATE_UI,
		JsonUi:      form.ToJSON(),
	}
	return nil
}

func (p *Base[T]) ShowDialog(form *ui.Form) error {
	p.queue <- &ExtensionResponse{
		ExtensionId: p.id,
		Type:        ExtensionResponseType_SHOW_DIALOG,
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
