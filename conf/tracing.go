package conf

import (
	"fmt"

	"github.com/opentracing/opentracing-go"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/opentracer"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

type TracingConfig struct {
	Enabled     bool              `toml:"enabled"`
	Host        string            `toml:"host"`
	Port        string            `toml:"port"`
	ServiceName string            `toml:"service_name"`
	Tags        map[string]string `toml:"tags"`
}

func (tc *TracingConfig) tracingAddr() string {
	return fmt.Sprintf("%s:%s", tc.Host, tc.Port)
}

func ConfigureTracing(tc *TracingConfig) {
	var t opentracing.Tracer = opentracing.NoopTracer{}
	if tc.Enabled {
		tracerOps := []tracer.StartOption{
			tracer.WithServiceName(tc.ServiceName),
			tracer.WithAgentAddr(tc.tracingAddr()),
		}

		for k, v := range tc.Tags {
			tracerOps = append(tracerOps, tracer.WithGlobalTag(k, v))
		}

		t = opentracer.New(tracerOps...)
	}
	opentracing.SetGlobalTracer(t)
}
