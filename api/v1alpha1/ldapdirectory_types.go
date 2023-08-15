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

package v1alpha1

import (
	"context"
	"strings"

	"github.com/gpu-ninja/operator-utils/reference"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type LDAPDirectoryPhase string

const (
	LDAPDirectoryPhasePending LDAPDirectoryPhase = "Pending"
	LDAPDirectoryPhaseReady   LDAPDirectoryPhase = "Ready"
	LDAPDirectoryPhaseFailed  LDAPDirectoryPhase = "Failed"
)

type LDAPDirectoryConditionType string

const (
	LDAPDirectoryConditionTypePending LDAPDirectoryConditionType = "Pending"
	LDAPDirectoryConditionTypeReady   LDAPDirectoryConditionType = "Ready"
	LDAPDirectoryConditionTypeFailed  LDAPDirectoryConditionType = "Failed"
)

// LDAPDirectoryStorageSpec defines the storage configuration for the LDAP directory.
type LDAPDirectoryStorageSpec struct {
	// Size is the size of the persistent volume that will be
	// used to store the LDAP database.
	Size string `json:"size"`
	// StorageClassName is the name of the storage class that will be
	// used to provision the persistent volume.
	StorageClassName *string `json:"storageClassName,omitempty"`
}

// LDAPDirectorySpec defines the desired state of the LDAP directory.
type LDAPDirectorySpec struct {
	// Image is the container image that will be used to run the LDAP directory.
	Image string `json:"image"`
	// Domain is the domain of the organization that owns the LDAP directory.
	Domain string `json:"domain"`
	// Organization is the name of the organization that owns the LDAP directory.
	Organization string `json:"organization"`
	// AdminPasswordSecretRef is a reference to a secret that contains the
	// password for the admin user.
	AdminPasswordSecretRef reference.LocalSecretReference `json:"adminPasswordSecretRef"`
	// CertificateSecretRef is a reference to a secret that contains the
	// TLS certificate and key that will be used to secure the LDAP directory.
	CertificateSecretRef reference.LocalSecretReference `json:"certificateSecretRef"`
	// DebugLevel controls the verbosity of the directory logs.
	DebugLevel *int `json:"debugLevel,omitempty"`
	// FileDescriptorLimit controls the maximum number of file
	// descriptors that the LDAP directory can open.
	// See: https://github.com/docker/docker/issues/8231
	FileDescriptorLimit *int `json:"fileDescriptorLimit,omitempty"`
	// Storage defines the persistent volume that will be used
	// to store the LDAP database.
	Storage LDAPDirectoryStorageSpec `json:"storage"`
	// AddressOverride is an optional address that will be used to
	// access the LDAP directory.
	AddressOverride string `json:"addressOverride,omitempty"`
}

// LDAPDirectoryStatus defines the observed state of the LDAP directory.
type LDAPDirectoryStatus struct {
	// Phase is the current state of the LDAP directory.
	Phase LDAPDirectoryPhase `json:"phase,omitempty"`
	// ObservedGeneration is the most recent generation observed for this LDAP directory by the controller.
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	// Conditions represents the latest available observations of the LDAP directories current state.
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// LDAPDirectory is a LDAP directory.
// +kubebuilder:object:root=true
// +kubebuilder:resource:path=ldapdirectories,scope=Namespaced
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type LDAPDirectory struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LDAPDirectorySpec   `json:"spec,omitempty"`
	Status LDAPDirectoryStatus `json:"status,omitempty"`
}

// LDAPDirectoryList contains a list of LDAPDirectory.
// +kubebuilder:object:root=true
type LDAPDirectoryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LDAPDirectory `json:"items"`
}

func (s *LDAPDirectory) GetDistinguishedName(_ context.Context, _ client.Reader, _ *runtime.Scheme) (string, error) {
	return "dc=" + strings.Join(strings.Split(s.Spec.Domain, "."), ",dc="), nil
}

func (s *LDAPDirectory) ResolveReferences(ctx context.Context, reader client.Reader, scheme *runtime.Scheme) error {
	_, err := s.Spec.AdminPasswordSecretRef.Resolve(ctx, reader, scheme, s)
	if err != nil {
		return err
	}

	_, err = s.Spec.CertificateSecretRef.Resolve(ctx, reader, scheme, s)
	if err != nil {
		return err
	}

	return nil
}

func init() {
	SchemeBuilder.Register(&LDAPDirectory{}, &LDAPDirectoryList{})
}
