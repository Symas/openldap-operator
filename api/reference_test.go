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

package api_test

import (
	"context"
	"testing"

	"github.com/gpu-ninja/openldap-operator/api"
	openldapv1alpha1 "github.com/gpu-ninja/openldap-operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestResolveGenericReference(t *testing.T) {
	clientScheme := runtime.NewScheme()
	_ = corev1.AddToScheme(clientScheme)
	_ = openldapv1alpha1.AddToScheme(clientScheme)

	reader := fake.NewClientBuilder().WithScheme(clientScheme).WithObjects(&openldapv1alpha1.LDAPServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo",
			Namespace: "openldap",
		},
		Spec: openldapv1alpha1.LDAPServerSpec{
			Domain: "demo.example.com",
		},
	}, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo-pod",
			Namespace: "openldap",
		},
		Spec: corev1.PodSpec{},
	}).Build()

	ctx := context.Background()

	// We leave out the core types for our tests.
	scheme := runtime.NewScheme()
	_ = openldapv1alpha1.AddToScheme(scheme)

	parent := &openldapv1alpha1.LDAPOrganizationalUnit{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo",
			Namespace: "openldap",
		},
	}

	t.Run("Registered Type", func(t *testing.T) {
		ref := api.ObjectReference{
			Name:      "demo",
			Namespace: "openldap",
			Kind:      "LDAPServer",
		}

		obj, err := ref.Resolve(ctx, reader, scheme, parent)
		require.NoError(t, err)

		assert.IsType(t, &openldapv1alpha1.LDAPServer{}, obj)

		server := obj.(api.NamedLDAPObject)
		distinguishedName, err := server.GetDistinguishedName(ctx, reader, scheme)
		require.NoError(t, err)

		assert.Equal(t, "dc=demo,dc=example,dc=com", distinguishedName)
	})

	t.Run("Unregistered Type", func(t *testing.T) {
		ref := api.ObjectReference{
			Name:       "demo-pod",
			APIVersion: "v1",
			Kind:       "Pod",
		}

		obj, err := ref.Resolve(ctx, reader, scheme, parent)
		require.NoError(t, err)

		assert.IsType(t, &unstructured.Unstructured{}, obj)
	})
}
