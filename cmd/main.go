package main

import (
	"fmt"
	"maritime_traffic/cmd/server"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	var rootCmd = &cobra.Command{Use: "maritime-traffic"}

	rootCmd.AddCommand(
		server.NewServerCmd(),
	)
	err := rootCmd.Execute()
	if err != nil {
		_, _ = fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}
}
