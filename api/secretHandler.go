package api

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/sirupsen/logrus"
	admission "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
)

type patchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

func (s *Server) secretHandler(w http.ResponseWriter, r *http.Request) {

	logger := s.Logger.WithField("handler", "secret")
	logger.Debug("request received, handling")

	// Read request body
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logger.WithError(err).Error("failed to read request body")
		http.Error(w, http.StatusText(500), 500)
		secretFailed.Inc()
		return
	}
	defer r.Body.Close()

	// Parse review request
	var admissionReview admission.AdmissionReview
	err = json.Unmarshal(body, &admissionReview)
	if err != nil {
		logger.WithError(err).Error("failed to unmarshal request")
		http.Error(w, http.StatusText(500), 500)
		secretFailed.Inc()
		return
	}

	logger = logger.WithFields(logrus.Fields{
		"kubernetes_admissionreview_kind":       admissionReview.Kind,
		"kubernetes_admissionreview_apiversion": admissionReview.APIVersion,
	})

	// Validate admission request type
	if admissionReview.Kind != "AdmissionReview" || admissionReview.APIVersion != "admission.k8s.io/v1beta1" {

		logger.Debug("not an admissionreview request, ignoring")
		s.sendAdmissionReview(w, admissionReview)
		secretIgnored.Inc()

		return
	}

	logger = logger.WithFields(logrus.Fields{
		"kubernetes_admissionreview_request_kind":       admissionReview.Request.Kind,
		"kubernetes_admissionreview_request_apiversion": admissionReview.Request.Kind.Version,
	})

	// Validate object type
	if admissionReview.Request.Kind.Kind != "Secret" || admissionReview.Request.Kind.Version != "v1" {

		logger.Debug("not a secret object, ignoring")
		s.sendAdmissionReview(w, admissionReview)
		secretIgnored.Inc()

		return
	}

	// Parse secret object
	var secret corev1.Secret
	err = json.Unmarshal(admissionReview.Request.Object.Raw, &secret)
	if err != nil {
		logger.WithError(err).Error("failed to unmarshal secret")
		http.Error(w, http.StatusText(500), 500)
		secretFailed.Inc()
		return
	}

	logger = logger.WithFields(logrus.Fields{
		"kubernetes_secret_name":      secret.Name,
		"kubernetes_secret_namespace": secret.Namespace,
	})

	// List of patchs on secret
	patch, err := s.mutateSecretData(secret)
	if err != nil {
		s.sendAdmissionReviewError(w, err)
		secretFailed.Inc()
		return
	}

	// Marshal patches
	patchBytes, err := json.Marshal(patch)
	if err != nil {
		logger.WithError(err).Error("failed to marshal patches")
		http.Error(w, http.StatusText(500), 500)
		secretFailed.Inc()
		return
	}

	// Attach admission response to admission review
	admissionReview.Response = &admission.AdmissionResponse{
		Allowed: true,
		Patch:   patchBytes,
		PatchType: func() *admission.PatchType {
			pt := admission.PatchTypeJSONPatch
			return &pt
		}(),
	}

	// Send admission review back to kubernetes
	s.sendAdmissionReview(w, admissionReview)
}
