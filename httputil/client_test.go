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

var (
	CorrelationIDHeader = "correlation-request-id"
)

func TestOutgoingRequestLogging(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		corrID := req.Header.Get(CorrelationIDHeader)

		assert.NotEmpty(t, corrID, "Should not have correlation ID")

		rw.Write([]byte(`OK`))
	}))

	defer server.Close()

	ctx := correlation.CreateCtxFromRequest(&http.Request{})

	logrus.SetLevel(logrus.DebugLevel)
	testHook := logrustest.NewGlobal()

	req, err := http.NewRequest("GET", server.URL, nil)
	require.NoError(t, err, "Should not get error while creating reqeust")

	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")

	c := NewHTTPClient(true, true, false)
	resp, err := c.Do(req)
	require.NoError(t, err, "Should not get error for server response")
	assert.Equal(t, 200, resp.StatusCode, "Should get OK status code.")

	assert.Equal(t, 2, len(testHook.Entries), "Should have correct number of outgoing logs")

	correlationID := getValueFromLog(testHook.Entries[0], "correlationID", t)
	assert.NotEmpty(t, correlationID, "Should get a correlationID")

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

func getValueFromLog(entry logrus.Entry, key string, t *testing.T) string {
	val, ok := entry.Data[key].(string)
	require.True(t, ok, "Should have succesful cast.")

	return val
}

func TestNoCorrelation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		corrID := req.Header.Get(CorrelationIDHeader)

		assert.Empty(t, corrID, "Should not have correlation ID")

		// Send response to be tested
		rw.Write([]byte(`OK`))
	}))
	// Close the server when test finishes
	defer server.Close()

	c := NewHTTPClient(true, false, false)
	resp, err := c.Get(server.URL)
	require.NoError(t, err, "Should not get error while creating reqeust")

	assert.Equal(t, 200, resp.StatusCode, "Should get OK status code.")
}

func TestClientWithCorrelation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		corrID := req.Header.Get(CorrelationIDHeader)

		assert.NotEmpty(t, corrID, "Should not have correlation ID")

		rw.Write([]byte(`OK`))
	}))

	defer server.Close()

	ctx := correlation.CreateCtxFromRequest(&http.Request{})

	req, err := http.NewRequest("GET", server.URL, nil)

	req = req.WithContext(ctx)

	c := NewHTTPClient(true, false, false)
	resp, err := c.Do(req)
	require.NoError(t, err, "Should not get error for server response")

	assert.Equal(t, 200, resp.StatusCode, "Should get OK status code.")
}
