package api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
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

	// Read body
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

		resp, err := json.Marshal(admissionReview)
		if err != nil {
			s.Logger.Errorf("failed to marshal response: %s", err)
			http.Error(w, http.StatusText(500), 500)
			return
		}

		_, err = w.Write(resp)
		if err != nil {
			s.Logger.Errorf("failed to write response: %s", err)
			return
		}

		return
	}

	// Validate object type
	if admissionReview.Request.Kind.Kind != "Secret" || admissionReview.Request.Kind.Version != "v1" {

		s.Logger.Debug("not a secret object, ignoring")

		resp, err := json.Marshal(admissionReview)
		if err != nil {
			s.Logger.Errorf("failed to marshal response: %s", err)
			http.Error(w, http.StatusText(500), 500)
			return
		}

		_, err = w.Write(resp)
		if err != nil {
			s.Logger.Errorf("failed to write response: %s", err)
			return
		}

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
	patch := []patchOperation{}

	// Check each data key for secret to mutate
	for key, val := range secret.Data {

		// Ignore if no "vault:" prefix on secret value
		if !strings.HasPrefix(string(val), "vault:") {
			s.Logger.Debugf("value of key '%s' doesn't start by 'vault:', ignoring", key)
			continue
		}

		// Extract Vault secret path and key
		re := regexp.MustCompile(`^vault:(.*)#(.*)$`)
		sub := re.FindStringSubmatch(string(val))
		if len(sub) != 3 {
			s.Logger.Errorf("invalid vault path no regex match")
			http.Error(w, http.StatusText(500), 500)
			return
		}
		secretPath := sub[1]
		secretKey := sub[2]

		// Template vault secret path
		pathTemplate, err := template.New("path").Funcs(sprig.TxtFuncMap()).Parse(s.VaultPattern)
		if err != nil {
			s.Logger.Errorf("failed to parse template vault path: %s", err)
			http.Error(w, http.StatusText(500), 500)
			return
		}

		var vaultSecretPath bytes.Buffer
		err = pathTemplate.Execute(&vaultSecretPath, struct {
			Name      string
			Namespace string
			Secret    string
		}{
			Name:      secret.Name,      // Kubernetes secret name
			Namespace: secret.Namespace, // Kubernetes secret namespace
			Secret:    secretPath,       // Kubernetes secret parsed value
		})
		if err != nil {
			s.Logger.Errorf("failed to execute template vault path: %s", err)
			http.Error(w, http.StatusText(500), 500)
			return
		}

		// Read secret from Vault
		vaultSecretValue, err := s.Vault.Read(vaultSecretPath.String(), secretKey)
		if err != nil {
			s.Logger.Errorf("failed to read secret '%s' in vault: %s", s.VaultPattern, err)
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

	// Marshal admission review with response
	resp, err := json.Marshal(admissionReview)
	if err != nil {
		s.Logger.Errorf("failed to marshal response: %s", err)
		http.Error(w, http.StatusText(500), 500)
		return
	}

	// Send admission review back to kubernetes
	_, err = w.Write(resp)
	if err != nil {
		s.Logger.Errorf("failed to write response")
		return
	}
}
