package tracing

import (
	"os"
	"strconv"
	"strings"

	"contrib.go.opencensus.io/exporter/jaeger"
	"go.opencensus.io/trace"
	"go.uber.org/zap"
)

func RegisterTraceExporter(logger *zap.Logger, collectorEndpoint, serviceName string) error {
	if len(collectorEndpoint) == 0 {
		logger.Info("skipping trace exporter registration")
		return nil
	}

	exporter, err := jaeger.NewExporter(jaeger.Options{
		CollectorEndpoint: collectorEndpoint,
		Process: jaeger.Process{
			ServiceName: serviceName,
			Tags: []jaeger.Tag{
				jaeger.BoolTag("fission", true),
			},
		},
	})
	if err != nil {
		return err
	}

	trace.RegisterExporter(exporter)

	if strings.EqualFold(serviceName, "Fission-Fetcher") {
		trace.ApplyConfig(trace.Config{DefaultSampler: trace.AlwaysSample()})
	} else {
		samplingRate, err := strconv.ParseFloat(os.Getenv("TRACING_SAMPLING_RATE"), 32)
		if err != nil {
			return err
		}
		trace.ApplyConfig(trace.Config{DefaultSampler: trace.ProbabilitySampler(samplingRate)})
	}

	return nil
}
