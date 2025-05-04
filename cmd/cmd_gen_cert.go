package cmd

import (
	"os"

	"github.com/hiddify/hiddify-core/v2/hutils"
	"github.com/spf13/cobra"
)

var commandGenerateCertification = &cobra.Command{
	Use:   "gen-cert",
	Short: "Generate certification for web server",
	Run: func(cmd *cobra.Command, args []string) {
		err := os.MkdirAll("cert", 0o644)
		if err != nil {
			panic("Error: " + err.Error())
		}
		hutils.GenerateCertificateFile("data/cert/server-cert.pem", "data/cert/server-key.pem", true, true)
		hutils.GenerateCertificateFile("data/cert/client-cert.pem", "data/cert/client-key.pem", false, true)
	},
}
