package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	v2 "github.com/hiddify/hiddify-core/v2"

	"github.com/hiddify/hiddify-core/utils"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"google.golang.org/grpc"
)

func StartTestExtensionServer() {
	v2.Setup("./tmp", "./", "./tmp", 0, false)
	StartExtensionServer()
}

func StartExtensionServer() {
	grpc_server, _ := v2.StartCoreGrpcServer("127.0.0.1:12345")
	fmt.Printf("Waiting for CTRL+C to stop\n")
	runWebserver(grpc_server)
}

func allowCors(resp http.ResponseWriter, req *http.Request) {
	resp.Header().Set("Access-Control-Allow-Origin", "*")
	resp.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	resp.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if req.Method == "OPTIONS" {
		resp.WriteHeader(http.StatusOK)
		return
	}
}

func runWebserver(grpcServer *grpc.Server) {
	// Context for cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Channels to signal termination
	grpcTerminated := make(chan struct{})
	grpcWebTerminated := make(chan struct{})

	// Specify the directory to serve static files
	dir := "./extension/html/"

	// Wrapping gRPC server with grpc-web
	grpcWeb := grpcweb.WrapServer(grpcServer)

	// HTTP multiplexer
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(resp http.ResponseWriter, req *http.Request) {
		allowCors(resp, req)
		if grpcWeb.IsGrpcWebRequest(req) || grpcWeb.IsAcceptableGrpcCorsRequest(req) {
			grpcWeb.ServeHTTP(resp, req)
		} else {
			http.DefaultServeMux.ServeHTTP(resp, req)
		}
	})

	// File server for static files
	fs := http.FileServer(http.Dir(dir))
	http.Handle("/", http.StripPrefix("/", fs))

	// HTTP server for grpc-web
	rpcWebServer := &http.Server{
		Handler: mux,
		Addr:    ":12346",
	}
	log.Println("Serving grpc-web from https://localhost:12346/")

	// Add a goroutine for the grpc-web server
	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		defer wg.Done()
		utils.GenerateCertificate("cert/server-cert.pem", "cert/server-key.pem", true, true)
		if err := rpcWebServer.ListenAndServeTLS("cert/server-cert.pem", "cert/server-key.pem"); err != nil && err != http.ErrServerClosed {
			// if err := rpcWebServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Web server (gRPC-web) shutdown with error: %s", err)
		}
		grpcServer.Stop()
		close(grpcWebTerminated) // Server terminated
	}()

	// Signal handling to gracefully shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-ctx.Done(): // Context canceled
		log.Println("Context canceled, shutting down servers...")
	case sig := <-sigChan: // OS signal received
		log.Printf("Received signal: %s, shutting down servers...", sig)
	case <-grpcTerminated: // Unexpected gRPC termination
		log.Println("gRPC server terminated unexpectedly")
	case <-grpcWebTerminated: // Unexpected gRPC-web termination
		log.Println("gRPC-web server terminated unexpectedly")
	}

	// Graceful shutdown of the servers
	if err := rpcWebServer.Shutdown(ctx); err != nil {
		log.Printf("gRPC-web server shutdown with error: %s", err)
	}
	<-grpcWebTerminated

	// Ensure all routines finish
	wg.Wait()
	log.Println("Server shutdown complete")
}
