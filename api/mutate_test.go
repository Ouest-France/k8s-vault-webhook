package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

// Fake Vault client for testing
type fakeVaultClient struct {
	Value  string
	Absent bool
}

// Fake Vault read method for testing
func (f fakeVaultClient) Read(path, key string) (string, error) {
	if f.Value == "error" {
		return "", errors.New("failed to read key in vault")
	}

	if f.Absent {
		return fmt.Sprintf("Secret %q does not exist in Vault", f.Value), nil
	}

	return f.Value, nil
}

func TestServer_mutateSecretData(t *testing.T) {

	var mutateTests = []struct {
		description  string
		vaultClient  VaultClient
		vaultPattern string
		secret       string
		patch        []patchOperation
		errorString  string
	}{
		{
			"Test secret that doesn't need to be mutated",
			fakeVaultClient{},
			"secret/data/{{.Secret}}",
			`{"metadata":{"name":"test-secret","namespace":"test-namespace","creationTimestamp":null,"annotations":{"kubectl.kubernetes.io/last-applied-configuration":"{\"apiVersion\":\"v1\",\"data\":{\"foo\":\"YmFy\"},\"kind\":\"Secret\",\"metadata\":{\"annotations\":{},\"name\":\"test-secret\",\"namespace\":\"test-namespace\"},\"type\":\"Opaque\"}\n"}},"data":{"foo":"YmFy"},"type":"Opaque"}`,
			[]patchOperation{},
			"",
		},
		{
			"Test secret with invalid path pattern",
			fakeVaultClient{},
			"secret/data/{{.Secret}}",
			`{"metadata":{"name":"test-secret","namespace":"test-namespace","creationTimestamp":null,"annotations":{"kubectl.kubernetes.io/last-applied-configuration":"{\"apiVersion\":\"v1\",\"data\":{\"foo\":\"dmF1bHQ6YmFy\"},\"kind\":\"Secret\",\"metadata\":{\"annotations\":{},\"name\":\"test-secret\",\"namespace\":\"test-namespace\"},\"type\":\"Opaque\"}\n"}},"data":{"foo":"dmF1bHQ6YmFy"},"type":"Opaque"}`,
			[]patchOperation{},
			"vault placeholder 'vault:bar' doesn't match regex '^vault:(.*)#(.*)$'",
		},
		{
			"Test secret with empty name",
			fakeVaultClient{},
			"secret/data/{{.Secret}}",
			`{"metadata":{"name":"","namespace":"test-namespace","creationTimestamp":null,"annotations":{"kubectl.kubernetes.io/last-applied-configuration":"{\"apiVersion\":\"v1\",\"data\":{\"foo\":\"dmF1bHQ6Zm9vI2Jhcg==\"},\"kind\":\"Secret\",\"metadata\":{\"annotations\":{},\"name\":\"\",\"namespace\":\"test-namespace\"},\"type\":\"Opaque\"}\n"}},"data":{"foo":"dmF1bHQ6Zm9vI2Jhcg=="},"type":"Opaque"}`,
			[]patchOperation{},
			"secret field name cannot be empty",
		},
		{
			"Test secret with empty namespace",
			fakeVaultClient{},
			"secret/data/{{.Secret}}",
			`{"metadata":{"name":"test-secret","namespace":"","creationTimestamp":null,"annotations":{"kubectl.kubernetes.io/last-applied-configuration":"{\"apiVersion\":\"v1\",\"data\":{\"foo\":\"dmF1bHQ6Zm9vI2Jhcg==\"},\"kind\":\"Secret\",\"metadata\":{\"annotations\":{},\"name\":\"test-secret\",\"namespace\":\"\"},\"type\":\"Opaque\"}\n"}},"data":{"foo":"dmF1bHQ6Zm9vI2Jhcg=="},"type":"Opaque"}`,
			[]patchOperation{},
			"secret field namespace cannot be empty",
		},
		{
			"Test invalid vault pattern",
			fakeVaultClient{},
			"secret/data/{{.Secret}}/{{.InvalidKey}}",
			`{"metadata":{"name":"test-secret","namespace":"test-namespace","creationTimestamp":null,"annotations":{"kubectl.kubernetes.io/last-applied-configuration":"{\"apiVersion\":\"v1\",\"data\":{\"foo\":\"dmF1bHQ6Zm9vI2Jhcg==\"},\"kind\":\"Secret\",\"metadata\":{\"annotations\":{},\"name\":\"test-secret\",\"namespace\":\"test-namespace\"},\"type\":\"Opaque\"}\n"}},"data":{"foo":"dmF1bHQ6Zm9vI2Jhcg=="},"type":"Opaque"}`,
			[]patchOperation{},
			"failed to execute template function on vault path pattern",
		},
		{
			"Test secret that doesn't exists in vault",
			fakeVaultClient{Absent: true, Value: "absentsecret"},
			"secret/data/{{.Secret}}",
			`{"metadata":{"name":"test-secret","namespace":"test-namespace","creationTimestamp":null,"annotations":{"kubectl.kubernetes.io/last-applied-configuration":"{\"apiVersion\":\"v1\",\"data\":{\"foo\":\"dmF1bHQ6Zm9vI2Jhcg==\"},\"kind\":\"Secret\",\"metadata\":{\"annotations\":{},\"name\":\"test-secret\",\"namespace\":\"test-namespace\"},\"type\":\"Opaque\"}\n"}},"data":{"foo":"dmF1bHQ6Zm9vI2Jhcg=="},"type":"Opaque"}`,
			[]patchOperation{{Op: "replace", Path: "/data/foo", Value: "U2VjcmV0ICJhYnNlbnRzZWNyZXQiIGRvZXMgbm90IGV4aXN0IGluIFZhdWx0"}},
			"",
		},
		{
			"Test valid secret defined in vault",
			fakeVaultClient{Value: "bar"},
			"secret/data/{{.Secret}}",
			`{"metadata":{"name":"test-secret","namespace":"test-namespace","creationTimestamp":null,"annotations":{"kubectl.kubernetes.io/last-applied-configuration":"{\"apiVersion\":\"v1\",\"data\":{\"foo\":\"dmF1bHQ6Zm9vI2Jhcg==\"},\"kind\":\"Secret\",\"metadata\":{\"annotations\":{},\"name\":\"test-secret\",\"namespace\":\"test-namespace\"},\"type\":\"Opaque\"}\n"}},"data":{"foo":"dmF1bHQ6Zm9vI2Jhcg=="},"type":"Opaque"}`,
			[]patchOperation{{Op: "replace", Path: "/data/foo", Value: "YmFy"}},
			"",
		},
		{
			"Test valid secret defined in vault + one simple secret",
			fakeVaultClient{Value: "bar"},
			"secret/data/{{.Secret}}",
			`{"metadata":{"name":"test-secret","namespace":"test-namespace","creationTimestamp":null,"annotations":{"kubectl.kubernetes.io/last-applied-configuration":"{\"apiVersion\":\"v1\",\"data\":{\"foo\":\"dmF1bHQ6Zm9vI2Jhcg==\",\"simple\":\"test\"},\"kind\":\"Secret\",\"metadata\":{\"annotations\":{},\"name\":\"test-secret\",\"namespace\":\"test-namespace\"},\"type\":\"Opaque\"}\n"}},"data":{"foo":"dmF1bHQ6Zm9vI2Jhcg==","simple":"test"},"type":"Opaque"}`,
			[]patchOperation{{Op: "replace", Path: "/data/foo", Value: "YmFy"}},
			"",
		},
		{
			"Test multi valid secrets defined in vault + one simple secret",
			fakeVaultClient{Value: "bar"},
			"secret/data/{{.Secret}}",
			`{"metadata":{"name":"test-secret","namespace":"test-namespace","creationTimestamp":null,"annotations":{"kubectl.kubernetes.io/last-applied-configuration":"{\"apiVersion\":\"v1\",\"data\":{\"foo\":\"dmF1bHQ6Zm9vI2Jhcg==\",\"simple\":\"test\",\"foo2\":\"dmF1bHQ6Zm9vI2JhcjI=\"},\"kind\":\"Secret\",\"metadata\":{\"annotations\":{},\"name\":\"test-secret\",\"namespace\":\"test-namespace\"},\"type\":\"Opaque\"}\n"}},"data":{"foo":"dmF1bHQ6Zm9vI2Jhcg==","simple":"test","foo2":"dmF1bHQ6Zm9vI2JhcjI="},"type":"Opaque"}`,
			[]patchOperation{{Op: "replace", Path: "/data/foo", Value: "YmFy"}, {Op: "replace", Path: "/data/foo2", Value: "YmFy"}},
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
			require.Nil(t, err, test.description)
		} else {
			require.EqualError(t, err, test.errorString, test.description)
		}

		// Sort patch and test.patch to avoid random order
		sort.Slice(patch, func(i, j int) bool {
			return patch[i].Path < patch[j].Path
		})
		sort.Slice(test.patch, func(i, j int) bool {
			return test.patch[i].Path < test.patch[j].Path
		})

		require.Equal(t, patch, test.patch, test.description)
	}
}
