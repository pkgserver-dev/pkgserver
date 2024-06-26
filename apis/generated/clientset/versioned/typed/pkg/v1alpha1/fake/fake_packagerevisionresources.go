/*
Copyright 2024 Nokia.

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
// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	v1alpha1 "github.com/pkgserver-dev/pkgserver/apis/pkg/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakePackageRevisionResourceses implements PackageRevisionResourcesInterface
type FakePackageRevisionResourceses struct {
	Fake *FakePkgV1alpha1
	ns   string
}

var packagerevisionresourcesesResource = v1alpha1.SchemeGroupVersion.WithResource("packagerevisionresourceses")

var packagerevisionresourcesesKind = v1alpha1.SchemeGroupVersion.WithKind("PackageRevisionResources")

// Get takes name of the packageRevisionResources, and returns the corresponding packageRevisionResources object, and an error if there is any.
func (c *FakePackageRevisionResourceses) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.PackageRevisionResources, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(packagerevisionresourcesesResource, c.ns, name), &v1alpha1.PackageRevisionResources{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.PackageRevisionResources), err
}

// List takes label and field selectors, and returns the list of PackageRevisionResourceses that match those selectors.
func (c *FakePackageRevisionResourceses) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.PackageRevisionResourcesList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(packagerevisionresourcesesResource, packagerevisionresourcesesKind, c.ns, opts), &v1alpha1.PackageRevisionResourcesList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.PackageRevisionResourcesList{ListMeta: obj.(*v1alpha1.PackageRevisionResourcesList).ListMeta}
	for _, item := range obj.(*v1alpha1.PackageRevisionResourcesList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested packageRevisionResourceses.
func (c *FakePackageRevisionResourceses) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(packagerevisionresourcesesResource, c.ns, opts))

}

// Create takes the representation of a packageRevisionResources and creates it.  Returns the server's representation of the packageRevisionResources, and an error, if there is any.
func (c *FakePackageRevisionResourceses) Create(ctx context.Context, packageRevisionResources *v1alpha1.PackageRevisionResources, opts v1.CreateOptions) (result *v1alpha1.PackageRevisionResources, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(packagerevisionresourcesesResource, c.ns, packageRevisionResources), &v1alpha1.PackageRevisionResources{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.PackageRevisionResources), err
}

// Update takes the representation of a packageRevisionResources and updates it. Returns the server's representation of the packageRevisionResources, and an error, if there is any.
func (c *FakePackageRevisionResourceses) Update(ctx context.Context, packageRevisionResources *v1alpha1.PackageRevisionResources, opts v1.UpdateOptions) (result *v1alpha1.PackageRevisionResources, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(packagerevisionresourcesesResource, c.ns, packageRevisionResources), &v1alpha1.PackageRevisionResources{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.PackageRevisionResources), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakePackageRevisionResourceses) UpdateStatus(ctx context.Context, packageRevisionResources *v1alpha1.PackageRevisionResources, opts v1.UpdateOptions) (*v1alpha1.PackageRevisionResources, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(packagerevisionresourcesesResource, "status", c.ns, packageRevisionResources), &v1alpha1.PackageRevisionResources{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.PackageRevisionResources), err
}

// Delete takes name of the packageRevisionResources and deletes it. Returns an error if one occurs.
func (c *FakePackageRevisionResourceses) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(packagerevisionresourcesesResource, c.ns, name, opts), &v1alpha1.PackageRevisionResources{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakePackageRevisionResourceses) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(packagerevisionresourcesesResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.PackageRevisionResourcesList{})
	return err
}

// Patch applies the patch and returns the patched packageRevisionResources.
func (c *FakePackageRevisionResourceses) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.PackageRevisionResources, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(packagerevisionresourcesesResource, c.ns, name, pt, data, subresources...), &v1alpha1.PackageRevisionResources{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.PackageRevisionResources), err
}
