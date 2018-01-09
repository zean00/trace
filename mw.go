package trace

import (
	"context"
	"net/http"

	"github.com/labstack/echo"

	opentracing "github.com/opentracing/opentracing-go"
)

type metaKey struct{}

// Metadata is our way of representing request headers internally.
// They're used at the RPC level and translate back and forth
// from Transport headers.
type Metadata map[string]string

func MetaFromContext(ctx context.Context) (Metadata, bool) {
	md, ok := ctx.Value(metaKey{}).(Metadata)
	return md, ok
}

func MetaNewContext(ctx context.Context, md Metadata) context.Context {
	return context.WithValue(ctx, metaKey{}, md)
}

func StartSpanFromContext(ctx context.Context, name string) (opentracing.Span, context.Context) {
	md, _ := MetaFromContext(ctx)
	var sp opentracing.Span
	tr := opentracing.GlobalTracer()
	wireContext, err := tr.Extract(opentracing.TextMap, opentracing.TextMapCarrier(md))
	if err != nil {
		sp = tr.StartSpan(name)
	} else {
		sp = tr.StartSpan(name, opentracing.ChildOf(wireContext))
	}
	if err := sp.Tracer().Inject(sp.Context(), opentracing.TextMap, opentracing.TextMapCarrier(md)); err != nil {
		return nil, ctx
	}
	ctx = MetaNewContext(ctx, md)
	return sp, ctx
}

func StartFollowFromContext(ctx context.Context, name string) (opentracing.Span, context.Context) {
	md, _ := MetaFromContext(ctx)
	var sp opentracing.Span
	tr := opentracing.GlobalTracer()
	wireContext, err := tr.Extract(opentracing.TextMap, opentracing.TextMapCarrier(md))
	if err != nil {
		sp = tr.StartSpan(name)
	} else {
		sp = tr.StartSpan(name, opentracing.FollowsFrom(wireContext))
	}
	if err := sp.Tracer().Inject(sp.Context(), opentracing.TextMap, opentracing.TextMapCarrier(md)); err != nil {
		return nil, ctx
	}
	ctx = MetaNewContext(ctx, md)
	return sp, ctx
}

func NewEcho() echo.MiddlewareFunc {
	inside := func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			name := "HTTP " + r.Method + " " + r.URL.Path
			md, ok := MetaFromContext(ctx)
			if !ok {
				md = make(map[string]string)
			}
			var sp opentracing.Span
			tr := opentracing.GlobalTracer()
			wireContext, err := tr.Extract(opentracing.TextMap, opentracing.TextMapCarrier(md))
			if err != nil {
				sp = tr.StartSpan(name)
			} else {
				sp = tr.StartSpan(name, opentracing.ChildOf(wireContext))
			}
			err = sp.Tracer().Inject(sp.Context(), opentracing.TextMap, opentracing.TextMapCarrier(md))
			if err != nil {
				return
			}
			ctx = MetaNewContext(ctx, md)
			//put ctx inside r
			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
			sp.Finish()
		}
		return http.HandlerFunc(fn)
	}
	return echo.WrapMiddleware(inside)
}
