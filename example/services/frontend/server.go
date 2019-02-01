package frontend

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/samkreter/go-core/httputil"
	"github.com/samkreter/go-core/log"
)

const (
	defaultAddr = "localhost:8081"
)

// Server holds configuration for the frontend server
type Server struct {
	frontendAddr string
	customerAddr string
	httpClient   *http.Client
}

// NewServer creates a new frontend server
func NewServer(addr, customerAddr string) (*Server, error) {
	if addr == "" {
		addr = defaultAddr
	}

	if customerAddr == "" {
		return nil, fmt.Errorf("Must supply custosmer Addr")
	}

	return &Server{
		frontendAddr: addr,
		customerAddr: customerAddr,
		httpClient:   httputil.NewHTTPClient(true, true, true),
	}, nil
}

// Run start the frontend server
func (s *Server) Run() {
	router := mux.NewRouter()

	router.Handle("/", http.FileServer(http.Dir("static")))
	router.HandleFunc("/create", s.handleCreate).Methods("POST")

	tracingRouter := httputil.SetUpHandler(router, &httputil.HandlerConfig{
		CorrelationEnabled: true,
		LoggingEnabled:     true,
		TracingEnabled:     true,
	})

	log.G(context.TODO()).WithField("address: ", s.frontendAddr).Info("Starting Frontend Server:")
	log.G(context.TODO()).Fatal(http.ListenAndServe(s.frontendAddr, tracingRouter))
}

func (s *Server) handleCreate(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	var jsonStr = []byte(`[ { "url": "http://blank.org", "arguments": [] } ]`)

	r, err := http.NewRequest("POST", "http://blank.org", bytes.NewBuffer(jsonStr))
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.G(ctx).WithError(err).Error("Failed to create request")
		return
	}

	r.Header.Set("Content-Type", "application/json")

	// Propagate the trace header info in the outgoing requests.
	r = r.WithContext(ctx)
	resp, err := s.httpClient.Do(r)
	if err != nil {
		log.G(context.TODO()).WithError(err).Error("failed outgoing request")
	} else {
		// TODO: handle response
		resp.Body.Close()
	}

	r2, err := http.NewRequest("GET", "http://"+s.customerAddr+"/customer", nil)
	if err != nil {
		http.Error(w, "Failed to create customer reqeust", http.StatusInternalServerError)
		log.G(context.TODO()).WithError(err).Error("failed createing customer request")
		return
	}
	r2 = r2.WithContext(r.Context())
	cResp, err := s.httpClient.Do(r2)
	if err != nil {
		http.Error(w, "Failed to retrieve customer data", http.StatusInternalServerError)
		log.G(context.TODO()).WithError(err).Error("failed to retreieve customer data")
		return
	}

	defer cResp.Body.Close()

	io.Copy(w, cResp.Body)
}
