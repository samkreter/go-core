package main

import (
	"context"
	"flag"

	"github.com/samkreter/go-core/example/services/customers"
	"github.com/samkreter/go-core/log"
	"github.com/samkreter/go-core/trace"
	"github.com/sirupsen/logrus"
)

const (
	customerAddr = ":8082"
	serviceName  = "customers"
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

	// Set up the default tracing wiht Jaeger
	err = trace.SetupTracing(serviceName, "jaeger")
	if err != nil {
		log.G(context.TODO()).WithError(err).Fatal("Failed to initialize tracing")
	}

	// Start the customers service
	c, err := customers.NewServer(customerAddr)
	if err != nil {
		log.G(context.TODO()).WithError(err).Fatal("failed to create customer server")
	}
	c.Run()
}
