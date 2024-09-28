package main

import (
	"os"

	"github.com/hiddify/hiddify-core/cmd"
)

type UpdateRequest struct {
	Description     string `json:"description,omitempty"`
	PrivatePods     bool   `json:"private_pods"`
	OperatingMode   string `json:"operating_mode,omitempty"`
	ActivationState string `json:"activation_state,omitempty"`
}

func main() {
	cmd.ParseCli(os.Args[1:])

	// var request UpdateRequest
	// // jsonTag, err2 := validation.ErrorFieldName(&request, &request.OperatingMode)
	// jsonTag, err2 := request.ValName(&request.OperatingMode)

	// fmt.Println(jsonTag, err2)
	// RegisterExtension("com.example.extension", NewExampleExtension())
	// ex := extensionsMap["com.example.extension"].(*Extension[struct])
	// fmt.Println(NewExampleExtension().Get())

	// fmt.Println(ex.Get())
}
