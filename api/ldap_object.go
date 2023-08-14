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

	"github.com/gpu-ninja/operator-utils/reference"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NamedLDAPObject is implemented by all LDAP objects that have a distinguished name.
type NamedLDAPObject interface {
	GetDistinguishedName(ctx context.Context, reader client.Reader, scheme *runtime.Scheme) (string, error)
}

// LDAPObject is implemented by all LDAP objects.
type LDAPObject interface {
	metav1.Object
	runtime.Object
	NamedLDAPObject
	reference.ObjectWithReferences
	GetLDAPObjectSpec() *LDAPObjectSpec
	SetStatus(status SimpleStatus)
	GetPhase() Phase
}

// +kubebuilder:object:generate=true
type LDAPObjectSpec struct {
	// ServerRef is a reference to the server that owns this object.
	ServerRef LDAPServerReference `json:"serverRef"`
	// ParentRef is an optional reference to the parent of this object (typically an organizational unit).
	ParentRef *reference.LocalObjectReference `json:"parentRef,omitempty"`
}

// Phase is the current phase of the object.
type Phase string

const (
	// PhasePending means the object has been created but is not ready for use.
	PhasePending Phase = "Pending"
	// PhaseReady means the object is ready for use.
	PhaseReady Phase = "Ready"
	// PhaseFailed means the object has failed.
	PhaseFailed Phase = "Failed"
)

// SimpleStatus is a basic status type that can be reused across multiple types.
type SimpleStatus struct {
	// Phase is the current phase of the object.
	Phase Phase `json:"phase,omitempty"`
	// ObservedGeneration is the most recent generation observed for this object by the controller.
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	// Message is a human readable message indicating details about why the object is in this condition.
	Message string `json:"message,omitempty"`
}

// LDAPServerReference is a reference to an LDAPServer.
// +kubebuilder:object:generate=true
type LDAPServerReference struct {
	// Name of the referenced LDAPServer.
	Name string `json:"name"`
}

func (ref *LDAPServerReference) Resolve(ctx context.Context, reader client.Reader, scheme *runtime.Scheme, parent runtime.Object) (runtime.Object, error) {
	objRef := &reference.ObjectReference{
		Name: ref.Name,
		Kind: "LDAPServer",
	}

	return objRef.Resolve(ctx, reader, scheme, parent)
}
