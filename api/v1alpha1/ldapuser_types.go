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
	"fmt"

	"github.com/gpu-ninja/openldap-operator/api"
	"github.com/gpu-ninja/operator-utils/reference"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type LDAPUserSpec struct {
	api.LDAPObjectSpec `json:",inline"`
	// Username is the username (uid) for this user.
	Username string `json:"username"`
	// Name is the full name of this user (commonName).
	Name string `json:"name"`
	// Surname is the surname of this user.
	Surname string `json:"surname"`
	// Email is an optional email address of this user.
	Email string `json:"email,omitempty"`
	// PasswordSecretRef is an optional reference to a secret containing the password of the user.
	PaswordSecretRef *reference.LocalSecretReference `json:"passwordSecretRef,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// LDAPUser is a LDAP user.
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type LDAPUser struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LDAPUserSpec     `json:"spec,omitempty"`
	Status api.SimpleStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// LDAPUserList contains a list of LDAPUser
type LDAPUserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LDAPUser `json:"items"`
}

func (u *LDAPUser) GetDistinguishedName(ctx context.Context, reader client.Reader, scheme *runtime.Scheme) (string, error) {
	if u.Spec.ParentRef != nil {
		parent, err := u.Spec.ParentRef.Resolve(ctx, reader, scheme, u)
		if err != nil {
			return "", err
		}

		parentObj, ok := parent.(api.NamedLDAPObject)
		if !ok {
			return "", fmt.Errorf("parent is not a named ldap object")
		}

		parentDN, err := parentObj.GetDistinguishedName(ctx, reader, scheme)
		if err != nil {
			return "", err
		}

		return "uid=" + u.Spec.Username + "," + parentDN, nil
	}

	server, err := u.Spec.ServerRef.Resolve(ctx, reader, scheme, u)
	if err != nil {
		return "", err
	}

	serverObj, ok := server.(api.NamedLDAPObject)
	if !ok {
		return "", fmt.Errorf("server is not a named ldap object")
	}

	serverDN, err := serverObj.GetDistinguishedName(ctx, reader, scheme)
	if err != nil {
		return "", err
	}

	return "uid=" + u.Spec.Username + "," + serverDN, nil
}

func (u *LDAPUser) ResolveReferences(ctx context.Context, reader client.Reader, scheme *runtime.Scheme) error {
	if _, err := (&(&u.Spec).ServerRef).Resolve(ctx, reader, scheme, u); err != nil {
		return err
	}

	if u.Spec.ParentRef != nil {
		if _, err := (&u.Spec).ParentRef.Resolve(ctx, reader, scheme, u); err != nil {
			return err
		}
	}

	if u.Spec.PaswordSecretRef != nil {
		if _, err := (&u.Spec).PaswordSecretRef.Resolve(ctx, reader, scheme, u); err != nil {
			return err
		}
	}

	return nil
}

func (u *LDAPUser) GetLDAPObjectSpec() *api.LDAPObjectSpec {
	return &u.Spec.LDAPObjectSpec
}

func (u *LDAPUser) SetStatus(status api.SimpleStatus) {
	u.Status = status
}

func (u *LDAPUser) GetPhase() api.Phase {
	return u.Status.Phase
}

func init() {
	SchemeBuilder.Register(&LDAPUser{}, &LDAPUserList{})
}
