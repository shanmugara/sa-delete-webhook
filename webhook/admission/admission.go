package admission

import (
	"encoding/json"
	"fmt"

	"github.com/shanmugara/sa-delete-webhook/webhook/validation"
	"github.com/sirupsen/logrus"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
)

var (
	Logger *logrus.Logger
)

type Admitter struct {
	Logger  *logrus.Entry
	Request *admissionv1.AdmissionRequest
}

func (a *Admitter) ValidateSaReview() (*admissionv1.AdmissionResponse, error) {
	a.Logger.Info("Validating ServiceAccount")
	sa, err := a.Sa()
	if err != nil {
		return nil, err
	}
	a.Logger.Infof("ServiceAccount to validate: %s in namespace %s", sa.Name, sa.Namespace)

	validator := validation.CheckInUseValidator{
		Logger: a.Logger,
	}

	return validator.ValidateSaDeletion(a.Request)
}

func (a *Admitter) Sa() (*corev1.ServiceAccount, error) {
	// Check if the object in the request is a ServiceAccount
	if a.Request.Kind.Kind != "ServiceAccount" {
		a.Logger.Error("The object in the request is not a ServiceAccount")
		return nil, fmt.Errorf("object in the request is not a ServiceAccount")
	}

	sa := corev1.ServiceAccount{}
	if err := json.Unmarshal(a.Request.OldObject.Raw, &sa); err != nil {
		a.Logger.Error("Failed to unmarshal the ServiceAccount object")
		return nil, err
	}

	return &sa, nil

}
