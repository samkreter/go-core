package main

import (
	"context"
	"flag"

	"github.com/samkreter/trace-example/example/services/frontend"
	"github.com/samkreter/trace-example/log"
	"github.com/samkreter/trace-example/trace"
	"github.com/sirupsen/logrus"
)

const (
	frontendAddr = ":8081"
	customerAddr = "customers:8082"
	serviceName  = "frontend"
)

func main() {
	logLevel := flag.String("log-level", "info", `set the log level, e.g. "trace", debug", "info", "warn", "error"`)
	flag.Parse()

	level, err := log.ParseLevel(*logLevel)
	if err != nil {
		log.G(context.TODO()).WithError(err).Fatal("Failed to parse log level")
	}

	logrus.SetLevel(level)
	log.L = logrus.WithField("service", serviceName)

	err = trace.SetupTracing(serviceName, "jaeger")
	if err != nil {
		log.G(context.TODO()).WithError(err).Fatal("Failed to initialize tracing")
	}

	// Start the frontend service
	f, err := frontend.NewServer(frontendAddr, customerAddr)
	if err != nil {
		log.G(context.TODO()).WithError(err).Fatal("failed to create frontend server")
	}
	f.Run()
}
