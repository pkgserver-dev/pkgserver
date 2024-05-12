package client

import (
	configv1alpha1 "github.com/pkgserver-dev/pkgserver/apis/config/v1alpha1"
	pkgv1alpha1 "github.com/pkgserver-dev/pkgserver/apis/pkg/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Client interface {
	client.Client
}

func CreateClient(config *rest.Config) (Client, error) {
	scheme, err := createScheme()
	if err != nil {
		return nil, err
	}

	c, err := client.New(config, client.Options{
		Scheme: scheme,
		Mapper: createRESTMapper(),
	})
	if err != nil {
		return nil, err
	}

	return c, nil
}

func CreateClientWithFlags(flags *genericclioptions.ConfigFlags) (client.Client, error) {
	config, err := flags.ToRESTConfig()
	if err != nil {
		return nil, err
	}

	return CreateClient(config)
}

func CreateRESTClient(flags *genericclioptions.ConfigFlags) (rest.Interface, error) {
	config, err := flags.ToRESTConfig()
	if err != nil {
		return nil, err
	}

	scheme, err := createScheme()
	if err != nil {
		return nil, err
	}

	codecs := serializer.NewCodecFactory(scheme)

	gv := pkgv1alpha1.SchemeGroupVersion
	config.GroupVersion = &gv
	config.APIPath = "/apis"
	config.NegotiatedSerializer = codecs.WithoutConversion()

	return rest.RESTClientFor(config)
}

func createScheme() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()

	for _, api := range (runtime.SchemeBuilder{
		pkgv1alpha1.AddToScheme,
		configv1alpha1.AddToScheme,
	}) {
		if err := api(scheme); err != nil {
			return nil, err
		}
	}
	return scheme, nil
}

func createRESTMapper() meta.RESTMapper {
	rm := meta.NewDefaultRESTMapper([]schema.GroupVersion{
		configv1alpha1.SchemeGroupVersion,
		pkgv1alpha1.SchemeGroupVersion,
	})

	for _, r := range []struct {
		kind             schema.GroupVersionKind
		plural, singular string
	}{
		//{kind: configv1alpha1.SchemeGroupVersion.WithKind("Repository"), plural: "repositories", singular: "repository"},
		{kind: pkgv1alpha1.SchemeGroupVersion.WithKind("PackageRevision"), plural: "packagerevisions", singular: "packagerevision"},
		{kind: pkgv1alpha1.SchemeGroupVersion.WithKind("PackageRevisionResources"), plural: "packagerevisionresources", singular: "packagerevisionresources"},
		{kind: corev1.SchemeGroupVersion.WithKind("Secret"), plural: "secrets", singular: "secret"},
		{kind: metav1.SchemeGroupVersion.WithKind("Table"), plural: "tables", singular: "table"},
	} {
		rm.AddSpecific(
			r.kind,
			r.kind.GroupVersion().WithResource(r.plural),
			r.kind.GroupVersion().WithResource(r.singular),
			meta.RESTScopeNamespace,
		)
	}

	return rm
}
