package trace

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/labstack/echo"
	"github.com/opentracing-contrib/go-stdlib/nethttp"
	opentracing "github.com/opentracing/opentracing-go"
)

//TagSpan is cool for fast tagging a span
func TagSpan(ctx context.Context, key, value string) error {
	span := opentracing.SpanFromContext(ctx)
	log.Println("status")
	if span == nil {
		return errors.New("the span is nil")
	}
	span.SetTag(key, value)
	return nil
}

//TagEcho is the implementation specific for the labstack/echo framework
func TagEcho(c echo.Context, key, value string) error {
	return TagSpan(c.Request().Context(), key, value)
}

//WithEcho is the implementation specific for the labstack/echo framework
func WithEcho(c echo.Context, method, url string) (int, string) {
	ctx := c.Request().Context()
	return TracedCall(ctx, method, url)
}

//WithContext will be the main function to call with a context (this can be used across frameworks)
func WithContext(ctx context.Context, method, url string) (int, string) {
	return TracedCall(ctx, method, url)
}

//TracedCall is an API call with tracing enabled
func TracedCall(ctx context.Context, method, url string) (int, string) {
	tracer := opentracing.SpanFromContext(ctx).Tracer()
	req, _ := http.NewRequest(method, url, nil)
	req = req.WithContext(ctx)
	req, ps := nethttp.TraceRequest(tracer, req)
	defer ps.Finish()
	client := &http.Client{Transport: &nethttp.Transport{}}
	rsp, err := client.Do(req)
	if err != nil {
		return 500, "something went wrong calling the status"
	}
	defer rsp.Body.Close()
	data, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		return 500, "something went wrong parsing the body"
	}
	return rsp.StatusCode, string(data)
}

//PostForm is very similar to the net/http implementation apart from the context
func PostForm(ctx context.Context, url string, u url.Values) (int, string) {
	return Call(ctx, http.MethodPost, url, strings.NewReader(u.Encode()))
}

//Call is an API call with tracing enabled
func Call(ctx context.Context, method, url string, body io.Reader) (int, string) {
	tracer := opentracing.SpanFromContext(ctx).Tracer()
	req, _ := http.NewRequest(method, url, body)
	req = req.WithContext(ctx)
	req, ps := nethttp.TraceRequest(tracer, req)
	defer ps.Finish()
	client := &http.Client{Transport: &nethttp.Transport{}}
	rsp, err := client.Do(req)
	if err != nil {
		return 500, "something went wrong calling the status"
	}
	defer rsp.Body.Close()
	data, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		return 500, "something went wrong parsing the body"
	}
	return rsp.StatusCode, string(data)
}

// //TracedCall is an API call with tracing enabled
// func TracedCall(ctx context.Context, method, url string) (int, string) {
// 	tracer := opentracing.SpanFromContext(ctx).Tracer()
// 	req, _ := http.NewRequest(method, url, nil)
// 	req = req.WithContext(ctx)
// 	req, ps := nethttp.TraceRequest(tracer, req)
// 	defer ps.Finish()
// 	client := &http.Client{Transport: &nethttp.Transport{}}
// 	rsp, err := client.Do(req)
// 	if err != nil {
// 		return 500, "something went wrong calling the status"
// 	}
// 	defer rsp.Body.Close()
// 	data, err := ioutil.ReadAll(rsp.Body)
// 	if err != nil {
// 		return 500, "something went wrong parsing the body"
// 	}
// 	return rsp.StatusCode, string(data)
// }

//WithoutContext is used in those situations where a context with a span inside is not given
func WithoutContext(method, url string) (int, string) {
	span := opentracing.GlobalTracer().StartSpan(url)
	defer span.Finish()
	ctx := opentracing.ContextWithSpan(context.Background(), span)
	return TracedCall(ctx, method, url)
}

//FromContext is a function that start the tracing of a sequential execution
func FromContext(ctx context.Context, name string) func() {
	span, _ := opentracing.StartSpanFromContext(ctx, name)
	return span.Finish
}

//FromEchoContext is a function that start the tracing of a sequential execution inside the echo framework
func FromEchoContext(ctx echo.Context, name string) func() {
	span, _ := opentracing.StartSpanFromContext(ctx.Request().Context(), name)
	return span.Finish
}
