package main

import (
	"flag"

	"github.com/maistra/ior/pkg/galley"
	"github.com/spf13/cobra"
	"istio.io/istio/pkg/log"
)

var (
	loggingOptions = log.DefaultOptions()
	galleyAddr     = "istio-galley.istio-system:9901"
)

func getRootCmd(args []string) *cobra.Command {

	rootCmd := &cobra.Command{
		Use:   "server",
		Short: "Connects to Galley and manages OpenShift Routes based on Istio Gateways",
		Run: func(cmd *cobra.Command, args []string) {
			log.Configure(loggingOptions)
			galley.ConnectToGalley(galleyAddr)
		},
	}

	rootCmd.SetArgs(args)
	rootCmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
	rootCmd.PersistentFlags().StringVarP(&galleyAddr, "mcp-address", "", galleyAddr,
		"Galley's MCP server address")

	loggingOptions.AttachCobraFlags(rootCmd)

	return rootCmd
}
