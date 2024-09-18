package db

import (
	"context"
	"log/slog"
	"os"

	"github.com/bartekpacia/fhome/api"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
)

func DBListener(client *api.Client) {
	// Here we listen to events from the database and send them to F&Home.

	influxURL := MustGetenv("INFLUXDB_URL")
	influxToken := MustGetenv("INFLUXDB_TOKEN")
	influxBucket := MustGetenv("INFLUXDB_BUCKET")
	influxOrg := MustGetenv("INFLUXDB_ORG")

	influxClient := influxdb2.NewClient(influxURL, influxToken)
	_, err := influxClient.Health(context.TODO())
	if err != nil {
		slog.Error("error connecting to influx database", slog.Any("error", err))
		os.Exit(1)
	}
	slog.Info("connected to Influx database", "url", influxURL)
}

func MustGetenv(varname string) string {
	value := os.Getenv(varname)
	if value == "" {
		slog.Error(varname + " env var is empty or not set")
		os.Exit(1)
	}
	return value
}
