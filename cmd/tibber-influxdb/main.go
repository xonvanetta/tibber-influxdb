package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	influxdb "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
	"github.com/koding/multiconfig"
	"github.com/machinebox/graphql"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/xonvanetta/shutdown/pkg/shutdown"
	"github.com/xonvanetta/tibber-influxdb/pkg/metrics"
)

type Config struct {
	Port     int
	LogLevel string
	Interval time.Duration
	Tibber   TibberConfig
	InfluxDB InfluxDBConfig
}

type InfluxDBConfig struct {
	Token  string
	Url    string
	Org    string
	Bucket string
}

func main() {
	config := &Config{
		Port:     8080,
		LogLevel: "info",
		Interval: time.Hour,
		Tibber: TibberConfig{
			Endpoint: "https://api.tibber.com/v1-beta/gql",
		},
	}
	multiconfig.MustLoad(config)
	l, err := logrus.ParseLevel(config.LogLevel)
	if err != nil {
		logrus.Fatalf("failed to parse LogLevel: %s", err)
	}
	logrus.SetLevel(l)
	logrus.SetFormatter(&logrus.JSONFormatter{})

	ctx := shutdown.Context()

	graphqlClient := graphql.NewClient(config.Tibber.Endpoint)
	if l == logrus.DebugLevel {
		graphqlClient.Log = func(s string) { logrus.Debug(s) }
	}

	influxdbClient := influxdb.NewClient(config.InfluxDB.Url, config.InfluxDB.Token)
	ticker := time.NewTicker(config.Interval)
	run := func() error {
		return run(ctx, config, graphqlClient, influxdbClient)
	}

	go func() {
		err := metrics.Scrape(run)
		if err != nil {
			logrus.Errorf("failed to run: %s", err)
		}
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
			}
			err := metrics.Scrape(run)
			if err != nil {
				logrus.Errorf("failed to run: %s", err)
			}
		}
	}()

	go func() {
		err := http.ListenAndServe(fmt.Sprintf(":%d", config.Port), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/health":
				w.WriteHeader(http.StatusOK)
			case "/metrics":
				promhttp.Handler().ServeHTTP(w, r)
			}
		}))
		if err != nil && err != http.ErrServerClosed {
			logrus.Fatalf("failed to start http server: %s", err)
		}
	}()

	<-ctx.Done()
}

func run(ctx context.Context, config *Config, graphqlClient *graphql.Client, influxdbClient influxdb.Client) error {
	response, err := scrape(ctx, config.Tibber.Token, graphqlClient)
	if err != nil {
		return fmt.Errorf("failed to scrape tibber: %w", err)
	}

	//Safe for reuse
	writer := influxdbClient.WriteAPI(config.InfluxDB.Org, config.InfluxDB.Bucket)
	defer writer.Flush()
	updateInfluxdb(response, writer)

	return nil
}

func updateInfluxdb(response response, writeAPI api.WriteAPI) {
	for _, home := range response.Viewer.Homes {
		for _, node := range home.Consumption.Nodes {
			if node.ConsumptionUnit != "kWh" {
				logrus.Errorf("skipping consumption as unit is wrong")
				continue
			}
			writeAPI.WritePoint(write.NewPointWithMeasurement("consumption_nodes_wh").
				SetTime(node.From).
				AddTag("home_id", home.ID).
				AddTag("home_timezone", home.TimeZone).
				AddTag("home_address_address1", home.Address.Address1).
				AddTag("home_address_address2", home.Address.Address2).
				AddTag("home_address_address3", home.Address.Address3).
				AddTag("home_address_city", home.Address.City).
				AddTag("home_address_postal_code", home.Address.PostalCode).
				AddTag("home_address_country", home.Address.Country).
				AddTag("home_address_latitude", home.Address.Latitude).
				AddTag("home_address_longitude", home.Address.Longitude).
				AddTag("currency", node.Currency).
				AddField("cost", node.Cost).
				AddField("unit_price", node.UnitPrice).
				AddField("unit_price_vat", node.UnitPriceVAT).
				AddField("consumption", node.Consumption*1000))
		}

		currency := home.CurrentSubscription.PriceRating.Hourly.Currency
		for _, entry := range home.CurrentSubscription.PriceRating.Hourly.Entries {
			writeAPI.WritePoint(write.NewPointWithMeasurement("price").
				SetTime(entry.Time).
				AddTag("home_id", home.ID).
				AddTag("home_timezone", home.TimeZone).
				AddTag("home_address_address1", home.Address.Address1).
				AddTag("home_address_address2", home.Address.Address2).
				AddTag("home_address_address3", home.Address.Address3).
				AddTag("home_address_city", home.Address.City).
				AddTag("home_address_postal_code", home.Address.PostalCode).
				AddTag("home_address_country", home.Address.Country).
				AddTag("home_address_latitude", home.Address.Latitude).
				AddTag("home_address_longitude", home.Address.Longitude).
				AddTag("currency", currency).
				AddTag("level", entry.Level).
				AddField("difference", entry.Difference).
				AddField("tax", entry.Tax).
				AddField("energy", entry.Energy).
				AddField("total", entry.Total))
		}
	}
}
