package extension

import (
	"fmt"

	"github.com/hiddify/hiddify-core/v2/db"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"

	"github.com/hiddify/hiddify-core/v2/service_manager"
)

var (
	allExtensionsMap     = make(map[string]ExtensionFactory)
	enabledExtensionsMap = make(map[string]*Extension)
)

func RegisterExtension(factory ExtensionFactory) error {
	if _, ok := allExtensionsMap[factory.Id]; ok {
		err := fmt.Errorf("Extension with ID %s already exists", factory.Id)
		log.Warn(err)
		return err
	}

	allExtensionsMap[factory.Id] = factory

	return nil
}

func isEnable(id string) bool {
	table := db.GetTable[extensionData]()
	extdata, err := table.Get(id)
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

var _ service_manager.HService = &extensionService{}

type extensionService struct {
	// Storage *CacheFile
}

// Init implements service_manager.HService.
func (s *extensionService) Init() error {
	table := db.GetTable[extensionData]()

	for _, factory := range allExtensionsMap {
		data, err := table.Get(factory.Id)

		if data == nil || err != nil {
			log.Warn("Data of Extension ", factory.Id, " not found, creating new one")
			data = &extensionData{Id: factory.Id, Enable: false}
			if err := table.UpdateInsert(data); err != nil {
				log.Warn("Failed to create new extension data: ", err, " ", factory.Id)
				return err
			}
		}

		if data.Enable {
			if err := loadExtension(factory); err != nil {
				return fmt.Errorf("failed to load extension %s: %w", data.Id, err)
			}
		}
	}

	return nil
}

// Dispose implements service_manager.HService.
func (s *extensionService) Dispose() error {
	for _, extension := range enabledExtensionsMap {
		if err := (*extension).OnUIClose(); err != nil {
			return err
		}
	}
	return nil
}

// OnMainServicePreStart implements service_manager.HService.
func (s *extensionService) OnMainServicePreStart(singconfig *option.Options) error {
	for _, extension := range enabledExtensionsMap {
		if err := (*extension).OnMainServicePreStart(singconfig); err != nil {
			return fmt.Errorf("failed to prestart extension %s: %w", (*extension).getId(), err)
		}
	}
	return nil
}

// OnMainServiceStart implements service_manager.HService.
func (s *extensionService) OnMainServiceStart() error {
	for _, extension := range enabledExtensionsMap {
		if err := (*extension).OnMainServiceStart(); err != nil {
			return fmt.Errorf("failed to start extension %s: %w", (*extension).getId(), err)
		}
	}
	return nil
}

// OnMainServiceClose implements service_manager.HService.
func (s *extensionService) OnMainServiceClose() error {
	for _, extension := range enabledExtensionsMap {
		if err := (*extension).OnMainServiceClose(); err != nil {
			return fmt.Errorf("failed to close extension %s: %w", (*extension).getId(), err)
		}
	}
	return nil
}

func init() {
	// service_manager.Register(&extensionService{})
}
