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
	"fmt"
	"strings"
	"sync"

	openldapv1alpha1 "github.com/gpu-ninja/openldap-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type fakeClientBuilder struct {
	db sync.Map
}

func NewFakeClientBuilder() ClientBuilder {
	return &fakeClientBuilder{}
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
		db: &b.db,
	}, nil
}

type fakeClient struct {
	db *sync.Map
}

func (c *fakeClient) Ping() error {
	return nil
}

func (c *fakeClient) GetEntry(dn string, entry any) error {
	v, ok := c.db.Load(dn)
	if !ok {
		return fmt.Errorf("entry not found")
	}

	switch e := entry.(type) {
	case *OrganizationalUnit:
		*e = *(v.(*OrganizationalUnit))
	case *Group:
		*e = *(v.(*Group))
	case *User:
		*e = *(v.(*User))
	default:
		return fmt.Errorf("unknown entry type")
	}

	return nil
}

func (c *fakeClient) CreateOrUpdateEntry(entry any) (bool, error) {
	var exists bool
	switch e := entry.(type) {
	case *OrganizationalUnit:
		_, exists = c.db.Load(e.DistinguishedName)
		c.db.Store(e.DistinguishedName, e)
	case *Group:
		_, exists = c.db.Load(e.DistinguishedName)
		c.db.Store(e.DistinguishedName, e)
	case *User:
		_, exists = c.db.Load(e.DistinguishedName)
		c.db.Store(e.DistinguishedName, e)
	default:
		return false, fmt.Errorf("unknown entry type")
	}

	return !exists, nil
}

func (c *fakeClient) DeleteEntry(dn string, cascading bool) error {
	if cascading {
		c.db.Range(func(key, value any) bool {
			if strings.HasPrefix(key.(string), dn) {
				c.db.Delete(key)
			}

			return true
		})
	} else {
		c.db.Delete(dn)
	}

	return nil
}
