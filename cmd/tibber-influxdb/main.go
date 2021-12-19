package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/influxdata/influxdb-client-go/v2/api/write"

	"github.com/sirupsen/logrus"

	influxdb "github.com/influxdata/influxdb-client-go/v2"
	"github.com/koding/multiconfig"
	"github.com/machinebox/graphql"
	"github.com/xonvanetta/shutdown/pkg/shutdown"
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
		Port:     9501,
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
	go func() {
		err := run(ctx, config, graphqlClient, influxdbClient)
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
			err := run(ctx, config, graphqlClient, influxdbClient)
			if err != nil {
				logrus.Errorf("failed to run: %s", err)
			}
		}
	}()

	go func() {
		err := http.ListenAndServe(fmt.Sprintf(":%d", config.Port), http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			writer.WriteHeader(http.StatusOK)
		}))
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("failed to start http server: %s", err)
		}
	}()

	<-ctx.Done()
}

func run(ctx context.Context, config *Config, graphqlClient *graphql.Client, influxdbClient influxdb.Client) error {
	logrus.Debug("running scrape")
	response, err := scrape(ctx, config.Tibber.Token, graphqlClient)
	if err != nil {
		return fmt.Errorf("failed to scrape tibber: %w", err)
	}
	err = updateInfluxdb(response, influxdbClient.WriteAPI(config.InfluxDB.Org, config.InfluxDB.Bucket))
	if err != nil {
		return fmt.Errorf("failed to update influxdb: %w", err)
	}
	return nil
}

func updateInfluxdb(response *response, writeAPI api.WriteAPI) error {
	defer writeAPI.Flush()

	for _, home := range response.Viewer.Homes {
		writeAPI.WritePoint(write.NewPointWithMeasurement("home").
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
			AddField("online", 1))

		for _, node := range home.Consumption.Nodes {
			if node.ConsumptionUnit != "kWh" {
				logrus.Errorf("skipping consumption as unit is wrong")
				continue
			}
			writeAPI.WritePoint(write.NewPointWithMeasurement("consumption_nodes_wh").
				SetTime(node.From).
				AddTag("home_id", home.ID).
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
				AddTag("currency", currency).
				AddTag("level", entry.Level).
				AddField("difference", entry.Difference).
				AddField("tax", entry.Tax).
				AddField("energy", entry.Energy).
				AddField("total", entry.Total))
		}
	}
	return nil
}
