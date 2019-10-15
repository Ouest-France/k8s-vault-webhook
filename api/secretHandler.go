package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/valyala/fasttemplate"
	"k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
)

type patchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

func (s *Server) secretHandler(w http.ResponseWriter, r *http.Request) {

	// Read body
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("failed read request body")
		http.Error(w, http.StatusText(500), 500)
	}
	defer r.Body.Close()

	// Parse review request
	var admissionReview v1beta1.AdmissionReview
	err = json.Unmarshal(body, &admissionReview)
	if err != nil {
		log.Printf("failed to unmarshal request")
		http.Error(w, http.StatusText(500), 500)
	}

	// Validate admission request type
	if admissionReview.Kind != "AdmissionReview" || admissionReview.APIVersion != "admission.k8s.io/v1beta1" {

		log.Printf("not an admissionreview request, ignoring")

		resp, err := json.Marshal(admissionReview)
		if err != nil {
			log.Printf("failed to marshal response")
			http.Error(w, http.StatusText(500), 500)
			return
		}

		_, err = w.Write(resp)
		if err != nil {
			log.Printf("failed to write response")
			return
		}

		return
	}

	// Validate object type
	if admissionReview.Request.Kind.Kind != "Secret" || admissionReview.Request.Kind.Version != "v1" {

		log.Printf("not a secret object, ignoring")

		resp, err := json.Marshal(admissionReview)
		if err != nil {
			log.Printf("failed to marshal response")
			http.Error(w, http.StatusText(500), 500)
			return
		}

		_, err = w.Write(resp)
		if err != nil {
			log.Printf("failed to write response")
			return
		}

		return
	}

	// Parse secret object
	var secret corev1.Secret
	err = json.Unmarshal(admissionReview.Request.Object.Raw, &secret)
	if err != nil {
		log.Printf("failed to unmarshal secret: %s", err)
		http.Error(w, http.StatusText(500), 500)
		return
	}

	// List of patchs on secret
	patch := []patchOperation{}

	// Check each data key for secret to mutate
	for key, val := range secret.Data {

		// Ignore if no "vault:" prefix on secret value
		if !strings.HasPrefix(string(val), "vault:") {
			continue
		}

		// Extract Vault secret path and key
		re := regexp.MustCompile(`^vault:(.*)#(.*)$`)
		sub := re.FindStringSubmatch(string(val))
		if len(sub) != 3 {
			log.Printf("invalid vault path no regex match")
			http.Error(w, http.StatusText(500), 500)
			return
		}
		secretPath := sub[1]
		secretKey := sub[2]

		t := fasttemplate.New(s.VaultPattern, "{{", "}}")
		pattern := t.ExecuteString(map[string]interface{}{"namespace": secret.Namespace})
		vaultSecretPath := fmt.Sprintf("%s/data/%s/%s", s.VaultBackend, pattern, secretPath)

		// Read secret from Vault
		vaultSecretValue, err := s.Vault.Read(vaultSecretPath, secretKey)
		if err != nil {
			log.Printf("failed to read secret %q in vault: %s", pattern, err)
			http.Error(w, http.StatusText(500), 500)
			return
		}

		// Create patch to mutate secret value with vault value
		patch = append(
			patch,
			patchOperation{
				Op:    "replace",
				Path:  fmt.Sprintf("/data/%s", key),
				Value: base64.StdEncoding.EncodeToString([]byte(vaultSecretValue)),
			},
		)
	}

	// Marshal patches
	patchBytes, err := json.Marshal(patch)
	if err != nil {
		log.Printf("failed marshal patches: %s", err)
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

	// Marshal admission review with response
	resp, err := json.Marshal(admissionReview)
	if err != nil {
		log.Printf("failed to marshal response: %s", err)
		http.Error(w, http.StatusText(500), 500)
		return
	}

	// Send admission review back to kubernetes
	_, err = w.Write(resp)
	if err != nil {
		log.Printf("failed to write response")
		return
	}
}
