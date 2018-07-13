// Copyright © 2018 The Kubernetes Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1alpha1

import (
	"bytes"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"github.com/enxebre/cluster-api-provider-aws/awsproviderconfig"
)

// +k8s:deepcopy-gen=false
type AWSProviderConfigCodec struct {
	encoder runtime.Encoder
	decoder runtime.Decoder
}

const GroupName = "awsproviderconfig"

var SchemeGroupVersion = schema.GroupVersion{Group: GroupName, Version: "v1alpha1"}

var (
	SchemeBuilder      runtime.SchemeBuilder
	localSchemeBuilder = &SchemeBuilder
	AddToScheme        = localSchemeBuilder.AddToScheme
)

func init() {
	localSchemeBuilder.Register(addKnownTypes)
}

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&AWSMachineProviderConfig{},
	)
	scheme.AddKnownTypes(SchemeGroupVersion,
		&AWSClusterProviderConfig{},
	)
	scheme.AddKnownTypes(SchemeGroupVersion,
		&AWSMachineProviderStatus{},
	)
	scheme.AddKnownTypes(SchemeGroupVersion,
		&AWSClusterProviderStatus{},
	)
	return nil
}

func NewScheme() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	if err := AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := awsproviderconfig.AddToScheme(scheme); err != nil {
		return nil, err
	}
	return scheme, nil
}

func NewCodec() (*AWSProviderConfigCodec, error) {
	scheme, err := NewScheme()
	if err != nil {
		return nil, err
	}
	codecFactory := serializer.NewCodecFactory(scheme)
	encoder, err := newEncoder(&codecFactory)
	if err != nil {
		return nil, err
	}
	codec := AWSProviderConfigCodec{
		encoder: encoder,
		decoder: codecFactory.UniversalDecoder(SchemeGroupVersion),
	}
	return &codec, nil
}

func (codec *AWSProviderConfigCodec) DecodeFromProviderConfig(providerConfig clusterv1.ProviderConfig, out runtime.Object) error {
	if providerConfig.Value != nil {
		_, _, err := codec.decoder.Decode(providerConfig.Value.Raw, nil, out)
		if err != nil {
			return fmt.Errorf("decoding failure: %v", err)
		}
	}
	return nil
}

func (codec *AWSProviderConfigCodec) EncodeToProviderConfig(in runtime.Object) (*clusterv1.ProviderConfig, error) {
	var buf bytes.Buffer
	if err := codec.encoder.Encode(in, &buf); err != nil {
		return nil, fmt.Errorf("encoding failed: %v", err)
	}
	return &clusterv1.ProviderConfig{
		Value: &runtime.RawExtension{Raw: buf.Bytes()},
	}, nil
}

func (codec *AWSProviderConfigCodec) EncodeProviderStatus(in runtime.Object) (*runtime.RawExtension, error) {
	var buf bytes.Buffer
	if err := codec.encoder.Encode(in, &buf); err != nil {
		return nil, fmt.Errorf("encoding failed: %v", err)
	}

	return &runtime.RawExtension{Raw: buf.Bytes()}, nil
}

func (codec *AWSProviderConfigCodec) DecodeProviderStatus(providerStatus *runtime.RawExtension, out runtime.Object) error {
	if providerStatus != nil {
		_, _, err := codec.decoder.Decode(providerStatus.Raw, nil, out)
		if err != nil {
			return fmt.Errorf("decoding failure: %v", err)
		}
	}
	return nil
}

func newEncoder(codecFactory *serializer.CodecFactory) (runtime.Encoder, error) {
	serializerInfos := codecFactory.SupportedMediaTypes()
	if len(serializerInfos) == 0 {
		return nil, fmt.Errorf("unable to find any serlializers")
	}
	encoder := codecFactory.EncoderForVersion(serializerInfos[0].Serializer, SchemeGroupVersion)
	return encoder, nil
}






func (codec *AWSProviderConfigCodec) MachineProviderFromProviderConfig(providerConfig clusterv1.ProviderConfig) (*AWSMachineProviderConfig, error) {
	var config AWSMachineProviderConfig
	err := codec.DecodeFromProviderConfig(providerConfig, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func (codec *AWSProviderConfigCodec) ClusterProviderFromProviderConfig(providerConfig clusterv1.ProviderConfig) (*AWSClusterProviderConfig, error) {
	var config AWSClusterProviderConfig
	err := codec.DecodeFromProviderConfig(providerConfig, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func (codec *AWSProviderConfigCodec) ClusterProviderStatusFromProviderConfig(providerConfig clusterv1.ProviderConfig) (*AWSClusterProviderStatus, error) {
	var config AWSClusterProviderStatus
	err := codec.DecodeFromProviderConfig(providerConfig, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func (codec *AWSProviderConfigCodec) MachineProviderStatusFromProviderConfig(providerConfig clusterv1.ProviderConfig) (*AWSMachineProviderStatus, error) {
	var config AWSMachineProviderStatus
	err := codec.DecodeFromProviderConfig(providerConfig, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}


// AWSMachineProviderStatusFromClusterAPIMachine gets the cluster-operator MachineSetSpec from the
// specified cluster-api MachineSet.
func (codec *AWSProviderConfigCodec) AWSMachineProviderStatusFromClusterAPIMachine(m *clusterv1.Machine) (*AWSMachineProviderStatus, error) {
	if m.Status.ProviderStatus == nil {
		return &AWSMachineProviderStatus{}, nil
	}
	obj, gvk, err := codec.decoder.Decode([]byte(m.Status.ProviderStatus.Raw), nil, nil)
	if err != nil {
		return nil, err
	}
	status, ok := obj.(*AWSMachineProviderStatus)
	if !ok {
		return nil, fmt.Errorf("Unexpected object: %#v", gvk)
	}
	return status, nil
}

// ClusterAPIMachineProviderStatusFromAWSMachineProviderStatus gets the cluster-api ProviderConfig for a Machine template
// to store the cluster-operator MachineSetSpec.
func (codec *AWSProviderConfigCodec) ClusterAPIMachineProviderStatusFromAWSMachineProviderStatus(awsStatus *AWSMachineProviderStatus) (*runtime.RawExtension, error) {
	awsStatus.TypeMeta = metav1.TypeMeta{
		APIVersion: SchemeGroupVersion.String(),
		Kind:       "AWSMachineProviderStatus",
	}

	raw, err := codec.EncodeProviderStatus(awsStatus)
	if err != nil {
		return nil, err
	}
	return raw, nil
}