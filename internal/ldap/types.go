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

package ldap

type Entry interface {
	*OrganizationalUnit | *Group | *User
}

// OrganizationalUnit represents an organizational unit in the directory.
// Organizational units are containers that can hold users, groups, and other
// organizational units, allowing for a hierarchy of these objects.
type OrganizationalUnit struct {
	// DistinguishedName is the unique identifier for this organizational unit within the directory.
	DistinguishedName string
	// Name is the common name for this organizational unit.
	Name string
	// Description is an optional description of this organizational unit.
	Description string
}

// Group represents a group of names in the directory. Groups are used
// to manage permissions and may contain users, other groups, or organizational
// units. Members of a group can be granted access to resources or permissions
// based on their group membership.
type Group struct {
	// DistinguishedName is the unique identifier for this group within the directory.
	DistinguishedName string
	// Name is the common name for this group.
	Name string
	// Description is an optional description of this group.
	Description string
	// Members is a list of distinguished names representing the members of this group.
	Members []string
}

// User represents a user in the directory. Users are individual
// account entities that may log into a system, be assigned permissions, and
// be included in groups.
type User struct {
	// DistinguishedName is the unique identifier for this user within the directory.
	DistinguishedName string
	// Username is the username (uid) for this user.
	Username string
	// Name is the full name of this user (commonName).
	Name string
	// Surname is the surname of this user.
	Surname string
	// Email is an optional email address of this user.
	Email string
	// Password is an optional password for this user.
	Password string
}
