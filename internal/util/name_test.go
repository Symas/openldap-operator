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
	"regexp"
	"testing"

	"github.com/gpu-ninja/openldap-operator/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateName(t *testing.T) {
	prefix := "test"
	generatedName := util.GenerateName(prefix)

	expectedPattern := "^" + prefix + "-[a-z0-9]{5}$"
	valid, err := regexp.MatchString(expectedPattern, generatedName)
	require.NoError(t, err)

	assert.True(t, valid)

	anotherName := util.GenerateName(prefix)
	assert.NotEqual(t, generatedName, anotherName)
}
