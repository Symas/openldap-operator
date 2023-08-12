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

package directory

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"time"

	"github.com/go-ldap/ldap/v3"
	"k8s.io/apimachinery/pkg/util/sets"
)

// Client is an LDAP directory client.
type Client interface {
	Ping() error
	GetEntry(dn string, entry any) error
	CreateOrUpdateEntry(entry any) (created bool, err error)
	DeleteEntry(dn string, cascading bool) error
}

type clientImpl struct {
	serverAddress string
	caBundle      *x509.CertPool
	adminUsername string
	adminPassword string
	baseDN        string
}

// Ping checks if the directory is available and responding to requests.
func (c *clientImpl) Ping() error {
	conn, err := c.connect()
	if err != nil {
		return err
	}
	defer conn.Close()

	searchRequest := ldap.NewSearchRequest(
		c.baseDN,
		ldap.ScopeBaseObject, ldap.DerefAlways, 0, 0, false,
		"(objectClass=*)",
		[]string{"1.1"},
		nil,
	)

	_, err = conn.Search(searchRequest)
	if err != nil {
		return err
	}

	return nil
}

func (c *clientImpl) GetEntry(dn string, entry any) error {
	switch entry := entry.(type) {
	case *OrganizationalUnit:
		ou, err := c.getOrganizationalUnit(dn)
		if err != nil {
			return err
		}

		*entry = *ou
	case *Group:
		g, err := c.getGroup(dn)
		if err != nil {
			return err
		}

		*entry = *g
	case *User:
		u, err := c.getUser(dn)
		if err != nil {
			return err
		}

		*entry = *u
	default:
		return fmt.Errorf("unsupported entry type: %T", entry)
	}

	return nil
}

func (c *clientImpl) CreateOrUpdateEntry(entry any) (bool, error) {
	switch entry := entry.(type) {
	case *OrganizationalUnit:
		return c.createOrUpdateOrganizationalUnit(entry)
	case *Group:
		return c.createOrUpdateGroup(entry)
	case *User:
		return c.createOrUpdateUser(entry)
	default:
		return false, fmt.Errorf("unsupported entry type: %T", entry)
	}
}

// DeleteEntry deletes an entry from the directory (if it exists).
// If cascading is true, all child entries will also be deleted.
func (c *clientImpl) DeleteEntry(dn string, cascading bool) error {
	conn, err := c.connect()
	if err != nil {
		return err
	}
	defer conn.Close()

	if cascading {
		// Search for child entries
		searchRequest := ldap.NewSearchRequest(
			dn,
			ldap.ScopeSingleLevel, ldap.NeverDerefAliases, 0, 0, false,
			"(objectClass=*)",
			[]string{"dn"},
			nil,
		)
		searchResult, err := conn.Search(searchRequest)
		if err != nil && !ldap.IsErrorWithCode(err, ldap.LDAPResultNoSuchObject) {
			return err
		}

		// Delete child entries if any
		for _, entry := range searchResult.Entries {
			childDN := entry.DN
			if err := c.DeleteEntry(childDN, true); err != nil {
				return err
			}
		}
	}

	delRequest := ldap.NewDelRequest(dn, nil)
	err = conn.Del(delRequest)
	if err != nil && !ldap.IsErrorWithCode(err, ldap.LDAPResultNoSuchObject) {
		return err
	}

	return nil
}

func (c *clientImpl) getOrganizationalUnit(dn string) (*OrganizationalUnit, error) {
	conn, err := c.connect()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	searchRequest := ldap.NewSearchRequest(
		dn,
		ldap.ScopeBaseObject, ldap.NeverDerefAliases, 1, 0, false,
		"(&(objectClass=organizationalUnit))",
		[]string{"dn", "ou", "description"},
		nil,
	)

	searchResult, err := conn.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to search for organizational unit: %w", err)
	}

	return &OrganizationalUnit{
		DistinguishedName: searchResult.Entries[0].DN,
		Name:              searchResult.Entries[0].GetAttributeValue("ou"),
		Description:       searchResult.Entries[0].GetAttributeValue("description"),
	}, nil
}

func (c *clientImpl) createOrUpdateOrganizationalUnit(ou *OrganizationalUnit) (bool, error) {
	conn, err := c.connect()
	if err != nil {
		return false, err
	}
	defer conn.Close()

	searchRequest := ldap.NewSearchRequest(
		ou.DistinguishedName,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		fmt.Sprintf("(&(objectClass=organizationalUnit)(ou=%s))", ou.Name),
		[]string{"dn", "ou", "description"},
		nil,
	)

	searchResult, err := conn.Search(searchRequest)
	if err != nil && !ldap.IsErrorWithCode(err, ldap.LDAPResultNoSuchObject) {
		return false, fmt.Errorf("failed to search for organizational unit: %w", err)
	}

	// If the organizational unit does not exist, create it.
	if len(searchResult.Entries) == 0 {
		addRequest := ldap.NewAddRequest(ou.DistinguishedName, nil)
		addRequest.Attribute("objectClass", []string{"top", "organizationalUnit"})
		addRequest.Attribute("ou", []string{ou.Name})
		if ou.Description != "" {
			addRequest.Attribute("description", []string{ou.Description})
		}

		if err := conn.Add(addRequest); err != nil {
			return false, fmt.Errorf("failed to create organizational unit: %w", err)
		}

		return true, nil
	}

	entry := searchResult.Entries[0]

	modifyRequest := ldap.NewModifyRequest(ou.DistinguishedName, nil)

	existingDescription := entry.GetAttributeValue("description")
	optionalAttributeModifications(modifyRequest, "description", existingDescription, ou.Description)

	if len(modifyRequest.Changes) > 0 {
		if err := conn.Modify(modifyRequest); err != nil {
			return false, fmt.Errorf("failed to update organizational unit: %w", err)
		}
	}

	return false, nil
}

func (c *clientImpl) getGroup(dn string) (*Group, error) {
	conn, err := c.connect()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	searchRequest := ldap.NewSearchRequest(
		dn,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 1, 0, false,
		"(objectClass=groupOfNames)",
		[]string{"dn", "cn", "description", "member"},
		nil,
	)

	searchResult, err := conn.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to search for group: %w", err)
	}

	return &Group{
		DistinguishedName: searchResult.Entries[0].DN,
		Name:              searchResult.Entries[0].GetAttributeValue("cn"),
		Description:       searchResult.Entries[0].GetAttributeValue("description"),
		Members:           searchResult.Entries[0].GetAttributeValues("member"),
	}, nil
}

func (c *clientImpl) createOrUpdateGroup(group *Group) (bool, error) {
	conn, err := c.connect()
	if err != nil {
		return false, err
	}
	defer conn.Close()

	searchRequest := ldap.NewSearchRequest(
		group.DistinguishedName,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		fmt.Sprintf("(&(objectClass=groupOfNames)(cn=%s))", group.Name),
		[]string{"dn", "cn", "description", "member"},
		nil,
	)

	searchResult, err := conn.Search(searchRequest)
	if err != nil && !ldap.IsErrorWithCode(err, ldap.LDAPResultNoSuchObject) {
		return false, fmt.Errorf("failed to search for group: %w", err)
	}

	// If the group does not exist, create it.
	if len(searchResult.Entries) == 0 {
		addRequest := ldap.NewAddRequest(group.DistinguishedName, nil)
		addRequest.Attribute("objectClass", []string{"top", "groupOfNames"})
		addRequest.Attribute("cn", []string{group.Name})
		addRequest.Attribute("member", group.Members)
		if group.Description != "" {
			addRequest.Attribute("description", []string{group.Description})
		}

		if err := conn.Add(addRequest); err != nil {
			return false, fmt.Errorf("failed to create group: %w", err)
		}

		return true, nil
	}

	entry := searchResult.Entries[0]

	modifyRequest := ldap.NewModifyRequest(group.DistinguishedName, nil)

	existingDescription := entry.GetAttributeValue("description")
	optionalAttributeModifications(modifyRequest, "description", existingDescription, group.Description)

	existingMembers := entry.GetAttributeValues("member")
	if !sets.New(existingMembers...).Equal(sets.New(group.Members...)) {
		modifyRequest.Replace("member", group.Members)
	}

	if len(modifyRequest.Changes) > 0 {
		if err := conn.Modify(modifyRequest); err != nil {
			return false, fmt.Errorf("failed to update group: %w", err)
		}
	}

	return false, nil
}

func (c *clientImpl) getUser(dn string) (*User, error) {
	conn, err := c.connect()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	searchRequest := ldap.NewSearchRequest(
		dn,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 1, 0, false,
		"(objectClass=inetOrgPerson)",
		[]string{"dn", "uid", "cn", "sn", "mail", "userPassword"},
		nil,
	)

	searchResult, err := conn.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to search for user: %w", err)
	}

	return &User{
		DistinguishedName: searchResult.Entries[0].DN,
		Username:          searchResult.Entries[0].GetAttributeValue("uid"),
		Name:              searchResult.Entries[0].GetAttributeValue("cn"),
		Surname:           searchResult.Entries[0].GetAttributeValue("sn"),
		Email:             searchResult.Entries[0].GetAttributeValue("mail"),
		Password:          searchResult.Entries[0].GetAttributeValue("userPassword"),
	}, nil
}

func (c *clientImpl) createOrUpdateUser(user *User) (bool, error) {
	conn, err := c.connect()
	if err != nil {
		return false, err
	}
	defer conn.Close()

	searchRequest := ldap.NewSearchRequest(
		user.DistinguishedName,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		fmt.Sprintf("(&(objectClass=inetOrgPerson)(uid=%s))", user.Username),
		[]string{"dn", "uid", "cn", "sn", "mail"},
		nil,
	)

	searchResult, err := conn.Search(searchRequest)
	if err != nil && !ldap.IsErrorWithCode(err, ldap.LDAPResultNoSuchObject) {
		return false, fmt.Errorf("failed to search for user: %w", err)
	}

	// If the user does not exist, create it.
	if len(searchResult.Entries) == 0 {
		addRequest := ldap.NewAddRequest(user.DistinguishedName, nil)
		addRequest.Attribute("objectClass", []string{"top", "person", "organizationalPerson", "inetOrgPerson"})
		addRequest.Attribute("uid", []string{user.Username})
		addRequest.Attribute("cn", []string{user.Name})
		addRequest.Attribute("sn", []string{user.Surname})

		if user.Email != "" {
			addRequest.Attribute("mail", []string{user.Email})
		}

		if err := conn.Add(addRequest); err != nil {
			return false, fmt.Errorf("failed to create user: %w", err)
		}

		if user.Password != "" {
			passwordModifyRequest := ldap.NewPasswordModifyRequest(user.DistinguishedName, "", user.Password)
			if _, err := conn.PasswordModify(passwordModifyRequest); err != nil {
				return false, fmt.Errorf("failed to set user password: %w", err)
			}
		}

		return true, nil
	}

	entry := searchResult.Entries[0]

	modifyRequest := ldap.NewModifyRequest(user.DistinguishedName, nil)

	existingCommonName := entry.GetAttributeValue("cn")
	if existingCommonName != user.Name {
		modifyRequest.Replace("cn", []string{user.Name})
	}

	existingSurname := entry.GetAttributeValue("sn")
	if existingSurname != user.Surname {
		modifyRequest.Replace("sn", []string{user.Surname})
	}

	existingEmail := entry.GetAttributeValue("mail")
	optionalAttributeModifications(modifyRequest, "mail", existingEmail, user.Email)

	if len(modifyRequest.Changes) > 0 {
		if err := conn.Modify(modifyRequest); err != nil {
			return false, fmt.Errorf("failed to update user: %w", err)
		}
	}

	connUser, err := c.connect()
	if err != nil {
		return false, fmt.Errorf("failed to connect to verify password: %w", err)
	}
	defer connUser.Close()

	var passwordChanged bool
	if err := connUser.Bind(user.DistinguishedName, user.Password); err != nil {
		if ldap.IsErrorWithCode(err, ldap.LDAPResultInvalidCredentials) {
			passwordChanged = true
		} else {
			return false, fmt.Errorf("failed to verify password: %w", err)
		}
	}

	if passwordChanged {
		passwordModifyRequest := ldap.NewPasswordModifyRequest(user.DistinguishedName, "", user.Password)
		if _, err := conn.PasswordModify(passwordModifyRequest); err != nil {
			return false, fmt.Errorf("failed to set user password: %w", err)
		}
	}

	return false, nil
}

func (c *clientImpl) connect() (*ldap.Conn, error) {
	conn, err := ldap.DialURL(c.serverAddress, ldap.DialWithTLSConfig(&tls.Config{
		RootCAs: c.caBundle,
	}))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to ldap server: %w", err)
	}

	conn.SetTimeout(5 * time.Second)

	if err := conn.Bind(c.adminUsername, c.adminPassword); err != nil {
		return nil, fmt.Errorf("failed to bind to ldap server: %w", err)
	}

	return conn, nil
}

func optionalAttributeModifications(modifyRequest *ldap.ModifyRequest, attributeName string, existing, desired string) {
	if desired == "" {
		if existing != "" {
			modifyRequest.Delete(attributeName, []string{})
		}
	} else if existing == "" {
		modifyRequest.Add(attributeName, []string{desired})
	} else {
		modifyRequest.Replace(attributeName, []string{desired})
	}
}
