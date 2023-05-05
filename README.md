# k8s-vault-webhook

k8s-vault-webhook is a Kubernetes webhook that retrieves secrets from Hashicorp Vault. This webhook allows your Kubernetes applications to automatically access secrets stored in Vault without having to directly query Vault themselves.

## Features

- Retrieve secrets from Hashicorp Vault and inject them into Kubernetes secrets
- Configurable Vault search pattern
- Easy deployment using Helm chart
- Customizable logging format (text or JSON)

## Deploy with Helm

The simplest way to deploy the webhook is to use the provider Helm Chart: [k8s-vault-webhook chart](https://github.com/Ouest-France/k8s-vault-webhook/tree/master/charts/k8svaultwebhook)
