#!/bin/bash
set -eu

if [ ! -e /var/lib/ldap/bootstrapped ]; then
  echo 'Configuring slapd'

  cat <<EOF | debconf-set-selections
# Organization name
slapd shared/organization string ${LDAP_ORGANIZATION}
# DNS domain name
slapd slapd/domain string ${LDAP_DOMAIN}
# Dump databases to file on upgrade (always, when needed, never)
slapd slapd/dump_database select when needed
# Directory to use for dumped databases
slapd slapd/dump_database_destdir string /var/lib/ldap/backups/slapd-VERSION
# Encrypted admin password
slapd slapd/internal/adminpw password ${LDAP_ADMIN_PASSWORD}
# Generated admin password
slapd slapd/internal/generated_adminpw password ${LDAP_ADMIN_PASSWORD}
# Administrator password
slapd slapd/password1 password ${LDAP_ADMIN_PASSWORD}
# Confirm password
slapd slapd/password2 password ${LDAP_ADMIN_PASSWORD}
# Do you want the database to be removed when slapd is purged?
slapd slapd/purge_database boolean false
# Move old database
slapd slapd/move_old_database boolean true
# Omit OpenLDAP server configuration
slapd slapd/no_configuration boolean false
# Retry configuration
slapd slapd/invalid_config boolean true
EOF

  dpkg-reconfigure -f noninteractive slapd

  echo 'Enabling Argon2 password hashing'

  cat <<EOF | slapmodify -n 0
dn: cn=module{0},cn=config
changetype: modify
add: olcModuleLoad
olcModuleLoad: /usr/lib/ldap/argon2.so
EOF

  cat <<EOF | slapmodify -n 0
dn: olcDatabase={-1}frontend,cn=config
changetype: modify
replace: olcPasswordHash
olcPasswordHash: {ARGON2}
EOF

  LDAP_ADMIN_PASSWORD_HASH=$(echo -n ${LDAP_ADMIN_PASSWORD} | argon2 $(openssl rand -hex 16) -e)

  cat <<EOF | slapmodify -n 0
dn: olcDatabase={1}mdb,cn=config
changetype: modify
replace: olcRootPW
olcRootPW: {ARGON2}${LDAP_ADMIN_PASSWORD_HASH}
EOF

  if [ -v LDAP_TLS_CERT ]; then
    echo 'Configuring slapd TLS'

    cat <<EOF | slapmodify -n 0
dn: cn=config
changetype: modify
replace: olcTLSCertificateFile
olcTLSCertificateFile: ${LDAP_TLS_CERT}
-
replace: olcTLSCertificateKeyFile
olcTLSCertificateKeyFile: ${LDAP_TLS_KEY}
-
replace: olcTLSCACertificateFile
olcTLSCACertificateFile: ${LDAP_TLS_CA_CERTS}
EOF
  fi

  chown -R openldap:openldap /etc/ldap/slapd.d /var/lib/ldap

  touch /var/lib/ldap/bootstrapped
fi