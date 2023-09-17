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

package main_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
)

func TestOperator(t *testing.T) {
	t.Log("Creating example resources")

	rootDir := os.Getenv("ROOT_DIR")

	require.NoError(t, createExampleResources(filepath.Join(rootDir, "examples")))

	kubeconfig := filepath.Join(clientcmd.RecommendedConfigDir, "config")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	require.NoError(t, err)

	dynClient, err := dynamic.NewForConfig(config)
	require.NoError(t, err)

	userGVR := schema.GroupVersionResource{
		Group:    "ldap.gpu-ninja.com",
		Version:  "v1alpha1",
		Resource: "ldapusers",
	}

	groupGVR := schema.GroupVersionResource{
		Group:    "ldap.gpu-ninja.com",
		Version:  "v1alpha1",
		Resource: "ldapgroups",
	}

	t.Log("Waiting for LDAPUser and LDAPGroup to be created")

	ctx := context.Background()
	err = wait.PollUntilContextTimeout(ctx, 5*time.Second, 5*time.Minute, true, func(ctx context.Context) (bool, error) {
		ldapUser, err := dynClient.Resource(userGVR).Namespace("default").Get(ctx, "demo", metav1.GetOptions{})
		if err != nil {
			if !apierrors.IsNotFound(err) {
				return true, err
			}

			return false, nil
		}

		phase, ok, err := unstructured.NestedString(ldapUser.Object, "status", "phase")
		if err != nil {
			return false, err
		}

		if !ok || phase != "Ready" {
			t.Log("LDAPUser not ready")

			return false, nil
		}

		ldapGroup, err := dynClient.Resource(groupGVR).Namespace("default").Get(ctx, "admins", metav1.GetOptions{})
		if err != nil {
			if !apierrors.IsNotFound(err) {
				return true, err
			}

			return false, nil
		}

		phase, ok, err = unstructured.NestedString(ldapGroup.Object, "status", "phase")
		if err != nil {
			return false, err
		}

		if !ok || phase != "Ready" {
			t.Log("LDAPGroup not ready")

			return false, nil
		}

		return true, nil
	})
	require.NoError(t, err)
}

func createExampleResources(examplesDir string) error {
	cmd := exec.Command("kapp", "deploy", "-y", "-a", "ldap-operator-examples", "-f", examplesDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
