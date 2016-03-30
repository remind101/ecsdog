package main

import (
	"flag"
	"log"

	"github.com/remind101/ecsdog"
)

func main() {
	var (
		cluster = flag.String("cluster", "", "Cluster to scrape metrics from")
		addr    = flag.String("statsd", "127.0.0.1:8125", "Statsd address")
	)
	flag.Parse()
	if err := ecsdog.Scrape(*cluster, *addr); err != nil {
		log.Fatal(err)
	}
}
