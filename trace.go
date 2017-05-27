package trace

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/micro/go-micro/metadata"
	"github.com/micro/go-micro/server"
	opentracing "github.com/opentracing/opentracing-go"
	jaeger "github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	jaegerlog "github.com/uber/jaeger-client-go/log"
	"github.com/uber/jaeger-client-go/rpcmetrics"
	"github.com/uber/jaeger-lib/metrics"
)

//Initialization initialise the Jeager tracer (compatible with opentracing.Tracer interface)
func Initialization(name string) (io.Closer, error) {
	var traceAddress string
	traceAddress, ok := os.LookupEnv("JAEGER_ADDRESS")
	if !ok {
		traceAddress = "127.0.0.1"
	}

	// Sample configuration for testing. Use constant sampling to sample every trace
	// and enable LogSpan to log every span via configured Logger.
	cfg := jaegercfg.Configuration{
		Sampler: &jaegercfg.SamplerConfig{
			Type:  jaeger.SamplerTypeConst,
			Param: 1,
		},
		Reporter: &jaegercfg.ReporterConfig{
			LogSpans:           true,
			LocalAgentHostPort: traceAddress + ":5775",
		},
	}

	// Example logger and metrics factory. Use github.com/uber/jaeger-client-go/log
	// and github.com/uber/jaeger-lib/metrics respectively to bind to real logging and metrics
	// frameworks.
	jLogger := jaegerlog.StdLogger
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

//MicroSubscriber is usefull to track those calls made from subscription
func MicroSubscriber() server.SubscriberWrapper {
	return func(next server.SubscriberFunc) server.SubscriberFunc {
		return func(ctx context.Context, msg server.Publication) error {
			md, _ := metadata.FromContext(ctx)
			var sp opentracing.Span
			tr := opentracing.GlobalTracer()
			name := "SUBS " + msg.Topic() + " " + fmt.Sprint(msg.Message())
			wireContext, err := tr.Extract(opentracing.TextMap, opentracing.TextMapCarrier(md))
			if err != nil {
				sp = tr.StartSpan(name)
			} else {
				sp = tr.StartSpan(name, opentracing.FollowsFrom(wireContext))
			}
			if err := sp.Tracer().Inject(sp.Context(), opentracing.TextMap, opentracing.TextMapCarrier(md)); err != nil {
				return err
			}
			ctx = metadata.NewContext(ctx, md)
			err = next(ctx, msg)
			if err != nil {
				return err
			}

			//this happens after the subsciber handler has finish
			sp.Finish()
			return nil
		}
	}
}
