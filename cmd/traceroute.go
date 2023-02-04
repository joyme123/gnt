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

	"github.com/joyme123/gnt/traceroute"
)

// tracerouteCmd represents the traceroute command
var tracerouteCmd = &cobra.Command{
	Use:   "traceroute",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			log.Println("must specify a target address to traceroute")
			os.Exit(1)
		}

		ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		trace := traceroute.NewTraceRouter(opt, args[0])
		trace.SetDebugLogger(DebugLogger)
		if err := trace.Run(ctx); err != nil {
			log.Println(err.Error())
			os.Exit(1)
		}
	},
}

var opt traceroute.Options

func init() {
	rootCmd.AddCommand(tracerouteCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// tracerouteCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// tracerouteCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	tracerouteCmd.Flags().BoolVarP(&opt.IPv4, "ipv4", "4", false, "")
	tracerouteCmd.Flags().BoolVarP(&opt.IPv6, "ipv6", "6", false, "")
	tracerouteCmd.Flags().BoolVarP(&opt.ICMP, "icmp", "I", false, "")
	tracerouteCmd.Flags().BoolVarP(&opt.TCP, "tcp", "T", false, "")
	tracerouteCmd.Flags().BoolVarP(&opt.UDP, "udp", "U", false, "")
	tracerouteCmd.Flags().Uint8VarP(&opt.FirstTTL, "first", "f", 1, "Start from the first_ttl hop (instead from 1)")
	tracerouteCmd.Flags().Uint8VarP(&opt.MaxTTL, "max-hops", "m", 30, "Set the max number of hops (max TTL to be reached). Default is 30")
	tracerouteCmd.Flags().IntVarP(&opt.Squeries, "sim-queries", "N", 16, "Set the number of probes to be tried simultaneously (default is 16)")
	tracerouteCmd.Flags().IntVarP(&opt.Nqueries, "queries", "q", 3, "Set the number of probes per each hop. Default is 3")
	tracerouteCmd.Flags().IntVarP(&opt.Port, "port", "p", 0,
		`Set the destination port to use. It is either
initial udp port value for "default" method
(incremented by each probe, default is 33434), or
initial seq for "icmp" (incremented as well,
default from 1), or some constant destination
port for other methods (with default of 80 for
"tcp", 53 for "udp", etc.)`)
	tracerouteCmd.Flags().IntVarP(&opt.SendWait, "sendwait", "z", 0, "Minimal time interval between probes (default 0). If the value is more than 10, then it specifies a number in milliseconds, else it is a number of seconds (float point values allowed too)")
	tracerouteCmd.Flags().BoolVarP(&opt.Unprivileged, "unprivileged", "u", true, "unprivileged mode")
}
