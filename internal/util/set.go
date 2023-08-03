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

package util

// Equal returns true if the two slices contain the same elements, regardless of order.
func Equal[T comparable](a []T, b []T) bool {
	if len(a) != len(b) {
		return false
	}

	countsA := make(map[T]int)
	countsB := make(map[T]int)

	for _, v := range a {
		countsA[v]++
	}

	for _, v := range b {
		countsB[v]++
	}

	for key, count := range countsA {
		if countsB[key] != count {
			return false
		}
	}

	return true
}

// Contains returns true if the slice contains the given element.
func Contains[T comparable](elems []T, v T) bool {
	for _, s := range elems {
		if v == s {
			return true
		}
	}

	return false
}
