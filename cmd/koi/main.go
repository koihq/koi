package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/koihq/koi/pkg/koiapp"
)

var version = "dev"
var revision = "unknown"

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version":
			fmt.Printf("koi %s (%s)\n", version, revision)
			return
		case "healthcheck":
			if err := healthcheck(); err != nil {
				slog.Error("healthcheck failed", "err", err)
				os.Exit(1)
			}
			return
		default:
			slog.Error("unknown command", "command", os.Args[1])
			os.Exit(2)
		}
	}

	app, err := koiapp.NewKoiApp()
	if err != nil {
		slog.Error("koi exited", "err", err)
		os.Exit(1)
	}

	slog.Info("koi starting", "version", version, "revision", revision)

	if err := app.Run(); err != nil {
		slog.Error("koi exited", "err", err)
		os.Exit(1)
	}
}

func healthcheck() error {
	addr := strings.TrimSpace(os.Getenv("KOI_PORT"))
	if addr == "" {
		addr = ":8080"
	}

	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("KOI_PORT: %w", err)
	}

	return checkHealth("http://" + net.JoinHostPort("127.0.0.1", port) + "/health")
}

func checkHealth(url string) error {
	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health returned HTTP %d", resp.StatusCode)
	}

	var status struct {
		Status string `json:"status"`
		DB     string `json:"db"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return fmt.Errorf("decode health response: %w", err)
	}
	if status.Status != "ok" || status.DB != "ok" {
		return fmt.Errorf("health returned status=%q db=%q", status.Status, status.DB)
	}

	return nil
}
