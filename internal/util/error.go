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

type retryableError struct {
	error
}

// Retryable wraps an error to indicate that it is retryable.
func Retryable(err error) error {
	return retryableError{
		error: err,
	}
}

// IsRetryable returns true if the error is retryable.
func IsRetryable(err error) bool {
	_, ok := err.(retryableError)

	return ok
}
