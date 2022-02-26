package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"time"
)

var (
	scrapeStatus = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "scrape_status",
			Help: "Status of the scrapes.",
		},
	)
	scrapeDuration = prometheus.NewSummary(
		prometheus.SummaryOpts{
			Name: "scrape_duration_seconds",
			Help: "Duration of the scrapes.",
		},
	)
)

func init() {
	prometheus.MustRegister(scrapeStatus, scrapeDuration)
}

func Scrape(f func() error) error {
	start := time.Now()
	var err error
	err = f()
	if err != nil {
		scrapeStatus.Set(0)
	} else {
		scrapeStatus.Set(1)
	}
	scrapeDuration.Observe(time.Since(start).Seconds())

	return err
}
