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

package mapper

import (
	"context"

	"github.com/gpu-ninja/openldap-operator/api"
	openldapv1alpha1 "github.com/gpu-ninja/openldap-operator/api/v1alpha1"
	"github.com/gpu-ninja/openldap-operator/internal/directory"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Mapper[T api.LDAPObject, E directory.Entry] func(ctx context.Context, reader client.Reader, scheme *runtime.Scheme, dn string, obj T) (entry E, err error)

func OrganizationalUnitToEntry(_ context.Context, _ client.Reader, _ *runtime.Scheme, dn string, obj *openldapv1alpha1.LDAPOrganizationalUnit) (*directory.OrganizationalUnit, error) {
	return &directory.OrganizationalUnit{
		DistinguishedName: dn,
		Name:              obj.Spec.Name,
		Description:       obj.Spec.Description,
	}, nil
}

func GroupToEntry(_ context.Context, _ client.Reader, _ *runtime.Scheme, dn string, obj *openldapv1alpha1.LDAPGroup) (*directory.Group, error) {
	return &directory.Group{
		DistinguishedName: dn,
		Name:              obj.Spec.Name,
		Description:       obj.Spec.Description,
		Members:           obj.Spec.Members,
	}, nil
}

func UserToEntry(ctx context.Context, reader client.Reader, scheme *runtime.Scheme, dn string, obj *openldapv1alpha1.LDAPUser) (*directory.User, error) {
	var password string
	if obj.Spec.PaswordSecretRef != nil {
		passwordSecret, err := obj.Spec.PaswordSecretRef.Resolve(ctx, reader, scheme, obj)
		if err != nil {
			return nil, err
		}

		password = string(passwordSecret.(*corev1.Secret).Data["LDAP_USER_PASSWORD"])
	}

	return &directory.User{
		DistinguishedName: dn,
		Username:          obj.Spec.Username,
		Name:              obj.Spec.Name,
		Surname:           obj.Spec.Surname,
		Email:             obj.Spec.Email,
		Password:          password,
	}, nil
}
