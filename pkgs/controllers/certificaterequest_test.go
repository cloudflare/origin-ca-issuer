package controllers

import (
	"context"
	"testing"
	"time"

	cmutil "github.com/cert-manager/cert-manager/pkg/api/util"
	cmapi "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	cmgen "github.com/cert-manager/cert-manager/test/unit/gen"
	"github.com/cloudflare/origin-ca-issuer/internal/cfapi"
	v1 "github.com/cloudflare/origin-ca-issuer/pkgs/apis/v1"
	"gopkg.in/dnaeon/go-vcr.v4/pkg/cassette"
	"gopkg.in/dnaeon/go-vcr.v4/pkg/recorder"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/golden"
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
		recorder      *recorder.Recorder
		expected      cmapi.CertificateRequestStatus
		error         string
		namespaceName types.NamespacedName
	}{
		{
			name: "working OriginIssuer with serviceKeyRef",
			objects: []runtime.Object{
				cmgen.CertificateRequest("foobar",
					cmgen.SetCertificateRequestNamespace("default"),
					cmgen.SetCertificateRequestDuration(&metav1.Duration{Duration: 7 * 24 * time.Hour}),
					cmgen.SetCertificateRequestCSR(golden.Get(t, "csr.golden")),
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
						RequestType: v1.RequestTypeOriginECC,
						Auth: v1.OriginIssuerAuthentication{
							ServiceKeyRef: &v1.SecretKeySelector{
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
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "service-key-issuer",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"key": []byte("djEuMC0weDAwQkFCMTBD"),
					},
				},
			},
			recorder: RecorderMust(t, "testdata/working"),
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
				Certificate: golden.Get(t, "certificate.golden"),
				CA:          eccCAPEM,
			},
			namespaceName: types.NamespacedName{
				Namespace: "default",
				Name:      "foobar",
			},
		},
		{
			name: "working ClusterOriginIssuer with serviceKeyRef",
			objects: []runtime.Object{
				cmgen.CertificateRequest("foobar",
					cmgen.SetCertificateRequestNamespace("default"),
					cmgen.SetCertificateRequestDuration(&metav1.Duration{Duration: 7 * 24 * time.Hour}),
					cmgen.SetCertificateRequestCSR(golden.Get(t, "csr.golden")),
					cmgen.SetCertificateRequestIssuer(cmmeta.ObjectReference{
						Name:  "foobar",
						Kind:  "ClusterOriginIssuer",
						Group: "cert-manager.k8s.cloudflare.com",
					}),
				),
				&v1.ClusterOriginIssuer{
					ObjectMeta: metav1.ObjectMeta{
						Name: "foobar",
					},
					Spec: v1.OriginIssuerSpec{
						RequestType: v1.RequestTypeOriginECC,
						Auth: v1.OriginIssuerAuthentication{
							ServiceKeyRef: &v1.SecretKeySelector{
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
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "service-key-issuer",
						Namespace: "super-secret",
					},
					Data: map[string][]byte{
						"key": []byte("djEuMC0weDAwQkFCMTBD"),
					},
				},
			},
			recorder: RecorderMust(t, "testdata/working"),
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
				Certificate: golden.Get(t, "certificate.golden"),
				CA:          eccCAPEM,
			},
			namespaceName: types.NamespacedName{
				Namespace: "default",
				Name:      "foobar",
			},
		},
		{
			name: "working OriginIssuer with tokenRef",
			objects: []runtime.Object{
				cmgen.CertificateRequest("foobar",
					cmgen.SetCertificateRequestNamespace("default"),
					cmgen.SetCertificateRequestDuration(&metav1.Duration{Duration: 7 * 24 * time.Hour}),
					cmgen.SetCertificateRequestCSR(golden.Get(t, "csr.golden")),
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
						RequestType: v1.RequestTypeOriginECC,
						Auth: v1.OriginIssuerAuthentication{
							TokenRef: &v1.SecretKeySelector{
								Name: "token-issuer",
								Key:  "token",
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
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "token-issuer",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"token": []byte("api-token"),
					},
				},
			},
			recorder: RecorderMust(t, "testdata/working"),
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
				Certificate: golden.Get(t, "certificate.golden"),
				CA:          eccCAPEM,
			},
			namespaceName: types.NamespacedName{
				Namespace: "default",
				Name:      "foobar",
			},
		},
		{
			name: "working ClusterOriginIssuer with tokenRef",
			objects: []runtime.Object{
				cmgen.CertificateRequest("foobar",
					cmgen.SetCertificateRequestNamespace("default"),
					cmgen.SetCertificateRequestDuration(&metav1.Duration{Duration: 7 * 24 * time.Hour}),
					cmgen.SetCertificateRequestCSR(golden.Get(t, "csr.golden")),
					cmgen.SetCertificateRequestIssuer(cmmeta.ObjectReference{
						Name:  "foobar",
						Kind:  "ClusterOriginIssuer",
						Group: "cert-manager.k8s.cloudflare.com",
					}),
				),
				&v1.ClusterOriginIssuer{
					ObjectMeta: metav1.ObjectMeta{
						Name: "foobar",
					},
					Spec: v1.OriginIssuerSpec{
						RequestType: v1.RequestTypeOriginECC,
						Auth: v1.OriginIssuerAuthentication{
							TokenRef: &v1.SecretKeySelector{
								Name: "token-issuer",
								Key:  "token",
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
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "token-issuer",
						Namespace: "super-secret",
					},
					Data: map[string][]byte{
						"token": []byte("api-token"),
					},
				},
			},
			recorder: RecorderMust(t, "testdata/working"),
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
				Certificate: golden.Get(t, "certificate.golden"),
				CA:          eccCAPEM,
			},
			namespaceName: types.NamespacedName{
				Namespace: "default",
				Name:      "foobar",
			},
		},
		{
			name: "ignores CertificateRequests with empty Issuer group reference",
			objects: []runtime.Object{
				cmgen.CertificateRequest("foobar",
					cmgen.SetCertificateRequestNamespace("default"),
					cmgen.SetCertificateRequestDuration(&metav1.Duration{Duration: 7 * 24 * time.Hour}),
					cmgen.SetCertificateRequestCSR(golden.Get(t, "csr.golden")),
					cmgen.SetCertificateRequestIssuer(cmmeta.ObjectReference{
						Name: "foobar",
						Kind: "StepIssuer", // 👋 hello friends!
					}),
				),
				&v1.ClusterOriginIssuer{
					ObjectMeta: metav1.ObjectMeta{
						Name: "foobar",
					},
					Spec: v1.OriginIssuerSpec{
						RequestType: v1.RequestTypeOriginECC,
						Auth: v1.OriginIssuerAuthentication{
							TokenRef: &v1.SecretKeySelector{
								Name: "token-issuer",
								Key:  "token",
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
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "token-issuer",
						Namespace: "super-secret",
					},
					Data: map[string][]byte{
						"token": []byte("api-token"),
					},
				},
			},
			recorder: RecorderMust(t, "testdata/working"),
			expected: cmapi.CertificateRequestStatus{},
			namespaceName: types.NamespacedName{
				Namespace: "default",
				Name:      "foobar",
			},
		},
		{
			name: "OriginIssuer without authentication",
			objects: []runtime.Object{
				cmgen.CertificateRequest("foobar",
					cmgen.SetCertificateRequestNamespace("default"),
					cmgen.SetCertificateRequestDuration(&metav1.Duration{Duration: 7 * 24 * time.Hour}),
					cmgen.SetCertificateRequestCSR(golden.Get(t, "csr.golden")),
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
					Spec: v1.OriginIssuerSpec{},
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
			namespaceName: types.NamespacedName{
				Namespace: "default",
				Name:      "foobar",
			},
			error: "issuer foobar does not have an authentication method configured",
		},
		{
			name: "requeue after API error",
			objects: []runtime.Object{
				cmgen.CertificateRequest("foobar",
					cmgen.SetCertificateRequestNamespace("default"),
					cmgen.SetCertificateRequestDuration(&metav1.Duration{Duration: 7 * 24 * time.Hour}),
					cmgen.SetCertificateRequestCSR(golden.Get(t, "csr.golden")),
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
						RequestType: v1.RequestTypeOriginECC,
						Auth: v1.OriginIssuerAuthentication{
							ServiceKeyRef: &v1.SecretKeySelector{
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
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "service-key-issuer",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"key": []byte("djEuMC0weDAwQkFCMTBD"),
					},
				},
			},
			recorder: RecorderMust(t, "testdata/database-failure"),
			namespaceName: types.NamespacedName{
				Namespace: "default",
				Name:      "foobar",
			},
			error: "unable to sign request: Cloudflare API Error code=1100 message=Failed to write certificate to Database ray_id=0123456789abcdef-ABC",
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

			if tt.recorder != nil {
				defer tt.recorder.Stop()
			}

			controller := &CertificateRequestController{
				Client:                   client,
				Reader:                   client,
				ClusterResourceNamespace: "super-secret",
				Log:                      logf.Log,
				Builder:                  cfapi.NewBuilder().WithClient(tt.recorder.GetDefaultClient()),
			}

			_, err := reconcile.AsReconciler(client, controller).Reconcile(context.Background(), reconcile.Request{
				NamespacedName: tt.namespaceName,
			})

			if err != nil {
				assert.Error(t, err, tt.error)
			} else {
				assert.NilError(t, err)
			}

			got := &cmapi.CertificateRequest{}
			assert.NilError(t, client.Get(context.TODO(), tt.namespaceName, got))
			assert.DeepEqual(t, got.Status, tt.expected)
		})
	}
}

func RecorderMust(t *testing.T, name string) *recorder.Recorder {
	t.Helper()
	recorder, err := recorder.New(name,
		recorder.WithHook(func(i *cassette.Interaction) error {
			delete(i.Response.Headers, "Set-Cookie")
			delete(i.Response.Headers, "Cf-Auditlog-Id")
			i.Response.Headers.Set("Cf-Ray", "0123456789abcdef-ABC")
			return nil
		}, recorder.BeforeSaveHook),
		recorder.WithSkipRequestLatency(true),
	)
	if err != nil {
		t.Fatal(err)
	}

	return recorder
}
