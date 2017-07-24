package trace

import (
	"context"
	"fmt"

	"github.com/micro/go-micro/metadata"

	"github.com/micro/go-micro/server"
	opentracing "github.com/opentracing/opentracing-go"
)

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
