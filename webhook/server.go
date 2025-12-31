package webhook

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/shanmugara/sa-delete-webhook/webhook/admission"
	"github.com/sirupsen/logrus"
	admissionv1 "k8s.io/api/admission/v1"
)

var (
	Logger *logrus.Logger
)

func RunWebhookServer(ctx context.Context, certFile, keyFile string, port int) error {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		Logger.Errorf("Failed to load TLS cert/key: %v", err)
		return err
	}

	Logger.Infof("Starting webhook server on port %d", port)
	mux := http.NewServeMux()
	mux.HandleFunc("/validate", validateSa)
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{cert},
		},
		ErrorLog: log.Default(),
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ListenAndServeTLS("", "")
	}()

	select {
	case <-ctx.Done():
		Logger.Info("Shutdown signal received, shutting down webhook server...")
		shutdownCtx := ctx
		return server.Shutdown(shutdownCtx)
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			Logger.Errorf("Webhook server error: %v", err)
			return err
		}
		return nil
	}
}

func validateSa(w http.ResponseWriter, r *http.Request) {
	Logger.Infof("Received request: %s %s", r.Method, r.URL.Path)

	admissionReview, err := parseRequest(*r)
	if err != nil {
		Logger.Errorf("Failed to parse admission review: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	admitter := admission.Admitter{
		Logger:  Logger.WithField("component", "admitter"),
		Request: admissionReview.Request,
	}

	admissionResponse, err := admitter.ValidateSaReview()
	if err != nil {
		Logger.Errorf("Validation error: %v", err)
		http.Error(w, "Validation error", http.StatusInternalServerError)
		return
	}
	Logger.Infof("Admission response: %+v", admissionResponse)

	// Build the AdmissionReview response
	respReview := admissionv1.AdmissionReview{
		TypeMeta: admissionReview.TypeMeta,
		Response: admissionResponse,
	}
	// Set the UID to match the request
	if admissionReview.Request != nil {
		respReview.Response.UID = admissionReview.Request.UID
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(respReview); err != nil {
		Logger.Errorf("Failed to encode response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func init() {
	Logger = logrus.New()
	Logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
}

// parseRequest extracts an AdmissionReview from an http.Request if possible
func parseRequest(r http.Request) (*admissionv1.AdmissionReview, error) {
	Logger.Infof("Parsing admission review request")
	contentType := r.Header.Get("Content-Type")

	if contentType != "application/json" {
		return nil, fmt.Errorf("Content-Type: %q should be %q", contentType, "application/json")
	}

	bodybuf := new(bytes.Buffer)
	_, err := bodybuf.ReadFrom(r.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read request body: %v", err)
	}
	body := bodybuf.Bytes()

	if len(body) == 0 {
		return nil, fmt.Errorf("admission request body is empty")
	}

	var a admissionv1.AdmissionReview

	if err := json.Unmarshal(body, &a); err != nil {
		return nil, fmt.Errorf("could not parse admission review request: %v", err)
	}

	if a.Request == nil {
		return nil, fmt.Errorf("admission review can't be used: Request field is nil")
	}

	return &a, nil
}
