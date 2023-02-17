//go:build !ignore_autogenerated
// +build !ignore_autogenerated

/*
Copyright The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/errors"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *APIEndpoint) DeepCopyInto(out *APIEndpoint) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new APIEndpoint.
func (in *APIEndpoint) DeepCopy() *APIEndpoint {
	if in == nil {
		return nil
	}
	out := new(APIEndpoint)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ControlPlaneServiceTemplate) DeepCopyInto(out *ControlPlaneServiceTemplate) {
	*out = *in
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ControlPlaneServiceTemplate.
func (in *ControlPlaneServiceTemplate) DeepCopy() *ControlPlaneServiceTemplate {
	if in == nil {
		return nil
	}
	out := new(ControlPlaneServiceTemplate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubevirtCluster) DeepCopyInto(out *KubevirtCluster) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubevirtCluster.
func (in *KubevirtCluster) DeepCopy() *KubevirtCluster {
	if in == nil {
		return nil
	}
	out := new(KubevirtCluster)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *KubevirtCluster) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubevirtClusterList) DeepCopyInto(out *KubevirtClusterList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]KubevirtCluster, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubevirtClusterList.
func (in *KubevirtClusterList) DeepCopy() *KubevirtClusterList {
	if in == nil {
		return nil
	}
	out := new(KubevirtClusterList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *KubevirtClusterList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubevirtClusterSpec) DeepCopyInto(out *KubevirtClusterSpec) {
	*out = *in
	out.ControlPlaneEndpoint = in.ControlPlaneEndpoint
	in.ControlPlaneServiceTemplate.DeepCopyInto(&out.ControlPlaneServiceTemplate)
	in.SshKeys.DeepCopyInto(&out.SshKeys)
	if in.InfraClusterSecretRef != nil {
		in, out := &in.InfraClusterSecretRef, &out.InfraClusterSecretRef
		*out = new(v1.ObjectReference)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubevirtClusterSpec.
func (in *KubevirtClusterSpec) DeepCopy() *KubevirtClusterSpec {
	if in == nil {
		return nil
	}
	out := new(KubevirtClusterSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubevirtClusterStatus) DeepCopyInto(out *KubevirtClusterStatus) {
	*out = *in
	if in.FailureDomains != nil {
		in, out := &in.FailureDomains, &out.FailureDomains
		*out = make(v1beta1.FailureDomains, len(*in))
		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
		}
	}
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make(v1beta1.Conditions, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubevirtClusterStatus.
func (in *KubevirtClusterStatus) DeepCopy() *KubevirtClusterStatus {
	if in == nil {
		return nil
	}
	out := new(KubevirtClusterStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubevirtMachine) DeepCopyInto(out *KubevirtMachine) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubevirtMachine.
func (in *KubevirtMachine) DeepCopy() *KubevirtMachine {
	if in == nil {
		return nil
	}
	out := new(KubevirtMachine)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *KubevirtMachine) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubevirtMachineList) DeepCopyInto(out *KubevirtMachineList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]KubevirtMachine, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubevirtMachineList.
func (in *KubevirtMachineList) DeepCopy() *KubevirtMachineList {
	if in == nil {
		return nil
	}
	out := new(KubevirtMachineList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *KubevirtMachineList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubevirtMachineSpec) DeepCopyInto(out *KubevirtMachineSpec) {
	*out = *in
	in.VirtualMachineTemplate.DeepCopyInto(&out.VirtualMachineTemplate)
	if in.ProviderID != nil {
		in, out := &in.ProviderID, &out.ProviderID
		*out = new(string)
		**out = **in
	}
	if in.InfraClusterSecretRef != nil {
		in, out := &in.InfraClusterSecretRef, &out.InfraClusterSecretRef
		*out = new(v1.ObjectReference)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubevirtMachineSpec.
func (in *KubevirtMachineSpec) DeepCopy() *KubevirtMachineSpec {
	if in == nil {
		return nil
	}
	out := new(KubevirtMachineSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubevirtMachineStatus) DeepCopyInto(out *KubevirtMachineStatus) {
	*out = *in
	if in.Addresses != nil {
		in, out := &in.Addresses, &out.Addresses
		*out = make([]v1beta1.MachineAddress, len(*in))
		copy(*out, *in)
	}
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make(v1beta1.Conditions, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.FailureReason != nil {
		in, out := &in.FailureReason, &out.FailureReason
		*out = new(errors.MachineStatusError)
		**out = **in
	}
	if in.FailureMessage != nil {
		in, out := &in.FailureMessage, &out.FailureMessage
		*out = new(string)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubevirtMachineStatus.
func (in *KubevirtMachineStatus) DeepCopy() *KubevirtMachineStatus {
	if in == nil {
		return nil
	}
	out := new(KubevirtMachineStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubevirtMachineTemplate) DeepCopyInto(out *KubevirtMachineTemplate) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubevirtMachineTemplate.
func (in *KubevirtMachineTemplate) DeepCopy() *KubevirtMachineTemplate {
	if in == nil {
		return nil
	}
	out := new(KubevirtMachineTemplate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *KubevirtMachineTemplate) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubevirtMachineTemplateList) DeepCopyInto(out *KubevirtMachineTemplateList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]KubevirtMachineTemplate, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubevirtMachineTemplateList.
func (in *KubevirtMachineTemplateList) DeepCopy() *KubevirtMachineTemplateList {
	if in == nil {
		return nil
	}
	out := new(KubevirtMachineTemplateList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *KubevirtMachineTemplateList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubevirtMachineTemplateResource) DeepCopyInto(out *KubevirtMachineTemplateResource) {
	*out = *in
	in.Spec.DeepCopyInto(&out.Spec)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubevirtMachineTemplateResource.
func (in *KubevirtMachineTemplateResource) DeepCopy() *KubevirtMachineTemplateResource {
	if in == nil {
		return nil
	}
	out := new(KubevirtMachineTemplateResource)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubevirtMachineTemplateSpec) DeepCopyInto(out *KubevirtMachineTemplateSpec) {
	*out = *in
	in.Template.DeepCopyInto(&out.Template)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubevirtMachineTemplateSpec.
func (in *KubevirtMachineTemplateSpec) DeepCopy() *KubevirtMachineTemplateSpec {
	if in == nil {
		return nil
	}
	out := new(KubevirtMachineTemplateSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SSHKeys) DeepCopyInto(out *SSHKeys) {
	*out = *in
	if in.ConfigRef != nil {
		in, out := &in.ConfigRef, &out.ConfigRef
		*out = new(v1.ObjectReference)
		**out = **in
	}
	if in.DataSecretName != nil {
		in, out := &in.DataSecretName, &out.DataSecretName
		*out = new(string)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SSHKeys.
func (in *SSHKeys) DeepCopy() *SSHKeys {
	if in == nil {
		return nil
	}
	out := new(SSHKeys)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ServiceSpecTemplate) DeepCopyInto(out *ServiceSpecTemplate) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ServiceSpecTemplate.
func (in *ServiceSpecTemplate) DeepCopy() *ServiceSpecTemplate {
	if in == nil {
		return nil
	}
	out := new(ServiceSpecTemplate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VirtualMachineTemplateSpec) DeepCopyInto(out *VirtualMachineTemplateSpec) {
	*out = *in
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VirtualMachineTemplateSpec.
func (in *VirtualMachineTemplateSpec) DeepCopy() *VirtualMachineTemplateSpec {
	if in == nil {
		return nil
	}
	out := new(VirtualMachineTemplateSpec)
	in.DeepCopyInto(out)
	return out
}
