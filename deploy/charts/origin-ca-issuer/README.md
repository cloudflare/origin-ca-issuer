# origin-ca-issuer

origin-ca-issuer is a Kubernetes addon to automate issuance and renewals of Cloudflare Origin CA certificates with cert-manager.

## Prerequisites

* Kubernetes 1.16+
* cert-manager 1.0.0+

## Installing the Chart

Before installing the chart, you must first install [cert-manager](https://cert-manager.io/docs/installation/), and the origin-ca-issuer CustomResourceDefinition resources.

```shell
VERSION="v0.6.1"
kubectl apply -f https://raw.githubusercontent.com/cloudflare/origin-ca-issuer/${VERSION}/deploy/crds/cert-manager.k8s.cloudflare.com_originissuers.yaml
```

To install the chart with the release name `my-release`:

``` shell
helm install --name my-release --namespace origin-ca-issuer .
```

In order to begin issuer certificates from the Cloudflare Origin CA you will need to setup an OriginIssuer. For more information, see the [documentation](https://github.com/cloudflare/origin-ca-issuer/blob/trunk/README.org).

## Uninstalling the Chart

To uninstall/delete the `my-release` deployment:

``` shell
helm delete my-release
```
If you want to completely uninstall origin-ca-issuer from your cluster, you also need to delete the previously installed CustomResourceDefinition resources:

``` shell
VERSION="v0.6.1"
kubectl delete -f https://raw.githubusercontent.com/cloudflare/origin-ca-issuer/${VERSION}/deploy/crds/cert-manager.k8s.cloudflare.com_originissuers.yaml
```

## Configuration

The following table lists the configurable parameters of the origin-ca-issuer chart and their default values.

| Parameter                             | Description                                                                             | Default                          |
|---------------------------------------|-----------------------------------------------------------------------------------------|----------------------------------|
| `global.imagePullSecrets`             | Reference to one or more secrets to be used when pulling images                         | `[]`                             |
| `global.rbac.create`                  | If `true`, create and use RBAC resources                                                | `true`                           |
| `global.priorityClassName`            | Priority class name for origin-ca-issuer pods                                           | `""`                             |
| `image.repository`                    | Image repository                                                                        | `cloudflare/origin-ca-issuer`    |
| `image.tag`                           | Image tag                                                                               | `""`                             |
| `image.digest`                        | Image digest                                                                            | `"sha256:{{ MANIFEST_DIGEST }}"` |
| `image.pullPolicy`                    | Image pull policy                                                                       | `Always`                         |
| `controller.deploymentAnnotations`    | Annotations to add to the origin-ca-issuer deployment                                   | `{}`                             |
| `controller.deploymentLabels`         | Labels to add to the origin-ca-issuer deployment                                        | `{}`                             |
| `controller.podAnntoations`           | Annotations to add to the origin-ca-issuer pods                                         | `{}`                             |
| `controller.podLabels`                | Labels to add to the origin-ca-issuer pods.                                             | `{}`                             |
| `controller.replicaCount`             | Number of origin-ca-issuer controller replicas                                          | `1`                              |
| `controller.featureGates`             | Comma-seperated list of feature gates to enable on the controller pold                  | `""`                             |
| `controller.extraArgs`                | Optional flags for origin-ca-issuer                                                     | `[]`                             |
| `controller.extraEnv`                 | Optional environment variables for origin-ca-issuer                                     | `[]`                             |
| `controller.serviceAccount.enable`    | If `true`, create a new service account                                                 | `true`                           |
| `controller.serviceAccount.name`      | Service account to be used. If not set, a name is generated using the fullname template |                                  |
| `controller.volumes`                  | Optional volumes for origin-ca-issuer                                                   | `[]`                             |
| `controller.volumeMounts`             | Optional volume mounts for origin-ca-issuer                                             | `[]`                             |
| `controller.securityContext`          | Optional security context. The YAML block should adhere to the SecurityContext spec     | `{}`                             |
| `controller.containerSecurityContext` | Optional container security context                                                     | `{}`                             |
| `controller.nodeSelector`             | Node labels for pod assignment                                                          | `{}`                             |
| `controller.affinity`                 | Node (anti-)affinity for pod assignemt                                                  | `{}`                             |
| `controller.tolerations`              | Node tolerations for pod assignment                                                     | `{}`                             |
| `controller.disableApprovedCheck`     | Disable waiting for CertificateRequests to be Approved before signing                   | `false`                          |

Specify each parameter using the `--set key=value[,key=value]` argument to `helm install`.

Alternatively, a YAML value that specifies the values for the above parameters can be provided while installing the chart. For example.

``` shell
helm install --name my-release -f values.yaml .
```

## Contributing

This chart is maintained at [github.com/cloudflare/origin-ca-issuer](https://github.com/cloudflare/origin-ca-issuer).
