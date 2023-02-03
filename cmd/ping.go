/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/joyme123/gnt/ping"
)

var pinger = ping.Pinger{}

// pingCmd represents the ping command
var pingCmd = &cobra.Command{
	Use:   "ping",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			log.Println("must specify a target address to ping")
			os.Exit(1)
		}
		pinger.TargetAddr = args[0]

		pinger.SetLogger(log.Default())
		pinger.SetDebugLogger(DebugLogger)

		ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		if err := pinger.Run(ctx); err != nil {
			log.Printf("ping failed: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(pingCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// pingCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	pingCmd.Flags().IntVarP(&pinger.Count, "count", "c", 0, "times of sending icmp echo request")
	pingCmd.Flags().IntVarP(&pinger.Interval, "interval", "i", 1, "interval")
	pingCmd.Flags().StringVarP(&pinger.Interface, "interface", "I", "", "interface")
	pingCmd.Flags().IntVarP(&pinger.TTL, "ttl", "t", 64, "ttl")
	pingCmd.Flags().IntVarP(&pinger.Timeout, "timeout", "W", 0, "timeout")
	pingCmd.Flags().IntVarP(&pinger.Deadline, "deadline", "w", 1, "deadline")
	pingCmd.Flags().BoolVarP(&pinger.UDP, "udp", "u", false, "udp")

}
