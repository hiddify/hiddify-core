package extension

import (
	"context"
	"fmt"
	"log"

	pb "github.com/hiddify/hiddify-core/hiddifyrpc"
	"github.com/hiddify/hiddify-core/v2/common"
	"google.golang.org/grpc"
)

type ExtensionHostService struct {
	pb.UnimplementedExtensionHostServiceServer
}

func (ExtensionHostService) ListExtensions(ctx context.Context, empty *pb.Empty) (*pb.ExtensionList, error) {
	extensionList := &pb.ExtensionList{
		Extensions: make([]*pb.Extension, 0),
	}

	for _, extension := range allExtensionsMap {
		extensionList.Extensions = append(extensionList.Extensions, &pb.Extension{
			Id:          extension.Id,
			Title:       extension.Title,
			Description: extension.Description,
			Enable:      generalExtensionData.ExtensionStatusMap[extension.Id],
		})
	}
	return extensionList, nil
}

func (e ExtensionHostService) Connect(req *pb.ExtensionRequest, stream grpc.ServerStreamingServer[pb.ExtensionResponse]) error {
	// Get the extension from the map using the Extension ID
	if extension, ok := enabledExtensionsMap[req.GetExtensionId()]; ok {

		log.Printf("Connecting stream for extension %s", req.GetExtensionId())
		log.Printf("Extension data: %+v", extension)
		// Handle loading the UI for the extension
		// Call extension-specific logic to generate UI data
		// if err := platform.connect(stream); err != nil {
		// 	log.Printf("Error connecting stream for extension %s: %v", req.GetExtensionId(), err)
		// }
		if err := (*extension).UpdateUI((*extension).GetUI()); err != nil {
			log.Printf("Error updating UI for extension %s: %v", req.GetExtensionId(), err)
		}
		// info := <-platform.queue

		// stream.Send(info)
		// (*platform.extension).SubmitData(map[string]string{})
		// log.Printf("Extension info: %+v", info)
		// 	// Handle submitting data to the extension
		// case pb.ExtensionRequestType_SUBMIT_DATA:
		// 	// Handle submitting data to the extension
		// 	// Process the provided data
		// 	err := extension.SubmitData(req.GetData())
		// 	if err != nil {
		// 		log.Printf("Error submitting data for extension %s: %v", req.GetExtensionId(), err)
		// 		// continue
		// 	}

		// case hiddifyrpc.ExtensionRequestType_CANCEL:
		// 	// Handle canceling the current operation in the extension
		// 	extension.Stop()
		// 	log.Printf("Operation canceled for extension %s", req.GetExtensionId())

		// default:
		// 	log.Printf("Unknown request type: %v", req.GetType())
		// }

		for {
			select {
			case <-stream.Context().Done():
				return nil
			case info := <-(*extension).getQueue():
				stream.Send(info)
				if info.GetType() == pb.ExtensionResponseType_END {
					return nil
				}
			}
		}

		// break
		// 	case <-stopCh:
		// 		break
		// 	// case info := <-sub:
		// 	// 	stream.Send(&info)
		// 	case <-time.After(1000 * time.Millisecond):
		// }

		// extension := extensionsMap[data.GetExtensionId()]
		// ui := extension.GetUI(data.Data)

		//	return &pb.UI{
		//		ExtensionId: data.GetExtensionId(),
		//		JsonUi:      ui.ToJSON(),
		//	}, nil
	} else {
		log.Printf("Extension with ID %s not found", req.GetExtensionId())
		return fmt.Errorf("Extension with ID %s not found", req.GetExtensionId())
	}
}

func (e ExtensionHostService) SubmitForm(ctx context.Context, req *pb.ExtensionRequest) (*pb.ExtensionActionResult, error) {
	if extension, ok := enabledExtensionsMap[req.GetExtensionId()]; ok {
		(*extension).SubmitData(req.GetData())

		return &pb.ExtensionActionResult{
			ExtensionId: req.ExtensionId,
			Code:        pb.ResponseCode_OK,
			Message:     "Success",
		}, nil
	}
	return nil, fmt.Errorf("Extension with ID %s not found", req.GetExtensionId())
}

func (e ExtensionHostService) Cancel(ctx context.Context, req *pb.ExtensionRequest) (*pb.ExtensionActionResult, error) {
	if extension, ok := enabledExtensionsMap[req.GetExtensionId()]; ok {
		(*extension).Cancel()

		return &pb.ExtensionActionResult{
			ExtensionId: req.ExtensionId,
			Code:        pb.ResponseCode_OK,
			Message:     "Success",
		}, nil
	}
	return nil, fmt.Errorf("Extension with ID %s not found", req.GetExtensionId())
}

func (e ExtensionHostService) Stop(ctx context.Context, req *pb.ExtensionRequest) (*pb.ExtensionActionResult, error) {
	if extension, ok := enabledExtensionsMap[req.GetExtensionId()]; ok {
		(*extension).Stop()
		(*extension).StoreData()
		return &pb.ExtensionActionResult{
			ExtensionId: req.ExtensionId,
			Code:        pb.ResponseCode_OK,
			Message:     "Success",
		}, nil
	}
	return nil, fmt.Errorf("Extension with ID %s not found", req.GetExtensionId())
}

func (e ExtensionHostService) EditExtension(ctx context.Context, req *pb.EditExtensionRequest) (*pb.ExtensionActionResult, error) {
	generalExtensionData.ExtensionStatusMap[req.GetExtensionId()] = req.Enable
	if !req.Enable {
		ext := *enabledExtensionsMap[req.GetExtensionId()]
		if ext != nil {
			ext.Stop()
			ext.StoreData()
		}
		delete(enabledExtensionsMap, req.GetExtensionId())
	} else {
		loadExtension(allExtensionsMap[req.GetExtensionId()])
	}
	common.Storage.SaveExtensionData("default", generalExtensionData)

	return &pb.ExtensionActionResult{
		ExtensionId: req.ExtensionId,
		Code:        pb.ResponseCode_OK,
		Message:     "Success",
	}, nil
}
