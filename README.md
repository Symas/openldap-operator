# openldap-operator

A Kubernetes operator for deploying and managing OpenLDAP directories.

## Getting Started

### Prerequisites

* [cert-manager](https://cert-manager.io/docs/installation/)
* [kapp](https://carvel.dev/kapp/)

### Installing

```shell
kapp deploy -a dex-operator -f https://github.com/gpu-ninja/openldap-operator/releases/latest/download/openldap-operator.yaml
```

### Starting an OpenLDAP directory

```shell
kubectl apply -f examples -l app.kubernetes.io/component=server
```

### Managed Resources

In the examples directory, there are a few examples of managed LDAP resources that can be created by the operator (eg. Organizational Units, Users, Groups, etc).

```shell
kubectl apply -f examples -l app.kubernetes.io/component=managed-resource
```
