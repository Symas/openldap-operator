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

	"github.com/gpu-ninja/ldap-operator/api"
	ldapv1alpha1 "github.com/gpu-ninja/ldap-operator/api/v1alpha1"
	"github.com/gpu-ninja/ldap-operator/internal/ldap"
	"github.com/gpu-ninja/ldap-operator/internal/mapper"
	"github.com/gpu-ninja/operator-utils/updater"
	"github.com/gpu-ninja/operator-utils/zaplogr"
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
//+kubebuilder:rbac:groups=ldap.gpu-ninja.com,resources=ldapgroups,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=ldap.gpu-ninja.com,resources=ldapgroups/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=ldap.gpu-ninja.com,resources=ldapgroups/finalizers,verbs=update

// LDAPOrganizationalUnits
//+kubebuilder:rbac:groups=ldap.gpu-ninja.com,resources=ldaporganizationalunits,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=ldap.gpu-ninja.com,resources=ldaporganizationalunits/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=ldap.gpu-ninja.com,resources=ldaporganizationalunits/finalizers,verbs=update

// LDAPUsers
//+kubebuilder:rbac:groups=ldap.gpu-ninja.com,resources=ldapusers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=ldap.gpu-ninja.com,resources=ldapusers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=ldap.gpu-ninja.com,resources=ldapusers/finalizers,verbs=update

type LDAPObjectReconciler[T api.LDAPObject, E ldap.Entry] struct {
	client.Client
	Scheme            *runtime.Scheme
	Recorder          record.EventRecorder
	LDAPClientBuilder ldap.ClientBuilder
	MapToEntry        mapper.Mapper[T, E]
}

func (r *LDAPObjectReconciler[T, E]) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := zaplogr.FromContext(ctx)

	logger.Info("Reconciling")

	obj := r.newInstance()
	if err := r.Get(ctx, req.NamespacedName, obj); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, err
	}

	if !controllerutil.ContainsFinalizer(obj, FinalizerName) {
		logger.Info("Adding Finalizer")

		_, err := controllerutil.CreateOrPatch(ctx, r.Client, obj, func() error {
			controllerutil.AddFinalizer(obj, FinalizerName)

			return nil
		})
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to add finalizer: %w", err)
		}
	}

	ok, err := obj.ResolveReferences(ctx, r.Client, r.Scheme)
	if !ok && err == nil {
		if !obj.GetDeletionTimestamp().IsZero() {
			// Parent has probably been removed by a cascading delete.
			// So there is probably no point in retrying.

			_, err := controllerutil.CreateOrPatch(ctx, r.Client, obj, func() error {
				controllerutil.RemoveFinalizer(obj, FinalizerName)

				return nil
			})
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to remove finalizer: %w", err)
			}

			return ctrl.Result{}, nil
		}

		logger.Info("Not all references are resolvable, requeuing")

		r.Recorder.Event(obj, corev1.EventTypeWarning,
			"NotReady", "Not all references are resolvable")

		if err := r.markPending(ctx, obj); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{RequeueAfter: reconcileRetryInterval}, nil
	} else if err != nil {
		r.Recorder.Eventf(obj, corev1.EventTypeWarning,
			"Failed", "Failed to resolve references: %s", err)

		r.markFailed(ctx, obj,
			fmt.Errorf("failed to resolve references: %w", err))

		return ctrl.Result{}, fmt.Errorf("failed to resolve references: %w", err)
	}

	objSpec := obj.GetLDAPObjectSpec()
	directoryObj, _, err := objSpec.DirectoryRef.Resolve(ctx, r.Client, r.Scheme, obj)
	if err != nil {
		logger.Error("Failed to resolve directory reference", zap.Error(err))

		r.Recorder.Eventf(obj, corev1.EventTypeWarning,
			"Failed", "Failed to resolve directory reference: %s", err)

		r.markFailed(ctx, obj,
			fmt.Errorf("failed to resolve directory reference: %w", err))

		return ctrl.Result{}, nil
	}
	directory := directoryObj.(*ldapv1alpha1.LDAPDirectory)

	if directory.Status.Phase != ldapv1alpha1.LDAPDirectoryPhaseReady {
		logger.Info("Referenced directory not ready",
			zap.String("namespace", directory.Namespace),
			zap.String("name", directory.Name))

		r.Recorder.Event(obj, corev1.EventTypeWarning,
			"NotReady", "Referenced directory is not ready")

		if err := r.markPending(ctx, obj); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{RequeueAfter: reconcileRetryInterval}, nil
	}

	var parent runtime.Object
	if objSpec.ParentRef != nil {
		parent, _, err = objSpec.ParentRef.Resolve(ctx, r.Client, r.Scheme, obj)
		if err != nil {
			// Should never happen as we've invoked ResolveReferences above.
			logger.Error("Failed to resolve parent reference", zap.Error(err))

			r.Recorder.Eventf(obj, corev1.EventTypeWarning,
				"Failed", "Failed to resolve parent reference: %s", err)

			r.markFailed(ctx, obj,
				fmt.Errorf("failed to resolve parent reference: %w", err))

			return ctrl.Result{}, nil
		}
	}

	if parent != nil {
		parentObj, ok := parent.(api.LDAPObject)
		if !ok {
			logger.Error("Parent is not an LDAP Object")

			r.Recorder.Event(obj, corev1.EventTypeWarning,
				"Failed", "Parent is not an ldap object")

			err := fmt.Errorf("parent is not an ldap object")
			r.markFailed(ctx, obj, err)

			return ctrl.Result{}, err
		}

		if parentObj.GetPhase() != api.PhaseReady {
			logger.Info("Referenced parent object is not ready",
				zap.String("namespace", parentObj.GetNamespace()), zap.String("name", parentObj.GetName()))

			r.Recorder.Event(obj, corev1.EventTypeWarning,
				"NotReady", "Referenced parent object is not ready")

			if err := r.markPending(ctx, obj); err != nil {
				return ctrl.Result{}, err
			}

			return ctrl.Result{RequeueAfter: reconcileRetryInterval}, nil
		}
	}

	ldapClient, err := r.LDAPClientBuilder.WithDirectory(directory).Build(ctx)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to create directory client: %w", err)
	}

	dn, err := obj.GetDistinguishedName(ctx, r.Client, r.Scheme)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get distinguished name: %w", err)
	}

	logger = logger.With(zap.String("dn", dn))

	if !obj.GetDeletionTimestamp().IsZero() {
		logger.Info("Deleting")

		if err := ldapClient.DeleteEntry(dn, true); err != nil {
			// Don't block deletion.
			logger.Error("Failed to delete LDAP entry, skipping deletion", zap.Error(err))
		}

		_, err := controllerutil.CreateOrPatch(ctx, r.Client, obj, func() error {
			controllerutil.RemoveFinalizer(obj, FinalizerName)

			return nil
		})
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to remove finalizer: %w", err)
		}

		return ctrl.Result{}, nil
	}

	logger.Info("Creating or updating")

	// Set owner reference to parent (if one is specified).
	var owner runtime.Object = directory
	if objSpec.ParentRef != nil {
		owner = parent
	}

	if err := r.setOwner(ctx, obj, owner); err != nil {
		logger.Error("Failed to set owner reference", zap.Error(err))

		r.Recorder.Eventf(obj, corev1.EventTypeWarning,
			"Failed", "Failed to set owner reference: %s", err)

		r.markFailed(ctx, obj,
			fmt.Errorf("failed to set owner reference: %w", err))

		return ctrl.Result{}, nil
	}

	entry, err := r.MapToEntry(ctx, r.Client, r.Scheme, dn, obj)
	if err != nil {
		logger.Error("Failed to map to LDAP entry", zap.Error(err))

		r.Recorder.Eventf(obj, corev1.EventTypeWarning,
			"Failed", "Failed to map to ldap entry: %s", err)

		r.markFailed(ctx, obj,
			fmt.Errorf("failed to map to ldap entry: %w", err))

		return ctrl.Result{}, nil
	}

	created, err := ldapClient.CreateOrUpdateEntry(entry)
	if err != nil {
		logger.Error("Failed to create or update LDAP entry", zap.Error(err))

		r.Recorder.Eventf(obj, corev1.EventTypeWarning,
			"Failed", "Failed to create or update ldap entry: %s", err)

		r.markFailed(ctx, obj,
			fmt.Errorf("failed to create or update ldap entry: %w", err))

		return ctrl.Result{}, nil
	}

	if created {
		r.Recorder.Event(obj, corev1.EventTypeNormal,
			"Created", "Successfully created")

		if err := r.markReady(ctx, obj); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *LDAPObjectReconciler[T, E]) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(r.newInstance()).
		Complete(r)
}

func (r *LDAPObjectReconciler[T, E]) newInstance() T {
	// Oh god, what have I done?
	return reflect.New(reflect.TypeOf((*T)(nil)).Elem().Elem()).Interface().(T)
}

func (r *LDAPObjectReconciler[T, E]) markPending(ctx context.Context, obj T) error {
	key := client.ObjectKeyFromObject(obj)
	err := updater.UpdateStatus(ctx, r.Client, key, obj, func() error {
		obj.SetStatus(api.SimpleStatus{
			Phase:              api.PhasePending,
			ObservedGeneration: obj.GetGeneration(),
		})

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to mark as pending: %w", err)
	}

	return nil
}

func (r *LDAPObjectReconciler[T, E]) markReady(ctx context.Context, obj T) error {
	key := client.ObjectKeyFromObject(obj)
	err := updater.UpdateStatus(ctx, r.Client, key, obj, func() error {
		obj.SetStatus(api.SimpleStatus{
			Phase:              api.PhaseReady,
			ObservedGeneration: obj.GetGeneration(),
		})

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to mark as ready: %w", err)
	}

	return nil
}

func (r *LDAPObjectReconciler[T, E]) markFailed(ctx context.Context, obj T, err error) {
	logger := zaplogr.FromContext(ctx)

	key := client.ObjectKeyFromObject(obj)
	updateErr := updater.UpdateStatus(ctx, r.Client, key, obj, func() error {
		obj.SetStatus(api.SimpleStatus{
			Phase:              api.PhaseFailed,
			ObservedGeneration: obj.GetGeneration(),
			Message:            err.Error(),
		})

		return nil
	})
	if updateErr != nil {
		logger.Error("Failed to mark as failed", zap.Error(updateErr))
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
