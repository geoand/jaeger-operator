package controller

import (
	"context"
	"testing"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"

	"github.com/jaegertracing/jaeger-operator/pkg/apis/io/v1alpha1"
)

func TestNewControllerForAllInOneAsDefault(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestNewControllerForAllInOneAsDefault")

	ctrl := NewController(context.TODO(), jaeger)
	ds := ctrl.Create()
	assert.Len(t, ds, 6)
}

func TestNewControllerForAllInOneAsExplicitValue(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestNewControllerForAllInOneAsExplicitValue")
	jaeger.Spec.Strategy = "ALL-IN-ONE" // same as 'all-in-one'

	ctrl := NewController(context.TODO(), jaeger)
	ds := ctrl.Create()
	assert.Len(t, ds, 6)
}

func TestNewControllerForProduction(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestNewControllerForProduction")
	jaeger.Spec.Strategy = "production"

	ctrl := NewController(context.TODO(), jaeger)
	ds := ctrl.Create()
	assert.Len(t, ds, 6)
}

func TestUnknownStorage(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestNewControllerForProduction")
	jaeger.Spec.Storage.Type = "unknown"

	NewController(context.TODO(), jaeger)
	assert.Equal(t, "memory", jaeger.Spec.Storage.Type)
}

func TestElasticsearchAsStorageOptions(t *testing.T) {
	jaeger := v1alpha1.NewJaeger("TestElasticsearchAsStorageOptions")
	jaeger.Spec.Strategy = "production"
	jaeger.Spec.Storage.Type = "elasticsearch"
	jaeger.Spec.Storage.Options = v1alpha1.NewOptions(map[string]interface{}{
		"es.server-urls": "http://elasticsearch-example-es-cluster:9200",
	})

	ctrl := NewController(context.TODO(), jaeger)
	ds := ctrl.Create()
	deps := getDeployments(ds)
	assert.Len(t, deps, 2) // query and collector, for a production setup
	counter := 0
	for _, dep := range deps {
		for _, arg := range dep.Spec.Template.Spec.Containers[0].Args {
			if arg == "--es.server-urls=http://elasticsearch-example-es-cluster:9200" {
				counter++
			}
		}
	}

	assert.Equal(t, len(deps), counter)
}

func TestDefaultName(t *testing.T) {
	jaeger := &v1alpha1.Jaeger{}
	NewController(context.TODO(), jaeger)
	assert.NotEmpty(t, jaeger.ObjectMeta.Name)
}

func getDeployments(objs []sdk.Object) []*appsv1.Deployment {
	var deps []*appsv1.Deployment

	for _, obj := range objs {
		switch obj.(type) {
		case *appsv1.Deployment:
			deps = append(deps, obj.(*appsv1.Deployment))
		}
	}

	return deps
}

func assertHasAllObjects(t *testing.T, name string, objs []sdk.Object, deployments map[string]bool, services map[string]bool, ingresses map[string]bool) {
	for _, obj := range objs {
		switch typ := obj.(type) {
		case *appsv1.Deployment:
			deployments[obj.(*appsv1.Deployment).ObjectMeta.Name] = true
		case *v1.Service:
			services[obj.(*v1.Service).ObjectMeta.Name] = true
		case *v1beta1.Ingress:
			ingresses[obj.(*v1beta1.Ingress).ObjectMeta.Name] = true
		default:
			assert.Failf(t, "unknown type to be deployed", "%v", typ)
		}
	}

	for k, v := range deployments {
		assert.True(t, v, "Expected %s to have been returned from the list of deployments", k)
	}

	for k, v := range services {
		assert.True(t, v, "Expected %s to have been returned from the list of services", k)
	}

	for k, v := range ingresses {
		assert.True(t, v, "Expected %s to have been returned from the list of ingress rules", k)
	}
}
