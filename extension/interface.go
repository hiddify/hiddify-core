package extension

import (
	"fmt"
	"log"

	"github.com/hiddify/hiddify-core/v2/common"
	"github.com/hiddify/hiddify-core/v2/service_manager"
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

	return nil
}

func loadExtension(factory ExtensionFactory) error {
	extension := factory.Builder()
	extension.init(factory.Id)

	// fmt.Printf("Registered extension: %+v\n", extension)
	enabledExtensionsMap[factory.Id] = &extension

	return nil
}

type extensionService struct {
	// Storage *CacheFile
}

func (s *extensionService) Start() error {
	common.Storage.GetExtensionData("default", &generalExtensionData)

	for id, factory := range allExtensionsMap {
		if val, ok := generalExtensionData.ExtensionStatusMap[id]; ok && val {
			loadExtension(factory)
		}
	}
	return nil
}

func (s *extensionService) Close() error {
	for _, extension := range enabledExtensionsMap {
		if err := (*extension).Stop(); err != nil {
			return err
		}
	}
	return nil
}

func init() {
	service_manager.Register(&extensionService{})
}
