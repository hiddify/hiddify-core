package main

/*
#include "stdint.h"
*/
import "C"
import (
	"encoding/json"
	"fmt"
	"os"
	"unsafe"

	"github.com/hiddify/hiddify-core/bridge"
	"github.com/hiddify/hiddify-core/config"
	pb "github.com/hiddify/hiddify-core/hiddifyrpc"
	v2 "github.com/hiddify/hiddify-core/v2"

	"github.com/sagernet/sing-box/log"
)

//export setupOnce
func setupOnce(api unsafe.Pointer) {
	bridge.InitializeDartApi(api)
}

//export setup
func setup(baseDir *C.char, workingDir *C.char, tempDir *C.char, statusPort C.longlong, debug bool) (CErr *C.char) {
	err := v2.Setup(C.GoString(baseDir), C.GoString(workingDir), C.GoString(tempDir), int64(statusPort), debug)

	return emptyOrErrorC(err)
}

//export parse
func parse(path *C.char, tempPath *C.char, debug bool) (CErr *C.char) {
	res, err := v2.Parse(&pb.ParseRequest{
		ConfigPath: C.GoString(path),
		TempPath:   C.GoString(tempPath),
	})
	if err != nil {
		log.Error(err.Error())
		return C.CString(err.Error())
	}

	err = os.WriteFile(C.GoString(path), []byte(res.Content), 0644)
	return emptyOrErrorC(err)
}

//export changeConfigOptions
func changeConfigOptions(configOptionsJson *C.char) (CErr *C.char) {

	_, err := v2.ChangeConfigOptions(&pb.ChangeConfigOptionsRequest{
		ConfigOptionsJson: C.GoString(configOptionsJson),
	})
	return emptyOrErrorC(err)
}

//export generateConfig
func generateConfig(path *C.char) (res *C.char) {
	_, err := v2.GenerateConfig(&pb.GenerateConfigRequest{
		Path: C.GoString(path),
	})
	return emptyOrErrorC(err)
}

//export start
func start(configPath *C.char, disableMemoryLimit bool) (CErr *C.char) {

	_, err := v2.Start(&pb.StartRequest{
		ConfigPath:             C.GoString(configPath),
		EnableOldCommandServer: true,
		DisableMemoryLimit:     disableMemoryLimit,
	})
	return emptyOrErrorC(err)
}

//export stop
func stop() (CErr *C.char) {

	_, err := v2.Stop()
	return emptyOrErrorC(err)
}

//export restart
func restart(configPath *C.char, disableMemoryLimit bool) (CErr *C.char) {

	_, err := v2.Restart(&pb.StartRequest{
		ConfigPath:             C.GoString(configPath),
		EnableOldCommandServer: true,
		DisableMemoryLimit:     disableMemoryLimit,
	})
	return emptyOrErrorC(err)
}

//export startCommandClient
func startCommandClient(command C.int, port C.longlong) *C.char {
	err := v2.StartCommand(int32(command), int64(port))
	return emptyOrErrorC(err)
}

//export stopCommandClient
func stopCommandClient(command C.int) *C.char {
	err := v2.StopCommand(int32(command))
	return emptyOrErrorC(err)
}

//export selectOutbound
func selectOutbound(groupTag *C.char, outboundTag *C.char) (CErr *C.char) {

	_, err := v2.SelectOutbound(&pb.SelectOutboundRequest{
		GroupTag:    C.GoString(groupTag),
		OutboundTag: C.GoString(outboundTag),
	})

	return emptyOrErrorC(err)
}

//export urlTest
func urlTest(groupTag *C.char) (CErr *C.char) {
	_, err := v2.UrlTest(&pb.UrlTestRequest{
		GroupTag: C.GoString(groupTag),
	})

	return emptyOrErrorC(err)
}

func emptyOrErrorC(err error) *C.char {
	if err == nil {
		return C.CString("")
	}
	log.Error(err.Error())
	return C.CString(err.Error())
}

//export generateWarpConfig
func generateWarpConfig(licenseKey *C.char, accountId *C.char, accessToken *C.char) (CResp *C.char) {
	res, err := v2.GenerateWarpConfig(&pb.GenerateWarpConfigRequest{
		LicenseKey:  C.GoString(licenseKey),
		AccountId:   C.GoString(accountId),
		AccessToken: C.GoString(accessToken),
	})

	if err != nil {
		return C.CString(fmt.Sprint("error: ", err.Error()))
	}
	warpAccount := config.WarpAccount{
		AccountID:   res.Account.AccountId,
		AccessToken: res.Account.AccessToken,
	}
	warpConfig := config.WarpWireguardConfig{
		PrivateKey:       res.Config.PrivateKey,
		LocalAddressIPv4: res.Config.LocalAddressIpv4,
		LocalAddressIPv6: res.Config.LocalAddressIpv6,
		PeerPublicKey:    res.Config.PeerPublicKey,
		ClientID:         res.Config.ClientId,
	}
	log := res.Log
	response := &config.WarpGenerationResponse{
		WarpAccount: warpAccount,
		Log:         log,
		Config:      warpConfig,
	}

	responseJson, err := json.Marshal(response)
	if err != nil {
		return C.CString("")
	}
	return C.CString(string(responseJson))

}

func main() {}
