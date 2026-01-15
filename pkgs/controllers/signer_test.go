package controllers

import (
	"testing"
	"testing/quick"
	"time"

	cmapi "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	cmgen "github.com/cert-manager/cert-manager/test/unit/gen"
	issuerv1alpha1 "github.com/cert-manager/issuer-lib/api/v1alpha1"
	"github.com/cert-manager/issuer-lib/controllers/signer"
	"github.com/cloudflare/origin-ca-issuer/internal/cfapi"
	v1 "github.com/cloudflare/origin-ca-issuer/pkgs/apis/v1"
	"gopkg.in/dnaeon/go-vcr.v4/pkg/cassette"
	"gopkg.in/dnaeon/go-vcr.v4/pkg/recorder"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/golden"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestCertificateRequestReconcile(t *testing.T) {
	assert.NilError(t, cmapi.AddToScheme(scheme.Scheme))
	assert.NilError(t, v1.AddToScheme(scheme.Scheme))

	tests := []struct {
		name     string
		request  *cmapi.CertificateRequest
		issuer   issuerv1alpha1.Issuer
		secret   *corev1.Secret
		recorder *recorder.Recorder
		expected signer.PEMBundle
		error    error
	}{
		{
			name: "working OriginIssuer with serviceKeyRef",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "service-key-issuer",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"key": []byte("v1.0-0x00BAB10C"),
				},
			},
			request: cmgen.CertificateRequest("foobar",
				cmgen.SetCertificateRequestNamespace("default"),
				cmgen.SetCertificateRequestDuration(&metav1.Duration{Duration: 7 * 24 * time.Hour}),
				cmgen.SetCertificateRequestCSR(golden.Get(t, "csr.golden")),
				cmgen.SetCertificateRequestIssuer(cmmeta.ObjectReference{
					Name:  "foobar",
					Kind:  "OriginIssuer",
					Group: "cert-manager.k8s.cloudflare.com",
				}),
			),
			issuer: &v1.OriginIssuer{
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
				Status: issuerv1alpha1.IssuerStatus{
					Conditions: []metav1.Condition{
						{
							Type:   string(cmapi.IssuerConditionReady),
							Status: metav1.ConditionTrue,
						},
					},
				},
			},
			recorder: RecorderMust(t, "testdata/working"),
			expected: signer.PEMBundle{
				ChainPEM: golden.Get(t, "certificate.golden"),
				CAPEM:    eccCAPEM,
			},
		},
		{
			name: "working ClusterOriginIssuer with serviceKeyRef",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "service-key-issuer",
					Namespace: "super-secret",
				},
				Data: map[string][]byte{
					"key": []byte("v1.0-0x00BAB10C"),
				},
			},
			request: cmgen.CertificateRequest("foobar",
				cmgen.SetCertificateRequestNamespace("default"),
				cmgen.SetCertificateRequestDuration(&metav1.Duration{Duration: 7 * 24 * time.Hour}),
				cmgen.SetCertificateRequestCSR(golden.Get(t, "csr.golden")),
				cmgen.SetCertificateRequestIssuer(cmmeta.ObjectReference{
					Name:  "foobar",
					Kind:  "ClusterOriginIssuer",
					Group: "cert-manager.k8s.cloudflare.com",
				}),
			),
			issuer: &v1.ClusterOriginIssuer{
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
				Status: issuerv1alpha1.IssuerStatus{
					Conditions: []metav1.Condition{
						{
							Type:   string(cmapi.IssuerConditionReady),
							Status: metav1.ConditionTrue,
						},
					},
				},
			},
			recorder: RecorderMust(t, "testdata/working"),
			expected: signer.PEMBundle{
				ChainPEM: golden.Get(t, "certificate.golden"),
				CAPEM:    eccCAPEM,
			},
		},
		{
			name: "working OriginIssuer with tokenRef",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "token-issuer",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"token": []byte("api-token"),
				},
			},
			request: cmgen.CertificateRequest("foobar",
				cmgen.SetCertificateRequestNamespace("default"),
				cmgen.SetCertificateRequestDuration(&metav1.Duration{Duration: 7 * 24 * time.Hour}),
				cmgen.SetCertificateRequestCSR(golden.Get(t, "csr.golden")),
				cmgen.SetCertificateRequestIssuer(cmmeta.ObjectReference{
					Name:  "foobar",
					Kind:  "OriginIssuer",
					Group: "cert-manager.k8s.cloudflare.com",
				}),
			),
			issuer: &v1.OriginIssuer{
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
				Status: issuerv1alpha1.IssuerStatus{
					Conditions: []metav1.Condition{
						{
							Type:   string(cmapi.IssuerConditionReady),
							Status: metav1.ConditionTrue,
						},
					},
				},
			},
			recorder: RecorderMust(t, "testdata/working"),
			expected: signer.PEMBundle{
				ChainPEM: golden.Get(t, "certificate.golden"),
				CAPEM:    eccCAPEM,
			},
		},
		{
			name: "working ClusterOriginIssuer with tokenRef",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "token-issuer",
					Namespace: "super-secret",
				},
				Data: map[string][]byte{
					"token": []byte("api-token"),
				},
			},
			request: cmgen.CertificateRequest("foobar",
				cmgen.SetCertificateRequestNamespace("default"),
				cmgen.SetCertificateRequestDuration(&metav1.Duration{Duration: 7 * 24 * time.Hour}),
				cmgen.SetCertificateRequestCSR(golden.Get(t, "csr.golden")),
				cmgen.SetCertificateRequestIssuer(cmmeta.ObjectReference{
					Name:  "foobar",
					Kind:  "ClusterOriginIssuer",
					Group: "cert-manager.k8s.cloudflare.com",
				}),
			),
			issuer: &v1.ClusterOriginIssuer{
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
				Status: issuerv1alpha1.IssuerStatus{
					Conditions: []metav1.Condition{
						{
							Type:   string(cmapi.IssuerConditionReady),
							Status: metav1.ConditionTrue,
						},
					},
				},
			},
			recorder: RecorderMust(t, "testdata/working"),
			expected: signer.PEMBundle{
				ChainPEM: golden.Get(t, "certificate.golden"),
				CAPEM:    eccCAPEM,
			},
		},
		{
			name:   "OriginIssuer without authentication",
			secret: &corev1.Secret{},
			request: cmgen.CertificateRequest("foobar",
				cmgen.SetCertificateRequestNamespace("default"),
				cmgen.SetCertificateRequestDuration(&metav1.Duration{Duration: 7 * 24 * time.Hour}),
				cmgen.SetCertificateRequestCSR(golden.Get(t, "csr.golden")),
				cmgen.SetCertificateRequestIssuer(cmmeta.ObjectReference{
					Name:  "foobar",
					Kind:  "OriginIssuer",
					Group: "cert-manager.k8s.cloudflare.com",
				}),
			),
			issuer: &v1.OriginIssuer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foobar",
					Namespace: "default",
				},
				Spec: v1.OriginIssuerSpec{},
				Status: issuerv1alpha1.IssuerStatus{
					Conditions: []metav1.Condition{
						{
							Type:   string(cmapi.IssuerConditionReady),
							Status: metav1.ConditionTrue,
						},
					},
				},
			},
			error: signer.PermanentError{
				Err: errNoAuthMethods,
			},
		},
		{
			name: "requeue after API error",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "service-key-issuer",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"key": []byte("djEuMC0weDAwQkFCMTBD"),
				},
			},
			request: cmgen.CertificateRequest("foobar",
				cmgen.SetCertificateRequestNamespace("default"),
				cmgen.SetCertificateRequestDuration(&metav1.Duration{Duration: 7 * 24 * time.Hour}),
				cmgen.SetCertificateRequestCSR(golden.Get(t, "csr.golden")),
				cmgen.SetCertificateRequestIssuer(cmmeta.ObjectReference{
					Name:  "foobar",
					Kind:  "OriginIssuer",
					Group: "cert-manager.k8s.cloudflare.com",
				}),
			),
			issuer: &v1.OriginIssuer{
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
				Status: issuerv1alpha1.IssuerStatus{
					Conditions: []metav1.Condition{
						{
							Type:   string(cmapi.IssuerConditionReady),
							Status: metav1.ConditionTrue,
						},
					},
				},
			},
			recorder: RecorderMust(t, "testdata/database-failure"),
			error:    &cfapi.APIError{Code: 1100},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fake.NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithRuntimeObjects(tt.secret, tt.issuer).
				WithStatusSubresource(&cmapi.CertificateRequest{}).
				Build()

			if tt.recorder != nil {
				defer tt.recorder.Stop()
			}

			s := Signer{
				Reader:                   client,
				ClusterResourceNamespace: "super-secret",
				Builder:                  cfapi.NewBuilder().WithClient(tt.recorder.GetDefaultClient()),
			}
			bundle, err := s.Sign(t.Context(), signer.CertificateRequestObjectFromCertificateRequest(tt.request), tt.issuer)

			if err != nil {
				assert.ErrorIs(t, err, tt.error)
			} else {
				assert.NilError(t, err)
			}

			assert.DeepEqual(t, bundle, tt.expected)
		})
	}
}

func TestClosest(t *testing.T) {
	index := func(x int, s []int) int {
		for i, n := range s {
			if x == n {
				return i
			}
		}

		return -1
	}

	f := func(x int) bool {
		d := closest(x, allowedValidity)
		return index(d, allowedValidity) >= 0
	}

	err := quick.Check(f, nil)
	assert.NilError(t, err)
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
		recorder.WithMatcher(cassette.NewDefaultMatcher(cassette.WithIgnoreUserAgent(true))),
	)
	if err != nil {
		t.Fatal(err)
	}

	return recorder
}
