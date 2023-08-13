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

	openldapv1alpha1 "github.com/gpu-ninja/openldap-operator/api/v1alpha1"
	"github.com/stretchr/testify/mock"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type fakeClientBuilder struct {
	m *mock.Mock
}

func NewFakeClientBuilder(m *mock.Mock) ClientBuilder {
	return &fakeClientBuilder{
		m: m,
	}
}

func (b *fakeClientBuilder) WithReader(_ client.Reader) ClientBuilder {
	return b
}

func (b *fakeClientBuilder) WithScheme(_ *runtime.Scheme) ClientBuilder {
	return b
}

func (b *fakeClientBuilder) WithServer(_ *openldapv1alpha1.LDAPServer) ClientBuilder {
	return b
}

func (b *fakeClientBuilder) Build(_ context.Context) (Client, error) {
	return &fakeClient{
		Mock: b.m,
	}, nil
}

type fakeClient struct {
	*mock.Mock
}

func (c *fakeClient) Ping() error {
	args := c.Called()
	return args.Error(0)
}

func (c *fakeClient) GetEntry(dn string, entry any) error {
	args := c.Called(dn, entry)
	return args.Error(0)
}

func (c *fakeClient) CreateOrUpdateEntry(entry any) (bool, error) {
	args := c.Called(entry)
	return args.Bool(0), args.Error(1)
}

func (c *fakeClient) DeleteEntry(dn string, cascading bool) error {
	args := c.Called(dn, cascading)
	return args.Error(0)
}
