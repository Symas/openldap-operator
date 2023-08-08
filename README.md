# openldap-operator

A Kubernetes operator for deploying and managing OpenLDAP directories.

## Getting Started

### Prerequisites

* [cert-manager](https://cert-manager.io/docs/installation/)
* [kapp](https://carvel.dev/kapp/)

### Installing

```shell
kapp deploy -a openldap-operator -f config/
```

### Starting an OpenLDAP directory

```shell
kubectl apply -f examples/issuer-selfsigned.yaml \
  -f examples/certificate-demo.yaml \
  -f examples/secret-admin-password.yaml \
  -f examples/ldapserver-demo.yaml
```

### Managed Resources

In the examples directory, there are a few examples of managed LDAP resources that can be created by the operator (eg. Organizational Units, Users, Groups, etc).