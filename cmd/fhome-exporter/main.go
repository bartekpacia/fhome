package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/bartekpacia/fhome/api"
	"github.com/bartekpacia/fhome/highlevel"
	"github.com/bartekpacia/fhome/internal"
	"github.com/lmittmann/tint"
	"github.com/urfave/cli/v3"
)

// This is set by GoReleaser, see https://goreleaser.com/cookbooks/using-main.version
var version = "dev"

func main() {
	app := &cli.Command{
		Name:            "fhome-exporter",
		Usage:           "Prometheus exporter for F&Home temperature data",
		Version:         version,
		HideHelpCommand: true,
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:  "port",
				Usage: "port to listen on",
				Value: 9222,
			},
			&cli.BoolFlag{
				Name:  "json",
				Usage: "output logs in JSON Lines format",
			},
			&cli.BoolFlag{
				Name:  "debug",
				Usage: "show debug logs",
			},
		},
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			var level slog.Level
			if cmd.Bool("debug") {
				level = slog.LevelDebug
			} else {
				level = slog.LevelInfo
			}

			if cmd.Bool("json") {
				handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
				slog.SetDefault(slog.New(handler))
			} else {
				handler := tint.NewHandler(os.Stdout, &tint.Options{Level: level, TimeFormat: time.TimeOnly})
				slog.SetDefault(slog.New(handler))
			}

			return ctx, nil
		},
		Action: run,
	}

	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt)
	err := app.Run(ctx, os.Args)
	if err != nil {
		slog.Error("exit", slog.Any("error", err))
		os.Exit(1)
	}
}

func run(ctx context.Context, cmd *cli.Command) error {
	config := internal.Load()
	port := cmd.Int("port")

	apiClient, err := highlevel.Connect(ctx, config, nil)
	if err != nil {
		return fmt.Errorf("connect to fhome: %v", err)
	}

	apiConfig, err := highlevel.GetConfigs(ctx, apiClient)
	if err != nil {
		return fmt.Errorf("get configs: %v", err)
	}

	slog.Info("connected to F&Home",
		slog.Int("panels", len(apiConfig.Panels)),
		slog.Int("cells", len(apiConfig.Cells())),
	)

	tempCells := filterTemperatureCells(apiConfig)
	slog.Info("found temperature cells", slog.Int("count", len(tempCells)))

	// TODO: api.Client is not concurrency-safe; add a mutex if concurrent scrapes become an issue
	// TODO: consider background polling if scrape latency becomes a problem
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><body><h1>F&Home Exporter</h1><p><a href="/metrics">Metrics</a></p></body></html>`)
	})
	mux.HandleFunc("GET /metrics", func(w http.ResponseWriter, r *http.Request) {
		slog.Debug("received request", slog.String("method", r.Method), slog.String("path", r.URL.Path), slog.String("remote_addr", r.RemoteAddr))
		handleMetrics(r.Context(), w, apiClient, tempCells)
	})

	addr := fmt.Sprintf("0.0.0.0:%d", port)
	server := &http.Server{Addr: addr, Handler: mux}

	errs := make(chan error)
	go func() {
		slog.Info("listening", "addr", fmt.Sprintf("http://%s", addr))
		err := server.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			slog.Info("HTTP server closed")
			errs <- nil
		} else {
			slog.Error("HTTP server error", slog.Any("error", err))
			errs <- err
		}
	}()

	go func() {
		<-ctx.Done()
		slog.Info("context done, shutting down HTTP server")
		errs <- server.Shutdown(context.Background())
	}()

	return <-errs
}

type tempCell struct {
	CellID    int
	CellName  string
	PanelName string
}

func filterTemperatureCells(cfg *api.Config) []tempCell {
	var cells []tempCell
	for _, panel := range cfg.Panels {
		for _, cell := range panel.Cells {
			if cell.DisplayType != string(api.Temperature) {
				continue
			}
			cells = append(cells, tempCell{
				CellID:    cell.ID,
				CellName:  cell.Name,
				PanelName: panel.Name,
			})
		}
	}
	return cells
}

func handleMetrics(ctx context.Context, w http.ResponseWriter, client *api.Client, tempCells []tempCell) {
	msg, err := client.SendAction(ctx, api.ActionStatusTouches)
	if err != nil {
		// TODO: implement reconnection logic
		slog.Error("failed to send statustouches", slog.Any("error", err))
		http.Error(w, "failed to collect metrics", http.StatusServiceUnavailable)
		return
	}

	var resp api.StatusTouchesChangedResponse
	if err := json.Unmarshal(msg.Raw, &resp); err != nil {
		slog.Error("failed to unmarshal statustouches response", slog.Any("error", err))
		http.Error(w, "failed to collect metrics", http.StatusServiceUnavailable)
		return
	}

	cellValues := make(map[string]api.CellValue, len(resp.Response.CellValues))
	for _, cv := range resp.Response.CellValues {
		cellValues[cv.ID] = cv
	}

	// TODO: consider prometheus/client_golang if more metrics are added
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

	fmt.Fprintln(w, "# HELP fhome_room_temperature_celsius Room temperature in degrees Celsius")
	fmt.Fprintln(w, "# TYPE fhome_room_temperature_celsius gauge")

	for _, tc := range tempCells {
		cellValue, ok := cellValues[strconv.Itoa(tc.CellID)]
		if !ok {
			slog.Debug("no value for temperature cell", slog.Int("cell_id", tc.CellID), slog.String("name", tc.CellName))
			continue
		}

		temp, err := api.DecodeTemperatureValueStr(cellValue.ValueStr)
		if err != nil {
			slog.Warn("failed to decode temperature",
				slog.Int("cell_id", tc.CellID),
				slog.String("value_str", cellValue.ValueStr),
				slog.Any("error", err),
			)
			continue
		}

		fmt.Fprintf(w, "fhome_room_temperature_celsius{panel=%q,room=%q,cell_id=%q} %g\n",
			tc.PanelName, tc.CellName, strconv.Itoa(tc.CellID), temp,
		)
	}
}
