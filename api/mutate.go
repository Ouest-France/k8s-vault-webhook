package api

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
)

// mutateSecretData iterates over all secret keys and replace values if necessary
// by secret values stored in Vault
func (s *Server) mutateSecretData(secret corev1.Secret) ([]patchOperation, error) {

	// Patchs list
	patch := []patchOperation{}

	// Check each data key for secret to mutate
	for k8sSecretKey, k8sSecretValue := range secret.Data {

		logger := s.Logger.WithFields(logrus.Fields{
			"kubernetes_secret_name":      secret.Name,
			"kubernetes_secret_namespace": secret.Namespace,
			"kubernetes_secret_key":       k8sSecretKey,
		})

		// Ignore if no "vault:" prefix on secret value
		if !strings.HasPrefix(string(k8sSecretValue), "vault:") {
			logger.Debug("value doesn't have 'vault:' prefix, ignoring")
			continue
		}

		// Extract Vault secret path and key
		re := regexp.MustCompile(`^vault:(.*)#(.*)$`)
		sub := re.FindStringSubmatch(string(k8sSecretValue))
		if len(sub) != 3 {
			logger.Errorf("vault placeholder '%s' doesn't match regex '^vault:(.*)#(.*)$'", string(k8sSecretValue))
			return []patchOperation{}, fmt.Errorf("vault placeholder '%s' doesn't match regex '^vault:(.*)#(.*)$'", string(k8sSecretValue))
		}
		vaultRawSecretPath := sub[1]
		vaultSecretKey := sub[2]

		// Check that required fields are not empty
		for key, val := range map[string]string{"name": secret.Name, "namespace": secret.Namespace} {
			if val == "" {
				logger.Errorf("secret field %s cannot be empty", key)
				return []patchOperation{}, fmt.Errorf("secret attribute %s cannot be empty", key)
			}
		}

		// Template vault secret path
		pathTemplate, err := template.New("path").Funcs(sprig.TxtFuncMap()).Parse(s.VaultPattern)
		if err != nil {
			logger.WithError(err).Errorf("failed to parse template vault path pattern")
			return []patchOperation{}, errors.New("failed to parse template vault path pattern")
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
			logger.WithError(err).Error("failed to execute template function on vault path pattern")
			return []patchOperation{}, errors.New("failed to execute template function on vault path pattern")
		}

		logger = logger.WithFields(logrus.Fields{
			"vault_secret_path": vaultSecretPath.String(),
			"vault_secret_key":  vaultSecretKey,
		})

		// Read secret from Vault
		vaultSecretValue, err := s.Vault.Read(vaultSecretPath.String(), vaultSecretKey)
		if err != nil {
			logger.WithError(err).Error("failed to read secret in vault")
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

		logger.Info("kubernetes secret mutated with vault value")
	}

	return patch, nil
}
