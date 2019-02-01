package trace

import (
	"context"
	"errors"
	"fmt"
	"os"

	"contrib.go.opencensus.io/exporter/ocagent"
	"go.opencensus.io/exporter/jaeger"
	"go.opencensus.io/trace"
)

const (
	defaultJaegerAgentEndpoint     = "localhost:6831"
	defaultJaegerCollectorEndpoint = "http://localhost:14268/api/traces"
)

func StartSpan(ctx context.Context, name string) (context.Context, *trace.Span) {
	return trace.StartSpan(ctx, name)
}

func StartSpanWithTags(ctx context.Context, name string, tags map[string]string) (context.Context, *trace.Span) {
	var attributes []trace.Attribute

	for key, value := range tags {
		attributes = append(attributes, trace.StringAttribute(key, value))
	}

	ctx, span := trace.StartSpan(ctx, name)
	span.AddAttributes(attributes...)

	return ctx, span
}

// SetupTracing setup default tracing
func SetupTracing(serviceName string, exporters ...string) error {
	trace.ApplyConfig(trace.Config{DefaultSampler: trace.AlwaysSample()})

	for _, exporter := range exporters {
		switch exporter {
		case "jaeger":
			if err := RegisterJaegerExporter(serviceName); err != nil {
				return err
			}
		default:
			return fmt.Errorf("invalid exporter: %s", exporter)
		}
	}

	return nil
}

// RegisterAppInsightsExportor registers the app insights exporter
func RegisterAppInsightsExportor(serviceName, agentEndpoint string) error {
	exporter, err := ocagent.NewExporter(ocagent.WithInsecure(), ocagent.WithServiceName(serviceName), ocagent.WithAddress(agentEndpoint))
	if err != nil {
		return err
	}

	trace.RegisterExporter(exporter)
	return nil
}

// RegisterJaegerExporter registers a new jaeger exporter.
func RegisterJaegerExporter(serviceName string) error {
	jOpts := jaeger.Options{
		CollectorEndpoint: os.Getenv("JAEGER_COLLECTION_ENDPOINT"),
		AgentEndpoint:     os.Getenv("JAEGER_AGENT_ENDPOINT"),
		Username:          os.Getenv("JAEGER_USER"),
		Password:          os.Getenv("JAEGER_PASSWORD"),
		ServiceName:       serviceName,
	}

	if jOpts.CollectorEndpoint == "" && jOpts.AgentEndpoint == "" {
		return errors.New("Must specify either JAEGER_COLLECTION_ENDPOINT or JAEGER_AGENT_ENDPOINT")
	}

	exporter, err := jaeger.NewExporter(jOpts)
	if err != nil {
		return err
	}

	trace.RegisterExporter(exporter)
	return nil
}

func addStdoutExporter() {
	trace.RegisterExporter(new(stdoutExporter))
}

type stdoutExporter struct{}

func (ce *stdoutExporter) ExportSpan(sd *trace.SpanData) {
	fmt.Printf("Name: %s\nTraceID: %x\nSpanID: %x\nParentSpanID: %x\nStartTime: %s\nEndTime: %s\nAnnotations: %+v\n\n",
		sd.Name, sd.TraceID, sd.SpanID, sd.ParentSpanID, sd.StartTime, sd.EndTime, sd.Annotations)
}
