package storagenode

import (
	"fmt"
	"reflect"
	"testing"

	corev1alpha1 "github.com/libopenstorage/operator/pkg/apis/core/v1alpha1"
	testutil "github.com/libopenstorage/operator/pkg/util/test"
	apiextensionsops "github.com/portworx/sched-ops/k8s/apiextensions"
	coreops "github.com/portworx/sched-ops/k8s/core"
	"github.com/stretchr/testify/require"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	fakeextclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakek8sclient "k8s.io/client-go/kubernetes/fake"
)

func TestRegisterCRD(t *testing.T) {
	fakeClient := fakek8sclient.NewSimpleClientset()
	fakeExtClient := fakeextclient.NewSimpleClientset()
	coreops.SetInstance(coreops.New(fakeClient))
	apiextensionsops.SetInstance(apiextensionsops.New(fakeExtClient))
	group := corev1alpha1.SchemeGroupVersion.Group
	storageNodeCRDName := corev1alpha1.StorageNodeResourcePlural + "." + group

	// When the CRDs are created, just updated their status so the validation
	// does not get stuck until timeout.

	go func() {
		err := testutil.ActivateCRDWhenCreated(fakeExtClient, storageNodeCRDName)
		require.NoError(t, err)
	}()

	controller := Controller{}

	// Should fail if the CRD specs are not found
	err := controller.RegisterCRD()
	require.Error(t, err)

	// Set the correct crd path
	crdBaseDir = func() string {
		return "../../../deploy/crds"
	}
	defer func() {
		crdBaseDir = getCRDBasePath
	}()

	err = controller.RegisterCRD()
	require.NoError(t, err)

	crds, err := fakeExtClient.ApiextensionsV1beta1().
		CustomResourceDefinitions().
		List(metav1.ListOptions{})
	require.NoError(t, err)
	require.Len(t, crds.Items, 1)

	subresource := &apiextensionsv1beta1.CustomResourceSubresources{
		Status: &apiextensionsv1beta1.CustomResourceSubresourceStatus{},
	}
	crd, err := fakeExtClient.ApiextensionsV1beta1().
		CustomResourceDefinitions().
		Get(storageNodeCRDName, metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, storageNodeCRDName, crd.Name)
	require.Equal(t, corev1alpha1.SchemeGroupVersion.Group, crd.Spec.Group)
	require.Len(t, crd.Spec.Versions, 1)
	require.Equal(t, corev1alpha1.SchemeGroupVersion.Version, crd.Spec.Versions[0].Name)
	require.True(t, crd.Spec.Versions[0].Served)
	require.True(t, crd.Spec.Versions[0].Storage)
	require.Equal(t, apiextensionsv1beta1.NamespaceScoped, crd.Spec.Scope)
	require.Equal(t, corev1alpha1.StorageNodeResourceName, crd.Spec.Names.Singular)
	require.Equal(t, corev1alpha1.StorageNodeResourcePlural, crd.Spec.Names.Plural)
	require.Equal(t, reflect.TypeOf(corev1alpha1.StorageNode{}).Name(), crd.Spec.Names.Kind)
	require.Equal(t, reflect.TypeOf(corev1alpha1.StorageNodeList{}).Name(), crd.Spec.Names.ListKind)
	require.Equal(t, []string{corev1alpha1.StorageNodeShortName}, crd.Spec.Names.ShortNames)
	require.Equal(t, subresource, crd.Spec.Subresources)
	require.NotEmpty(t, crd.Spec.Validation.OpenAPIV3Schema.Properties)

	// If CRDs are already present, then should not fail
	err = controller.RegisterCRD()
	require.NoError(t, err)

	crds, err = fakeExtClient.ApiextensionsV1beta1().
		CustomResourceDefinitions().
		List(metav1.ListOptions{})
	require.NoError(t, err)
	require.Len(t, crds.Items, 1)
	require.Equal(t, storageNodeCRDName, crds.Items[0].Name)
}

func TestRegisterCRDShouldRemoveNodeStatusCRD(t *testing.T) {
	nodeStatusCRDName := fmt.Sprintf("%s.%s",
		storageNodeStatusPlural,
		corev1alpha1.SchemeGroupVersion.Group,
	)
	nodeStatusCRD := &apiextensionsv1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: nodeStatusCRDName,
		},
	}
	fakeClient := fakek8sclient.NewSimpleClientset()
	fakeExtClient := fakeextclient.NewSimpleClientset(nodeStatusCRD)
	coreops.SetInstance(coreops.New(fakeClient))
	apiextensionsops.SetInstance(apiextensionsops.New(fakeExtClient))
	crdBaseDir = func() string {
		return "../../../deploy/crds"
	}
	defer func() {
		crdBaseDir = getCRDBasePath
	}()

	group := corev1alpha1.SchemeGroupVersion.Group
	storageClusterCRDName := corev1alpha1.StorageClusterResourcePlural + "." + group
	storageNodeCRDName := corev1alpha1.StorageNodeResourcePlural + "." + group

	// When the CRDs are created, just updated their status so the validation
	// does not get stuck until timeout.
	go func() {
		err := testutil.ActivateCRDWhenCreated(fakeExtClient, storageClusterCRDName)
		require.NoError(t, err)
	}()
	go func() {
		err := testutil.ActivateCRDWhenCreated(fakeExtClient, storageNodeCRDName)
		require.NoError(t, err)
	}()

	controller := Controller{}

	err := controller.RegisterCRD()
	require.NoError(t, err)

	crds, err := fakeExtClient.ApiextensionsV1beta1().
		CustomResourceDefinitions().
		List(metav1.ListOptions{})
	require.NoError(t, err)
	require.Len(t, crds.Items, 1)
	for _, crd := range crds.Items {
		require.NotEqual(t, nodeStatusCRDName, crd.Name)
	}
}
