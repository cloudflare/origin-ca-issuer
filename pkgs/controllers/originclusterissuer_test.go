package controllers

import (
	"context"
	"testing"
	"time"

	"github.com/cloudflare/origin-ca-issuer/internal/cfapi"
	v1 "github.com/cloudflare/origin-ca-issuer/pkgs/apis/v1"
	"github.com/cloudflare/origin-ca-issuer/pkgs/provisioners"
	"github.com/google/go-cmp/cmp"
	cmapi "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	fakeClock "k8s.io/utils/clock/testing"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestOriginClusterIssuerReconcile(t *testing.T) {
	if err := cmapi.AddToScheme(scheme.Scheme); err != nil {
		t.Fatal(err)
	}

	if err := v1.Install(scheme.Scheme); err != nil {
		t.Fatal(err)
	}

	clock := fakeClock.NewFakeClock(time.Now().Truncate(time.Second))
	now := metav1.NewTime(clock.Now())

	tests := []struct {
		name          string
		objects       []runtime.Object
		expected      v1.OriginIssuerStatus
		error         string
		namespaceName types.NamespacedName
	}{
		{
			name: "working with secrets",
			objects: []runtime.Object{
				&v1.OriginClusterIssuer{
					ObjectMeta: metav1.ObjectMeta{
						Name: "foo",
					},
					Spec: v1.OriginIssuerSpec{
						RequestType: v1.RequestTypeOriginRSA,
						Auth: v1.OriginIssuerAuthentication{
							ServiceKeyRef: v1.SecretKeySelector{
								Name: "issuer-service-key",
								Key:  "key",
							},
						},
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "issuer-service-key",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"key": []byte("djEuMC0weDAwQkFCMTBD"),
					},
				},
			},
			expected: v1.OriginIssuerStatus{
				Conditions: []v1.OriginIssuerCondition{
					{
						Type:               v1.ConditionReady,
						Status:             v1.ConditionTrue,
						LastTransitionTime: &now,
						Reason:             "Verified",
						Message:            "OriginClusterIssuer verified and ready to sign certificates",
					},
				},
			},
			namespaceName: types.NamespacedName{
				Name: "foo",
			},
		},
		{
			name: "missing secret",
			objects: []runtime.Object{
				&v1.OriginClusterIssuer{
					ObjectMeta: metav1.ObjectMeta{
						Name: "foo",
					},
					Spec: v1.OriginIssuerSpec{
						RequestType: v1.RequestTypeOriginRSA,
						Auth: v1.OriginIssuerAuthentication{
							ServiceKeyRef: v1.SecretKeySelector{
								Name: "issuer-service-key",
								Key:  "key",
							},
						},
					},
				},
			},
			expected: v1.OriginIssuerStatus{
				Conditions: []v1.OriginIssuerCondition{
					{
						Type:               v1.ConditionReady,
						Status:             v1.ConditionFalse,
						LastTransitionTime: &now,
						Reason:             "NotFound",
						Message:            `Failed to retrieve auth secret: secrets "issuer-service-key" not found`,
					},
				},
			},
			error: `secrets "issuer-service-key" not found`,
			namespaceName: types.NamespacedName{
				Name: "foo",
			},
		},
		{
			name: "secret missing key",
			objects: []runtime.Object{
				&v1.OriginClusterIssuer{
					ObjectMeta: metav1.ObjectMeta{
						Name: "foo",
					},
					Spec: v1.OriginIssuerSpec{
						RequestType: v1.RequestTypeOriginRSA,
						Auth: v1.OriginIssuerAuthentication{
							ServiceKeyRef: v1.SecretKeySelector{
								Name: "issuer-service-key",
								Key:  "key",
							},
						},
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "issuer-service-key",
						Namespace: "default",
					},
					Data: map[string][]byte{},
				},
			},
			expected: v1.OriginIssuerStatus{
				Conditions: []v1.OriginIssuerCondition{
					{
						Type:               v1.ConditionReady,
						Status:             v1.ConditionFalse,
						LastTransitionTime: &now,
						Reason:             "NotFound",
						Message:            `Failed to retrieve auth secret: secret issuer-service-key does not contain key "key"`,
					},
				},
			},
			error: `secret issuer-service-key does not contain key "key"`,
			namespaceName: types.NamespacedName{
				Name: "foo",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			client := fake.NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithRuntimeObjects(tt.objects...).
				Build()

			collection := provisioners.CollectionWith(nil)

			controller := &OriginClusterIssuerController{
				Client: client,
				Factory: cfapi.FactoryFunc(func(serviceKey []byte) (cfapi.Interface, error) {
					return nil, nil
				}),
				Clock:       clock,
				Log:         logf.Log,
				Collection:  collection,
				CRNamespace: "default",
			}

			_, err := controller.Reconcile(context.Background(), reconcile.Request{
				NamespacedName: tt.namespaceName,
			})

			if err != nil {
				if diff := cmp.Diff(err.Error(), tt.error); diff != "" {
					t.Fatalf("diff: (-wanted +got)\n%s", diff)
				}
			}

			got := &v1.OriginClusterIssuer{}
			if err := client.Get(context.TODO(), tt.namespaceName, got); err != nil {
				t.Fatalf("expected to retrieve issuer from client: %s", err)
			}
			if diff := cmp.Diff(got.Status, tt.expected); diff != "" {
				t.Fatalf("diff: (-want +got)\n%s", diff)
			}

			if tt.error == "" {
				if _, ok := controller.Collection.Load(tt.namespaceName); !ok {
					t.Fatal("was unable to find provisioner")
				}
			}
		})
	}
}
