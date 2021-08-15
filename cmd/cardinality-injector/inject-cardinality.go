package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var addr = flag.String("listen-address", ":8888", "The address to listen on for HTTP requests.")

func main() {
	flag.Parse()
	http.Handle("/metrics", promhttp.Handler())

	ticker := time.NewTicker(50 * time.Millisecond)
	go func() {
		for {
			select {
			case t := <-ticker.C:

				c := prometheus.NewCounter(prometheus.CounterOpts{
					Name: "high_cardinality_counter",
					ConstLabels: prometheus.Labels{
						"bad_label": t.Local().String(),
					},
				})
				prometheus.MustRegister(c)
				c.Inc()
			}
		}
	}()

	log.Fatal(http.ListenAndServe(*addr, nil))
}
