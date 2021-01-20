package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check health status of the target application.",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Todo
		if len(args) == 0 {
			fmt.Println("OK")
		} else {
			for _, arg := range args {
				fmt.Printf("%v: OK\n", arg)
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(healthCmd)
}
