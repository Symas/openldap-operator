# ldap-operator

A Kubernetes operator for deploying and managing LDAP directories.

## Getting Started

### Prerequisites

* [kapp](https://carvel.dev/kapp/)

### Installing

#### Cert-Manager

```shell
kapp deploy -a cert-manager -f https://github.com/cert-manager/cert-manager/releases/download/v1.12.0/cert-manager.yaml
```

#### Operator

```shell
kapp deploy -a ldap-operator -f https://github.com/gpu-ninja/ldap-operator/releases/latest/download/ldap-operator.yaml
```

### Starting a Directory

```shell
kubectl apply -f examples -l app.kubernetes.io/component=server
```

### Managed Resources

In the examples directory, there are a few examples of managed LDAP resources that can be created by the operator (eg. Organizational Units, Users, Groups, etc).

```shell
kubectl apply -f examples -l app.kubernetes.io/component=managed-resource
```
