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

package controller_test

import (
	"context"
	"testing"
	"time"

	"github.com/gpu-ninja/openldap-operator/api"
	openldapv1alpha1 "github.com/gpu-ninja/openldap-operator/api/v1alpha1"
	"github.com/gpu-ninja/openldap-operator/internal/constants"
	"github.com/gpu-ninja/openldap-operator/internal/controller"
	"github.com/gpu-ninja/openldap-operator/internal/directory"
	"github.com/gpu-ninja/openldap-operator/internal/mapper"
	"github.com/gpu-ninja/openldap-operator/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestLDAPObjectReconciler(t *testing.T) {
	ctrl.SetLogger(util.NewLogger(zaptest.NewLogger(t)))

	scheme := runtime.NewScheme()
	_ = openldapv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	ldapServer := &openldapv1alpha1.LDAPServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-server",
			Namespace: "default",
		},
		Spec: openldapv1alpha1.LDAPServerSpec{
			Domain: "example.com",
			AdminPasswordSecretRef: api.LocalSecretReference{
				Name: "admin-password",
			},
			CertificateSecretRef: api.LocalSecretReference{
				Name: "certificate",
			},
		},
		Status: openldapv1alpha1.LDAPServerStatus{
			Phase: openldapv1alpha1.LDAPServerPhaseReady,
		},
	}

	ldapUser := &openldapv1alpha1.LDAPUser{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-user",
			Namespace: "default",
			Finalizers: []string{
				constants.FinalizerName,
			},
		},
		Spec: openldapv1alpha1.LDAPUserSpec{
			LDAPObjectSpec: api.LDAPObjectSpec{
				ServerRef: api.LDAPServerReference{
					Name: "test-server",
				},
			},
			Username: "test-user",
			PaswordSecretRef: &api.LocalSecretReference{
				Name: "test-user-password",
			},
		},
	}

	userPassword := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-user-password",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"LDAP_USER_PASSWORD": []byte("test-password"),
		},
	}

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ldapServer, ldapUser, userPassword).
		WithStatusSubresource(ldapUser).
		Build()

	r := &controller.LDAPObjectReconciler[*openldapv1alpha1.LDAPUser, *directory.User]{
		Client:     client,
		Scheme:     scheme,
		MapToEntry: mapper.UserToEntry,
	}

	ctx := context.Background()

	t.Run("Create or Update", func(t *testing.T) {
		eventRecorder := record.NewFakeRecorder(2)
		r.EventRecorder = eventRecorder
		r.DirectoryClientBuilder = directory.NewFakeClientBuilder()

		_, err := r.Reconcile(ctx, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      ldapUser.Name,
				Namespace: ldapUser.Namespace,
			},
		})
		require.NoError(t, err)
		assert.Len(t, eventRecorder.Events, 1)

		directoryClient, _ := r.DirectoryClientBuilder.Build(ctx)

		var user directory.User
		err = directoryClient.GetEntry("uid=test-user,dc=example,dc=com", &user)
		require.NoError(t, err)

		assert.Equal(t, "test-user", user.Username)
	})

	t.Run("Delete", func(t *testing.T) {
		deletingLdapUser := ldapUser.DeepCopy()
		deletingLdapUser.DeletionTimestamp = &metav1.Time{Time: metav1.Now().Add(-1 * time.Second)}

		r.Client = fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(ldapServer, deletingLdapUser, userPassword).
			WithStatusSubresource(deletingLdapUser).
			Build()

		eventRecorder := record.NewFakeRecorder(2)
		r.EventRecorder = eventRecorder
		r.DirectoryClientBuilder = directory.NewFakeClientBuilder()

		directoryClient, _ := r.DirectoryClientBuilder.Build(ctx)

		dn := "uid=test-user,dc=example,dc=com"
		user := directory.User{
			DistinguishedName: dn,
			Username:          "test-user",
		}

		_, err := directoryClient.CreateOrUpdateEntry(&user)
		require.NoError(t, err)

		err = client.Delete(ctx, ldapUser)
		require.NoError(t, err)

		_, err = r.Reconcile(ctx, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      "test-user",
				Namespace: "default",
			},
		})
		require.NoError(t, err)

		assert.Empty(t, eventRecorder.Events)

		err = directoryClient.GetEntry(dn, &user)
		assert.Error(t, err)
	})

	t.Run("References Not Resolvable", func(t *testing.T) {
		r.Client = fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(ldapServer, ldapUser). // Note the missing secret.
			WithStatusSubresource(ldapUser).
			Build()

		eventRecorder := record.NewFakeRecorder(2)
		r.EventRecorder = eventRecorder
		r.DirectoryClientBuilder = directory.NewFakeClientBuilder()

		resp, err := r.Reconcile(ctx, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      ldapUser.Name,
				Namespace: ldapUser.Namespace,
			},
		})
		require.NoError(t, err)
		assert.Len(t, eventRecorder.Events, 1)

		assert.Equal(t, reconcile.Result{
			RequeueAfter: constants.ReconcileRetryInterval,
		}, resp)
	})
}
