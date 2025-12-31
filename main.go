package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	server "github.com/shanmugara/sa-delete-webhook/webhook"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var (
	certFile string
	keyFile  string
	port     int = 8443
)

func main() {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
	server.Logger = logger
	loggerEntry := logrus.NewEntry(logger)
	loggerEntry.Info("Initializing SA Delete Webhook Server")

	pflag.StringVar(&certFile, "cert-file", "cert.crt", "TLS Cert file for the server")
	pflag.StringVar(&keyFile, "key-file", "key.key", "TLS Key file for the server")
	pflag.IntVar(&port, "port", 8443, "Port for the webhook server")

	viper.BindPFlags(pflag.CommandLine)
	viper.SetEnvPrefix("SDW")
	viper.AutomaticEnv()

	pflag.Parse()
	certFile = viper.GetString("cert-file")
	keyFile = viper.GetString("key-file")
	port = viper.GetInt("port")

	// Setup signal handling for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	loggerEntry.Info("Starting the webhook server...")
	if err := server.RunWebhookServer(ctx, certFile, keyFile, port); err != nil {
		loggerEntry.Errorf("Webhook server exited with error: %v", err)
		os.Exit(1)
	}
	loggerEntry.Info("Webhook server shut down cleanly.")
}
