/* SPDX-License-Identifier: Apache-2.0
 *
 * Copyright 2023 Damian Peckett <damian@pecke.tt>.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package ldap_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gpu-ninja/operator-utils/name"
	"github.com/gpu-ninja/operator-utils/reference"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	openldapv1alpha1 "github.com/gpu-ninja/ldap-operator/api/v1alpha1"
	"github.com/gpu-ninja/ldap-operator/internal/ldap"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	image = "ghcr.io/gpu-ninja/openldap-operator/ldap:v0.1.1"
)

func TestClient(t *testing.T) {
	certsDir := t.TempDir()

	err := generateCertificates(certsDir)
	require.NoError(t, err)

	caCertPEM, err := os.ReadFile(filepath.Join(certsDir, "ca.crt"))
	require.NoError(t, err)

	mounts := testcontainers.ContainerMounts{
		testcontainers.ContainerMount{
			Source: testcontainers.GenericVolumeMountSource{
				Name: name.Generate("openldap-config"),
			},
			Target: "/etc/ldap/slapd.d",
		},
		testcontainers.ContainerMount{
			Source: testcontainers.GenericVolumeMountSource{
				Name: name.Generate("openldap-data"),
			},
			Target: "/var/lib/ldap",
		},
		testcontainers.ContainerMount{
			Source: testcontainers.GenericBindMountSource{
				HostPath: certsDir,
			},
			Target: "/etc/ldap/certs",
		},
	}

	req := testcontainers.ContainerRequest{
		Image:      image,
		Mounts:     mounts,
		WaitingFor: wait.ForExit(),
		Cmd:        []string{"/bootstrap.sh"},
	}

	// Bootstrap the database.
	ctx := context.Background()
	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)
	require.NoError(t, c.Terminate(ctx))

	req = testcontainers.ContainerRequest{
		Image:        image,
		ExposedPorts: []string{"636/tcp"},
		Mounts:       mounts,
		Env: map[string]string{
			"LDAP_DEBUG_LEVEL": "255",
		},
		WaitingFor: wait.ForExposedPort(),
	}

	// Start the openldap directory.
	c, err = testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)
	defer func() {
		if err := c.Terminate(ctx); err != nil {
			t.Errorf("failed to terminate container: %v", err)
		}
	}()

	directory := openldapv1alpha1.LDAPDirectory{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: openldapv1alpha1.LDAPDirectorySpec{
			Domain: "example.com",
			AdminPasswordSecretRef: reference.LocalSecretReference{
				Name: "admin-password",
			},
			CertificateSecretRef: reference.LocalSecretReference{
				Name: "directory-cert",
			},
		},
	}

	_ = corev1.AddToScheme(scheme.Scheme)
	client := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "admin-password",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"password": []byte("admin"),
		},
	}, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "directory-cert",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"ca.crt": caCertPEM,
		},
	}).Build()

	baseDN, err := directory.GetDistinguishedName(ctx, client, scheme.Scheme)
	require.NoError(t, err)

	endpoint, err := c.Endpoint(ctx, "")
	require.NoError(t, err)

	directory.Spec.AddressOverride = fmt.Sprintf("ldaps://%s", endpoint)

	ldapClient, err := ldap.NewClientBuilder().
		WithReader(client).
		WithScheme(scheme.Scheme).
		WithDirectory(&directory).
		Build(ctx)
	require.NoError(t, err)

	err = ldapClient.Ping()
	require.NoError(t, err)

	t.Run("Organizational Units", func(t *testing.T) {
		organizationalUnitName := name.Generate("users")
		dn := fmt.Sprintf("ou=%s,%s", organizationalUnitName, baseDN)

		created, err := ldapClient.CreateOrUpdateEntry(&ldap.OrganizationalUnit{
			DistinguishedName: dn,
			Name:              organizationalUnitName,
			Description:       "Test Users",
		})
		assert.True(t, created)
		assert.NoError(t, err)

		username := name.Generate("user")
		created, err = ldapClient.CreateOrUpdateEntry(&ldap.User{
			DistinguishedName: fmt.Sprintf("uid=%s,%s", username, dn),
			Username:          username,
			Name:              "Test User",
			Surname:           "User",
		})
		assert.True(t, created)
		assert.NoError(t, err)

		var ou ldap.OrganizationalUnit
		err = ldapClient.GetEntry(dn, &ou)
		assert.NoError(t, err)

		assert.Equal(t, dn, ou.DistinguishedName)
		assert.Equal(t, organizationalUnitName, ou.Name)
		assert.Equal(t, "Test Users", ou.Description)

		// Remove the description.
		created, err = ldapClient.CreateOrUpdateEntry(&ldap.OrganizationalUnit{
			DistinguishedName: dn,
			Name:              organizationalUnitName,
		})
		assert.False(t, created)
		assert.NoError(t, err)

		err = ldapClient.GetEntry(dn, &ou)
		assert.NoError(t, err)

		assert.Equal(t, "", ou.Description)

		err = ldapClient.DeleteEntry(dn, true)
		assert.NoError(t, err)

		err = ldapClient.GetEntry(dn, &ou)
		assert.Error(t, err)
	})

	t.Run("Groups", func(t *testing.T) {
		groupName := name.Generate("admins")
		dn := fmt.Sprintf("cn=%s,%s", groupName, baseDN)

		created, err := ldapClient.CreateOrUpdateEntry(&ldap.Group{
			DistinguishedName: dn,
			Name:              groupName,
			Description:       "Test Admins",
			Members:           []string{"cn=admin," + baseDN, "cn=other,ou=users," + baseDN},
		})
		assert.True(t, created)
		assert.NoError(t, err)

		var group ldap.Group
		err = ldapClient.GetEntry(dn, &group)
		assert.NoError(t, err)

		assert.Equal(t, dn, group.DistinguishedName)
		assert.Equal(t, groupName, group.Name)
		assert.Equal(t, "Test Admins", group.Description)
		assert.Len(t, group.Members, 2)

		// Remove the description and a member.
		created, err = ldapClient.CreateOrUpdateEntry(&ldap.Group{
			DistinguishedName: dn,
			Name:              groupName,
			Members: []string{
				"cn=admin," + baseDN,
			},
		})
		assert.False(t, created)
		assert.NoError(t, err)

		err = ldapClient.GetEntry(dn, &group)
		assert.NoError(t, err)

		assert.Equal(t, "", group.Description)
		assert.Len(t, group.Members, 1)

		err = ldapClient.DeleteEntry(dn, false)
		assert.NoError(t, err)

		err = ldapClient.GetEntry(dn, &group)
		assert.Error(t, err)
	})

	t.Run("Users", func(t *testing.T) {
		username := name.Generate("other")
		dn := fmt.Sprintf("uid=%s,%s", username, baseDN)

		created, err := ldapClient.CreateOrUpdateEntry(&ldap.User{
			DistinguishedName: dn,
			Username:          username,
			Name:              "John Doe",
			Surname:           "Doe",
			Email:             "john@example.com",
			Password:          "changeme",
		})
		assert.True(t, created)
		assert.NoError(t, err)

		var user ldap.User
		err = ldapClient.GetEntry(dn, &user)
		assert.NoError(t, err)

		assert.Equal(t, dn, user.DistinguishedName)
		assert.Equal(t, username, user.Username)
		assert.Equal(t, "John Doe", user.Name)
		assert.Equal(t, "Doe", user.Surname)
		assert.Equal(t, "john@example.com", user.Email)
		assert.True(t, strings.HasPrefix(user.Password, "{ARGON2}$argon2"))

		passwordHashBeforeModify := user.Password

		// Remove the email.
		created, err = ldapClient.CreateOrUpdateEntry(&ldap.User{
			DistinguishedName: dn,
			Username:          username,
			Name:              "John Doe",
			Surname:           "Doe",
			Password:          "changeme",
		})
		assert.False(t, created)
		assert.NoError(t, err)

		err = ldapClient.GetEntry(dn, &user)
		assert.NoError(t, err)

		assert.Equal(t, "", user.Email)
		assert.Equal(t, passwordHashBeforeModify, user.Password)

		err = ldapClient.DeleteEntry(dn, false)
		assert.NoError(t, err)

		err = ldapClient.GetEntry(dn, &user)
		assert.Error(t, err)
	})
}
