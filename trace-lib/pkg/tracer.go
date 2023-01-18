package pkg

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
)

const serviceName = "cronos-tracing"
var JaegerCollectorURL = "http://localhost:14278/api/traces"

type BlockInfo struct {
	Height int64
	Round int32
}

var CurrentBlockInfo *BlockInfo

func (b *BlockInfo) ShouldRefreshInfo(height int64, round int32) (bool) {
	if height > b.Height {
		return true
	}

	return false
}

type TracerN struct {
	trace.Tracer
	ctx context.Context
	Shutdown func()
}

var GlobalTracer *TracerN
var EthermintTracer *TracerN
var CosmosTracer *TracerN

func NewEthermintTracer() (*TracerN) {
	aTracer, shutdown, err := InitTracer("ethermint", "instance")
	if err != nil {
		log.Fatal(err)
	}

	EthermintTracer = &TracerN{aTracer, GlobalTracer.ctx, shutdown}	

	return EthermintTracer
}

func NewCosmosTracer() (*TracerN) {
	aTracer, shutdown, err := InitTracer("cosmos-sdk", "instance")
	if err != nil {
		log.Fatal(err)
	}

	CosmosTracer = &TracerN{aTracer, GlobalTracer.ctx, shutdown}	

	return CosmosTracer
}

func NewTracer(service string, instance string) (*TracerN) {
	aTracer, shutdown, err := InitTracer(service, instance)
	if err != nil {
		log.Fatal(err)
	}

	GlobalTracer = &TracerN{aTracer, context.Background(), shutdown}

	return GlobalTracer
}

func (t *TracerN) StartSpan(name string) (trace.Span) {

	opts := []oteltrace.SpanStartOption{
		oteltrace.WithSpanKind(oteltrace.SpanKindServer),
	}

	ctx, span := t.Start(t.ctx, name, opts...)

	t.ctx = ctx;

	return span
}

func InitTracer(serviceName, instanceName string) (trace.Tracer, func(), error) {

	tp, err := newTracerProvider(serviceName, instanceName)
	if err != nil {
		return nil, nil, fmt.Errorf("couldn't initialize tracer provider: %w", err)
	}

	otel.SetTracerProvider(tp)

	// Cleanly shutdown and flush telemetry when the application exits.
	shutdown := func() {
		// Do not make the application hang when it is shutdown.
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		if err := tp.Shutdown(ctx); err != nil {
			log.Fatal(err)
		}
	}

	return tp.Tracer(serviceName), shutdown, err
}

func newTracerProvider(serviceName, instanceName string) (*tracesdk.TracerProvider, error) {
	// Create the Jaeger exporter
	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(JaegerCollectorURL)))
	if err != nil {
		return nil, err
	}
	return tracesdk.NewTracerProvider(
		tracesdk.WithSampler(tracesdk.AlwaysSample()),
		// Always be sure to batch in production.
		tracesdk.WithBatcher(exp),
		// Record information about this application in a Resource.
		tracesdk.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(serviceName),
			attribute.String("service-instance", instanceName),
		)),
	), nil
}


func GetTracer(t ... *TracerN) (*TracerN) {
	var tracer *TracerN
	if t != nil {
		tracer = t[0]
	}
	return tracer
}
