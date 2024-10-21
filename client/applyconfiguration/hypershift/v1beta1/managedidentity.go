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

package v1beta1

// ManagedIdentityApplyConfiguration represents an declarative configuration of the ManagedIdentity type for use
// with apply.
type ManagedIdentityApplyConfiguration struct {
	ClientID        *string `json:"clientID,omitempty"`
	CertificateName *string `json:"certificateName,omitempty"`
}

// ManagedIdentityApplyConfiguration constructs an declarative configuration of the ManagedIdentity type for use with
// apply.
func ManagedIdentity() *ManagedIdentityApplyConfiguration {
	return &ManagedIdentityApplyConfiguration{}
}

// WithClientID sets the ClientID field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ClientID field is set to the value of the last call.
func (b *ManagedIdentityApplyConfiguration) WithClientID(value string) *ManagedIdentityApplyConfiguration {
	b.ClientID = &value
	return b
}

// WithCertificateName sets the CertificateName field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the CertificateName field is set to the value of the last call.
func (b *ManagedIdentityApplyConfiguration) WithCertificateName(value string) *ManagedIdentityApplyConfiguration {
	b.CertificateName = &value
	return b
}
