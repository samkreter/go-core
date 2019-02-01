package customers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	otrace "go.opencensus.io/trace"

	"github.com/samkreter/trace-example/httputil"
	"github.com/samkreter/trace-example/log"
	"github.com/samkreter/trace-example/trace"
)

const (
	defaultAddr = ":8082"
)

var (
	defaultCustomer = customer{
		Name:  "tester",
		Email: "tester@example.com",
	}
)

type customer struct {
	Name  string
	Email string
	ID    string
}

// Server stores configuration for the customers microservice
type Server struct {
	customerAddr string
	httpClient   *http.Client
}

// NewServer creates a new customers server instance
func NewServer(addr string) (*Server, error) {
	if addr == "" {
		addr = defaultAddr
	}

	return &Server{
		customerAddr: addr,
		httpClient:   httputil.NewHTTPClient(true, true, true),
	}, nil
}

// Run start the customer microservice server
func (s *Server) Run() {
	router := mux.NewRouter()

	router.HandleFunc("/customer", s.handleCusomter).Methods("GET")

	tracingRouter := httputil.SetUpHandler(router, &httputil.HandlerConfig{
		CorrelationEnabled: true,
		LoggingEnabled:     true,
		TracingEnabled:     true,
	})

	log.G(context.TODO()).WithField("address: ", s.customerAddr).Info("Starting Customer Server:")
	log.G(context.TODO()).Fatal(http.ListenAndServe(s.customerAddr, tracingRouter))
}

func (s *Server) handleCusomter(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	if err := json.NewEncoder(w).Encode(defaultCustomer); err != nil {
		http.Error(w, fmt.Sprintf("json parse error: %v", err), http.StatusInternalServerError)
		log.G(ctx).WithError(err).Error("Failed to parsing json")
		return
	}

	ctx, span := trace.StartSpanWithTags(ctx, "Main Work", map[string]string{
		"importantInfo": "this is important information",
	})

	defer span.End()

	DoWork(ctx, "2")

	DoWork(ctx, "3")
}

func DoWork(ctx context.Context, workNum string) {
	_, span := trace.StartSpan(ctx, "DoingWork:"+workNum)
	defer span.End()

	span.SetStatus(otrace.Status{
		Code:    otrace.StatusCodePermissionDenied,
		Message: "You do not have permission to access this."})

	time.Sleep(time.Second)
}
