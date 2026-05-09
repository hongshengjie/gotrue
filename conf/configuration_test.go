package conf

import (
	"os"
	"testing"

	"github.com/opentracing/opentracing-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeTempTOML(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp("", "gotrue-test-*.toml")
	require.NoError(t, err)
	t.Cleanup(func() { os.Remove(f.Name()) })
	_, err = f.WriteString(content)
	require.NoError(t, err)
	f.Close()
	return f.Name()
}

func TestGlobal(t *testing.T) {
	path := writeTempTOML(t, `
operator_token = "token"

[api]
request_id_header = "X-Request-ID"

[db]
driver = "mysql"
url = "fake"
`)
	gc, err := LoadGlobal(path)
	require.NoError(t, err)
	require.NotNil(t, gc)
	assert.Equal(t, "X-Request-ID", gc.API.RequestIDHeader)
}

func TestTracing(t *testing.T) {
	path := writeTempTOML(t, `
operator_token = "token"

[db]
driver = "mysql"
url = "fake"

[tracing]
enabled = false
service_name = "identity"
port = "8126"
host = "127.0.0.1"

[tracing.tags]
tag1 = "value1"
tag2 = "value2"
`)
	gc, err := LoadGlobal(path)
	require.NoError(t, err)

	tc := opentracing.GlobalTracer()

	assert.Equal(t, opentracing.NoopTracer{}, tc)
	assert.Equal(t, false, gc.Tracing.Enabled)
	assert.Equal(t, "identity", gc.Tracing.ServiceName)
	assert.Equal(t, "8126", gc.Tracing.Port)
	assert.Equal(t, "127.0.0.1", gc.Tracing.Host)
	assert.Equal(t, map[string]string{"tag1": "value1", "tag2": "value2"}, gc.Tracing.Tags)
}
