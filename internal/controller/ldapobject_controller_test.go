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
	"github.com/gpu-ninja/openldap-operator/internal/controller"
	"github.com/gpu-ninja/openldap-operator/internal/ldap"
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
				controller.FinalizerName,
			},
		},
		Spec: openldapv1alpha1.LDAPUserSpec{
			LDAPObjectSpec: api.LDAPObjectSpec{
				DirectoryRef: api.LocalLDAPDirectoryReference{
					Name: "test-directory",
				},
				ParentRef: &reference.LocalObjectReference{
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
				DirectoryRef: api.LocalLDAPDirectoryReference{
					Name: "test-directory",
				},
			},
			Name: "users",
		},
		Status: api.SimpleStatus{
			Phase: api.PhaseReady,
		},
	}

	directory := &openldapv1alpha1.LDAPDirectory{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-directory",
			Namespace: "default",
		},
		Spec: openldapv1alpha1.LDAPDirectorySpec{
			Domain: "example.com",
			AdminPasswordSecretRef: reference.LocalSecretReference{
				Name: "admin-password",
			},
			CertificateSecretRef: reference.LocalSecretReference{
				Name: "certificate",
			},
		},
		Status: openldapv1alpha1.LDAPDirectoryStatus{
			Phase: openldapv1alpha1.LDAPDirectoryPhaseReady,
		},
	}

	subResourceClient := fakeutils.NewSubResourceClient(scheme)

	interceptorFuncs := interceptor.Funcs{
		SubResource: func(client client.WithWatch, subResource string) client.SubResourceClient {
			return subResourceClient
		},
	}

	r := &controller.LDAPObjectReconciler[*openldapv1alpha1.LDAPUser, *ldap.User]{
		Scheme:     scheme,
		MapToEntry: mapper.UserToEntry,
	}

	ctx := context.Background()

	t.Run("Create or Update", func(t *testing.T) {
		eventRecorder := record.NewFakeRecorder(2)
		r.Recorder = eventRecorder

		subResourceClient.Reset()

		r.Client = fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(user, userPassword, orgUnit, directory).
			WithStatusSubresource(user, orgUnit, directory).
			WithInterceptorFuncs(interceptorFuncs).
			Build()

		var m mock.Mock
		r.LDAPClientBuilder = ldap.NewFakeClientBuilder(&m)

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
		deletingUser.Finalizers = []string{controller.FinalizerName}

		eventRecorder := record.NewFakeRecorder(2)
		r.Recorder = eventRecorder

		subResourceClient.Reset()

		r.Client = fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(deletingUser, userPassword, orgUnit, directory).
			WithStatusSubresource(deletingUser, orgUnit, directory).
			WithInterceptorFuncs(interceptorFuncs).
			Build()

		var m mock.Mock
		r.LDAPClientBuilder = ldap.NewFakeClientBuilder(&m)

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
		r.Recorder = eventRecorder

		subResourceClient.Reset()

		r.Client = fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(user, orgUnit, directory). // Note the missing secrets.
			WithStatusSubresource(user, orgUnit, directory).
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

	t.Run("Directory Not Ready", func(t *testing.T) {
		eventRecorder := record.NewFakeRecorder(2)
		r.Recorder = eventRecorder

		subResourceClient.Reset()

		notReadyDirectory := directory.DeepCopy()
		notReadyDirectory.Status.Phase = openldapv1alpha1.LDAPDirectoryPhasePending

		r.Client = fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(user, userPassword, orgUnit, notReadyDirectory).
			WithStatusSubresource(user, orgUnit, notReadyDirectory).
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
		assert.Equal(t, "Warning NotReady Referenced directory is not ready", event)

		updatedUser := user.DeepCopy()
		err = subResourceClient.Get(ctx, user, updatedUser)
		require.NoError(t, err)

		assert.Equal(t, api.PhasePending, updatedUser.Status.Phase)
	})

	t.Run("Parent Not Ready", func(t *testing.T) {
		eventRecorder := record.NewFakeRecorder(2)
		r.Recorder = eventRecorder

		subResourceClient.Reset()

		notReadyOrgUnit := orgUnit.DeepCopy()
		notReadyOrgUnit.Status.Phase = api.PhasePending

		r.Client = fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(user, userPassword, notReadyOrgUnit, directory).
			WithStatusSubresource(user, notReadyOrgUnit, directory).
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
		r.Recorder = eventRecorder

		subResourceClient.Reset()

		r.Client = fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(user, userPassword, orgUnit, directory).
			WithStatusSubresource(user, orgUnit, directory).
			WithInterceptorFuncs(interceptorFuncs).
			Build()

		var m mock.Mock
		r.LDAPClientBuilder = ldap.NewFakeClientBuilder(&m)

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
		assert.Equal(t, "Warning Failed Failed to create or update ldap entry: bang", event)

		updateduser := user.DeepCopy()
		err = subResourceClient.Get(ctx, user, updateduser)
		require.NoError(t, err)

		assert.Equal(t, api.PhaseFailed, updateduser.Status.Phase)
	})
}
