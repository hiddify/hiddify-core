package extension

import (
	"context"
	"fmt"
	"log"

	pb "github.com/hiddify/hiddify-core/hiddifyrpc"
	"github.com/hiddify/hiddify-core/v2/db"
	"google.golang.org/grpc"
)

type ExtensionHostService struct {
	pb.UnimplementedExtensionHostServiceServer
}

func (ExtensionHostService) ListExtensions(ctx context.Context, empty *pb.Empty) (*pb.ExtensionList, error) {
	extensionList := &pb.ExtensionList{
		Extensions: make([]*pb.Extension, 0),
	}
	allext, err := db.GetTable[extensionData]().All()
	if err != nil {
		return nil, err
	}
	for _, dbext := range allext {
		if ext, ok := allExtensionsMap[dbext.Id]; ok {
			extensionList.Extensions = append(extensionList.Extensions, &pb.Extension{
				Id:          ext.Id,
				Title:       ext.Title,
				Description: ext.Description,
				Enable:      dbext.Enable,
			})
		}
	}

	return extensionList, nil
}

func getExtension(id string) (*Extension, error) {
	if !isEnable(id) {
		return nil, fmt.Errorf("Extension with ID %s is not enabled", id)
	}
	if extension, ok := enabledExtensionsMap[id]; ok {
		return extension, nil
	}
	return nil, fmt.Errorf("Extension with ID %s not found", id)
}

func (e ExtensionHostService) Connect(req *pb.ExtensionRequest, stream grpc.ServerStreamingServer[pb.ExtensionResponse]) error {
	extension, err := getExtension(req.GetExtensionId())
	if err != nil {
		log.Printf("Error connecting stream for extension %s: %v", req.GetExtensionId(), err)
		return err
	}

	log.Printf("Connecting stream for extension %s", req.GetExtensionId())
	log.Printf("Extension data: %+v", extension)

	if err := (*extension).UpdateUI((*extension).GetUI()); err != nil {
		log.Printf("Error updating UI for extension %s: %v", req.GetExtensionId(), err)
	}

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
}

func (e ExtensionHostService) SubmitForm(ctx context.Context, req *pb.SendExtensionDataRequest) (*pb.ExtensionActionResult, error) {
	extension, err := getExtension(req.GetExtensionId())
	if err != nil {
		log.Println(err)
		return &pb.ExtensionActionResult{
			ExtensionId: req.ExtensionId,
			Code:        pb.ResponseCode_FAILED,
			Message:     err.Error(),
		}, err
	}
	(*extension).SubmitData(req.Button, req.GetData())

	return &pb.ExtensionActionResult{
		ExtensionId: req.ExtensionId,
		Code:        pb.ResponseCode_OK,
		Message:     "Success",
	}, nil
}

func (e ExtensionHostService) Close(ctx context.Context, req *pb.ExtensionRequest) (*pb.ExtensionActionResult, error) {
	extension, err := getExtension(req.GetExtensionId())
	if err != nil {
		log.Println(err)
		return &pb.ExtensionActionResult{
			ExtensionId: req.ExtensionId,
			Code:        pb.ResponseCode_FAILED,
			Message:     err.Error(),
		}, err
	}
	(*extension).Close()
	(*extension).StoreData()
	return &pb.ExtensionActionResult{
		ExtensionId: req.ExtensionId,
		Code:        pb.ResponseCode_OK,
		Message:     "Success",
	}, nil
}

func (e ExtensionHostService) EditExtension(ctx context.Context, req *pb.EditExtensionRequest) (*pb.ExtensionActionResult, error) {
	if !req.Enable {
		extension, _ := getExtension(req.GetExtensionId())
		if extension != nil {
			(*extension).Close()
			(*extension).StoreData()
		}
		delete(enabledExtensionsMap, req.GetExtensionId())
	}
	table := db.GetTable[extensionData]()
	data, err := table.Get(req.GetExtensionId())
	if err != nil {
		return nil, err
	}
	data.Enable = req.Enable
	table.UpdateInsert(data)

	if req.Enable {
		loadExtension(allExtensionsMap[req.GetExtensionId()])
	}

	return &pb.ExtensionActionResult{
		ExtensionId: req.ExtensionId,
		Code:        pb.ResponseCode_OK,
		Message:     "Success",
	}, nil
}

type extensionData struct {
	Id       string `json:"id"`
	Enable   bool   `json:"enable"`
	JsonData []byte
}
