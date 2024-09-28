package extension

import (
	"fmt"
	"log"

	"github.com/hiddify/hiddify-core/common"
)

var (
	allExtensionsMap     = make(map[string]ExtensionFactory)
	enabledExtensionsMap = make(map[string]*Extension)
	generalExtensionData = mustSaveExtensionData{
		ExtensionStatusMap: make(map[string]bool),
	}
)

type mustSaveExtensionData struct {
	ExtensionStatusMap map[string]bool `json:"extensionStatusMap"`
}

func RegisterExtension(factory ExtensionFactory) error {
	if _, ok := allExtensionsMap[factory.Id]; ok {
		err := fmt.Errorf("Extension with ID %s already exists", factory.Id)
		log.Fatal(err)
		return err
	}
	allExtensionsMap[factory.Id] = factory
	common.Storage.GetExtensionData("default", &generalExtensionData)

	if val, ok := generalExtensionData.ExtensionStatusMap[factory.Id]; ok && val {
		loadExtension(factory)
	}
	return nil
}

func loadExtension(factory ExtensionFactory) error {
	extension := factory.Builder()
	extension.init(factory.Id)

	// fmt.Printf("Registered extension: %+v\n", extension)
	enabledExtensionsMap[factory.Id] = &extension

	return nil
}
