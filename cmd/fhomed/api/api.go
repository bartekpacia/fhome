// Package api implements a few simple HTTP endpoints for discovery and control of smart home devices.
package api

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
	"strconv"

	"github.com/bartekpacia/fhome/api"
)

//go:embed assets/*
var assets embed.FS

//go:embed templates/*
var templates embed.FS

var tmpl = template.Must(template.ParseFS(templates, "templates/*"))

func New(fhomeClient *api.Client, homeConfig *api.Config) *API {
	return &API{
		fhomeClient: fhomeClient,
		homeConfig:  homeConfig,
	}
}

type API struct {
	fhomeClient *api.Client
	homeConfig  *api.Config
}

func (a *API) Run(ctx context.Context, port int, apiPassphrase string) error {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /gate", a.gate)
	mux.HandleFunc("GET /api/index", a.index)
	mux.HandleFunc("GET /api/devices", a.getDevices)
	mux.HandleFunc("/api/devices/{id}", a.toggleDevice)
	mux.Handle("GET /public", http.StripPrefix("/public/", http.FileServer(http.FS(assets))))

	authMux := withPassphrase(mux, apiPassphrase)
	addr := fmt.Sprint("0.0.0.0:", port)
	httpServer := http.Server{Addr: addr, Handler: authMux}

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

// gate is a hacky workaround for myself to open my gate from my phone.
func (a *API) gate(w http.ResponseWriter, r *http.Request) {
	var result string
	err := a.fhomeClient.SendEvent(r.Context(), 260, api.ValueToggle)
	if err != nil {
		result = fmt.Sprintf("Failed to send event: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
	}

	if result != "" {
		log.Print(result)
		fmt.Fprint(w, result)
	}
}

func (a *API) index(w http.ResponseWriter, r *http.Request) {
	slog.Info("got request", slog.String("method", r.Method), slog.String("path", r.URL.Path))

	data := map[string]interface{}{
		"Panels": a.homeConfig.Panels,
		"Cells":  a.homeConfig.Cells(),
	}

	err := tmpl.ExecuteTemplate(w, "index.html.tmpl", data)
	if err != nil {
		slog.Error("failed to execute template", slog.Any("error", err))
	}
}

func (a *API) getDevices(w http.ResponseWriter, r *http.Request) {
	userConfig, err := a.fhomeClient.GetUserConfig(r.Context())
	if err != nil {
		http.Error(w, "failed to get user config"+err.Error(), http.StatusInternalServerError)
		return
	}

	response := make([]device, 0)
	for _, cell := range userConfig.Cells {
		response = append(response, device{
			Name: cell.Name,
			ID:   cell.ObjectID,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, "failed to encode user into json"+err.Error(), http.StatusInternalServerError)
		return
	}
}

func (a *API) toggleDevice(w http.ResponseWriter, r *http.Request) {
	objectID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = a.fhomeClient.SendEvent(r.Context(), int(objectID), api.ValueToggle)
	if err != nil {
		msg := fmt.Sprintf("failed to send event to object with id %d: %v\n", objectID, err)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}
}

func withPassphrase(next http.Handler, passphrase string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info("new request", "method", r.Method, "url", r.URL.String(), "remote_addr", r.RemoteAddr)

		if r.Header.Get("Authorization") != "Passphrase: "+passphrase {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

type device struct {
	Name string
	ID   int
}
