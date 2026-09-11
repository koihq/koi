package main

import (
	"log/slog"
	"os"

	"github.com/koihq/koi/pkg/koiapp"
)

func main() {
	if err := koiapp.NewKoiApp().Run(); err != nil {
		slog.Error("koi exited", "err", err)
		os.Exit(1)
	}
}
