package cmd

import (
	"github.com/rnemeth90/yahba/internal/coordinator"
	"github.com/rnemeth90/yahba/internal/logger"
	"github.com/spf13/cobra"
)

var coordinatorPort string

var coordinatorCmd = &cobra.Command{
	Use:   "coordinator",
	Short: "Start a coordinator for distributed load testing",
	Long: `Start a coordinator server that manages agents and distributes
load test work across them. Agents connect to this coordinator,
and tests are submitted via 'yahba run --distributed'.`,
	Run: func(cmd *cobra.Command, args []string) {
		log := logger.New("info", "stdout", false)
		coord := coordinator.New(coordinatorPort, log)
		if err := coord.Run(); err != nil {
			log.Error("Coordinator error: %v", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(coordinatorCmd)
	coordinatorCmd.Flags().StringVarP(&coordinatorPort, "port", "p", ":9090", "Port to listen on")
}
