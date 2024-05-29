/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"net"
	"os"

	"github.com/spf13/cobra"

	"github.com/smarthall/nodem/internal/modem"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "nodem",
	Short: "Nodem is NOT a modem",
	Long:  `Nodem is a tool to emulate a Hayes modem. You can use it with Qemu to create a dial-up connection.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Testing...")

		// Open the unix socket
		c, err := net.Dial("unix", cmd.Flag("socket").Value.String())
		if err != nil {
			fmt.Println("Error connecting to socket:", err)
			os.Exit(1)
		}

		// Initalise the Modem
		m := modem.New(c)

		// Run the Modem
		m.Run()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.nodem.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	//rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	rootCmd.Flags().StringP("socket", "s", "/tmp/nodem.sock", "Socket file to listen on")
}
