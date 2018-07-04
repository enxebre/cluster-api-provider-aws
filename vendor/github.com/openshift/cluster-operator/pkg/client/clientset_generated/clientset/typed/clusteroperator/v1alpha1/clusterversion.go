/*
Copyright 2018 The Kubernetes Authors.

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

package v1alpha1

import (
	v1alpha1 "github.com/openshift/cluster-operator/pkg/apis/clusteroperator/v1alpha1"
	scheme "github.com/openshift/cluster-operator/pkg/client/clientset_generated/clientset/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// ClusterVersionsGetter has a method to return a ClusterVersionInterface.
// A group's client should implement this interface.
type ClusterVersionsGetter interface {
	ClusterVersions(namespace string) ClusterVersionInterface
}

// ClusterVersionInterface has methods to work with ClusterVersion resources.
type ClusterVersionInterface interface {
	Create(*v1alpha1.ClusterVersion) (*v1alpha1.ClusterVersion, error)
	Update(*v1alpha1.ClusterVersion) (*v1alpha1.ClusterVersion, error)
	UpdateStatus(*v1alpha1.ClusterVersion) (*v1alpha1.ClusterVersion, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha1.ClusterVersion, error)
	List(opts v1.ListOptions) (*v1alpha1.ClusterVersionList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.ClusterVersion, err error)
	ClusterVersionExpansion
}

// clusterVersions implements ClusterVersionInterface
type clusterVersions struct {
	client rest.Interface
	ns     string
}

// newClusterVersions returns a ClusterVersions
func newClusterVersions(c *ClusteroperatorV1alpha1Client, namespace string) *clusterVersions {
	return &clusterVersions{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the clusterVersion, and returns the corresponding clusterVersion object, and an error if there is any.
func (c *clusterVersions) Get(name string, options v1.GetOptions) (result *v1alpha1.ClusterVersion, err error) {
	result = &v1alpha1.ClusterVersion{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("clusterversions").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of ClusterVersions that match those selectors.
func (c *clusterVersions) List(opts v1.ListOptions) (result *v1alpha1.ClusterVersionList, err error) {
	result = &v1alpha1.ClusterVersionList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("clusterversions").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested clusterVersions.
func (c *clusterVersions) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("clusterversions").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a clusterVersion and creates it.  Returns the server's representation of the clusterVersion, and an error, if there is any.
func (c *clusterVersions) Create(clusterVersion *v1alpha1.ClusterVersion) (result *v1alpha1.ClusterVersion, err error) {
	result = &v1alpha1.ClusterVersion{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("clusterversions").
		Body(clusterVersion).
		Do().
		Into(result)
	return
}

// Update takes the representation of a clusterVersion and updates it. Returns the server's representation of the clusterVersion, and an error, if there is any.
func (c *clusterVersions) Update(clusterVersion *v1alpha1.ClusterVersion) (result *v1alpha1.ClusterVersion, err error) {
	result = &v1alpha1.ClusterVersion{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("clusterversions").
		Name(clusterVersion.Name).
		Body(clusterVersion).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *clusterVersions) UpdateStatus(clusterVersion *v1alpha1.ClusterVersion) (result *v1alpha1.ClusterVersion, err error) {
	result = &v1alpha1.ClusterVersion{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("clusterversions").
		Name(clusterVersion.Name).
		SubResource("status").
		Body(clusterVersion).
		Do().
		Into(result)
	return
}

// Delete takes name of the clusterVersion and deletes it. Returns an error if one occurs.
func (c *clusterVersions) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("clusterversions").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *clusterVersions) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("clusterversions").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched clusterVersion.
func (c *clusterVersions) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.ClusterVersion, err error) {
	result = &v1alpha1.ClusterVersion{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("clusterversions").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
