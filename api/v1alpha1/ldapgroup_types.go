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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type LDAPGroupSpec struct {
	api.LDAPObjectSpec `json:",inline"`
	// Name is the common name for this group.
	Name string `json:"name"`
	// Description is an optional description of this group.
	Description string `json:"description,omitempty"`
	// Members is a list of distinguished names representing the members of this group.
	//+kubebuilder:validation:MinItems=1
	Members []string `json:"members"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// LDAPGroup is a LDAP group of names.
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type LDAPGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LDAPGroupSpec    `json:"spec,omitempty"`
	Status api.SimpleStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// LDAPGroupList contains a list of LDAPGroup
type LDAPGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LDAPGroup `json:"items"`
}

func (g *LDAPGroup) GetDistinguishedName(ctx context.Context, reader client.Reader, scheme *runtime.Scheme) (string, error) {
	if g.Spec.ParentRef != nil {
		parent, err := g.Spec.ParentRef.Resolve(ctx, reader, scheme, g)
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

		return "cn=" + g.Spec.Name + "," + parentDN, nil
	}

	server, err := g.Spec.ServerRef.Resolve(ctx, reader, scheme, g)
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

	return "cn=" + g.Spec.Name + "," + serverDN, nil
}

func (g *LDAPGroup) ResolveReferences(ctx context.Context, reader client.Reader, scheme *runtime.Scheme) error {
	if _, err := g.Spec.ServerRef.Resolve(ctx, reader, scheme, g); err != nil {
		return err
	}

	if g.Spec.ParentRef != nil {
		if _, err := g.Spec.ParentRef.Resolve(ctx, reader, scheme, g); err != nil {
			return err
		}
	}

	return nil
}

func (g *LDAPGroup) GetLDAPObjectSpec() *api.LDAPObjectSpec {
	return &g.Spec.LDAPObjectSpec
}

func (g *LDAPGroup) SetStatus(status api.SimpleStatus) {
	g.Status = status
}

func (g *LDAPGroup) GetPhase() api.Phase {
	return g.Status.Phase
}

func init() {
	SchemeBuilder.Register(&LDAPGroup{}, &LDAPGroupList{})
}
