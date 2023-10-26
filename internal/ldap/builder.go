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

	openldapv1alpha1 "github.com/symas/openldap-operator/api/v1alpha1"
	"github.com/symas/operator-utils/k8sutils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ClientBuilder interface {
	WithReader(reader client.Reader) ClientBuilder
	WithScheme(scheme *runtime.Scheme) ClientBuilder
	WithDirectory(directory *openldapv1alpha1.LDAPDirectory) ClientBuilder
	Build(ctx context.Context) (Client, error)
}

type clientBuilderImpl struct {
	reader    client.Reader
	scheme    *runtime.Scheme
	directory *openldapv1alpha1.LDAPDirectory
}

func NewClientBuilder() ClientBuilder {
	return &clientBuilderImpl{}
}

func (b *clientBuilderImpl) WithReader(reader client.Reader) ClientBuilder {
	return &clientBuilderImpl{
		reader:    reader,
		scheme:    b.scheme,
		directory: b.directory,
	}
}

func (b *clientBuilderImpl) WithScheme(scheme *runtime.Scheme) ClientBuilder {
	return &clientBuilderImpl{
		reader:    b.reader,
		scheme:    scheme,
		directory: b.directory,
	}
}

func (b *clientBuilderImpl) WithDirectory(directory *openldapv1alpha1.LDAPDirectory) ClientBuilder {
	return &clientBuilderImpl{
		reader:    b.reader,
		scheme:    b.scheme,
		directory: directory,
	}
}

func (b *clientBuilderImpl) Build(ctx context.Context) (Client, error) {
	// Get the admin password.
	adminPasswordSecret, ok, err := b.directory.Spec.AdminPasswordSecretRef.Resolve(ctx, b.reader, b.scheme, b.directory)
	if !ok && err == nil {
		return nil, fmt.Errorf("referenced admin password secret not found")
	} else if err != nil {
		return nil, fmt.Errorf("failed to resolve admin password secret reference: %w", err)
	}

	adminPassword := string(adminPasswordSecret.(*corev1.Secret).Data["password"])

	// Get the CA certificate..
	certificateSecret, ok, err := b.directory.Spec.CertificateSecretRef.Resolve(ctx, b.reader, b.scheme, b.directory)
	if !ok && err == nil {
		return nil, fmt.Errorf("referenced certificate secret not found")
	} else if err != nil {
		return nil, fmt.Errorf("failed to resolve certificate secret reference: %w", err)
	}

	caBundle := x509.NewCertPool()
	if ok := caBundle.AppendCertsFromPEM(certificateSecret.(*corev1.Secret).Data["ca.crt"]); !ok {
		return nil, fmt.Errorf("failed to construct ca bundle")
	}

	directoryAddress := fmt.Sprintf("ldaps://%s.%s.svc.%s", b.directory.Name, b.directory.Namespace, k8sutils.GetClusterDomain())
	if b.directory.Spec.AddressOverride != "" {
		directoryAddress = b.directory.Spec.AddressOverride
	}

	baseDN := "dc=" + strings.ReplaceAll(b.directory.Spec.Domain, ".", ",dc=")

	return &clientImpl{
		directoryAddress: directoryAddress,
		caBundle:         caBundle,
		adminUsername:    "cn=admin," + baseDN,
		adminPassword:    adminPassword,
		baseDN:           baseDN,
	}, nil
}
