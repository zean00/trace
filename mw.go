package trace

import (
	"context"
	"net/http"

	"github.com/labstack/echo"
	"github.com/micro/go-micro/metadata"

	opentracing "github.com/opentracing/opentracing-go"
)

func StartSpanFromContext(ctx context.Context, name string) (opentracing.Span, context.Context) {
	md, _ := metadata.FromContext(ctx)
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
	ctx = metadata.NewContext(ctx, md)
	return sp, ctx
}

func StartFollowFromContext(ctx context.Context, name string) (opentracing.Span, context.Context) {
	md, _ := metadata.FromContext(ctx)
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
	ctx = metadata.NewContext(ctx, md)
	return sp, ctx
}

func NewEcho() echo.MiddlewareFunc {
	inside := func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			name := "HTTP " + r.Method + " " + r.URL.Path
			md, ok := metadata.FromContext(ctx)
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
			ctx = metadata.NewContext(ctx, md)
			//put ctx inside r
			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
			sp.Finish()
		}
		return http.HandlerFunc(fn)
	}
	return echo.WrapMiddleware(inside)
}
