package extension

import (
	"fmt"
	"log"

	"github.com/hiddify/hiddify-core/extension/ui_elements"
	pb "github.com/hiddify/hiddify-core/hiddifyrpc"
)

var (
	extensionsMap      = make(map[string]*Extension)
	extensionStatusMap = make(map[string]bool)
)

type Extension interface {
	GetTitle() string
	GetDescription() string
	GetUI() ui_elements.Form
	SubmitData(data map[string]string) error
	Cancel() error
	Stop() error
	UpdateUI(form ui_elements.Form) error
	init(id string)
	getQueue() chan *pb.ExtensionResponse
	getId() string
}

type BaseExtension struct {
	id string
	// responseStream grpc.ServerStreamingServer[pb.ExtensionResponse]
	queue chan *pb.ExtensionResponse
}

// func (b *BaseExtension) mustEmbdedBaseExtension() {
// }

func (b *BaseExtension) init(id string) {
	b.id = id
	b.queue = make(chan *pb.ExtensionResponse, 1)
}

func (b *BaseExtension) getQueue() chan *pb.ExtensionResponse {
	return b.queue
}

func (b *BaseExtension) getId() string {
	return b.id
}

func (p *BaseExtension) UpdateUI(form ui_elements.Form) error {
	p.queue <- &pb.ExtensionResponse{
		ExtensionId: p.id,
		Type:        pb.ExtensionResponseType_UPDATE_UI,
		JsonUi:      form.ToJSON(),
	}
	return nil
}

func (p *BaseExtension) ShowDialog(form ui_elements.Form) error {
	p.queue <- &pb.ExtensionResponse{
		ExtensionId: p.id,
		Type:        pb.ExtensionResponseType_SHOW_DIALOG,
		JsonUi:      form.ToJSON(),
	}
	// log.Printf("Updated UI for extension %s: %s", err, p.id)
	return nil
}

func RegisterExtension(id string, extension Extension) error {
	if _, ok := extensionsMap[id]; ok {
		err := fmt.Errorf("Extension with ID %s already exists", id)
		log.Fatal(err)
		return err
	}
	if val, ok := extensionStatusMap[id]; ok && !val {
		err := fmt.Errorf("Extension with ID %s is not enabled", id)
		log.Fatal(err)
		return err
	}
	extension.init(id)

	fmt.Printf("Registered extension: %+v\n", extension)
	extensionsMap[id] = &extension
	return nil
}
