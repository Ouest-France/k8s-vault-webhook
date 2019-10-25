package api

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestServer_mutateSecretData(t *testing.T) {

	var mutateTests = []struct {
		vaultClient  VaultClient
		vaultPattern string
		secret       string
		patch        []patchOperation
		errorString  string
	}{
		{
			// Test secret that doesn't need to be mutated
			fakeVaultClient{},
			"secret/data/{{.Secret}}",
			`{"metadata":{"name":"test-secret","namespace":"test-namespace","creationTimestamp":null,"annotations":{"kubectl.kubernetes.io/last-applied-configuration":"{\"apiVersion\":\"v1\",\"data\":{\"foo\":\"YmFy\"},\"kind\":\"Secret\",\"metadata\":{\"annotations\":{},\"name\":\"test-secret\",\"namespace\":\"test-namespace\"},\"type\":\"Opaque\"}\n"}},"data":{"foo":"YmFy"},"type":"Opaque"}`,
			[]patchOperation{},
			"",
		},
		{
			// Test secret with invalid path pattern
			fakeVaultClient{},
			"secret/data/{{.Secret}}",
			`{"metadata":{"name":"test-secret","namespace":"test-namespace","creationTimestamp":null,"annotations":{"kubectl.kubernetes.io/last-applied-configuration":"{\"apiVersion\":\"v1\",\"data\":{\"foo\":\"dmF1bHQ6YmFy\"},\"kind\":\"Secret\",\"metadata\":{\"annotations\":{},\"name\":\"test-secret\",\"namespace\":\"test-namespace\"},\"type\":\"Opaque\"}\n"}},"data":{"foo":"dmF1bHQ6YmFy"},"type":"Opaque"}`,
			[]patchOperation{},
			"vault placeholder 'vault:bar' doesn't match regex '^vault:(.*)#(.*)$'",
		},
		{
			// Test secret with empty name
			fakeVaultClient{},
			"secret/data/{{.Secret}}",
			`{"metadata":{"name":"","namespace":"test-namespace","creationTimestamp":null,"annotations":{"kubectl.kubernetes.io/last-applied-configuration":"{\"apiVersion\":\"v1\",\"data\":{\"foo\":\"dmF1bHQ6Zm9vI2Jhcg==\"},\"kind\":\"Secret\",\"metadata\":{\"annotations\":{},\"name\":\"\",\"namespace\":\"test-namespace\"},\"type\":\"Opaque\"}\n"}},"data":{"foo":"dmF1bHQ6Zm9vI2Jhcg=="},"type":"Opaque"}`,
			[]patchOperation{},
			"secret field name cannot be empty",
		},
		{
			// Test secret with empty namespace
			fakeVaultClient{},
			"secret/data/{{.Secret}}",
			`{"metadata":{"name":"test-secret","namespace":"","creationTimestamp":null,"annotations":{"kubectl.kubernetes.io/last-applied-configuration":"{\"apiVersion\":\"v1\",\"data\":{\"foo\":\"dmF1bHQ6Zm9vI2Jhcg==\"},\"kind\":\"Secret\",\"metadata\":{\"annotations\":{},\"name\":\"test-secret\",\"namespace\":\"\"},\"type\":\"Opaque\"}\n"}},"data":{"foo":"dmF1bHQ6Zm9vI2Jhcg=="},"type":"Opaque"}`,
			[]patchOperation{},
			"secret field namespace cannot be empty",
		},
		{
			// Test invalid vault pattern
			fakeVaultClient{},
			"secret/data/{{.Secret}}/{{.InvalidKey}}",
			`{"metadata":{"name":"test-secret","namespace":"test-namespace","creationTimestamp":null,"annotations":{"kubectl.kubernetes.io/last-applied-configuration":"{\"apiVersion\":\"v1\",\"data\":{\"foo\":\"dmF1bHQ6Zm9vI2Jhcg==\"},\"kind\":\"Secret\",\"metadata\":{\"annotations\":{},\"name\":\"test-secret\",\"namespace\":\"test-namespace\"},\"type\":\"Opaque\"}\n"}},"data":{"foo":"dmF1bHQ6Zm9vI2Jhcg=="},"type":"Opaque"}`,
			[]patchOperation{},
			"failed to execute template function on vault path pattern: template: path:1:26: executing \"path\" at <.InvalidKey>: can't evaluate field InvalidKey in type struct { Name string; Namespace string; Secret string }",
		},
		{
			// Test secret that doesn't exists in vault
			fakeVaultClient{"error"},
			"secret/data/{{.Secret}}",
			`{"metadata":{"name":"test-secret","namespace":"test-namespace","creationTimestamp":null,"annotations":{"kubectl.kubernetes.io/last-applied-configuration":"{\"apiVersion\":\"v1\",\"data\":{\"foo\":\"dmF1bHQ6Zm9vI2Jhcg==\"},\"kind\":\"Secret\",\"metadata\":{\"annotations\":{},\"name\":\"test-secret\",\"namespace\":\"test-namespace\"},\"type\":\"Opaque\"}\n"}},"data":{"foo":"dmF1bHQ6Zm9vI2Jhcg=="},"type":"Opaque"}`,
			[]patchOperation{},
			"failed to read secret 'secret/data/foo' in vault: failed to read key in vault",
		},
		{
			// Test valid secret defined in vault
			fakeVaultClient{"bar"},
			"secret/data/{{.Secret}}",
			`{"metadata":{"name":"test-secret","namespace":"test-namespace","creationTimestamp":null,"annotations":{"kubectl.kubernetes.io/last-applied-configuration":"{\"apiVersion\":\"v1\",\"data\":{\"foo\":\"dmF1bHQ6Zm9vI2Jhcg==\"},\"kind\":\"Secret\",\"metadata\":{\"annotations\":{},\"name\":\"test-secret\",\"namespace\":\"test-namespace\"},\"type\":\"Opaque\"}\n"}},"data":{"foo":"dmF1bHQ6Zm9vI2Jhcg=="},"type":"Opaque"}`,
			[]patchOperation{patchOperation{Op: "replace", Path: "/data/foo", Value: "YmFy"}},
			"",
		},
		{
			// Test valid secret defined in vault + one simple secret
			fakeVaultClient{"bar"},
			"secret/data/{{.Secret}}",
			`{"metadata":{"name":"test-secret","namespace":"test-namespace","creationTimestamp":null,"annotations":{"kubectl.kubernetes.io/last-applied-configuration":"{\"apiVersion\":\"v1\",\"data\":{\"foo\":\"dmF1bHQ6Zm9vI2Jhcg==\",\"simple\":\"test\"},\"kind\":\"Secret\",\"metadata\":{\"annotations\":{},\"name\":\"test-secret\",\"namespace\":\"test-namespace\"},\"type\":\"Opaque\"}\n"}},"data":{"foo":"dmF1bHQ6Zm9vI2Jhcg==","simple":"test"},"type":"Opaque"}`,
			[]patchOperation{patchOperation{Op: "replace", Path: "/data/foo", Value: "YmFy"}},
			"",
		},
		{
			// Test multi valid secrets defined in vault + one simple secret
			fakeVaultClient{"bar"},
			"secret/data/{{.Secret}}",
			`{"metadata":{"name":"test-secret","namespace":"test-namespace","creationTimestamp":null,"annotations":{"kubectl.kubernetes.io/last-applied-configuration":"{\"apiVersion\":\"v1\",\"data\":{\"foo\":\"dmF1bHQ6Zm9vI2Jhcg==\",\"simple\":\"test\",\"foo2\":\"dmF1bHQ6Zm9vI2JhcjI=\"},\"kind\":\"Secret\",\"metadata\":{\"annotations\":{},\"name\":\"test-secret\",\"namespace\":\"test-namespace\"},\"type\":\"Opaque\"}\n"}},"data":{"foo":"dmF1bHQ6Zm9vI2Jhcg==","simple":"test","foo2":"dmF1bHQ6Zm9vI2JhcjI="},"type":"Opaque"}`,
			[]patchOperation{patchOperation{Op: "replace", Path: "/data/foo", Value: "YmFy"}, patchOperation{Op: "replace", Path: "/data/foo2", Value: "YmFy"}},
			"",
		},
	}

	for _, test := range mutateTests {

		s := Server{
			Listen:       ":8443",
			Cert:         "",
			Key:          "",
			Vault:        test.vaultClient,
			VaultPattern: test.vaultPattern,
			Logger:       logrus.New(),
		}

		// Parse secret object
		var secret corev1.Secret
		err := json.Unmarshal([]byte(test.secret), &secret)
		if err != nil {
			t.Fatal(err)
		}

		patch, err := s.mutateSecretData(secret)
		if test.errorString == "" {
			require.Nil(t, err)
		} else {
			require.EqualError(t, err, test.errorString)
		}

		require.Equal(t, patch, test.patch)
	}
}

type fakeVaultClient struct {
	Value string
}

func (f fakeVaultClient) Read(path, key string) (string, error) {
	if f.Value == "error" {
		return "", errors.New("failed to read key in vault")
	}

	return f.Value, nil
}
