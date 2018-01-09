package trace

import (
	"io"
	"log"

	jaeger "github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	jaegerlog "github.com/uber/jaeger-client-go/log"
	"github.com/uber/jaeger-client-go/rpcmetrics"
	"github.com/uber/jaeger-lib/metrics"
)

//Initialization initialise the Jeager tracer (compatible with opentracing.Tracer interface)
func Initialization(name, url string) (io.Closer, error) {
	traceAddress := "127.0.0.1:5775"
	if url != "" {
		traceAddress = url
	}

	// Sample configuration for testing. Use constant sampling to sample every trace
	// and enable LogSpan to log every span via configured Logger.
	cfg := jaegercfg.Configuration{
		Sampler: &jaegercfg.SamplerConfig{
			Type:  jaeger.SamplerTypeRateLimiting,
			Param: 1,
		},
		Reporter: &jaegercfg.ReporterConfig{
			LogSpans:           true,
			LocalAgentHostPort: traceAddress,
		},
	}

	// Example logger and metrics factory. Use github.com/uber/jaeger-client-go/log
	// and github.com/uber/jaeger-lib/metrics respectively to bind to real logging and metrics
	// frameworks.
	jLogger := jaegerlog.NullLogger
	jMetricsFactory := metrics.NullFactory
	// metricsFactory := xkit.Wrap("", expvar.NewFactory(10))
	// Initialize tracer with a logger and a metrics factory
	closer, err := cfg.InitGlobalTracer(
		name,
		jaegercfg.Logger(jLogger),
		// jaegercfg.Metrics(jMetricsFactory),
		jaegercfg.Observer(rpcmetrics.NewObserver(jMetricsFactory, rpcmetrics.DefaultNameNormalizer)),
	)
	if err != nil {
		log.Printf("Could not initialize jaeger tracer: %s", err.Error())
		return nil, err
	}
	return closer, nil
}
