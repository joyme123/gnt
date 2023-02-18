/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/joyme123/gnt/tcpdump"
)

// tcpdumpCmd represents the tcpdump command
var tcpdumpCmd = &cobra.Command{
	Use:   "tcpdump",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		bpfFilter := ""
		if len(args) > 0 {
			bpfFilter = strings.Join(args, " ")
		}

		dumper := tcpdump.NewTcpdump(&tcpdumpOpts, DebugLogger)

		ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		if err := dumper.Sniff(ctx, bpfFilter); err != nil {
			log.Printf("ping failed: %v\n", err)
			os.Exit(1)
		}
	},
}

var tcpdumpOpts tcpdump.Options

func init() {
	rootCmd.AddCommand(tcpdumpCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// tcpdumpCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// tcpdumpCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	tcpdumpCmd.Flags().StringVarP(&tcpdumpOpts.Interface, "interface", "i", "any", "interface to sniff")
	tcpdumpCmd.Flags().BoolVarP(&tcpdumpOpts.Ethernet, "ethernet", "e", false, "dump ethernet info")
}
