package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"dgx-spark-prometheus/collectors"
)

func main() {
	listenAddr := flag.String("listen", ":9835", "Address to listen on for Prometheus metrics")
	flag.Parse()

	// Resolve hostname for global "host" label
	hostname, err := os.Hostname()
	if err != nil {
		log.Fatalf("failed to get hostname: %v", err)
	}

	// Wrap the default registerer to add "host" label to all metrics
	registry := prometheus.WrapRegistererWith(
		prometheus.Labels{"host": hostname},
		prometheus.DefaultRegisterer,
	)

	// Register all collectors
	registry.MustRegister(collectors.NewCPUCollector())
	registry.MustRegister(collectors.NewGPUCollector())
	registry.MustRegister(collectors.NewMemoryCollector())
	registry.MustRegister(collectors.NewDiskCollector())
	registry.MustRegister(collectors.NewNetworkCollector())

	// Landing page
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<!DOCTYPE html>
<html>
<head><title>DGX Spark Prometheus Exporter</title></head>
<body>
<h1>DGX Spark Prometheus Exporter</h1>
<p><a href="/metrics">Metrics</a></p>
</body>
</html>`)
	})

	// Prometheus metrics endpoint
	http.Handle("/metrics", promhttp.Handler())

	log.Printf("DGX Spark Prometheus Exporter listening on %s", *listenAddr)
	log.Fatal(http.ListenAndServe(*listenAddr, nil))
}
