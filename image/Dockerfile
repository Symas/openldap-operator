FROM debian:bookworm-slim

ENV LDAP_NOFILE=1024 \
    LDAP_DEBUG_LEVEL=0 \
    LDAP_DOMAIN=example.com \
    LDAP_ORGANIZATION="Acme Widgets Inc." \
    LDAP_ADMIN_PASSWORD=admin \
    LDAP_TLS_CERT=/etc/ldap/certs/tls.crt \
    LDAP_TLS_KEY=/etc/ldap/certs/tls.key \
    LDAP_TLS_CA_CERTS=/etc/ldap/certs/ca.crt

RUN apt update \
  && apt install -y openssl slapd ldap-utils argon2

COPY bootstrap.sh /bootstrap.sh
RUN chmod +x /bootstrap.sh

# OpenLDAP config
VOLUME /etc/ldap/slapd.d
# OpenLDAP database
VOLUME /var/lib/ldap

EXPOSE 636/tcp

ENTRYPOINT ["/bin/sh", "-c"]
CMD ["ulimit -n ${LDAP_NOFILE} && slapd -u openldap -g openldap -d ${LDAP_DEBUG_LEVEL} -h 'ldaps:/// ldapi:///'"]
