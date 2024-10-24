/*


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
// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1alpha1

import (
	v1alpha1 "github.com/openshift/hypershift/api/hypershift/v1alpha1"
)

// DiagnosticsApplyConfiguration represents an declarative configuration of the Diagnostics type for use
// with apply.
type DiagnosticsApplyConfiguration struct {
	StorageAccountType *v1alpha1.AzureDiagnosticsStorageAccountType `json:"storageAccountType,omitempty"`
	UserManaged        *UserManagedDiagnosticsApplyConfiguration    `json:"userManaged,omitempty"`
}

// DiagnosticsApplyConfiguration constructs an declarative configuration of the Diagnostics type for use with
// apply.
func Diagnostics() *DiagnosticsApplyConfiguration {
	return &DiagnosticsApplyConfiguration{}
}

// WithStorageAccountType sets the StorageAccountType field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the StorageAccountType field is set to the value of the last call.
func (b *DiagnosticsApplyConfiguration) WithStorageAccountType(value v1alpha1.AzureDiagnosticsStorageAccountType) *DiagnosticsApplyConfiguration {
	b.StorageAccountType = &value
	return b
}

// WithUserManaged sets the UserManaged field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the UserManaged field is set to the value of the last call.
func (b *DiagnosticsApplyConfiguration) WithUserManaged(value *UserManagedDiagnosticsApplyConfiguration) *DiagnosticsApplyConfiguration {
	b.UserManaged = value
	return b
}
