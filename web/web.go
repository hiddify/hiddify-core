package web

import (
	"crypto/tls"
	"fmt"
	"github.com/hiddify/libcore/global"
	"github.com/hiddify/libcore/utils"
	"net/http"
	"strconv"
)

const (
	serverCertPath = "cert/server-cert.pem"
	serverKeyPath  = "cert/server-key.pem"
	clientCertPath = "cert/client-cert.pem"
	clientKeyPath  = "cert/client-key.pem"
)

func StartWebServer(Port int) {
	http.HandleFunc("/start", startHandler)
	http.HandleFunc("/stop", StopHandler)
	server := &http.Server{
		Addr: "127.0.0.1:" + fmt.Sprintf("%d", Port),
		TLSConfig: &tls.Config{
			MinVersion:   tls.VersionTLS12,
			Certificates: []tls.Certificate{utils.LoadCertificate(serverCertPath, serverKeyPath)},
			ClientAuth:   tls.RequireAndVerifyClientCert,
			ClientCAs:    utils.LoadClientCA(clientCertPath),
		},
	}
	err := server.ListenAndServeTLS(serverCertPath, serverKeyPath)
	if err != nil {
		panic("Error: " + err.Error())
	}
}
func startHandler(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	Ipv6 := queryParams.Get("Ipv6")
	ServerPort := queryParams.Get("ServerPort")
	StrictRoute := queryParams.Get("StrictRoute")
	EndpointIndependentNat := queryParams.Get("EndpointIndependentNat")
	TheStack := queryParams.Get("Stack")

	ipv6, err := strconv.ParseBool(Ipv6)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusBadRequest)
		return
	}
	serverPort, err := strconv.Atoi(ServerPort)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusBadRequest)
		return
	}
	strictRoute, err := strconv.ParseBool(StrictRoute)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusBadRequest)
		return
	}
	endpointIndependentNat, err := strconv.ParseBool(EndpointIndependentNat)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusBadRequest)
		return
	}
	theStack := GetStack(TheStack)
	if theStack == "UNKNOWN" {
		http.Error(w, fmt.Sprintf("Error: %s", "Stack is not valid"), http.StatusBadRequest)
		return
	}

	parameters := global.Parameters{Ipv6: ipv6, ServerPort: serverPort, StrictRoute: strictRoute, EndpointIndependentNat: endpointIndependentNat, Stack: GetStack(TheStack)}

	err = global.WriteParameters(parameters.Ipv6, parameters.ServerPort, parameters.StrictRoute, parameters.EndpointIndependentNat, GetStringFromStack(parameters.Stack))
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusBadRequest)
		return
	}
	err = global.SetupC("./", "./work", "./tmp", false)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusBadRequest)
		return
	}
	err = global.StartServiceC(true, global.MakeConfig(parameters.Ipv6, parameters.ServerPort, parameters.StrictRoute, parameters.EndpointIndependentNat, GetStringFromStack(parameters.Stack)))
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusBadRequest)
		return
	}

}
func StopHandler(w http.ResponseWriter, r *http.Request) {
	err := global.StopService()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusBadRequest)
		return
	}
}
func GetStack(stack string) global.Stack {
	switch stack {
	case "system":
		return global.System
	case "gVisor":
		return global.GVisor
	case "mixed":
		return global.Mixed
	case "LWIP":
		return global.LWIP
	default:
		return "UNKNOWN"
	}
}
func GetStringFromStack(stack global.Stack) string {
	switch stack {
	case global.System:
		return "system"
	case global.GVisor:
		return "gVisor"
	case global.Mixed:
		return "mixed"
	case global.LWIP:
		return "LWIP"
	default:
		return "UNKNOWN"
	}
}
