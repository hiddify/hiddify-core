package admin_service

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/hiddify/libcore/global"
	"github.com/hiddify/libcore/utils"
)

const (
	serverCertPath = "cert/server-cert.pem"
	serverKeyPath  = "cert/server-key.pem"
	clientCertPath = "cert/client-cert.pem"
	clientKeyPath  = "cert/client-key.pem"
)

func StartWebServer(Port int, TLS bool) {
	http.HandleFunc("/start", startHandler)
	http.HandleFunc("/stop", StopHandler)
	http.HandleFunc("/status", StatusHandler)
	http.HandleFunc("/exit", ExitHandler)
	server := &http.Server{
		Addr: "127.0.0.1:" + fmt.Sprintf("%d", Port),
	}
	var err error
	if TLS {
		server.TLSConfig = &tls.Config{
			MinVersion:   tls.VersionTLS12,
			Certificates: []tls.Certificate{utils.LoadCertificate(serverCertPath, serverKeyPath)},
			ClientAuth:   tls.RequireAndVerifyClientCert,
			ClientCAs:    utils.LoadClientCA(clientCertPath),
		}
		err = server.ListenAndServeTLS(serverCertPath, serverKeyPath)
	} else {
		err = server.ListenAndServe()
	}
	if err != nil {
		panic("Error: " + err.Error())
	}
}
func startHandler(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	ipv6, err := strconv.ParseBool(queryParams.Get("Ipv6"))
	if err != nil {
		fmt.Printf("ipv6 Error: %v ==>using false\n", err)
		ipv6 = false
	}
	serverPort, err := strconv.Atoi(queryParams.Get("ServerPort"))
	if err != nil {
		fmt.Printf("serverPort Error: %v ==>using 2334\n", err)
		serverPort = 2334
	}
	strictRoute, err := strconv.ParseBool(queryParams.Get("StrictRoute"))
	if err != nil {
		fmt.Printf("strictRoute Error: %v ==>using false\n", err)
		strictRoute = false
	}
	endpointIndependentNat, err := strconv.ParseBool(queryParams.Get("EndpointIndependentNat"))
	if err != nil {
		fmt.Printf("endpointIndependentNat Error: %v ==>using false\n", err)
		endpointIndependentNat = false
	}
	theStack := GetStack(queryParams.Get("Stack"))

	parameters := global.Parameters{Ipv6: ipv6, ServerPort: serverPort, StrictRoute: strictRoute, EndpointIndependentNat: endpointIndependentNat, Stack: theStack}

	// err = global.WriteParameters(parameters.Ipv6, parameters.ServerPort, parameters.StrictRoute, parameters.EndpointIndependentNat, GetStringFromStack(parameters.Stack))
	// if err != nil {
	// http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusBadRequest)
	// return
	// }
	err = global.SetupC("./", "./", "./tmp", false)

	if err != nil {
		http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusBadRequest)
		return
	}
	err = global.StartServiceC(true, global.MakeConfig(parameters.Ipv6, parameters.ServerPort, parameters.StrictRoute, parameters.EndpointIndependentNat, GetStringFromStack(parameters.Stack)))
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusBadRequest)
		return
	}
	http.Error(w, fmt.Sprintf("Ok"), http.StatusOK)
}
func StatusHandler(w http.ResponseWriter, r *http.Request) {
}
func StopHandler(w http.ResponseWriter, r *http.Request) {
	err := global.StopServiceC()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusBadRequest)
		return
	}
	http.Error(w, fmt.Sprintf("Ok"), http.StatusOK)
}
func ExitHandler(w http.ResponseWriter, r *http.Request) {
	global.StopServiceC()
	os.Exit(0)
	http.Error(w, fmt.Sprintf("Ok"), http.StatusOK)
}
func GetStack(stack string) global.Stack {

	switch stack {
	case "system":
		return global.System
	case "gvisor":
		return global.GVisor
	case "mixed":
		return global.Mixed
	// case "LWIP":
	// 	return global.LWIP
	default:
		fmt.Printf("Stack Error: %s is not valid==> using GVisor\n", stack)
		return global.GVisor
	}
}
func GetStringFromStack(stack global.Stack) string {
	switch stack {
	case global.System:
		return "system"
	case global.GVisor:
		return "gvisor"
	case global.Mixed:
		return "mixed"
	// case global.LWIP:
	// 	return "LWIP"
	default:
		return "UNKNOWN"
	}
}
