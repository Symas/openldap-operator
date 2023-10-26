//go:build !ignore_autogenerated
// +build !ignore_autogenerated

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

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	"github.com/symas/operator-utils/reference"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LDAPDirectory) DeepCopyInto(out *LDAPDirectory) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LDAPDirectory.
func (in *LDAPDirectory) DeepCopy() *LDAPDirectory {
	if in == nil {
		return nil
	}
	out := new(LDAPDirectory)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *LDAPDirectory) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LDAPDirectoryList) DeepCopyInto(out *LDAPDirectoryList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]LDAPDirectory, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LDAPDirectoryList.
func (in *LDAPDirectoryList) DeepCopy() *LDAPDirectoryList {
	if in == nil {
		return nil
	}
	out := new(LDAPDirectoryList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *LDAPDirectoryList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LDAPDirectorySpec) DeepCopyInto(out *LDAPDirectorySpec) {
	*out = *in
	out.AdminPasswordSecretRef = in.AdminPasswordSecretRef
	out.CertificateSecretRef = in.CertificateSecretRef
	if in.DebugLevel != nil {
		in, out := &in.DebugLevel, &out.DebugLevel
		*out = new(int)
		**out = **in
	}
	if in.FileDescriptorLimit != nil {
		in, out := &in.FileDescriptorLimit, &out.FileDescriptorLimit
		*out = new(int)
		**out = **in
	}
	in.Storage.DeepCopyInto(&out.Storage)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LDAPDirectorySpec.
func (in *LDAPDirectorySpec) DeepCopy() *LDAPDirectorySpec {
	if in == nil {
		return nil
	}
	out := new(LDAPDirectorySpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LDAPDirectoryStatus) DeepCopyInto(out *LDAPDirectoryStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]v1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LDAPDirectoryStatus.
func (in *LDAPDirectoryStatus) DeepCopy() *LDAPDirectoryStatus {
	if in == nil {
		return nil
	}
	out := new(LDAPDirectoryStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LDAPDirectoryStorageSpec) DeepCopyInto(out *LDAPDirectoryStorageSpec) {
	*out = *in
	if in.StorageClassName != nil {
		in, out := &in.StorageClassName, &out.StorageClassName
		*out = new(string)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LDAPDirectoryStorageSpec.
func (in *LDAPDirectoryStorageSpec) DeepCopy() *LDAPDirectoryStorageSpec {
	if in == nil {
		return nil
	}
	out := new(LDAPDirectoryStorageSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LDAPGroup) DeepCopyInto(out *LDAPGroup) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LDAPGroup.
func (in *LDAPGroup) DeepCopy() *LDAPGroup {
	if in == nil {
		return nil
	}
	out := new(LDAPGroup)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *LDAPGroup) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LDAPGroupList) DeepCopyInto(out *LDAPGroupList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]LDAPGroup, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LDAPGroupList.
func (in *LDAPGroupList) DeepCopy() *LDAPGroupList {
	if in == nil {
		return nil
	}
	out := new(LDAPGroupList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *LDAPGroupList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LDAPGroupSpec) DeepCopyInto(out *LDAPGroupSpec) {
	*out = *in
	in.LDAPObjectSpec.DeepCopyInto(&out.LDAPObjectSpec)
	if in.Members != nil {
		in, out := &in.Members, &out.Members
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LDAPGroupSpec.
func (in *LDAPGroupSpec) DeepCopy() *LDAPGroupSpec {
	if in == nil {
		return nil
	}
	out := new(LDAPGroupSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LDAPOrganizationalUnit) DeepCopyInto(out *LDAPOrganizationalUnit) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LDAPOrganizationalUnit.
func (in *LDAPOrganizationalUnit) DeepCopy() *LDAPOrganizationalUnit {
	if in == nil {
		return nil
	}
	out := new(LDAPOrganizationalUnit)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *LDAPOrganizationalUnit) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LDAPOrganizationalUnitList) DeepCopyInto(out *LDAPOrganizationalUnitList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]LDAPOrganizationalUnit, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LDAPOrganizationalUnitList.
func (in *LDAPOrganizationalUnitList) DeepCopy() *LDAPOrganizationalUnitList {
	if in == nil {
		return nil
	}
	out := new(LDAPOrganizationalUnitList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *LDAPOrganizationalUnitList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LDAPOrganizationalUnitSpec) DeepCopyInto(out *LDAPOrganizationalUnitSpec) {
	*out = *in
	in.LDAPObjectSpec.DeepCopyInto(&out.LDAPObjectSpec)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LDAPOrganizationalUnitSpec.
func (in *LDAPOrganizationalUnitSpec) DeepCopy() *LDAPOrganizationalUnitSpec {
	if in == nil {
		return nil
	}
	out := new(LDAPOrganizationalUnitSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LDAPUser) DeepCopyInto(out *LDAPUser) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LDAPUser.
func (in *LDAPUser) DeepCopy() *LDAPUser {
	if in == nil {
		return nil
	}
	out := new(LDAPUser)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *LDAPUser) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LDAPUserList) DeepCopyInto(out *LDAPUserList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]LDAPUser, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LDAPUserList.
func (in *LDAPUserList) DeepCopy() *LDAPUserList {
	if in == nil {
		return nil
	}
	out := new(LDAPUserList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *LDAPUserList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LDAPUserSpec) DeepCopyInto(out *LDAPUserSpec) {
	*out = *in
	in.LDAPObjectSpec.DeepCopyInto(&out.LDAPObjectSpec)
	if in.PaswordSecretRef != nil {
		in, out := &in.PaswordSecretRef, &out.PaswordSecretRef
		*out = new(reference.LocalSecretReference)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LDAPUserSpec.
func (in *LDAPUserSpec) DeepCopy() *LDAPUserSpec {
	if in == nil {
		return nil
	}
	out := new(LDAPUserSpec)
	in.DeepCopyInto(out)
	return out
}
