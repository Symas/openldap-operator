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
	"strconv"

	"dario.cat/mergo"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	openldapv1alpha1 "github.com/gpu-ninja/openldap-operator/api/v1alpha1"
	"github.com/gpu-ninja/openldap-operator/internal/constants"
	"github.com/gpu-ninja/openldap-operator/internal/util"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

// Need to be able to read secrets to get the TLS certificates / passwords, etc.
//+kubebuilder:rbac:groups=v1,resources=secrets,verbs=get;list;watch

//+kubebuilder:rbac:groups=openldap.gpu-ninja.com,resources=ldapservers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=openldap.gpu-ninja.com,resources=ldapservers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=openldap.gpu-ninja.com,resources=ldapservers/finalizers,verbs=update

// LDAPServerReconciler reconciles a LDAPServer object
type LDAPServerReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	EventRecorder record.EventRecorder
}

func (r *LDAPServerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := util.LoggerFromContext(ctx)

	logger.Info("Reconciling server")

	var server openldapv1alpha1.LDAPServer
	if err := r.Get(ctx, req.NamespacedName, &server); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, err
	}

	statefulSetNamespaceName := types.NamespacedName{Name: server.Name, Namespace: server.Namespace}
	statefulSet := appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      statefulSetNamespaceName.Name,
			Namespace: statefulSetNamespaceName.Namespace,
			Labels:    server.ObjectMeta.Labels,
		},
	}

	if !controllerutil.ContainsFinalizer(&server, constants.FinalizerName) {
		logger.Info("Adding Finalizer")

		_, err := controllerutil.CreateOrPatch(ctx, r.Client, &server, func() error {
			controllerutil.AddFinalizer(&server, constants.FinalizerName)

			return nil
		})
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to add finalizer: %w", err)
		}
	}

	if !server.ObjectMeta.DeletionTimestamp.IsZero() {
		logger.Info("Deleting server")

		if controllerutil.ContainsFinalizer(&server, constants.FinalizerName) {
			logger.Info("Removing Finalizer")

			_, err := controllerutil.CreateOrPatch(ctx, r.Client, &server, func() error {
				controllerutil.RemoveFinalizer(&server, constants.FinalizerName)

				return nil
			})
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to remove finalizer: %w", err)
			}
		}

		return ctrl.Result{}, nil
	}

	// Make sure all references are resolvable.
	if err := server.ResolveReferences(ctx, r.Client, r.Scheme); err != nil {
		if util.IsRetryable(err) {
			logger.Info("Not all references are resolvable, requeuing")

			r.EventRecorder.Event(&server, corev1.EventTypeWarning,
				"NotReady", "Not all references are resolvable")

			if err := r.markPending(ctx, &server); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to mark as pending: %w", err)
			}

			return ctrl.Result{RequeueAfter: constants.ReconcileRetryInterval}, nil
		}

		logger.Error("Failed to resolve references", zap.Error(err))

		r.EventRecorder.Eventf(&server, corev1.EventTypeWarning,
			"Failed", "Failed to resolve references: %s", err)

		r.markFailed(ctx, &server, err)

		return ctrl.Result{}, nil
	}

	var creatingStatefulSet bool
	if err := r.Get(ctx, statefulSetNamespaceName, &statefulSet); err != nil && errors.IsNotFound(err) {
		creatingStatefulSet = true
	}

	statefulSetOpResult, err := controllerutil.CreateOrPatch(ctx, r.Client, &statefulSet, func() error {
		if err := controllerutil.SetOwnerReference(&server, &statefulSet, r.Scheme); err != nil {
			return fmt.Errorf("failed to set owner reference: %w", err)
		}

		storageSize, err := resource.ParseQuantity(server.Spec.Storage.Size)
		if err != nil {
			return fmt.Errorf("failed to parse requested storage size: %w", err)
		}

		envVars := []corev1.EnvVar{
			{
				Name:  "LDAP_DOMAIN",
				Value: server.Spec.Domain,
			},
			{
				Name:  "LDAP_ORGANIZATION",
				Value: server.Spec.Organization,
			},
		}

		if server.Spec.FileDescriptorLimit != nil {
			envVars = append(envVars, corev1.EnvVar{
				Name:  "LDAP_NOFILE",
				Value: strconv.Itoa(*server.Spec.FileDescriptorLimit),
			})
		}
		if server.Spec.DebugLevel != nil {
			envVars = append(envVars, corev1.EnvVar{
				Name:  "LDAP_DEBUG_LEVEL",
				Value: strconv.Itoa(*server.Spec.DebugLevel),
			})
		}

		spec := appsv1.StatefulSetSpec{
			Replicas:        ptr.To(int32(1)),
			ServiceName:     "openldap",
			MinReadySeconds: 10,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/name":     "openldap",
					"app.kubernetes.io/instance": server.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/name":     "openldap",
						"app.kubernetes.io/instance": server.Name,
					},
				},
				Spec: corev1.PodSpec{
					TerminationGracePeriodSeconds: ptr.To(int64(10)),
					SecurityContext: &corev1.PodSecurityContext{
						// The default Debian openldap group.
						FSGroup: ptr.To(int64(101)),
					},
					InitContainers: []corev1.Container{
						{
							Name:  "openldap-init",
							Image: server.Spec.Image,
							Command: []string{
								"/bootstrap.sh",
							},
							Env: envVars,
							EnvFrom: []corev1.EnvFromSource{
								{
									SecretRef: &corev1.SecretEnvSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: server.Spec.AdminPasswordSecretRef.Name,
										},
									},
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "openldap-config",
									MountPath: "/etc/ldap/slapd.d",
								},
								{
									Name:      "openldap-data",
									MountPath: "/var/lib/ldap",
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:  "openldap",
							Image: server.Spec.Image,
							Env:   envVars,
							Ports: []corev1.ContainerPort{
								{
									Name:          "ldaps",
									ContainerPort: 636,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									TCPSocket: &corev1.TCPSocketAction{
										Port: intstr.IntOrString{IntVal: 636},
									},
								},
								InitialDelaySeconds: 5,
								PeriodSeconds:       10,
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "openldap-config",
									MountPath: "/etc/ldap/slapd.d",
								},
								{
									Name:      "openldap-data",
									MountPath: "/var/lib/ldap",
								},
								{
									Name:      "openldap-certs",
									MountPath: "/etc/ldap/certs",
								},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("32Mi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("1"),
									corev1.ResourceMemory: resource.MustParse("256Mi"),
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "openldap-certs",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName:  server.Spec.CertificateSecretRef.Name,
									DefaultMode: ptr.To(int32(0o400)),
								},
							},
						},
					},
				},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "openldap-config",
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{
							corev1.ReadWriteOnce,
						},
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: resource.MustParse("10Mi"),
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "openldap-data",
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						StorageClassName: server.Spec.Storage.StorageClassName,
						AccessModes: []corev1.PersistentVolumeAccessMode{
							corev1.ReadWriteOnce,
						},
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: storageSize,
							},
						},
					},
				},
			},
		}

		if creatingStatefulSet {
			statefulSet.Spec = spec
		} else if err := mergo.Merge(&statefulSet.Spec, spec, mergo.WithOverride, mergo.WithSliceDeepCopy); err != nil {
			return fmt.Errorf("failed to merge spec: %w", err)
		}

		return nil
	})
	if err != nil {
		logger.Error("Failed to reconcile openldap statefulset", zap.Error(err))

		r.EventRecorder.Eventf(&server, corev1.EventTypeWarning,
			"Failed", "Failed to reconcile openldap statefulset: %v", err)

		r.markFailed(ctx, &server, err)

		return ctrl.Result{}, nil
	}

	if statefulSetOpResult != controllerutil.OperationResultNone {
		logger.Info("OpenLDAP StatefulSet successfully reconciled, marking as pending",
			zap.String("operation", string(statefulSetOpResult)))

		if err := r.markPending(ctx, &server); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to mark as pending: %w", err)
		}
	}

	serviceNamespaceName := types.NamespacedName{Name: server.Name, Namespace: server.Namespace}
	service := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceNamespaceName.Name,
			Namespace: serviceNamespaceName.Namespace,
			Labels:    server.ObjectMeta.Labels,
		},
	}

	// check if service already exists
	var creatingService bool
	if err := r.Client.Get(ctx, serviceNamespaceName, &service); err != nil && errors.IsNotFound(err) {
		creatingService = true
	}

	serviceOpResult, err := controllerutil.CreateOrPatch(ctx, r.Client, &service, func() error {
		if err := controllerutil.SetControllerReference(&server, &service, r.Scheme); err != nil {
			return fmt.Errorf("failed to set controller reference: %w", err)
		}

		spec := corev1.ServiceSpec{
			Selector: map[string]string{
				"app.kubernetes.io/name":     "openldap",
				"app.kubernetes.io/instance": server.Name,
			},
			Ports: []corev1.ServicePort{
				{
					Port:       636,
					TargetPort: intstr.FromInt(636),
					Name:       "ldaps",
					Protocol:   corev1.ProtocolTCP,
				},
			},
		}

		if creatingService {
			service.Spec = spec
		} else {
			if err := mergo.Merge(&service.Spec, spec, mergo.WithOverride, mergo.WithSliceDeepCopy); err != nil {
				return fmt.Errorf("failed to merge spec: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		logger.Error("Failed to reconcile openldap service", zap.Error(err))

		r.EventRecorder.Eventf(&server, corev1.EventTypeWarning,
			"Failed", "Failed to reconcile openldap statefulset: %v", err)

		r.markFailed(ctx, &server, err)

		return ctrl.Result{}, nil
	}

	if serviceOpResult != controllerutil.OperationResultNone {
		logger.Info("OpenLDAP Service successfully reconciled",
			zap.String("operation", string(serviceOpResult)))
	}

	if statefulSet.Status.ReadyReplicas != *statefulSet.Spec.Replicas {
		logger.Info("Waiting for LDAPServer to become ready")

		if err := r.markPending(ctx, &server); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to mark as pending: %w", err)
		}

		return ctrl.Result{RequeueAfter: constants.ReconcileRetryInterval}, nil
	}

	if server.Status.Phase != openldapv1alpha1.LDAPServerPhaseReady {
		r.EventRecorder.Event(&server, corev1.EventTypeNormal,
			"Created", "Successfully created")

		if err := r.markReady(ctx, &server); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to mark as ready: %w", err)
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *LDAPServerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&openldapv1alpha1.LDAPServer{}).
		Complete(r)
}

func (r *LDAPServerReconciler) markPending(ctx context.Context, server *openldapv1alpha1.LDAPServer) error {
	updatedConditions := make([]openldapv1alpha1.LDAPServerCondition, len(server.Status.Conditions))
	copy(updatedConditions, server.Status.Conditions)

	if server.Status.Phase != openldapv1alpha1.LDAPServerPhasePending {
		updatedConditions = append(updatedConditions, openldapv1alpha1.LDAPServerCondition{
			Type:               openldapv1alpha1.LDAPServerConditionTypePending,
			Status:             corev1.ConditionTrue,
			LastTransitionTime: metav1.Now(),
			Reason:             "Pending",
			Message:            "LDAP Server is pending",
		})
	}

	_, err := controllerutil.CreateOrPatch(ctx, r.Client, server, func() error {
		server.Status.ObservedGeneration = server.Generation
		server.Status.Phase = openldapv1alpha1.LDAPServerPhasePending
		server.Status.Conditions = updatedConditions

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	return nil
}

func (r *LDAPServerReconciler) markReady(ctx context.Context, server *openldapv1alpha1.LDAPServer) error {
	updatedConditions := make([]openldapv1alpha1.LDAPServerCondition, len(server.Status.Conditions)+1)
	copy(updatedConditions, server.Status.Conditions)

	updatedConditions[len(server.Status.Conditions)] = openldapv1alpha1.LDAPServerCondition{
		Type:               openldapv1alpha1.LDAPServerConditionTypeReady,
		Status:             corev1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		Reason:             "Ready",
		Message:            "LDAP Server is ready and running",
	}

	_, err := controllerutil.CreateOrPatch(ctx, r.Client, server, func() error {
		server.Status.ObservedGeneration = server.Generation
		server.Status.Phase = openldapv1alpha1.LDAPServerPhaseReady
		server.Status.Conditions = updatedConditions

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	return nil
}

func (r *LDAPServerReconciler) markFailed(ctx context.Context, server *openldapv1alpha1.LDAPServer, err error) {
	logger := util.LoggerFromContext(ctx)

	updatedConditions := make([]openldapv1alpha1.LDAPServerCondition, len(server.Status.Conditions)+1)
	copy(updatedConditions, server.Status.Conditions)

	updatedConditions[len(server.Status.Conditions)] = openldapv1alpha1.LDAPServerCondition{
		Type:               openldapv1alpha1.LDAPServerConditionTypeFailed,
		Status:             corev1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		Reason:             "Failed",
		Message:            err.Error(),
	}

	_, updateErr := controllerutil.CreateOrPatch(ctx, r.Client, server, func() error {
		server.Status.ObservedGeneration = server.Generation
		server.Status.Phase = openldapv1alpha1.LDAPServerPhaseFailed
		server.Status.Conditions = updatedConditions

		return nil
	})
	if updateErr != nil {
		logger.Error("Failed to update status", zap.Error(updateErr))
	}
}
