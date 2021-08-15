# Cardinanny

A cardinailty nanny for Promtheus.

Which will:

* Prevent high cardinality metrics from crashing prometheus (by dropping those labels)
* Remove old instances of the metric with high cardinality

TODO:

* Metric name cardinality as opposed to label cardinality
* Write notifications when cardinality is averted
* Better local setup

## Running locally

Open 3 terminals (sorry):

1. Run `docker compose up` to start prometheus
2. Run `go run cmd/cardinality-injector/inject-cardinality.go` to inject some cardinality into prometheus
3. Run `go run cmd/cardinanny/cardinanny.go` to start cardinanny