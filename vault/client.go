package vault

import (
	"fmt"
	"io/ioutil"

	vault "github.com/hashicorp/vault/api"
)

type Client struct {
	Client *vault.Client
	Token  string
}

func NewClient(address, tokenPath string) (Client, error) {
	vc, err := vault.NewClient(&vault.Config{Address: address})
	if err != nil {
		return Client{}, err
	}

	return Client{Client: vc, Token: tokenPath}, nil
}

func (c Client) Read(path, key string) (string, error) {

	// Load token from disk
	err := c.refreshToken()
	if err != nil {
		return "", fmt.Errorf("failed to refresh token: %s", err)
	}

	// Read vault secret
	secret, err := c.Client.Logical().Read(path)
	if err != nil {
		return "", fmt.Errorf("failed to read secret at %q: %s", path, err)
	}
	if secret == nil {
		return "", fmt.Errorf("secret %q does not exist", path)
	}

	// Check data key for KV2 is present
	_, ok := secret.Data["data"]
	if !ok {
		return "", fmt.Errorf("failed to read secret at %q: no data returned", path)
	}

	// Check if requested key is present
	data, ok := secret.Data["data"].(map[string]interface{})[key]
	if !ok || data == nil {
		return "", fmt.Errorf("key %q not found", key)
	}

	return data.(string), nil
}

func (c Client) refreshToken() error {
	token, err := ioutil.ReadFile(c.Token)
	if err != nil {
		return fmt.Errorf("failed to read token file: %s", err)
	}

	c.Client.SetToken(string(token))
	return nil
}
