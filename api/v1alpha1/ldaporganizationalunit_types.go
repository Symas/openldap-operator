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

	"github.com/symas/openldap-operator/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type LDAPOrganizationalUnitSpec struct {
	api.LDAPObjectSpec `json:",inline"`
	// Name is the common name for this organizational unit.
	Name string `json:"name"`
	// Description is an optional description of this organizational unit.
	Description string `json:"description,omitempty"`
}

// LDAPOrganizationalUnit is a LDAP organizational unit.
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type LDAPOrganizationalUnit struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LDAPOrganizationalUnitSpec `json:"spec,omitempty"`
	Status api.SimpleStatus           `json:"status,omitempty"`
}

// LDAPOrganizationalUnitList contains a list of LDAPOrganizationalUnit
// +kubebuilder:object:root=true
type LDAPOrganizationalUnitList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LDAPOrganizationalUnit `json:"items"`
}

func (ou *LDAPOrganizationalUnit) GetDistinguishedName(ctx context.Context, reader client.Reader, scheme *runtime.Scheme) (string, error) {
	if ou.Spec.ParentRef != nil {
		parent, ok, err := ou.Spec.ParentRef.Resolve(ctx, reader, scheme, ou)
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

		return "ou=" + ou.Spec.Name + "," + parentDN, nil
	}

	directory, ok, err := ou.Spec.DirectoryRef.Resolve(ctx, reader, scheme, ou)
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

	return "ou=" + ou.Spec.Name + "," + directoryDN, nil
}

func (ou *LDAPOrganizationalUnit) ResolveReferences(ctx context.Context, reader client.Reader, scheme *runtime.Scheme) (bool, error) {
	_, ok, err := ou.Spec.DirectoryRef.Resolve(ctx, reader, scheme, ou)
	if !ok || err != nil {
		return ok, err
	}

	if ou.Spec.ParentRef != nil {
		_, ok, err = ou.Spec.ParentRef.Resolve(ctx, reader, scheme, ou)
		if !ok || err != nil {
			return ok, err
		}
	}

	return true, nil
}

func (ou *LDAPOrganizationalUnit) GetLDAPObjectSpec() *api.LDAPObjectSpec {
	return &ou.Spec.LDAPObjectSpec
}

func (ou *LDAPOrganizationalUnit) SetStatus(status api.SimpleStatus) {
	ou.Status = status
}

func (ou *LDAPOrganizationalUnit) GetPhase() api.Phase {
	return ou.Status.Phase
}

func init() {
	SchemeBuilder.Register(&LDAPOrganizationalUnit{}, &LDAPOrganizationalUnitList{})
}
