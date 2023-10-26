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

	"github.com/symas/operator-utils/reference"
	"github.com/symas/openldap-operator/api"
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

// LDAPUser is a LDAP user.
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type LDAPUser struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LDAPUserSpec     `json:"spec,omitempty"`
	Status api.SimpleStatus `json:"status,omitempty"`
}

// LDAPUserList contains a list of LDAPUser
// +kubebuilder:object:root=true
type LDAPUserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LDAPUser `json:"items"`
}

func (u *LDAPUser) GetDistinguishedName(ctx context.Context, reader client.Reader, scheme *runtime.Scheme) (string, error) {
	if u.Spec.ParentRef != nil {
		parent, ok, err := u.Spec.ParentRef.Resolve(ctx, reader, scheme, u)
		if !ok && err == nil {
			return "", fmt.Errorf("referenced parent not found")
		} else if err != nil {
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

	directory, ok, err := u.Spec.DirectoryRef.Resolve(ctx, reader, scheme, u)
	if !ok && err == nil {
		return "", fmt.Errorf("referenced directory not found")
	} else if err != nil {
		return "", err
	}

	directoryObj, ok := directory.(api.NamedLDAPObject)
	if !ok {
		return "", fmt.Errorf("directory is not a named ldap object")
	}

	directoryDN, err := directoryObj.GetDistinguishedName(ctx, reader, scheme)
	if err != nil {
		return "", err
	}

	return "uid=" + u.Spec.Username + "," + directoryDN, nil
}

func (u *LDAPUser) ResolveReferences(ctx context.Context, reader client.Reader, scheme *runtime.Scheme) (bool, error) {
	_, ok, err := u.Spec.DirectoryRef.Resolve(ctx, reader, scheme, u)
	if !ok || err != nil {
		return ok, err
	}

	if u.Spec.ParentRef != nil {
		_, ok, err = u.Spec.ParentRef.Resolve(ctx, reader, scheme, u)
		if !ok || err != nil {
			return ok, err
		}
	}

	if u.Spec.PaswordSecretRef != nil {
		_, ok, err = u.Spec.PaswordSecretRef.Resolve(ctx, reader, scheme, u)
		if !ok || err != nil {
			return ok, err
		}
	}

	return true, nil
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
