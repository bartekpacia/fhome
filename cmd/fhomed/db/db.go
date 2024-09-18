package db

import (
	"context"
	"log/slog"
	"os"

	"github.com/bartekpacia/fhome/api"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
)

func DBListener(client *api.Client, influxClient influxdb2.Client) {
	// Here we listen to events from the database and send them to F&Home.

	influxURL := MustGetenv("INFLUXDB_URL")
	influxToken := MustGetenv("INFLUXDB_TOKEN")
	influxBucket := MustGetenv("INFLUXDB_BUCKET")
	influxOrg := MustGetenv("INFLUXDB_ORG")

	_, err := influxClient.Health(context.TODO())
	if err != nil {
		slog.Error("error performing healthcheck on influx database", slog.Any("error", err))
		os.Exit(1)
	}
	slog.Info("connected to Influx database", "url", influxURL)

	influxClient.QueryAPI(influxOrg).Query(ctx)
}
