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

// AzureKMSSpecApplyConfiguration represents an declarative configuration of the AzureKMSSpec type for use
// with apply.
type AzureKMSSpecApplyConfiguration struct {
	ActiveKey *AzureKMSKeyApplyConfiguration     `json:"activeKey,omitempty"`
	BackupKey *AzureKMSKeyApplyConfiguration     `json:"backupKey,omitempty"`
	KMS       *ManagedIdentityApplyConfiguration `json:"kms,omitempty"`
}

// AzureKMSSpecApplyConfiguration constructs an declarative configuration of the AzureKMSSpec type for use with
// apply.
func AzureKMSSpec() *AzureKMSSpecApplyConfiguration {
	return &AzureKMSSpecApplyConfiguration{}
}

// WithActiveKey sets the ActiveKey field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ActiveKey field is set to the value of the last call.
func (b *AzureKMSSpecApplyConfiguration) WithActiveKey(value *AzureKMSKeyApplyConfiguration) *AzureKMSSpecApplyConfiguration {
	b.ActiveKey = value
	return b
}

// WithBackupKey sets the BackupKey field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the BackupKey field is set to the value of the last call.
func (b *AzureKMSSpecApplyConfiguration) WithBackupKey(value *AzureKMSKeyApplyConfiguration) *AzureKMSSpecApplyConfiguration {
	b.BackupKey = value
	return b
}

// WithKMS sets the KMS field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the KMS field is set to the value of the last call.
func (b *AzureKMSSpecApplyConfiguration) WithKMS(value *ManagedIdentityApplyConfiguration) *AzureKMSSpecApplyConfiguration {
	b.KMS = value
	return b
}
