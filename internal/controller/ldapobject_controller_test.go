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
	"fmt"
	"testing"
	"time"

	"github.com/gpu-ninja/openldap-operator/api"
	openldapv1alpha1 "github.com/gpu-ninja/openldap-operator/api/v1alpha1"
	"github.com/gpu-ninja/openldap-operator/internal/constants"
	"github.com/gpu-ninja/openldap-operator/internal/controller"
	"github.com/gpu-ninja/openldap-operator/internal/directory"
	"github.com/gpu-ninja/openldap-operator/internal/mapper"
	fakeutils "github.com/gpu-ninja/operator-utils/fake"
	"github.com/gpu-ninja/operator-utils/reference"
	"github.com/gpu-ninja/operator-utils/zaplogr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestLDAPObjectReconciler(t *testing.T) {
	ctrl.SetLogger(zaplogr.New(zaptest.NewLogger(t)))

	scheme := runtime.NewScheme()

	err := corev1.AddToScheme(scheme)
	require.NoError(t, err)

	err = appsv1.AddToScheme(scheme)
	require.NoError(t, err)

	err = openldapv1alpha1.AddToScheme(scheme)
	require.NoError(t, err)

	user := &openldapv1alpha1.LDAPUser{
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
				ParentRef: &reference.ObjectReference{
					Name: "test-ou",
					Kind: "LDAPOrganizationalUnit",
				},
			},
			Username: "test-user",
			PaswordSecretRef: &reference.LocalSecretReference{
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
			"password": []byte("test-password"),
		},
	}

	orgUnit := &openldapv1alpha1.LDAPOrganizationalUnit{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-ou",
			Namespace: "default",
		},
		Spec: openldapv1alpha1.LDAPOrganizationalUnitSpec{
			LDAPObjectSpec: api.LDAPObjectSpec{
				ServerRef: api.LDAPServerReference{
					Name: "test-server",
				},
			},
			Name: "users",
		},
		Status: api.SimpleStatus{
			Phase: api.PhaseReady,
		},
	}

	server := &openldapv1alpha1.LDAPServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-server",
			Namespace: "default",
		},
		Spec: openldapv1alpha1.LDAPServerSpec{
			Domain: "example.com",
			AdminPasswordSecretRef: reference.LocalSecretReference{
				Name: "admin-password",
			},
			CertificateSecretRef: reference.LocalSecretReference{
				Name: "certificate",
			},
		},
		Status: openldapv1alpha1.LDAPServerStatus{
			Phase: openldapv1alpha1.LDAPServerPhaseReady,
		},
	}

	subResourceClient := fakeutils.NewSubResourceClient(scheme)

	interceptorFuncs := interceptor.Funcs{
		SubResource: func(client client.WithWatch, subResource string) client.SubResourceClient {
			return subResourceClient
		},
	}

	r := &controller.LDAPObjectReconciler[*openldapv1alpha1.LDAPUser, *directory.User]{
		Scheme:     scheme,
		MapToEntry: mapper.UserToEntry,
	}

	ctx := context.Background()

	t.Run("Create or Update", func(t *testing.T) {
		eventRecorder := record.NewFakeRecorder(2)
		r.EventRecorder = eventRecorder

		subResourceClient.Reset()

		r.Client = fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(user, userPassword, orgUnit, server).
			WithStatusSubresource(user, orgUnit, server).
			WithInterceptorFuncs(interceptorFuncs).
			Build()

		var m mock.Mock
		r.DirectoryClientBuilder = directory.NewFakeClientBuilder(&m)

		m.On("CreateOrUpdateEntry", mock.Anything).Return(true, nil)

		resp, err := r.Reconcile(ctx, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      user.Name,
				Namespace: user.Namespace,
			},
		})
		require.NoError(t, err)
		assert.Zero(t, resp)

		require.Len(t, eventRecorder.Events, 1)
		event := <-eventRecorder.Events
		assert.Equal(t, "Normal Created Successfully created", event)

		updatedUser := user.DeepCopy()
		err = subResourceClient.Get(ctx, user, updatedUser)
		require.NoError(t, err)

		assert.Equal(t, api.PhaseReady, updatedUser.Status.Phase)
	})

	t.Run("Delete", func(t *testing.T) {
		deletingUser := user.DeepCopy()
		deletingUser.DeletionTimestamp = &metav1.Time{Time: metav1.Now().Add(-1 * time.Second)}
		deletingUser.Finalizers = []string{constants.FinalizerName}

		eventRecorder := record.NewFakeRecorder(2)
		r.EventRecorder = eventRecorder

		subResourceClient.Reset()

		r.Client = fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(deletingUser, userPassword, orgUnit, server).
			WithStatusSubresource(deletingUser, orgUnit, server).
			WithInterceptorFuncs(interceptorFuncs).
			Build()

		var m mock.Mock
		r.DirectoryClientBuilder = directory.NewFakeClientBuilder(&m)

		m.On("DeleteEntry", mock.Anything, mock.Anything).Return(nil)

		resp, err := r.Reconcile(ctx, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      user.Name,
				Namespace: user.Namespace,
			},
		})
		require.NoError(t, err)
		assert.Zero(t, resp)

		assert.Len(t, eventRecorder.Events, 0)

		m.AssertCalled(t, "DeleteEntry", mock.Anything, mock.Anything)
	})

	t.Run("References Not Resolvable", func(t *testing.T) {
		eventRecorder := record.NewFakeRecorder(2)
		r.EventRecorder = eventRecorder

		subResourceClient.Reset()

		r.Client = fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(user, orgUnit, server). // Note the missing secrets.
			WithStatusSubresource(user, orgUnit, server).
			WithInterceptorFuncs(interceptorFuncs).
			Build()

		resp, err := r.Reconcile(ctx, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      user.Name,
				Namespace: user.Namespace,
			},
		})
		require.NoError(t, err)
		assert.NotZero(t, resp.RequeueAfter)

		require.Len(t, eventRecorder.Events, 1)
		event := <-eventRecorder.Events
		assert.Equal(t, "Warning NotReady Not all references are resolvable", event)

		updatedUser := user.DeepCopy()
		err = subResourceClient.Get(ctx, user, updatedUser)
		require.NoError(t, err)

		assert.Equal(t, api.PhasePending, updatedUser.Status.Phase)
	})

	t.Run("Server Not Ready", func(t *testing.T) {
		eventRecorder := record.NewFakeRecorder(2)
		r.EventRecorder = eventRecorder

		subResourceClient.Reset()

		notReadyServer := server.DeepCopy()
		notReadyServer.Status.Phase = openldapv1alpha1.LDAPServerPhasePending

		r.Client = fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(user, userPassword, orgUnit, notReadyServer).
			WithStatusSubresource(user, orgUnit, notReadyServer).
			WithInterceptorFuncs(interceptorFuncs).
			Build()

		resp, err := r.Reconcile(ctx, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      user.Name,
				Namespace: user.Namespace,
			},
		})
		require.NoError(t, err)
		assert.NotZero(t, resp.RequeueAfter)

		require.Len(t, eventRecorder.Events, 1)
		event := <-eventRecorder.Events
		assert.Equal(t, "Warning NotReady Referenced server is not ready", event)

		updatedUser := user.DeepCopy()
		err = subResourceClient.Get(ctx, user, updatedUser)
		require.NoError(t, err)

		assert.Equal(t, api.PhasePending, updatedUser.Status.Phase)
	})

	t.Run("Parent Not Ready", func(t *testing.T) {
		eventRecorder := record.NewFakeRecorder(2)
		r.EventRecorder = eventRecorder

		subResourceClient.Reset()

		notReadyOrgUnit := orgUnit.DeepCopy()
		notReadyOrgUnit.Status.Phase = api.PhasePending

		r.Client = fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(user, userPassword, notReadyOrgUnit, server).
			WithStatusSubresource(user, notReadyOrgUnit, server).
			WithInterceptorFuncs(interceptorFuncs).
			Build()

		resp, err := r.Reconcile(ctx, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      user.Name,
				Namespace: user.Namespace,
			},
		})
		require.NoError(t, err)
		assert.NotZero(t, resp.RequeueAfter)

		require.Len(t, eventRecorder.Events, 1)
		event := <-eventRecorder.Events
		assert.Equal(t, "Warning NotReady Referenced parent object is not ready", event)

		updatedUser := user.DeepCopy()
		err = subResourceClient.Get(ctx, user, updatedUser)
		require.NoError(t, err)

		assert.Equal(t, api.PhasePending, updatedUser.Status.Phase)
	})

	t.Run("Failure", func(t *testing.T) {
		eventRecorder := record.NewFakeRecorder(2)
		r.EventRecorder = eventRecorder

		subResourceClient.Reset()

		r.Client = fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(user, userPassword, orgUnit, server).
			WithStatusSubresource(user, orgUnit, server).
			WithInterceptorFuncs(interceptorFuncs).
			Build()

		var m mock.Mock
		r.DirectoryClientBuilder = directory.NewFakeClientBuilder(&m)

		m.On("CreateOrUpdateEntry", mock.Anything).Return(false, fmt.Errorf("bang"))

		resp, err := r.Reconcile(ctx, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      user.Name,
				Namespace: user.Namespace,
			},
		})
		require.NoError(t, err)
		assert.Zero(t, resp)

		require.Len(t, eventRecorder.Events, 1)
		event := <-eventRecorder.Events
		assert.Equal(t, "Warning Failed Failed to create or update ldap object: bang", event)

		updateduser := user.DeepCopy()
		err = subResourceClient.Get(ctx, user, updateduser)
		require.NoError(t, err)

		assert.Equal(t, api.PhaseFailed, updateduser.Status.Phase)
	})
}
