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

package controller

import (
	"context"
	"fmt"
	"reflect"

	"github.com/gpu-ninja/openldap-operator/api"
	openldapv1alpha1 "github.com/gpu-ninja/openldap-operator/api/v1alpha1"
	"github.com/gpu-ninja/openldap-operator/internal/constants"
	"github.com/gpu-ninja/openldap-operator/internal/directory"
	"github.com/gpu-ninja/openldap-operator/internal/mapper"
	"github.com/gpu-ninja/openldap-operator/internal/util"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// LDAPGroups
//+kubebuilder:rbac:groups=openldap.gpu-ninja.com,resources=ldapgroups,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=openldap.gpu-ninja.com,resources=ldapgroups/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=openldap.gpu-ninja.com,resources=ldapgroups/finalizers,verbs=update

// LDAPOrganizationalUnits
//+kubebuilder:rbac:groups=openldap.gpu-ninja.com,resources=ldaporganizationalunits,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=openldap.gpu-ninja.com,resources=ldaporganizationalunits/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=openldap.gpu-ninja.com,resources=ldaporganizationalunits/finalizers,verbs=update

// LDAPUsers
//+kubebuilder:rbac:groups=openldap.gpu-ninja.com,resources=ldapusers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=openldap.gpu-ninja.com,resources=ldapusers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=openldap.gpu-ninja.com,resources=ldapusers/finalizers,verbs=update

type LDAPObjectReconciler[T api.LDAPObject, E directory.Entry] struct {
	client.Client
	Scheme                 *runtime.Scheme
	EventRecorder          record.EventRecorder
	DirectoryClientBuilder directory.ClientBuilder
	MapToEntry             mapper.Mapper[T, E]
}

func (r *LDAPObjectReconciler[T, E]) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := util.LoggerFromContext(ctx)

	logger.Info("Reconciling LDAP object")

	obj := r.newInstance()
	if err := r.Get(ctx, req.NamespacedName, obj); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, err
	}

	if !controllerutil.ContainsFinalizer(obj, constants.FinalizerName) {
		logger.Info("Adding Finalizer")

		_, err := controllerutil.CreateOrPatch(ctx, r.Client, obj, func() error {
			controllerutil.AddFinalizer(obj, constants.FinalizerName)

			return nil
		})
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to add finalizer: %w", err)
		}
	}

	// Make sure all references are resolvable.
	if err := obj.ResolveReferences(ctx, r.Client, r.Scheme); err != nil {
		if util.IsRetryable(err) {
			if !obj.GetDeletionTimestamp().IsZero() {
				// Parent has probably been removed by a cascading delete.
				// So there is probably no point in retrying.

				_, err := controllerutil.CreateOrPatch(ctx, r.Client, obj, func() error {
					controllerutil.RemoveFinalizer(obj, constants.FinalizerName)

					return nil
				})
				if err != nil {
					return ctrl.Result{}, fmt.Errorf("failed to remove finalizer: %w", err)
				}

				return ctrl.Result{}, nil
			}

			logger.Info("Not all references are resolvable, requeuing")

			r.EventRecorder.Event(obj, corev1.EventTypeWarning,
				"NotReady", "Not all references are resolvable")

			if err := r.markPending(ctx, obj); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to mark as pending: %w", err)
			}

			return ctrl.Result{RequeueAfter: constants.ReconcileRetryInterval}, nil
		}

		logger.Error("Failed to resolve references", zap.Error(err))

		r.EventRecorder.Eventf(obj, corev1.EventTypeWarning,
			"Failed", "Failed to resolve references: %s", err)

		r.markFailed(ctx, obj, err)

		return ctrl.Result{}, nil
	}

	objSpec := obj.GetLDAPObjectSpec()
	serverObj, err := objSpec.ServerRef.Resolve(ctx, r.Client, r.Scheme, obj)
	if err != nil {
		logger.Error("Failed to resolve server reference", zap.Error(err))

		r.EventRecorder.Eventf(obj, corev1.EventTypeWarning,
			"Failed", "Failed to resolve server reference: %s", err)

		r.markFailed(ctx, obj, err)

		return ctrl.Result{}, nil
	}
	server := serverObj.(*openldapv1alpha1.LDAPServer)

	// Is the server ready?
	if err := r.isServerReady(ctx, obj, server); err != nil {
		if util.IsRetryable(err) {
			if err := r.markPending(ctx, obj); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to mark as pending: %w", err)
			}

			return ctrl.Result{RequeueAfter: constants.ReconcileRetryInterval}, nil
		}

		logger.Error("Failed to check if server is ready", zap.Error(err))

		r.EventRecorder.Event(obj, corev1.EventTypeWarning,
			"Failed", "Failed to check if server is ready")

		r.markFailed(ctx, obj, err)

		return ctrl.Result{}, nil
	}

	directoryClient, err := r.DirectoryClientBuilder.WithServer(server).Build(ctx)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to create directory client: %w", err)
	}

	dn, err := obj.GetDistinguishedName(ctx, r.Client, r.Scheme)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get distinguished name: %w", err)
	}

	logger = logger.With(zap.String("dn", dn))

	if !obj.GetDeletionTimestamp().IsZero() {
		logger.Info("Deleting LDAP object")

		if err := directoryClient.DeleteEntry(dn, true); err != nil {
			logger.Error("Failed to delete LDAP object", zap.Error(err))

			return ctrl.Result{}, fmt.Errorf("failed to delete ldap object: %w", err)
		}

		_, err := controllerutil.CreateOrPatch(ctx, r.Client, obj, func() error {
			controllerutil.RemoveFinalizer(obj, constants.FinalizerName)

			return nil
		})
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to remove finalizer: %w", err)
		}

		return ctrl.Result{}, nil
	}

	logger.Info("Creating or updating LDAP object")

	// Set owner reference to parent (if one is specified).
	if objSpec.ParentRef != nil {
		parent, _ := objSpec.ParentRef.Resolve(ctx, r.Client, r.Scheme, obj)
		err = r.setOwner(ctx, obj, parent)
	} else {
		err = r.setOwner(ctx, obj, server)
	}

	if err != nil {
		logger.Error("Failed to set owner reference", zap.Error(err))

		r.EventRecorder.Eventf(obj, corev1.EventTypeWarning,
			"Failed", "Failed to set owner reference: %s", err)

		r.markFailed(ctx, obj, err)

		return ctrl.Result{}, nil
	}

	entry, err := r.MapToEntry(ctx, r.Client, r.Scheme, dn, obj)
	if err != nil {
		logger.Error("Failed to map to entry", zap.Error(err))

		r.EventRecorder.Eventf(obj, corev1.EventTypeWarning,
			"Failed", "Failed to map to entry: %s", err)

		r.markFailed(ctx, obj, err)

		return ctrl.Result{}, nil
	}

	created, err := directoryClient.CreateOrUpdateEntry(entry)
	if err != nil {
		logger.Error("Failed to create or update LDAP object", zap.Error(err))

		r.EventRecorder.Eventf(obj, corev1.EventTypeWarning,
			"Failed", "Failed to create or update LDAP object: %s", err)

		r.markFailed(ctx, obj, err)

		return ctrl.Result{}, nil
	}

	if created {
		logger.Info("LDAP object created")

		r.EventRecorder.Event(obj, corev1.EventTypeNormal,
			"Created", "Successfully created")
	}

	if err := r.markReady(ctx, obj); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to mark as ready: %w", err)
	}

	return ctrl.Result{}, nil
}

func (r *LDAPObjectReconciler[T, E]) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(r.newInstance()).
		Complete(r)
}

func (r *LDAPObjectReconciler[T, E]) newInstance() T {
	return reflect.New(reflect.TypeOf((*T)(nil)).Elem().Elem()).Interface().(T)
}

func (r *LDAPObjectReconciler[T, E]) isServerReady(ctx context.Context, obj T, server *openldapv1alpha1.LDAPServer) error {
	logger := util.LoggerFromContext(ctx)

	if server.Status.Phase != openldapv1alpha1.LDAPServerPhaseReady {
		logger.Warn("Referenced server not ready",
			zap.String("serverName", server.Name),
			zap.String("serverNamespace", server.Namespace))

		r.EventRecorder.Eventf(obj, corev1.EventTypeWarning,
			"NotReady", "Server %s in namespace %s not ready",
			server.Name, server.Namespace)

		if err := r.markPending(ctx, obj); err != nil {
			return err
		}

		return util.Retryable(fmt.Errorf("referenced server not ready"))
	}

	return nil
}

func (r *LDAPObjectReconciler[T, E]) markPending(ctx context.Context, obj T) error {
	_, err := controllerutil.CreateOrPatch(ctx, r.Client, obj, func() error {
		obj.SetStatus(api.SimpleStatus{
			Phase:              api.PhasePending,
			ObservedGeneration: obj.GetGeneration(),
		})

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	return nil
}

func (r *LDAPObjectReconciler[T, E]) markReady(ctx context.Context, obj T) error {
	_, err := controllerutil.CreateOrPatch(ctx, r.Client, obj, func() error {
		obj.SetStatus(api.SimpleStatus{
			Phase:              api.PhaseReady,
			ObservedGeneration: obj.GetGeneration(),
		})

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	return nil
}

func (r *LDAPObjectReconciler[T, E]) markFailed(ctx context.Context, obj T, err error) {
	logger := util.LoggerFromContext(ctx)

	_, updateErr := controllerutil.CreateOrPatch(ctx, r.Client, obj, func() error {
		obj.SetStatus(api.SimpleStatus{
			Phase:              api.PhaseFailed,
			ObservedGeneration: obj.GetGeneration(),
			Message:            err.Error(),
		})

		return nil
	})
	if updateErr != nil {
		logger.Error("Failed to update status", zap.Error(updateErr))
	}
}

func (r *LDAPObjectReconciler[T, E]) setOwner(ctx context.Context, obj T, owner runtime.Object) error {
	_, err := controllerutil.CreateOrPatch(ctx, r.Client, obj, func() error {
		return controllerutil.SetControllerReference(owner.(metav1.Object), obj, r.Scheme)
	})
	if err != nil {
		return err
	}

	return nil
}
