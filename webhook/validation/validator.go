package validation

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	clientset     *kubernetes.Clientset
	clientsetOnce sync.Once
	clientsetErr  error
)

type CheckInUseValidator struct {
	Logger *logrus.Entry
}

// getClientset returns a singleton Kubernetes clientset
func getClientset() (*kubernetes.Clientset, error) {
	clientsetOnce.Do(func() {
		config, err := rest.InClusterConfig()
		if err != nil {
			clientsetErr = fmt.Errorf("failed to get in-cluster config: %w", err)
			return
		}
		clientset, clientsetErr = kubernetes.NewForConfig(config)
		if clientsetErr != nil {
			clientsetErr = fmt.Errorf("failed to create Kubernetes clientset: %w", clientsetErr)
		}
	})
	return clientset, clientsetErr
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

	inUse, podList, err := v.CheckForPods(&sa)
	if err != nil {
		return nil, err
	}

	admissionResponse := &admissionv1.AdmissionResponse{}
	if inUse {
		v.Logger.Infof("ServiceAccount %s is in use, denying deletion", sa.Name)
		admissionResponse.Allowed = false
		admissionResponse.Result = &metav1.Status{
			Message: fmt.Sprintf("ServiceAccount %s is in use by pods: %v", sa.Name, podList),
		}
	} else {
		v.Logger.Infof("ServiceAccount %s is not in use, allowing deletion", sa.Name)
		admissionResponse.Allowed = true
	}

	return admissionResponse, nil
}

// CheckForPods checks if there are any Pods using the given ServiceAccount and returns true if found, along with the list of pod names.
func (v *CheckInUseValidator) CheckForPods(sa *corev1.ServiceAccount) (bool, []string, error) {
	ns := sa.Namespace
	name := sa.Name

	v.Logger.Infof("Checking for Pods using ServiceAccount %s in namespace %s", name, ns)

	// Create context with timeout to prevent hanging
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get singleton Kubernetes client
	clientset, err := getClientset()
	if err != nil {
		v.Logger.Errorf("Failed to get Kubernetes client: %v", err)
		return false, nil, err
	}

	// List Pods in the namespace
	podList, err := clientset.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		v.Logger.Errorf("Failed to list pods in namespace %s: %v", ns, err)
		return false, nil, err
	}

	// Check if any pod is using the ServiceAccount
	podsList := []string{}
	for _, pod := range podList.Items {
		if pod.Spec.ServiceAccountName == name {
			podsList = append(podsList, pod.Name)
		}
	}

	if len(podsList) > 0 {
		v.Logger.Infof("ServiceAccount %s is used by %d pods: %v", name, len(podsList), podsList)
		return true, podsList, nil
	}

	v.Logger.Infof("No Pods are using ServiceAccount %s", name)
	return false, nil, nil
}
