package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/rnemeth90/yahba/internal/agent"
	"github.com/rnemeth90/yahba/internal/logger"
	"github.com/spf13/cobra"
)

var (
	agentCoordinator string
	agentName        string
)

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Start an agent for distributed load testing",
	Long: `Start an agent that connects to a coordinator and waits for
load test work. The agent registers itself, polls for assignments,
executes its share of requests, and reports results back.`,
	Run: func(cmd *cobra.Command, args []string) {
		log := logger.New("info", "stdout", false)

		if agentName == "" {
			hostname, err := os.Hostname()
			if err != nil {
				hostname = "agent"
			}
			agentName = hostname
		}

		a := agent.New(agentName, agentCoordinator, log)

		ctx, cancel := context.WithCancel(context.Background())
		shutdown := make(chan os.Signal, 1)
		signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-shutdown
			fmt.Println("\nShutting down agent...")
			cancel()
		}()

		if err := a.Run(ctx); err != nil {
			log.Error("Agent error: %v", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(agentCmd)
	agentCmd.Flags().StringVar(&agentCoordinator, "coordinator", "http://localhost:9090", "Coordinator address (e.g. http://host:9090)")
	agentCmd.Flags().StringVar(&agentName, "name", "", "Agent name (defaults to hostname)")
}
