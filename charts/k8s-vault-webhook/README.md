# k8s-vault-webhook

K8s-vault-webhook is a kubernetes webhook to get secrets from Hashicorp Vault.

## Prerequisites

- Kubernetes 1.9+
- Helm 2.13+

## Installing the Chart

If not already done you first have to add the chart repo:

```console
$ helm repo add ouestfrance https://charts.ouest-france.fr/
```

To install the chart with the release name `my-release`:

```console
$ helm install --name my-release ouestfrance/k8s-vault-webhook
```

The command deploys WordPress on the Kubernetes cluster in the default configuration. The [Parameters](#parameters) section lists the parameters that can be configured during installation.

## Uninstalling the Chart

To uninstall/delete the `my-release` deployment:

```console
$ helm delete my-release
```

The command removes all the Kubernetes components associated with the chart and deletes the release.

## Parameters

The following table lists the configurable parameters of the k8s-vault-webhook chart and their default values.

|            Parameter                          |                                  Description                    |                           Default                            |
| --------------------------------------------- | --------------------------------------------------------------- | ------------------------------------------------------------ |
| `replicaCount`                                | number of pod replicas                                          | `2`                                                          |
| `serviceAccount`                              | service account name                                            | `k8s-vault-webhook`                                          |
| `image.repository`                            | k8s-vault-webhook image repository                              | `ouestfrance/k8s-vault-webhook`                              |
| `image.tag`                                   | k8s-vault-webhook image tag                                     | `latest`                                                     |
| `image.pullPolicy`                            | k8s-vault-webhook image pull policy                             | `Always`                                                     |
| `loglevel`                                    | k8s-vault-webhook log level                                     | `info`                                                       |
| `logformat`                                   | k8s-vault-webhook log format (json or text)                     | `json`                                                       |
| `basicauth`                                   | k8s-vault-webhook basicauth list of authorized users            | `[]`                                                         |
| `vault.address`                               | vault server address                                            | `http://127.0.0.1:8200`                                      |
| `vault.pattern`                               | k8s-vault-webhook vault path template pattern                   | `secret/data/{{.Namespace}}/{{.Secret}}`                     |
| `resources.limits.cpu`                        | k8s-vault-webhook container cpu limit                           | `100m`                                                       |
| `resources.limits.memory`                     | k8s-vault-webhook container memory limit                        | `128Mi`                                                      |
| `resources.requests.cpu`                      | k8s-vault-webhook container cpu request                         | `100m`                                                       |
| `resources.requests.memory`                   | k8s-vault-webhook container memory request                      | `64Mi`                                                       |
| `vault.agent.mount`                           | vault kubernetes auth mount name                                | `kubernetes`                                                 |
| `vault.agent.role`                            | vault kubernetes auth role                                      | `k8s-vault-webhook`                                          |
| `vault.agent.resources.limits.cpu`            | vault-agent container cpu limit                                 | `100m`                                                       |
| `vault.agent.resources.limits.memory`         | vault-agent container memory limit                              | `128Mi`                                                      |
| `vault.agent.resources.requests.cpu`          | vault-agent container cpu request                               | `100m`                                                       |
| `vault.agent.resources.requests.memory`       | vault-agent container memory request                            | `64Mi`                                                       |
| `webhook.failurePolicy`                       | mutating webhook failure policy                                 | `Fail`                                                       |
| `webhook.namespaceSelector.matchLabels`       | mutating webhook labels for namespace selector                  | `{}`                                                         |
| `webhook.namespaceSelector.matchExpressions`  | mutating webhook expressions for namespace selector             | `[]`                                                         |
| `nameOverride`                                | chart name override                                             | ``                                                           |
| `fullnameOverride`                            | chart fullname override                                         | ``                                                           |
| `service.type`                                | service type                                                    | `ClusterIP`                                                  |
| `service.port`                                | service port                                                    | `443`                                                        |
| `nodeSelector`                                | pod node selector                                               | `{}`                                                         |
| `tolerations`                                 | pod tolerations                                                 | `[]`                                                         |
| `affinity`                                    | pod affinity                                                    | `{}`                                                         |
