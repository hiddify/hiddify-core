package cmd

import (
	hcore "github.com/hiddify/hiddify-core/v2/hcore"

	"github.com/spf13/cobra"
)

var commandRun = &cobra.Command{
	Use:   "run",
	Short: "run",
	Args:  cobra.OnlyValidArgs,
	Run:   runCommand,
}

func init() {
	// commandRun.PersistentFlags().BoolP("help", "", false, "help for this command")
	// commandRun.Flags().StringVarP(&hiddifySettingPath, "hiddify", "d", "", "Hiddify Setting JSON Path")

	addHConfigFlags(commandRun)

	mainCommand.AddCommand(commandRun)
}

func runCommand(cmd *cobra.Command, args []string) {
	hcore.Setup(
		hcore.SetupParameters{
			BasePath:          "./tmp",
			WorkingDir:        "./",
			TempDir:           "./tmp",
			FlutterStatusPort: 0,
			Debug:             false,
			Mode:              hcore.GRPC_NORMAL_INSECURE,
			Listen:            "127.0.0.1:17078",
		},
	)
	// conn, err := grpc.Dial("127.0.0.1:17078", grpc.WithInsecure())
	// if err != nil {
	// 	fmt.Printf("did not connect: %v", err)
	// }
	// defer conn.Close()
	// c := hello.NewHelloClient(conn)
	// ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	// defer cancel()
	// res, err := c.SayHello(ctx, &hello.HelloRequest{Name: "test"})
	// fmt.Println(res, err)
	// <-time.After(10 * time.Second)
	hcore.RunStandalone(hiddifySettingPath, configPath, defaultConfigs)
}
