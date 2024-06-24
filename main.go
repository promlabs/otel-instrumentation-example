package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/metric"
	sdk_metric "go.opentelemetry.io/otel/sdk/metric"
)

func main() {
	// Handle SIGINT (CTRL+C) gracefully.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)

	// Create an OTLP metric exporter that sends all metrics to the local Prometheus server.
	otlpMetricExporter, err := otlpmetrichttp.New(ctx, otlpmetrichttp.WithEndpointURL("http://localhost:9090/api/v1/otlp/v1/metrics"))
	if err != nil {
		log.Fatalf("Failed to create OTLP metric exporter: %v", err)
	}

	// OPTIONAL: Create a stdout exporter that periodically logs the metrics to stdout.
	stdoutMetricExporter, err := stdoutmetric.New(stdoutmetric.WithPrettyPrint())
	if err != nil {
		log.Fatalf("Failed to create stdout metric exporter: %v", err)
	}

	// Create a new MeterProvider with a reader that sends metrics to the OTLP exporter every 5 seconds.
	meterProvider := sdk_metric.NewMeterProvider(
		// Send metrics via OTLP.
		sdk_metric.WithReader(sdk_metric.NewPeriodicReader(otlpMetricExporter, sdk_metric.WithInterval(5*time.Second))),
		// OPTIONAL: Log metrics to stdout.
		sdk_metric.WithReader(sdk_metric.NewPeriodicReader(stdoutMetricExporter, sdk_metric.WithInterval(5*time.Second))),
	)
	// Ensure the MeterProvider is flushed and shut down properly when terminating the program.
	defer meterProvider.Shutdown(context.Background())

	// Set the global MeterProvider to the newly created MeterProvider.
	// This enables calls like otel.Meter() anywhere in the application rather than having to pass the MeterProvider around.
	otel.SetMeterProvider(meterProvider)

	// Create a new Meter.
	meter := otel.Meter("otel-instrumentation-example")

	// Create and record some example metrics.
	createAndRecordMetrics(ctx, meter)

	// Wait for interruption / first CTRL+C.
	<-ctx.Done()
	log.Println("Shutting down...")
	// Stop receiving further signal notifications as soon as possible.
	stop()
}

func createAndRecordMetrics(ctx context.Context, meter metric.Meter) {
	// This is where we'll add our metrics later.
}
