package controllers

import (
	"context"
	"crypto/x509"
	"testing"
	"time"

	cmutil "github.com/cert-manager/cert-manager/pkg/api/util"
	cmapi "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	cmgen "github.com/cert-manager/cert-manager/test/unit/gen"
	"github.com/cloudflare/origin-ca-issuer/internal/cfapi"
	fakeapi "github.com/cloudflare/origin-ca-issuer/internal/cfapi/testing"
	v1 "github.com/cloudflare/origin-ca-issuer/pkgs/apis/v1"
	"github.com/cloudflare/origin-ca-issuer/pkgs/provisioners"
	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	fakeClock "k8s.io/utils/clock/testing"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestCertificateRequestReconcile(t *testing.T) {
	if err := cmapi.AddToScheme(scheme.Scheme); err != nil {
		t.Fatal(err)
	}

	if err := v1.AddToScheme(scheme.Scheme); err != nil {
		t.Fatal(err)
	}

	clock := fakeClock.NewFakeClock(time.Now().Truncate(time.Second))
	now := metav1.NewTime(clock.Now())

	cmutil.Clock = clock

	tests := []struct {
		name          string
		objects       []runtime.Object
		collection    *provisioners.Collection
		expected      cmapi.CertificateRequestStatus
		error         string
		namespaceName types.NamespacedName
	}{
		{
			name: "working",
			objects: []runtime.Object{
				cmgen.CertificateRequest("foobar",
					cmgen.SetCertificateRequestNamespace("default"),
					cmgen.SetCertificateRequestDuration(&metav1.Duration{Duration: 7 * 24 * time.Hour}),
					cmgen.SetCertificateRequestCSR((func() []byte {
						csr, _, err := cmgen.CSR(x509.ECDSA)
						if err != nil {
							t.Fatalf("creating CSR: %s", err)
						}

						return csr
					})()),
					cmgen.SetCertificateRequestIssuer(cmmeta.ObjectReference{
						Name:  "foobar",
						Kind:  "OriginIssuer",
						Group: "cert-manager.k8s.cloudflare.com",
					}),
				),
				&v1.OriginIssuer{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foobar",
						Namespace: "default",
					},
					Spec: v1.OriginIssuerSpec{
						Auth: v1.OriginIssuerAuthentication{
							ServiceKeyRef: v1.SecretKeySelector{
								Name: "service-key-issuer",
								Key:  "key",
							},
						},
					},
					Status: v1.OriginIssuerStatus{
						Conditions: []v1.OriginIssuerCondition{
							{
								Type:   v1.ConditionReady,
								Status: v1.ConditionTrue,
							},
						},
					},
				},
			},
			collection: provisioners.CollectionWith([]provisioners.CollectionItem{
				{
					NamespacedName: types.NamespacedName{
						Name:      "foobar",
						Namespace: "default",
					},
					Provisioner: (func() *provisioners.Provisioner {
						c := &fakeapi.FakeClient{
							Response: &cfapi.SignResponse{
								Id:          "1",
								Certificate: "bogus",
								Hostnames:   []string{"example.com"},
								Expiration:  time.Time{},
								Type:        "colemak",
								Validity:    0,
								CSR:         "foobar",
							},
						}
						p, err := provisioners.New(c, v1.RequestTypeOriginRSA, logf.Log)
						if err != nil {
							t.Fatalf("error creating provisioner: %s", err)
						}

						return p
					}()),
				},
			}),
			expected: cmapi.CertificateRequestStatus{
				Conditions: []cmapi.CertificateRequestCondition{
					{
						Type:               cmapi.CertificateRequestConditionReady,
						Status:             cmmeta.ConditionTrue,
						LastTransitionTime: &now,
						Reason:             "Issued",
						Message:            "Certificate issued",
					},
				},
				Certificate: []byte("bogus"),
			},
			namespaceName: types.NamespacedName{
				Namespace: "default",
				Name:      "foobar",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			client := fake.NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithRuntimeObjects(tt.objects...).
				WithStatusSubresource(&cmapi.CertificateRequest{}).
				Build()

			controller := &CertificateRequestController{
				Client:     client,
				Log:        logf.Log,
				Collection: tt.collection,
			}

			_, err := controller.Reconcile(context.Background(), reconcile.Request{
				NamespacedName: tt.namespaceName,
			})

			if err != nil {
				if diff := cmp.Diff(err.Error(), tt.error); diff != "" {
					t.Fatalf("diff: (-wanted +got)\n%s", diff)
				}
			}

			got := &cmapi.CertificateRequest{}
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
