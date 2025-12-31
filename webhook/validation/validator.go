package validation

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sirupsen/logrus"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type CheckInUseValidator struct {
	Logger *logrus.Entry
}

// ValidateSaDeletion checks if the ServiceAccount is in use by any Pods before allowing deletion and returns an AdmissionResponse.
func (v *CheckInUseValidator) ValidateSaDeletion(request *admissionv1.AdmissionRequest) (*admissionv1.AdmissionResponse, error) {
	v.Logger.Info("Validating ServiceAccount deletion")

	// Unmarshal the ServiceAccount object from the request
	var sa corev1.ServiceAccount
	if err := json.Unmarshal(request.OldObject.Raw, &sa); err != nil {
		v.Logger.Error("Failed to unmarshal the ServiceAccount object")
		return nil, err
	}

	inUse, err := v.CheckForPods(&sa)
	if err != nil {
		return nil, err
	}

	admissionResponse := &admissionv1.AdmissionResponse{}
	if inUse {
		v.Logger.Infof("ServiceAccount %s is in use, denying deletion", sa.Name)
		admissionResponse.Allowed = false
		admissionResponse.Result = &metav1.Status{
			Message: fmt.Sprintf("ServiceAccount %s is in use and cannot be deleted", sa.Name),
		}
	} else {
		v.Logger.Infof("ServiceAccount %s is not in use, allowing deletion", sa.Name)
		admissionResponse.Allowed = true
	}

	return admissionResponse, nil
}

// CheckForPods checks if there are any Pods using the given ServiceAccount and returns true if found.
func (v *CheckInUseValidator) CheckForPods(sa *corev1.ServiceAccount) (bool, error) {
	// Check if the object in the request is a ServiceAccount
	ns := sa.Namespace
	name := sa.Name

	v.Logger.Infof("Checking for Pods using ServiceAccount %s in namespace %s", name, ns)

	ctx := context.Background()

	// Initialize Kubernetes client
	config, err := rest.InClusterConfig()
	if err != nil {
		v.Logger.Error("Failed to get in-cluster config")
		return false, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		v.Logger.Error("Failed to create Kubernetes clientset")
		return false, err
	}

	// List Pods in the namespace
	podList, err := clientset.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		v.Logger.Errorf("Failed to list pods in namespace %s: %v", ns, err)
		return false, err
	}

	// Check if any pod is using the ServiceAccount
	podsUsingSA := false
	for _, pod := range podList.Items {
		if pod.Spec.ServiceAccountName == name {
			podsUsingSA = true
			break
		}
	}

	return podsUsingSA, nil
}
