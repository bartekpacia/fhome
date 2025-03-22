package webserver

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"log/slog"
	"net/http"

	"github.com/bartekpacia/fhome/api"
)

//go:embed assets/*
var assets embed.FS

//go:embed templates/*
var templates embed.FS

var tmpl = template.Must(template.ParseFS(templates, "templates/*"))

const port = 9001

func Run(ctx context.Context, client *api.Client, homeConfig *api.Config, email string) error {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /index", func(w http.ResponseWriter, r *http.Request) {
		slog.Info("got request", slog.String("method", r.Method), slog.String("path", r.URL.Path))

		data := map[string]interface{}{
			"Email":  email,
			"Panels": homeConfig.Panels,
			"Cells":  homeConfig.Cells(),
		}

		tmpl.ExecuteTemplate(w, "index.html.tmpl", data)
	})
	// Hacky workaround for myself to open my gate from my phone.
	mux.HandleFunc("GET /gate", func(w http.ResponseWriter, r *http.Request) {
		var result string
		err := client.SendEvent(ctx, 260, api.ValueToggle)
		if err != nil {
			result = fmt.Sprintf("Failed to send event: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
		}

		if result != "" {
			log.Print(result)
			fmt.Fprint(w, result)
		}
	})

	mux.HandleFunc("GET /api/objects", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(homeConfig.Cells())
		if err != nil {
			http.Error(w, "failed to encode response to json", http.StatusInternalServerError)
			return
		}
	})

	mux.Handle("GET /public", http.StripPrefix("/public/", http.FileServer(http.FS(assets))))
	addr := fmt.Sprint("0.0.0.0:", port)
	httpServer := http.Server{Addr: addr, Handler: mux}

	errs := make(chan error)
	go func() {
		slog.Info("server will listen and serve", "addr", fmt.Sprint("http://", addr))
		err := httpServer.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			errs <- nil
		} else {
			slog.Warn("http server's 'listen and serve' failed", slog.Any("error", err))
			errs <- err
		}
	}()

	go func() {
		<-ctx.Done()
		err := httpServer.Shutdown(ctx)
		errs <- err
	}()

	return <-errs
}
