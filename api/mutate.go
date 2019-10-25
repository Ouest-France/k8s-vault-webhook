package api

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	corev1 "k8s.io/api/core/v1"
)

// mutateSecretData iterates over all secret keys and replace values if necessary
// by secret values stored in Vault
func (s *Server) mutateSecretData(secret corev1.Secret) ([]patchOperation, error) {

	// Patchs list
	patch := []patchOperation{}

	// Check each data key for secret to mutate
	for k8sSecretKey, k8sSecretValue := range secret.Data {

		// Ignore if no "vault:" prefix on secret value
		if !strings.HasPrefix(string(k8sSecretValue), "vault:") {
			s.Logger.Debugf("value of key '%s' doesn't start by 'vault:', ignoring", k8sSecretKey)
			continue
		}

		// Extract Vault secret path and key
		re := regexp.MustCompile(`^vault:(.*)#(.*)$`)
		sub := re.FindStringSubmatch(string(k8sSecretValue))
		if len(sub) != 3 {
			return []patchOperation{}, fmt.Errorf("vault placeholder '%s' doesn't match regex '^vault:(.*)#(.*)$'", string(k8sSecretValue))
		}
		vaultRawSecretPath := sub[1]
		vaultSecretKey := sub[2]

		// Check that required fields are not empty
		for key, val := range map[string]string{"name": secret.Name, "namespace": secret.Namespace} {
			if val == "" {
				return []patchOperation{}, fmt.Errorf("secret field %s cannot be empty", key)
			}
		}

		// Template vault secret path
		pathTemplate, err := template.New("path").Funcs(sprig.TxtFuncMap()).Parse(s.VaultPattern)
		if err != nil {
			return []patchOperation{}, fmt.Errorf("failed to parse template vault path pattern: %s", err)
		}

		var vaultSecretPath bytes.Buffer
		err = pathTemplate.Execute(&vaultSecretPath, struct {
			Name      string
			Namespace string
			Secret    string
		}{
			Name:      secret.Name,        // Kubernetes secret name
			Namespace: secret.Namespace,   // Kubernetes secret namespace
			Secret:    vaultRawSecretPath, // Kubernetes secret parsed value
		})
		if err != nil {
			return []patchOperation{}, fmt.Errorf("failed to execute template function on vault path pattern: %s", err)
		}

		// Read secret from Vault
		vaultSecretValue, err := s.Vault.Read(vaultSecretPath.String(), vaultSecretKey)
		if err != nil {
			return []patchOperation{}, fmt.Errorf("failed to read secret '%s' in vault: %s", vaultSecretPath.String(), err)
		}

		// Create patch to mutate secret value with vault value
		patch = append(
			patch,
			patchOperation{
				Op:    "replace",
				Path:  fmt.Sprintf("/data/%s", k8sSecretKey),
				Value: base64.StdEncoding.EncodeToString([]byte(vaultSecretValue)),
			},
		)

		s.Logger.Infof(
			"kubernetes secret '%s' key '%s' in namespace '%s', replaced by vault secret '%s' key '%s'",
			secret.Name, k8sSecretKey, secret.Namespace, vaultSecretPath.String(), vaultSecretKey)
	}

	return patch, nil
}
