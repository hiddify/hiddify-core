package cmd

import (
	"os"
	"os/signal"
	"syscall"

	v2 "github.com/hiddify/hiddify-core/v2"
	"github.com/sagernet/sing-box/log"
	"github.com/spf13/cobra"
)

var commandInstance = &cobra.Command{
	Use:   "instance",
	Short: "instance",
	Args:  cobra.OnlyValidArgs,
	Run: func(cmd *cobra.Command, args []string) {
		hiddifySetting := defaultConfigs
		if hiddifySettingPath != "" {
			hiddifySetting2, err := v2.ReadHiddifyOptionsAt(hiddifySettingPath)
			if err != nil {
				log.Fatal(err)
			}
			hiddifySetting = *hiddifySetting2
		}

		instance, err := v2.RunInstanceString(&hiddifySetting, configPath)
		if err != nil {
			log.Fatal(err)
		}
		defer instance.Close()
		ping, err := instance.PingAverage("http://cp.cloudflare.com", 4)
		if err != nil {
			// log.Fatal(err)
		}
		log.Info("Average Ping to Cloudflare : ", ping, "\n")

		for i := 1; i <= 4; i++ {
			ping, err := instance.PingCloudflare()
			if err != nil {
				log.Warn(i, " Error ", err, "\n")
			} else {
				log.Info(i, " Ping time: ", ping, " ms\n")
			}
		}
		log.Info("Instance is running on port socks5://127.0.0.1:", instance.ListenPort, "\n")
		log.Info("Press Ctrl+C to exit\n")
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		log.Info("CTRL+C recived-->stopping\n")
		instance.Close()
	},
}

func init() {
	mainCommand.AddCommand(commandInstance)
	addHConfigFlags(commandInstance)
}
