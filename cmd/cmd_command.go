package cmd

import (
	"context"
	"log"
	"time"

	"github.com/hiddify/hiddify-core/v2/hcommon"
	hcore "github.com/hiddify/hiddify-core/v2/hcore"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

var execCommand = &cobra.Command{
	Use:   "command",
	Short: "command",
	Args:  cobra.OnlyValidArgs,
	Run:   execCommandgrpc,
}

func init() {
	// commandRun.PersistentFlags().BoolP("help", "", false, "help for this command")
	// commandRun.Flags().StringVarP(&hiddifySettingPath, "hiddify", "d", "", "Hiddify Setting JSON Path")

	mainCommand.AddCommand(execCommand)
}

func execCommandgrpc(cmd *cobra.Command, args []string) {
	conn, err := grpc.Dial("127.0.0.1:17078", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := hcore.NewCoreClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*100)
	defer cancel()
	tream, err := c.CoreInfoListener(ctx, &hcommon.Empty{})
	stream2, err := c.OutboundsInfo(ctx, &hcommon.Empty{})
	stream, err := c.MainOutboundsInfo(ctx, &hcommon.Empty{})
	logstream, err := c.LogListener(ctx, &hcore.LogRequest{Level: hcore.LogLevel_DEBUG})
	if err != nil {
		log.Fatalf("could not stream: %v", err)
	}

	go coreLogger(tream, "coreinfo")
	go coreLogger(stream2, "alloutbounds")
	go coreLogger(stream, "mainoutbounds")
	go coreLogger(logstream, "log")
	for _, x := range []string{
		"m4 § 0",
		"warp in warp § 1",
		"LocalIP § 2",
		"WarpInWarp✅ § 3",
	} {
		c.UrlTest(ctx, &hcore.UrlTestRequest{Tag: "select"})
		log.Printf("Sending: %s", x)
		resp, err := c.SelectOutbound(ctx, &hcore.SelectOutboundRequest{
			GroupTag:    "select",
			OutboundTag: x,
		})
		if err != nil {
			log.Fatalf("could not greet: %s %v", x, err)
		}

		log.Printf("Received: %s %v", x, resp)
		<-time.After(1 * time.Second)
	}
	<-time.After(10 * time.Second)
}

func coreLogger[T any](stream grpc.ServerStreamingClient[T], name string) {
	for {
		c, err := stream.Recv()
		if err != nil {
			log.Printf("could not receive %s : %v", name, err)
		}
		log.Printf("Received: %s, %v", name, c)
	}
}
