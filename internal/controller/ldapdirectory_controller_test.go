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

	fakeutils "github.com/symas/operator-utils/fake"
	"github.com/symas/operator-utils/reference"
	"github.com/symas/operator-utils/zaplogr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	openldapv1alpha1 "github.com/symas/ldap-operator/api/v1alpha1"
	"github.com/symas/ldap-operator/internal/controller"
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

func TestLDAPDirectoryReconciler(t *testing.T) {
	ctrl.SetLogger(zaplogr.New(zaptest.NewLogger(t)))

	scheme := runtime.NewScheme()

	err := corev1.AddToScheme(scheme)
	require.NoError(t, err)

	err = appsv1.AddToScheme(scheme)
	require.NoError(t, err)

	err = openldapv1alpha1.AddToScheme(scheme)
	require.NoError(t, err)

	directory := &openldapv1alpha1.LDAPDirectory{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: openldapv1alpha1.LDAPDirectorySpec{
			Image:        "ghcr.io/symas/openldap-operator/ldap:latest",
			Domain:       "example.com",
			Organization: "Acme Widgets Inc.",
			AdminPasswordSecretRef: reference.LocalSecretReference{
				Name: "admin-password",
			},
			CertificateSecretRef: reference.LocalSecretReference{
				Name: "demo-tls",
			},
			Storage: openldapv1alpha1.LDAPDirectoryStorageSpec{
				Size: "1Gi",
			},
		},
	}

	directoryCertificate := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo-tls",
			Namespace: "default",
		},
		Type: corev1.SecretTypeTLS,
		StringData: map[string]string{
			"ca.crt":  "",
			"tls.crt": "",
			"tls.key": "",
		},
	}

	adminPassword := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "admin-password",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"password": []byte("password"),
		},
	}

	subResourceClient := fakeutils.NewSubResourceClient(scheme)

	interceptorFuncs := interceptor.Funcs{
		SubResource: func(client client.WithWatch, subResource string) client.SubResourceClient {
			return subResourceClient
		},
	}

	r := &controller.LDAPDirectoryReconciler{
		Scheme: scheme,
	}

	ctx := context.Background()

	t.Run("Create or Update", func(t *testing.T) {
		eventRecorder := record.NewFakeRecorder(2)
		r.Recorder = eventRecorder

		subResourceClient.Reset()

		r.Client = fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(directory, directoryCertificate, adminPassword).
			WithStatusSubresource(directory).
			WithInterceptorFuncs(interceptorFuncs).
			Build()

		resp, err := r.Reconcile(ctx, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      directory.Name,
				Namespace: directory.Namespace,
			},
		})
		require.NoError(t, err)
		assert.NotZero(t, resp.RequeueAfter)

		require.Len(t, eventRecorder.Events, 1)
		event := <-eventRecorder.Events
		assert.Equal(t, "Normal Pending Waiting for statefulset to become ready", event)

		updatedDirectory := directory.DeepCopy()
		err = subResourceClient.Get(ctx, directory, updatedDirectory)
		require.NoError(t, err)

		assert.Equal(t, openldapv1alpha1.LDAPDirectoryPhasePending, updatedDirectory.Status.Phase)
		assert.Len(t, updatedDirectory.Status.Conditions, 1)

		var svc corev1.Service
		err = r.Client.Get(ctx, types.NamespacedName{
			Name:      directory.Name,
			Namespace: directory.Namespace,
		}, &svc)
		require.NoError(t, err)

		var sts appsv1.StatefulSet
		err = r.Client.Get(ctx, types.NamespacedName{
			Name:      directory.Name,
			Namespace: directory.Namespace,
		}, &sts)
		require.NoError(t, err)

		sts.Status.ReadyReplicas = *sts.Spec.Replicas

		r.Client = fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(updatedDirectory, directoryCertificate, adminPassword, &sts).
			WithStatusSubresource(updatedDirectory, &sts).
			WithInterceptorFuncs(interceptorFuncs).
			Build()

		resp, err = r.Reconcile(ctx, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      directory.Name,
				Namespace: directory.Namespace,
			},
		})
		require.NoError(t, err)
		assert.Zero(t, resp)

		require.Len(t, eventRecorder.Events, 1)
		event = <-eventRecorder.Events
		assert.Equal(t, "Normal Created Successfully created", event)

		updatedDirectory = directory.DeepCopy()
		err = subResourceClient.Get(ctx, directory, updatedDirectory)
		require.NoError(t, err)

		assert.Equal(t, openldapv1alpha1.LDAPDirectoryPhaseReady, updatedDirectory.Status.Phase)
		assert.Len(t, updatedDirectory.Status.Conditions, 2)
	})

	t.Run("Delete", func(t *testing.T) {
		deletingDirectory := directory.DeepCopy()
		deletingDirectory.DeletionTimestamp = &metav1.Time{Time: metav1.Now().Add(-1 * time.Second)}
		deletingDirectory.Finalizers = []string{controller.FinalizerName}

		eventRecorder := record.NewFakeRecorder(2)
		r.Recorder = eventRecorder

		r.Client = fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(deletingDirectory, directoryCertificate, adminPassword).
			WithStatusSubresource(deletingDirectory).
			Build()

		resp, err := r.Reconcile(ctx, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      directory.Name,
				Namespace: directory.Namespace,
			},
		})
		require.NoError(t, err)
		assert.Zero(t, resp)

		assert.Len(t, eventRecorder.Events, 0)
	})

	t.Run("References Not Resolvable", func(t *testing.T) {
		eventRecorder := record.NewFakeRecorder(2)
		r.Recorder = eventRecorder

		subResourceClient.Reset()

		r.Client = fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(directory). // note the missing secrets
			WithStatusSubresource(directory).
			WithInterceptorFuncs(interceptorFuncs).
			Build()

		resp, err := r.Reconcile(ctx, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      directory.Name,
				Namespace: directory.Namespace,
			},
		})
		require.NoError(t, err)
		assert.NotZero(t, resp.RequeueAfter)

		require.Len(t, eventRecorder.Events, 1)
		event := <-eventRecorder.Events
		assert.Equal(t, "Warning NotReady Not all references are resolvable", event)

		updatedDirectory := directory.DeepCopy()
		err = subResourceClient.Get(ctx, directory, updatedDirectory)
		require.NoError(t, err)

		assert.Equal(t, openldapv1alpha1.LDAPDirectoryPhasePending, updatedDirectory.Status.Phase)
		assert.Len(t, updatedDirectory.Status.Conditions, 1)
	})

	t.Run("Failure", func(t *testing.T) {
		eventRecorder := record.NewFakeRecorder(2)
		r.Recorder = eventRecorder

		failOnStatefulSets := interceptorFuncs
		failOnStatefulSets.Get = func(ctx context.Context, client client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
			if _, ok := obj.(*appsv1.StatefulSet); ok {
				return fmt.Errorf("bang")
			}

			return client.Get(ctx, key, obj, opts...)
		}

		subResourceClient.Reset()

		r.Client = fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(directory, directoryCertificate, adminPassword).
			WithStatusSubresource(directory).
			WithInterceptorFuncs(failOnStatefulSets).
			Build()

		_, err := r.Reconcile(ctx, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      directory.Name,
				Namespace: directory.Namespace,
			},
		})
		require.Error(t, err)

		require.Len(t, eventRecorder.Events, 1)
		event := <-eventRecorder.Events
		assert.Equal(t, "Warning Failed Failed to reconcile statefulset: failed to get object: bang", event)

		updatedDirectory := directory.DeepCopy()
		err = subResourceClient.Get(ctx, directory, updatedDirectory)
		require.NoError(t, err)

		assert.Equal(t, openldapv1alpha1.LDAPDirectoryPhaseFailed, updatedDirectory.Status.Phase)
	})
}
