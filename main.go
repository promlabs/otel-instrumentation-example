package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/metric"
	sdk_metric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.25.0"
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

	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(semconv.SchemaURL,
			semconv.ServiceName("my-demo-service"),  // Becomes the "job" label.
			semconv.ServiceInstanceID("instance-a"), // Becomes the "instance" label.
			semconv.ServiceVersion("0.1.0"),
		),
	)
	if err != nil {
		log.Fatalf("failed to create resource: %v", err)
	}

	// Create a new MeterProvider with a reader that sends metrics to the OTLP exporter every 5 seconds.
	meterProvider := sdk_metric.NewMeterProvider(
		sdk_metric.WithResource(res),
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
	// Counter.
	counter, err := meter.Int64Counter("demo.handled_items")
	if err != nil {
		log.Fatalf("Failed to create Counter: %v", err)
	}
	counter.Add(ctx, 1)  // Increment the counter by 1.
	counter.Add(ctx, 23) // Increment the counter by 23.

	// UpDownCounter.
	upDownCounter, err := meter.Int64UpDownCounter("demo.queue_length")
	if err != nil {
		log.Fatalf("Failed to create UpDownCounter: %v", err)
	}
	upDownCounter.Add(ctx, 5)  // Increment by 5.
	upDownCounter.Add(ctx, -2) // Decrement by 2.

	// Gauge.
	gauge, err := meter.Int64Gauge("demo.start_time")
	if err != nil {
		log.Fatalf("Failed to create Gauge: %v", err)
	}
	gauge.Record(ctx, time.Now().Unix()) // Set to the current Unix timestamp in seconds.

	// Histogram.
	histogram, err := meter.Float64Histogram(
		"demo.request.duration",
		metric.WithDescription("The distribution of demo request durations."),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.05, 0.1, 0.25, 0.5, 1, 2, 5),
	)
	if err != nil {
		log.Fatalf("Failed to create Histogram: %v", err)
	}
	histogram.Record(ctx, 0.023) // Record a request that took 0.023 seconds.
	histogram.Record(ctx, 1.632) // Record a request that took 1.632 seconds.
	histogram.Record(ctx, 0.345) // Record a request that took 0.345 seconds.
	histogram.Record(ctx, 0.123) // Record a request that took 0.123 seconds.

	// Asynchronous Gauge.
	_, err = meter.Int64ObservableGauge(
		"demo.observed_value",
		metric.WithInt64Callback(func(ctx context.Context, result metric.Int64Observer) error {
			result.Observe(23) // Return 23 as the current value of the gauge.
			return nil
		}),
	)
	if err != nil {
		log.Fatalf("Failed to create ObservableGauge: %v", err)
	}

	// Request counter that is partitioned by the HTTP method and path.
	partitionedCounter, err := meter.Int64Counter("demo.request.count",
		metric.WithDescription("The number of requests handled by the server."),
	)
	if err != nil {
		log.Fatalf("Failed to create Counter: %v", err)
	}

	// Record a few requests for different method and path combinations.
	partitionedCounter.Add(ctx, 58, metric.WithAttributes(attribute.String("demo.method", "GET"), attribute.String("demo.path", "/items")))
	partitionedCounter.Add(ctx, 81, metric.WithAttributes(attribute.String("demo.method", "POST"), attribute.String("demo.path", "/items")))
	partitionedCounter.Add(ctx, 33, metric.WithAttributes(attribute.String("demo.method", "GET"), attribute.String("demo.path", "/users")))
	partitionedCounter.Add(ctx, 97, metric.WithAttributes(attribute.String("demo.method", "POST"), attribute.String("demo.path", "/users")))
}
