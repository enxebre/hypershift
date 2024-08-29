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

// AWSNodePoolPlatformApplyConfiguration represents an declarative configuration of the AWSNodePoolPlatform type for use
// with apply.
type AWSNodePoolPlatformApplyConfiguration struct {
	InstanceType    *string                                  `json:"instanceType,omitempty"`
	InstanceProfile *string                                  `json:"instanceProfile,omitempty"`
	Subnet          *AWSResourceReferenceApplyConfiguration  `json:"subnet,omitempty"`
	AMI             *string                                  `json:"ami,omitempty"`
	SecurityGroups  []AWSResourceReferenceApplyConfiguration `json:"securityGroups,omitempty"`
	RootVolume      *VolumeApplyConfiguration                `json:"rootVolume,omitempty"`
	ResourceTags    []AWSResourceTagApplyConfiguration       `json:"resourceTags,omitempty"`
	Tenancy         *string                                  `json:"tenancy,omitempty"`
}

// AWSNodePoolPlatformApplyConfiguration constructs an declarative configuration of the AWSNodePoolPlatform type for use with
// apply.
func AWSNodePoolPlatform() *AWSNodePoolPlatformApplyConfiguration {
	return &AWSNodePoolPlatformApplyConfiguration{}
}

// WithInstanceType sets the InstanceType field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the InstanceType field is set to the value of the last call.
func (b *AWSNodePoolPlatformApplyConfiguration) WithInstanceType(value string) *AWSNodePoolPlatformApplyConfiguration {
	b.InstanceType = &value
	return b
}

// WithInstanceProfile sets the InstanceProfile field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the InstanceProfile field is set to the value of the last call.
func (b *AWSNodePoolPlatformApplyConfiguration) WithInstanceProfile(value string) *AWSNodePoolPlatformApplyConfiguration {
	b.InstanceProfile = &value
	return b
}

// WithSubnet sets the Subnet field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Subnet field is set to the value of the last call.
func (b *AWSNodePoolPlatformApplyConfiguration) WithSubnet(value *AWSResourceReferenceApplyConfiguration) *AWSNodePoolPlatformApplyConfiguration {
	b.Subnet = value
	return b
}

// WithAMI sets the AMI field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the AMI field is set to the value of the last call.
func (b *AWSNodePoolPlatformApplyConfiguration) WithAMI(value string) *AWSNodePoolPlatformApplyConfiguration {
	b.AMI = &value
	return b
}

// WithSecurityGroups adds the given value to the SecurityGroups field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the SecurityGroups field.
func (b *AWSNodePoolPlatformApplyConfiguration) WithSecurityGroups(values ...*AWSResourceReferenceApplyConfiguration) *AWSNodePoolPlatformApplyConfiguration {
	for i := range values {
		if values[i] == nil {
			panic("nil value passed to WithSecurityGroups")
		}
		b.SecurityGroups = append(b.SecurityGroups, *values[i])
	}
	return b
}

// WithRootVolume sets the RootVolume field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the RootVolume field is set to the value of the last call.
func (b *AWSNodePoolPlatformApplyConfiguration) WithRootVolume(value *VolumeApplyConfiguration) *AWSNodePoolPlatformApplyConfiguration {
	b.RootVolume = value
	return b
}

// WithResourceTags adds the given value to the ResourceTags field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the ResourceTags field.
func (b *AWSNodePoolPlatformApplyConfiguration) WithResourceTags(values ...*AWSResourceTagApplyConfiguration) *AWSNodePoolPlatformApplyConfiguration {
	for i := range values {
		if values[i] == nil {
			panic("nil value passed to WithResourceTags")
		}
		b.ResourceTags = append(b.ResourceTags, *values[i])
	}
	return b
}

// WithTenancy sets the Tenancy field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Tenancy field is set to the value of the last call.
func (b *AWSNodePoolPlatformApplyConfiguration) WithTenancy(value string) *AWSNodePoolPlatformApplyConfiguration {
	b.Tenancy = &value
	return b
}
