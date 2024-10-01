package extension

import (
	"fmt"
	"log"

	"github.com/hiddify/hiddify-core/v2/db"

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
	table := db.GetTable[extensionData]()
	_, err := table.FirstOrInsert(func(data extensionData) bool { return data.Id == factory.Id }, func() extensionData { return extensionData{Id: factory.Id, Enable: false} })
	if err != nil {
		return err
	}
	allExtensionsMap[factory.Id] = factory

	return nil
}

func isEnable(id string) bool {
	table := db.GetTable[extensionData]()
	extdata, err := table.First(func(data extensionData) bool { return data.Id == id })
	if err != nil {
		return false
	}
	return extdata.Enable
}

func loadExtension(factory ExtensionFactory) error {
	if !isEnable(factory.Id) {
		return fmt.Errorf("Extension with ID %s is not enabled", factory.Id)
	}
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
	table := db.GetTable[extensionData]()
	extdata, err := table.Select(func(data extensionData) bool { return data.Enable })
	if err != nil {
		return fmt.Errorf("failed to select enabled extensions: %w", err)
	}
	for _, data := range extdata {
		if factory, ok := allExtensionsMap[data.Id]; ok {
			if err := loadExtension(factory); err != nil {
				return fmt.Errorf("failed to load extension %s: %w", data.Id, err)
			}
		} else {
			return fmt.Errorf("extension %s is enabled but not found", data.Id)
		}
	}
	return nil
}

func (s *extensionService) Close() error {
	for _, extension := range enabledExtensionsMap {
		if err := (*extension).Close(); err != nil {
			return err
		}
	}
	return nil
}

func init() {
	service_manager.Register(&extensionService{})
}
