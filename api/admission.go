package api

import (
	"encoding/json"
	"net/http"

	admission "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// sendAdmissionReviewError create an admission review with an
// error set as response message and write it to http.ResponseWriter
func (s *Server) sendAdmissionReviewError(w http.ResponseWriter, err error) {

	// Create empty AdmissionReview with an error
	// set as response message
	ar := admission.AdmissionReview{
		Response: &admission.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		},
	}

	// Marshal admission review with response
	arResp, err := json.Marshal(ar)
	if err != nil {
		s.Logger.Errorf("failed to marshal response: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Set http code to server error
	w.WriteHeader(500)

	// Send admission review back to kubernetes
	_, err = w.Write(arResp)
	if err != nil {
		s.Logger.Errorf("failed to write response: %s", err)
		return
	}
}

// sendAdmissionReview masharl and write to http.ResponseWriter an admission review
func (s *Server) sendAdmissionReview(w http.ResponseWriter, ar admission.AdmissionReview) {

	// Marshal Admission Rreview
	resp, err := json.Marshal(ar)
	if err != nil {
		s.Logger.Errorf("failed to marshal response: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Send admission review back to kubernetes
	_, err = w.Write(resp)
	if err != nil {
		s.Logger.Errorf("failed to write admission review response: %s", err)
		return
	}
}
