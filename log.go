package otelgokit

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	gokitlog "github.com/go-kit/log"

	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
)

type config struct {
	provider  log.LoggerProvider
	version   string
	schemaURL string
}

func newConfig(options []Option) config {
	var c config
	for _, opt := range options {
		c = opt.apply(c)
	}

	if c.provider == nil {
		c.provider = global.GetLoggerProvider()
	}

	return c
}

func (c config) logger(name string) log.Logger {
	var opts []log.LoggerOption
	if c.version != "" {
		opts = append(opts, log.WithInstrumentationVersion(c.version))
	}
	if c.schemaURL != "" {
		opts = append(opts, log.WithSchemaURL(c.schemaURL))
	}
	return c.provider.Logger(name, opts...)
}

// Option configures a [Core].
type Option interface {
	apply(config) config
}

type optFunc func(config) config

func (f optFunc) apply(c config) config { return f(c) }

// WithVersion returns an [Option] that configures the version of the
// [log.Logger] used by a [gokitlog.Logger]. The version should be the version of the
// package that is being logged.
func WithVersion(version string) Option {
	return optFunc(func(c config) config {
		c.version = version
		return c
	})
}

// WithSchemaURL returns an [Option] that configures the semantic convention
// schema URL of the [log.Logger] used by a [gokitlog.Logger]. The schemaURL should be
// the schema URL for the semantic conventions used in log records.
func WithSchemaURL(schemaURL string) Option {
	return optFunc(func(c config) config {
		c.schemaURL = schemaURL
		return c
	})
}

// WithLoggerProvider returns an [Option] that configures [log.LoggerProvider]
// used by a [gokitlog.Logger] to create its [log.Logger].
//
// By default if this Option is not provided, the Handler will use the global
// LoggerProvider.
func WithLoggerProvider(provider log.LoggerProvider) Option {
	return optFunc(func(c config) config {
		c.provider = provider
		return c
	})
}

type OTelLogger struct {
	logger log.Logger
	ctx    context.Context
}

var _ gokitlog.Logger = (*OTelLogger)(nil)

// NewOTelLogger creates a new [gokitlog.Logger]
// The name should be the package import path that is being logged.
func NewLogger(name string, opts ...Option) gokitlog.Logger {
	cfg := newConfig(opts)

	return &OTelLogger{
		logger: cfg.logger(name),
		ctx:    context.Background(),
	}
}

// Log method maps key-value pair to OTel logs and emits them
func (o *OTelLogger) Log(keyvals ...interface{}) error {
	r := log.Record{}
	for i := 0; i < len(keyvals); i += 2 {
		k, v := keyvals[i], keyvals[i+1]

		// sets timestamp.  This expects key to match the keyword "ts"
		if k == "ts" {
			r.SetTimestamp(v.(time.Time))
			continue
		}

		// sets context
		if ctx, ok := v.(context.Context); ok {
			o.ctx = ctx
			continue
		}

		// sets severityLevel and severityText. This expects key to match the keyword "level"
		if k == "level" {
			r.SetSeverity(convertLevel(v))
			r.SetSeverityText(v.(string))
			continue
		}

		// sets attributes
		r.AddAttributes(log.KeyValue{Key: k.(string), Value: convertValue(v)})
	}

	o.logger.Emit(o.ctx, r)
	return nil
}

func convertLevel(level interface{}) log.Severity {
	s := level.(string)
	s = strings.ToLower(s)

	switch s {
	case "debug":
		return log.SeverityDebug
	case "info":
		return log.SeverityInfo
	case "warn":
		return log.SeverityWarn
	case "error":
		return log.SeverityError
	case "panic":
		return log.SeverityFatal1
	case "fatal":
		return log.SeverityFatal2
	default:
		return log.SeverityUndefined
	}

}

func convertValue(v interface{}) log.Value {
	switch v := v.(type) {
	case bool:
		return log.BoolValue(v)
	case []byte:
		return log.BytesValue(v)
	case float64:
		return log.Float64Value(v)
	case int:
		return log.IntValue(v)
	case int64:
		return log.Int64Value(v)
	case string:
		return log.StringValue(v)
	}

	t := reflect.TypeOf(v)
	if t == nil {
		return log.Value{}
	}
	val := reflect.ValueOf(v)
	switch t.Kind() {
	case reflect.Struct:
		return log.StringValue(fmt.Sprintf("%+v", v))
	case reflect.Slice, reflect.Array:
		items := make([]log.Value, 0, val.Len())
		for i := 0; i < val.Len(); i++ {
			items = append(items, convertValue(val.Index(i).Interface()))
		}
		return log.SliceValue(items...)
	case reflect.Map:
		kvs := make([]log.KeyValue, 0, val.Len())
		for _, k := range val.MapKeys() {
			var key string
			// If the key is a struct, use %+v to print the struct fields.
			if k.Kind() == reflect.Struct {
				key = fmt.Sprintf("%+v", k.Interface())
			} else {
				key = fmt.Sprintf("%v", k.Interface())
			}
			kvs = append(kvs, log.KeyValue{
				Key:   key,
				Value: convertValue(val.MapIndex(k).Interface()),
			})
		}
		return log.MapValue(kvs...)
	case reflect.Ptr, reflect.Interface:
		return convertValue(val.Elem().Interface())
	}

	return log.StringValue(fmt.Sprintf("unhandled attribute type: (%s) %+v", t, v))
}
