package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "phono",
	Short: "DSP pipeline",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Usage()
	},
}

// Execute the root comand.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run(interrupt chan<- struct{}, onInterrupt func()) {
	sigint := make(chan os.Signal, 1)
	// interrupt and sigterm signal
	signal.Notify(sigint, os.Interrupt)
	signal.Notify(sigint, syscall.SIGTERM)

	// block until signal received
	<-sigint
	onInterrupt()
	close(interrupt)
}
