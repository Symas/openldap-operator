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

package util_test

import (
	"testing"

	"github.com/gpu-ninja/openldap-operator/internal/util"
)

func TestEqual(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		if !util.Equal([]int{}, []int{}) {
			t.Error("Expected empty slices to be equal")
		}
	})

	t.Run("Different Lengths", func(t *testing.T) {
		if util.Equal([]float64{1.2, 3.4}, []float64{1.2}) {
			t.Error("Expected slices with different lengths to be not equal")
		}
	})

	t.Run("Different Elements", func(t *testing.T) {
		if util.Equal([]int{1, 2, 3}, []int{1, 3, 4}) {
			t.Error("Expected slices with different elements to be not equal")
		}
	})

	t.Run("Matching Same Order", func(t *testing.T) {
		if !util.Equal([]string{"apple", "banana"}, []string{"apple", "banana"}) {
			t.Error("Expected slices with the same elements and the same order to be equal")
		}
	})

	t.Run("Matching Different Order", func(t *testing.T) {
		if !util.Equal([]string{"a", "b", "c"}, []string{"c", "b", "a"}) {
			t.Error("Expected slices with the same elements but different order to be equal")
		}
	})
}

func TestContains(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		if util.Contains([]int{}, 1) {
			t.Error("Expected empty slice to not contain anything")
		}
	})

	t.Run("Not Found", func(t *testing.T) {
		if util.Contains([]int{1, 2, 3}, 4) {
			t.Error("Expected slice to not contain the element")
		}
	})

	t.Run("Found", func(t *testing.T) {
		if !util.Contains([]int{1, 2, 3}, 2) {
			t.Error("Expected slice to contain the element")
		}
	})
}
