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

package directory

import (
	"context"
	"crypto/x509"
	"fmt"
	"strings"

	openldapv1alpha1 "github.com/gpu-ninja/openldap-operator/api/v1alpha1"
	"github.com/gpu-ninja/operator-utils/k8sutils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ClientBuilder interface {
	WithReader(reader client.Reader) ClientBuilder
	WithScheme(scheme *runtime.Scheme) ClientBuilder
	WithServer(server *openldapv1alpha1.LDAPServer) ClientBuilder
	Build(ctx context.Context) (Client, error)
}

type clientBuilderImpl struct {
	reader client.Reader
	scheme *runtime.Scheme
	server *openldapv1alpha1.LDAPServer
}

func NewClientBuilder() ClientBuilder {
	return &clientBuilderImpl{}
}

func (b *clientBuilderImpl) WithReader(reader client.Reader) ClientBuilder {
	return &clientBuilderImpl{
		reader: reader,
		scheme: b.scheme,
		server: b.server,
	}
}

func (b *clientBuilderImpl) WithScheme(scheme *runtime.Scheme) ClientBuilder {
	return &clientBuilderImpl{
		reader: b.reader,
		scheme: scheme,
		server: b.server,
	}
}

func (b *clientBuilderImpl) WithServer(server *openldapv1alpha1.LDAPServer) ClientBuilder {
	return &clientBuilderImpl{
		reader: b.reader,
		scheme: b.scheme,
		server: server,
	}
}

func (b *clientBuilderImpl) Build(ctx context.Context) (Client, error) {
	// Get the admin password.
	adminPasswordSecret, err := b.server.Spec.AdminPasswordSecretRef.Resolve(ctx, b.reader, b.scheme, b.server)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve admin password secret reference: %w", err)
	}

	adminPassword := string(adminPasswordSecret.(*corev1.Secret).Data["password"])

	// Get the CA certificate..
	certificateSecret, err := b.server.Spec.CertificateSecretRef.Resolve(ctx, b.reader, b.scheme, b.server)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve certificate secret reference: %w", err)
	}

	caBundle := x509.NewCertPool()
	if ok := caBundle.AppendCertsFromPEM(certificateSecret.(*corev1.Secret).Data["ca.crt"]); !ok {
		return nil, fmt.Errorf("failed to construct ca bundle")
	}

	serverAddress := fmt.Sprintf("ldaps://%s.%s.svc.%s", b.server.Name, b.server.Namespace, k8sutils.GetClusterDomain())
	if b.server.Spec.AddressOverride != "" {
		serverAddress = b.server.Spec.AddressOverride
	}

	baseDN := "dc=" + strings.ReplaceAll(b.server.Spec.Domain, ".", ",dc=")

	return &clientImpl{
		serverAddress: serverAddress,
		caBundle:      caBundle,
		adminUsername: "cn=admin," + baseDN,
		adminPassword: adminPassword,
		baseDN:        baseDN,
	}, nil
}
