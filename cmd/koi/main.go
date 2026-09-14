package main

import (
	"log/slog"
	"os"

	"github.com/koihq/koi/pkg/koiapp"
)

func main() {
	app, err := koiapp.NewKoiApp()
	if err != nil {
		slog.Error("koi exited", "err", err)
		os.Exit(1)
	}
	if err := app.Run(); err != nil {
		slog.Error("koi exited", "err", err)
		os.Exit(1)
	}
}
