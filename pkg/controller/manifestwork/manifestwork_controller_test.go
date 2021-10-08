// Copyright (c) Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package manifestwork

import (
	"context"
	"testing"

	"github.com/open-cluster-management/managedcluster-import-controller/pkg/helpers"
	testinghelpers "github.com/open-cluster-management/managedcluster-import-controller/pkg/helpers/testing"
	operatorfake "open-cluster-management.io/api/client/operator/clientset/versioned/fake"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	workv1 "open-cluster-management.io/api/work/v1"

	"github.com/openshift/library-go/pkg/operator/events/eventstesting"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	testscheme = scheme.Scheme
	now        = v1.Now()
)

func init() {
	testscheme.AddKnownTypes(clusterv1.SchemeGroupVersion, &clusterv1.ManagedCluster{})
	testscheme.AddKnownTypes(workv1.SchemeGroupVersion, &workv1.ManifestWork{})
	testscheme.AddKnownTypes(workv1.SchemeGroupVersion, &workv1.ManifestWorkList{})
}

func TestReconcile(t *testing.T) {
	cases := []struct {
		name         string
		startObjs    []client.Object
		request      reconcile.Request
		validateFunc func(t *testing.T, runtimeClient client.Client)
	}{
		{
			name:      "no managed clusters",
			startObjs: []client.Object{},
			request: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: "test",
				},
			},
			validateFunc: func(t *testing.T, runtimeClient client.Client) {
				// do nothing
			},
		},
		{
			name: "no manifest works",
			startObjs: []client.Object{
				&clusterv1.ManagedCluster{
					ObjectMeta: v1.ObjectMeta{
						Name:       "test",
						Finalizers: []string{manifestWorkFinalizer},
					},
				},
			},
			request: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: "test",
				},
			},
			validateFunc: func(t *testing.T, runtimeClient client.Client) {
				managedCluster := &clusterv1.ManagedCluster{}
				if err := runtimeClient.Get(context.TODO(), types.NamespacedName{Name: "test"}, managedCluster); err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if len(managedCluster.Finalizers) != 0 {
					t.Errorf("expected no finalizers, but failed")
				}
			},
		},
		{
			name: "manifest works are created",
			startObjs: []client.Object{
				&clusterv1.ManagedCluster{
					ObjectMeta: v1.ObjectMeta{
						Name: "test",
					},
				},
				&workv1.ManifestWork{
					ObjectMeta: v1.ObjectMeta{
						Name:      "test",
						Namespace: "test",
					},
				},
			},
			request: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: "test",
				},
			},
			validateFunc: func(t *testing.T, runtimeClient client.Client) {
				managedCluster := &clusterv1.ManagedCluster{}
				if err := runtimeClient.Get(context.TODO(), types.NamespacedName{Name: "test"}, managedCluster); err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if len(managedCluster.Finalizers) != 1 {
					t.Errorf("expected one finalizer, but failed")
				}
			},
		},
		{
			name: "managed clusters is deleting",
			startObjs: []client.Object{
				&clusterv1.ManagedCluster{
					ObjectMeta: v1.ObjectMeta{
						Name:              "test",
						Finalizers:        []string{manifestWorkFinalizer},
						DeletionTimestamp: &now,
					},
				},
				&workv1.ManifestWork{
					ObjectMeta: v1.ObjectMeta{
						Name:      "test1",
						Namespace: "test",
					},
				},
				&workv1.ManifestWork{
					ObjectMeta: v1.ObjectMeta{
						Name:      "test-klusterlet-crds",
						Namespace: "test",
					},
				},
				&workv1.ManifestWork{
					ObjectMeta: v1.ObjectMeta{
						Name:      "test-klusterlet",
						Namespace: "test",
					},
					Status: workv1.ManifestWorkStatus{
						Conditions: []v1.Condition{
							{
								Type:   "Applied",
								Status: v1.ConditionTrue,
							},
						},
					},
				},
			},
			request: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: "test",
				},
			},
			validateFunc: func(t *testing.T, runtimeClient client.Client) {
				manifestWorks := &workv1.ManifestWorkList{}
				if err := runtimeClient.List(context.TODO(), manifestWorks, &client.ListOptions{Namespace: "test"}); err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if len(manifestWorks.Items) != 1 {
					t.Errorf("expected one work, but failed %d", len(manifestWorks.Items))
				}
			},
		},
		{
			name: "only have crd works",
			startObjs: []client.Object{
				&clusterv1.ManagedCluster{
					ObjectMeta: v1.ObjectMeta{
						Name:       "test",
						Finalizers: []string{manifestWorkFinalizer},
						Labels: map[string]string{
							"local-cluster": "true",
						},
						DeletionTimestamp: &now,
					},
				},
				&workv1.ManifestWork{
					ObjectMeta: v1.ObjectMeta{
						Name:      "test1",
						Namespace: "test",
					},
				},
				&workv1.ManifestWork{
					ObjectMeta: v1.ObjectMeta{
						Name:      "test-klusterlet-crds",
						Namespace: "test",
					},
				},
			},
			request: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: "test",
				},
			},
			validateFunc: func(t *testing.T, runtimeClient client.Client) {
				manifestWorks := &workv1.ManifestWorkList{}
				if err := runtimeClient.List(context.TODO(), manifestWorks, &client.ListOptions{Namespace: "test"}); err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if len(manifestWorks.Items) != 0 {
					t.Errorf("expected one work, but failed %d", len(manifestWorks.Items))
				}
			},
		},
		{
			name: "managed clusters is deleting - only has klusterlet",
			startObjs: []client.Object{
				&clusterv1.ManagedCluster{
					ObjectMeta: v1.ObjectMeta{
						Name:              "test",
						Finalizers:        []string{manifestWorkFinalizer},
						DeletionTimestamp: &now,
					},
				},
				&workv1.ManifestWork{
					ObjectMeta: v1.ObjectMeta{
						Name:      "test-klusterlet",
						Namespace: "test",
					},
				},
			},
			request: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: "test",
				},
			},
			validateFunc: func(t *testing.T, runtimeClient client.Client) {
				manifestWorks := &workv1.ManifestWorkList{}
				if err := runtimeClient.List(context.TODO(), manifestWorks, &client.ListOptions{Namespace: "test"}); err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if len(manifestWorks.Items) != 1 {
					t.Errorf("expected one work, but failed %d", len(manifestWorks.Items))
				}
			},
		},
		{
			name: "managed clusters is deleting and managed clusters is offline",
			startObjs: []client.Object{
				&clusterv1.ManagedCluster{
					ObjectMeta: v1.ObjectMeta{
						Name:              "test",
						Finalizers:        []string{manifestWorkFinalizer},
						DeletionTimestamp: &now,
					},
					Status: clusterv1.ManagedClusterStatus{
						Conditions: []v1.Condition{
							{
								Type:   clusterv1.ManagedClusterConditionAvailable,
								Status: v1.ConditionFalse,
							},
						},
					},
				},
				&workv1.ManifestWork{
					ObjectMeta: v1.ObjectMeta{
						Name:       "test",
						Namespace:  "test",
						Finalizers: []string{"test"},
					},
				},
				&workv1.ManifestWork{
					ObjectMeta: v1.ObjectMeta{
						Name:       "test-crds",
						Namespace:  "test",
						Finalizers: []string{"test"},
					},
				},
				&workv1.ManifestWork{
					ObjectMeta: v1.ObjectMeta{
						Name:       "test-klusterlet",
						Namespace:  "test",
						Finalizers: []string{"test"},
					},
				},
			},
			request: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: "test",
				},
			},
			validateFunc: func(t *testing.T, runtimeClient client.Client) {
				manifestWorks := &workv1.ManifestWorkList{}
				if err := runtimeClient.List(context.TODO(), manifestWorks, &client.ListOptions{Namespace: "test"}); err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if len(manifestWorks.Items) != 0 {
					t.Errorf("expected no works, but failed")
				}
			},
		},
		{
			name: "apply klusterlet manifest works",
			startObjs: []client.Object{
				&clusterv1.ManagedCluster{
					ObjectMeta: v1.ObjectMeta{
						Name:       "test",
						Finalizers: []string{manifestWorkFinalizer},
					},
					Status: clusterv1.ManagedClusterStatus{
						Conditions: []v1.Condition{
							{
								Type:   clusterv1.ManagedClusterConditionJoined,
								Status: v1.ConditionTrue,
							},
						},
						Version: clusterv1.ManagedClusterVersion{Kubernetes: "v1.18.0"},
					},
				},
				testinghelpers.GetImportSecret("test"),
			},
			request: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: "test",
				},
			},
			validateFunc: func(t *testing.T, runtimeClient client.Client) {},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := &ReconcileManifestWork{
				clientHolder: &helpers.ClientHolder{
					RuntimeClient:  fake.NewClientBuilder().WithScheme(testscheme).WithObjects(c.startObjs...).Build(),
					OperatorClient: operatorfake.NewSimpleClientset(),
				},
				scheme:   testscheme,
				recorder: eventstesting.NewTestingEventRecorder(t),
			}

			_, err := r.Reconcile(context.TODO(), c.request)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			c.validateFunc(t, r.clientHolder.RuntimeClient)
		})
	}
}