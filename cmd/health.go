package cmd

import (
	"fmt"
	"grpcdebug/transport"

	"github.com/spf13/cobra"
)

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check health status of the target application.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			fmt.Println(transport.GetHealthStatus(""))
		}
		for _, service := range args {
			fmt.Fprintf(
				w, "%v:\t%v\t\n",
				service,
				transport.GetHealthStatus(service),
			)
			w.Flush()
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(healthCmd)
}
