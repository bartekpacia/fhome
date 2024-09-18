package main

import (
	"embed"
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

// Stupid webserver to display some state about my smart devices.
func webserver(client *api.Client, homeConfig *api.Config) {
	http.HandleFunc("GET /index", func(w http.ResponseWriter, r *http.Request) {
		slog.Info("got request", slog.String("method", r.Method), slog.String("path", r.URL.Path))

		data := map[string]interface{}{
			"Email":  config.Email,
			"Panels": homeConfig.Panels,
			"Cells":  homeConfig.Cells(),
		}

		tmpl.ExecuteTemplate(w, "index.html.tmpl", data)
	})

	// Hacky workaround for myself to open my gate from my phone.
	http.HandleFunc("GET /gate", func(w http.ResponseWriter, r *http.Request) {
		var result string
		err := client.SendEvent(260, api.ValueToggle)
		if err != nil {
			result = fmt.Sprintf("Failed to send event: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
		}

		if result != "" {
			log.Print(result)
			fmt.Fprint(w, result)
		}
	})

	http.Handle("GET /public", http.StripPrefix("/public/", http.FileServer(http.FS(assets))))

	addr := fmt.Sprint("0.0.0.0:", port)
	slog.Info("server will listen and serve", "addr", fmt.Sprint("http://", addr))
	log.Println("http server is listening and serving on port 9001")
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		panic(err)
	}
}
