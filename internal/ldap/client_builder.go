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

package ldap

import (
	"context"
	"crypto/x509"
	"fmt"
	"strings"

	ldapv1alpha1 "github.com/gpu-ninja/ldap-operator/api/v1alpha1"
	"github.com/gpu-ninja/operator-utils/k8sutils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ClientBuilder interface {
	WithClient(client client.Client) ClientBuilder
	WithScheme(scheme *runtime.Scheme) ClientBuilder
	WithDirectory(directory *ldapv1alpha1.LDAPDirectory) ClientBuilder
	Build(ctx context.Context) (Client, error)
}

type clientBuilderImpl struct {
	client    client.Client
	scheme    *runtime.Scheme
	directory *ldapv1alpha1.LDAPDirectory
}

func NewClientBuilder() ClientBuilder {
	return &clientBuilderImpl{}
}

func (b *clientBuilderImpl) WithClient(client client.Client) ClientBuilder {
	return &clientBuilderImpl{
		client:    client,
		scheme:    b.scheme,
		directory: b.directory,
	}
}

func (b *clientBuilderImpl) WithScheme(scheme *runtime.Scheme) ClientBuilder {
	return &clientBuilderImpl{
		client:    b.client,
		scheme:    scheme,
		directory: b.directory,
	}
}

func (b *clientBuilderImpl) WithDirectory(directory *ldapv1alpha1.LDAPDirectory) ClientBuilder {
	return &clientBuilderImpl{
		client:    b.client,
		scheme:    b.scheme,
		directory: directory,
	}
}

func (b *clientBuilderImpl) Build(ctx context.Context) (Client, error) {
	adminPasswordSecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("ldap-%s-admin-password", b.directory.Name),
			Namespace: b.directory.Namespace,
		},
	}

	// Get the admin password.
	if err := b.client.Get(ctx, client.ObjectKeyFromObject(&adminPasswordSecret), &adminPasswordSecret); err != nil {
		return nil, fmt.Errorf("failed to get admin password secret: %w", err)
	}

	// Get the CA certificate.
	certificateSecret, ok, err := b.directory.Spec.CertificateSecretRef.Resolve(ctx, b.client, b.scheme, b.directory)
	if !ok && err == nil {
		return nil, fmt.Errorf("referenced certificate secret not found")
	} else if err != nil {
		return nil, fmt.Errorf("failed to resolve certificate secret reference: %w", err)
	}

	caBundle := x509.NewCertPool()
	if ok := caBundle.AppendCertsFromPEM(certificateSecret.(*corev1.Secret).Data["ca.crt"]); !ok {
		return nil, fmt.Errorf("failed to construct ca bundle")
	}

	directoryAddress := fmt.Sprintf("ldaps://ldap-%s.%s.svc.%s", b.directory.Name, b.directory.Namespace, k8sutils.GetClusterDomain())
	if b.directory.Spec.AddressOverride != "" {
		directoryAddress = b.directory.Spec.AddressOverride
	}

	baseDN := "dc=" + strings.ReplaceAll(b.directory.Spec.Domain, ".", ",dc=")

	return &clientImpl{
		directoryAddress: directoryAddress,
		caBundle:         caBundle,
		adminUsername:    "cn=admin," + baseDN,
		adminPassword:    string(adminPasswordSecret.Data["password"]),
		baseDN:           baseDN,
	}, nil
}
