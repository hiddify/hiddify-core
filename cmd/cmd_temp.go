package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/hiddify/hiddify-core/v2/config"

	// "github.com/hiddify/hiddify-core/extension_repository/cleanip_scanner"
	"github.com/spf13/cobra"
)

var commandTemp = &cobra.Command{
	Use:   "temp",
	Short: "temp",
	Args:  cobra.MaximumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		// fmt.Printf("Ping time: %d ms\n", Ping())
		tmp := map[string][]string{
			"direct-dns-address":         {"1.1.1.1"},
			"tls-tricks.enable-fragment": {"true"},
			"tls-tricks.fragment-size":   {"2-4"},
		}
		h := config.GetOverridableHiddifyOptions(tmp)
		j, _ := json.Marshal(h)
		fmt.Println(string(j))
	},
}

func init() {
	mainCommand.AddCommand(commandTemp)
}
