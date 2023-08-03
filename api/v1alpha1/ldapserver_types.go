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

	"github.com/gpu-ninja/openldap-operator/api"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type LDAPServerPhase string

const (
	LDAPServerPhasePending LDAPServerPhase = "Pending"
	LDAPServerPhaseReady   LDAPServerPhase = "Ready"
	LDAPServerPhaseFailed  LDAPServerPhase = "Failed"
)

type LDAPServerConditionType string

const (
	LDAPServerConditionTypePending LDAPServerConditionType = "Pending"
	LDAPServerConditionTypeReady   LDAPServerConditionType = "Ready"
	LDAPServerConditionTypeFailed  LDAPServerConditionType = "Failed"
)

// LDAPServerStorageSpec defines the storage configuration for the LDAP server.
type LDAPServerStorageSpec struct {
	// Size is the size of the persistent volume that will be
	// used to store the LDAP database.
	Size string `json:"size"`
	// StorageClassName is the name of the storage class that will be
	// used to provision the persistent volume.
	StorageClassName *string `json:"storageClassName,omitempty"`
}

// LDAPServerSpec defines the desired state of LDAPServer.
type LDAPServerSpec struct {
	// Image is the container image that will be used to run the LDAP server.
	Image string `json:"image"`
	// Domain is the domain of the organization that owns the LDAP server.
	Domain string `json:"domain"`
	// Organization is the name of the organization that owns the LDAP server.
	Organization string `json:"organization"`
	// AdminPasswordSecretRef is a reference to a secret that contains the
	// password for the admin user.
	AdminPasswordSecretRef api.LocalSecretReference `json:"adminPasswordSecretRef"`
	// CertificateSecretRef is a reference to a secret that contains the
	// TLS certificate and key that will be used to secure the LDAP server.
	// Expected to be in cert-manager format, eg. tls.key, tls.crt, ca.crt.
	CertificateSecretRef api.LocalSecretReference `json:"certificateSecretRef"`
	// DebugLevel controls the verbosity of the server logs.
	DebugLevel *int `json:"debugLevel,omitempty"`
	// FileDescriptorLimit controls the maximum number of file
	// descriptors that the LDAP server can open.
	// See: https://github.com/docker/docker/issues/8231
	FileDescriptorLimit *int `json:"fileDescriptorLimit,omitempty"`
	// Storage defines the persistent volume that will be used
	// to store the LDAP database.
	Storage LDAPServerStorageSpec `json:"storage"`
	// AddressOverride is an optional address that will be used to
	// access the LDAP server
	AddressOverride string `json:"addressOverride,omitempty"`
}

// LDAPServerCondition contains details for the current condition of the LDAPServer.
type LDAPServerCondition struct {
	// Type is the type of the condition.
	Type LDAPServerConditionType `json:"type"`
	// Status is the status of the condition.
	Status corev1.ConditionStatus `json:"status"`
	// LastTransitionTime is the last time the condition transitioned from one status to another.
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// Reason is a unique, one-word, CamelCase reason for the condition's last transition.
	Reason string `json:"reason,omitempty"`
	// Message is a human readable message indicating details about the last transition.
	Message string `json:"message,omitempty"`
}

// LDAPServerStatus defines the observed state of the LDAPServer.
type LDAPServerStatus struct {
	// Phase is the current state of the LDAP server.
	Phase LDAPServerPhase `json:"phase,omitempty"`
	// ObservedGeneration is the most recent generation observed for this LDAPServer by the controller.
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	// Conditions represents the latest available observations of an LDAPServer's current state.
	Conditions []LDAPServerCondition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// LDAPServer is an OpenLDAP server.
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type LDAPServer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LDAPServerSpec   `json:"spec,omitempty"`
	Status LDAPServerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// LDAPServerList contains a list of LDAPServer
type LDAPServerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LDAPServer `json:"items"`
}

func (s *LDAPServer) GetDistinguishedName(_ context.Context, _ client.Reader, _ *runtime.Scheme) (string, error) {
	return "dc=" + strings.Join(strings.Split(s.Spec.Domain, "."), ",dc="), nil
}

func (s *LDAPServer) ResolveReferences(ctx context.Context, reader client.Reader, scheme *runtime.Scheme) error {
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
	SchemeBuilder.Register(&LDAPServer{}, &LDAPServerList{})
}
