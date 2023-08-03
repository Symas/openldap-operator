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

package api

import (
	"context"
	"fmt"

	"github.com/gpu-ninja/openldap-operator/internal/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Reference interface {
	// Resolve resolves the reference to its underlying resource.
	Resolve(ctx context.Context, reader client.Reader, scheme *runtime.Scheme, parent runtime.Object) (runtime.Object, error)
}

type ObjectWithReferences interface {
	// ResolveReferences resolves all references in the object.
	ResolveReferences(ctx context.Context, reader client.Reader, scheme *runtime.Scheme) error
}

// ObjectReference is a reference to an arbitrary Kubernetes resource.
// +kubebuilder:object:generate=true
type ObjectReference struct {
	// Name is the name of the resource.
	Name string `json:"name,omitempty"`
	// Namespace is the namespace of the resource.
	Namespace string `json:"namespace,omitempty"`
	// APIVersion is the API version of the resource.
	APIVersion string `json:"apiVersion,omitempty"`
	// Kind is the kind of the resource.
	Kind string `json:"kind,omitempty"`
}

// Resolve resolves the reference to its underlying resource.
func (ref *ObjectReference) Resolve(ctx context.Context, reader client.Reader, scheme *runtime.Scheme, parent runtime.Object) (runtime.Object, error) {
	var u unstructured.Unstructured
	apiVersion := ref.APIVersion
	if apiVersion == "" {
		gvks, _, err := scheme.ObjectKinds(parent)
		if err != nil {
			return nil, fmt.Errorf("failed to get object kinds: %w", err)
		}

		if len(gvks) == 0 {
			return nil, fmt.Errorf("no object kinds found")
		}

		apiVersion = gvks[0].GroupVersion().String()
	}

	u.SetAPIVersion(apiVersion)
	u.SetKind(ref.Kind)

	namespace := ref.Namespace
	if namespace == "" {
		parentMeta, err := meta.Accessor(parent)
		if err != nil {
			return nil, fmt.Errorf("failed to get accessor: %w", err)
		}

		namespace = parentMeta.GetNamespace()
	}

	err := reader.Get(ctx, client.ObjectKey{Name: ref.Name, Namespace: namespace}, &u)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, util.Retryable(err)
		}

		return nil, fmt.Errorf("failed to resolve reference: %w", err)
	}

	unstructuredBytes, err := u.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal unstructured object: %w", err)
	}

	dec := serializer.NewCodecFactory(scheme).UniversalDeserializer()
	obj, _, err := dec.Decode(unstructuredBytes, nil, nil)
	if err != nil {
		if runtime.IsNotRegisteredError(err) {
			return &u, nil
		}

		return nil, fmt.Errorf("failed to decode unstructured object: %w", err)
	}

	return obj, nil
}

// LocalSecretReference is a reference to a secret in the same namespace.
// +kubebuilder:object:generate=true
type LocalSecretReference struct {
	// Name is the name of the secret.
	Name string `json:"name"`
}

// Resolve resolves the reference to its underlying secret.
func (ref *LocalSecretReference) Resolve(ctx context.Context, reader client.Reader, scheme *runtime.Scheme, parent runtime.Object) (runtime.Object, error) {
	objRef := ObjectReference{
		Name:       ref.Name,
		APIVersion: "v1",
		Kind:       "Secret",
	}

	secret, err := objRef.Resolve(ctx, reader, scheme, parent)
	if err != nil {
		return nil, err
	}

	return secret.(*corev1.Secret), nil
}

// LDAPServerReference is a reference to an LDAPServer.
// +kubebuilder:object:generate=true
type LDAPServerReference struct {
	// Name of the referenced LDAPServer.
	Name string `json:"name"`
	// Namespace is the optional namespace of the referenced LDAPServer.
	Namespace string `json:"namespace,omitempty"`
}

func (ref *LDAPServerReference) Resolve(ctx context.Context, reader client.Reader, scheme *runtime.Scheme, parent runtime.Object) (runtime.Object, error) {
	objRef := &ObjectReference{
		Name:      ref.Name,
		Namespace: ref.Namespace,
		Kind:      "LDAPServer",
	}

	return objRef.Resolve(ctx, reader, scheme, parent)
}
