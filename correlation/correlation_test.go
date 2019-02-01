package correlation

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testCorrelationID = "test-correlation-id"
)

func TestAddHeadersFromContext(t *testing.T) {
	ctx := SetCorrelationID(context.Background(), testCorrelationID)

	req, err := http.NewRequest("GET", "example.com", nil)
	require.NoError(t, err, "Should not get error creating request.")

	AddHeadersFromContext(ctx, req)

	assert.Equal(t, testCorrelationID, req.Header.Get(CorrelationIDHeader), "Should add correct correlation Id to request")
}

func TestCreateCtxFromRequest(t *testing.T) {
	tt := []struct {
		name          string
		correlationID string
	}{
		{"With Passed in Correlation ID",
			testCorrelationID,
		},

		{"Generating Correaltion ID",
			"",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "example.com", nil)
			require.NoError(t, err, "Should not get error when creating a request")

			if tc.correlationID != "" {
				req.Header.Set("correlation-request-id", tc.correlationID)
			}

			AddStandardRequestHeaders(req)

			ctx := CreateCtxFromRequest(req)

			if tc.correlationID == "" {
				assert.NotEmpty(t, GetCorrelationID(ctx), "Correlation ID should have been generated.")
			} else {
				assert.Equal(t, tc.correlationID, GetCorrelationID(ctx), "Shoulld have correct correlation ID")
			}

			assert.NotEmpty(t, GetActivityID(ctx))

			ctxHeaders := GetMetadataHeaders(ctx)

			assert.Equal(t, "test-user-agent", ctxHeaders.Get("User-Agent"), "Should get correct user agent")
			assert.Equal(t, "test-langauge", ctxHeaders.Get("Accept-Language"), "Should get correct langauge")
		})
	}
}

func AddStandardRequestHeaders(req *http.Request) {
	req.Header.Set("User-Agent", "test-user-agent")
	req.Header.Set("Accept-Language", "test-langauge")
	req.Header.Set("Content-Type", "application/json")
}
