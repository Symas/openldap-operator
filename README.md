# ldap-operator

A Kubernetes operator for deploying and managing LDAP directories.

## Getting Started

### Prerequisites

* [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
* [kapp](https://carvel.dev/kapp/)

### Installing

#### Dependencies

```shell
PROMETHEUS_VERSION="v0.68.0"
CERT_MANAGER_VERSION="v1.12.0"

kapp deploy -y -a prometheus-crds -f "https://github.com/prometheus-operator/prometheus-operator/releases/download/${PROMETHEUS_VERSION}/stripped-down-crds.yaml"
kapp deploy -y -a cert-manager -f "https://github.com/cert-manager/cert-manager/releases/download/${CERT_MANAGER_VERSION}/cert-manager.yaml"
```

#### Operator

```shell
kapp deploy -a ldap-operator -f https://github.com/gpu-ninja/ldap-operator/releases/latest/download/ldap-operator.yaml
```

### Custom Resources

#### Start an LDAP Directory

```shell
kubectl apply -f examples -l app.kubernetes.io/component=server
```

#### Create Managed LDAP Resources

In the examples directory, there are a few examples of managed LDAP resources that can be created by the operator (eg. Organizational Units, Users, Groups, etc).

```shell
kubectl apply -f examples -l app.kubernetes.io/component=managed-resource
```
