package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/rnemeth90/yahba/internal/web"
	"github.com/spf13/cobra"
)

var (
	webAddr string
	webDB   string
)

var webCmd = &cobra.Command{
	Use:   "web",
	Short: "Launch the web GUI for load testing",
	RunE: func(cmd *cobra.Command, args []string) error {
		dbPath := webDB
		if dbPath == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("cannot locate home directory (use --db): %w", err)
			}
			dbPath = filepath.Join(home, ".yahba", "history.db")
		}
		if err := os.MkdirAll(filepath.Dir(dbPath), 0o700); err != nil {
			return err
		}

		store, err := web.OpenStore(dbPath)
		if err != nil {
			return fmt.Errorf("opening history database: %w", err)
		}
		defer store.Close()
		if err := store.MarkAbandoned(); err != nil {
			return err
		}

		runner := web.NewRunner(store)
		srv := &http.Server{
			Addr:              webAddr,
			Handler:           web.NewServer(store, runner),
			ReadHeaderTimeout: 10 * time.Second,
		}

		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		errCh := make(chan error, 1)
		go func() { errCh <- srv.ListenAndServe() }()
		fmt.Printf("YAHBA web UI listening on http://%s (history: %s)\n", webAddr, dbPath)

		select {
		case err := <-errCh:
			return err
		case <-ctx.Done():
		}

		runner.Shutdown()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	},
}

func init() {
	webCmd.Flags().StringVar(&webAddr, "addr", "127.0.0.1:8080", "Address to listen on (keep it on localhost; the UI can launch load tests)")
	webCmd.Flags().StringVar(&webDB, "db", "", "SQLite history file (default ~/.yahba/history.db)")
	rootCmd.AddCommand(webCmd)
}
