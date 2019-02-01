package httputil

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/samkreter/go-core/correlation"
)

const (
	testCorrelationID = "test-correlation-id"
)

func TestIncommingRequestLogging(t *testing.T) {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte(`OK`))
	})

	handler := SetUpHandler(testHandler, &HandlerConfig{
		CorrelationEnabled: true,
		LoggingEnabled:     true,
	})

	req, err := http.NewRequest("GET", "example.com", nil)
	require.NoError(t, err, "Should not get error when creating a request")
	AddStandardRequestHeaders(req)

	logrus.SetLevel(logrus.DebugLevel)
	testHook := logrustest.NewGlobal()

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, 2, len(testHook.Entries), "Should have correct number of outgoing logs")

	correlationID := getValueFromLog(testHook.Entries[0], "correlationID", t)
	assert.Equal(t, testCorrelationID, correlationID, "Should get correct correlationID")

	activityID := getValueFromLog(testHook.Entries[0], "activityID", t)
	assert.NotEmpty(t, activityID, "Should get a activityID")

	httpMethod := getValueFromLog(testHook.Entries[0], "httpMethod", t)
	assert.Equal(t, req.Method, httpMethod, "Should get correct httpMethod")

	targetURI := getValueFromLog(testHook.Entries[0], "targetUri", t)
	assert.Equal(t, req.URL.String(), targetURI, "Should get correct targetUri")

	hostName := getValueFromLog(testHook.Entries[0], "hostName", t)
	assert.Equal(t, req.Host, hostName, "Should get correct hostName")

	contentType := getValueFromLog(testHook.Entries[0], "contentType", t)
	assert.Equal(t, "application/json", contentType, "Should get correct contentType")

	httpStatusCode, ok := testHook.Entries[1].Data["httpStatusCode"].(int)
	require.True(t, ok, "Should have succesful cast.")

	assert.Equal(t, 200, httpStatusCode, "Should get correct statusCode")
}

func TestCorrelationMiddleware(t *testing.T) {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()

		assert.Equal(t, testCorrelationID, correlation.GetCorrelationID(ctx))
		assert.NotEmpty(t, correlation.GetActivityID(ctx))

		ctxHeaders := correlation.GetMetadataHeaders(ctx)

		assert.Equal(t, "test-user-agent", ctxHeaders.Get("User-Agent"), "Should get correct user agent")
		assert.Equal(t, "test-langauge", ctxHeaders.Get("Accept-Language"), "Should get correct langauge")

		w.Write([]byte(`OK`))
	})

	handler := SetUpHandler(testHandler, &HandlerConfig{
		CorrelationEnabled: true,
	})

	req, err := http.NewRequest("GET", "example.com", nil)
	require.NoError(t, err, "Should not get error when creating a request")

	AddStandardRequestHeaders(req)

	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
}

func AddStandardRequestHeaders(req *http.Request) {
	req.Header.Set("correlation-request-id", testCorrelationID)
	req.Header.Set("User-Agent", "test-user-agent")
	req.Header.Set("Accept-Language", "test-langauge")
	req.Header.Set("Content-Type", "application/json")
}
