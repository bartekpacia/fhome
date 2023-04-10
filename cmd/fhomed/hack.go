package main

import (
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/bartekpacia/fhome/api"
	"golang.org/x/exp/slog"
)

//go:embed assets/*
var assets embed.FS

//go:embed templates/*
var templates embed.FS

var tmpl = template.Must(template.ParseFS(templates, "templates/*"))

// Hacky workaround for myself to open my gate from my phone.
func serviceListener(client *api.Client) {
	http.HandleFunc("/gate", func(w http.ResponseWriter, r *http.Request) {
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

	err := http.ListenAndServe(":9000", nil)
	if err != nil {
		panic(err)
	}
}

// Stupid webserver to display some state about my smart devices.
func websiteListener(homeConfig *api.Config) {
	http.HandleFunc("/index", func(w http.ResponseWriter, r *http.Request) {
		slog.Info("got request", slog.String("method", r.Method), slog.String("path", r.URL.Path))

		data := map[string]interface{}{
			"Email":  config.Email,
			"Panels": homeConfig.Panels,
			"Cells":  homeConfig.Cells(),
		}

		tmpl.ExecuteTemplate(w, "index.html.tmpl", data)
	})

	http.Handle("/public/", http.StripPrefix("/public/", http.FileServer(http.FS(assets))))

	err := http.ListenAndServe(":9001", nil)
	if err != nil {
		panic(err)
	}
}
