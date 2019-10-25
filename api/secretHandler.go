package api

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
)

type patchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

func (s *Server) secretHandler(w http.ResponseWriter, r *http.Request) {

	s.Logger.Debug("secret request received, handling")

	// Read request body
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		s.Logger.Errorf("failed read request body: %s", err)
		http.Error(w, http.StatusText(500), 500)
		return
	}
	defer r.Body.Close()

	// Parse review request
	var admissionReview v1beta1.AdmissionReview
	err = json.Unmarshal(body, &admissionReview)
	if err != nil {
		s.Logger.Errorf("failed to unmarshal request: %s", err)
		http.Error(w, http.StatusText(500), 500)
		return
	}

	// Validate admission request type
	if admissionReview.Kind != "AdmissionReview" || admissionReview.APIVersion != "admission.k8s.io/v1beta1" {

		s.Logger.Debug("not an admissionreview request, ignoring")
		s.sendAdmissionReview(w, admissionReview)

		return
	}

	// Validate object type
	if admissionReview.Request.Kind.Kind != "Secret" || admissionReview.Request.Kind.Version != "v1" {

		s.Logger.Debug("not a secret object, ignoring")
		s.sendAdmissionReview(w, admissionReview)

		return
	}

	// Parse secret object
	var secret corev1.Secret
	err = json.Unmarshal(admissionReview.Request.Object.Raw, &secret)
	if err != nil {
		s.Logger.Errorf("failed to unmarshal secret: %s", err)
		http.Error(w, http.StatusText(500), 500)
		return
	}

	// List of patchs on secret
	patch, err := s.mutateSecretData(secret)
	if err != nil {
		s.Logger.Error(err)
		s.sendAdmissionReviewError(w, err)
		return
	}

	// Marshal patches
	patchBytes, err := json.Marshal(patch)
	if err != nil {
		s.Logger.Errorf("failed marshal patches: %s", err)
		http.Error(w, http.StatusText(500), 500)
		return
	}

	// Attach admission response to admission review
	admissionReview.Response = &v1beta1.AdmissionResponse{
		Allowed: true,
		Patch:   patchBytes,
		PatchType: func() *v1beta1.PatchType {
			pt := v1beta1.PatchTypeJSONPatch
			return &pt
		}(),
	}

	// Send admission review back to kubernetes
	s.sendAdmissionReview(w, admissionReview)
}
