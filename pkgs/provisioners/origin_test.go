package provisioners

import (
	"context"
	"crypto/x509"
	"errors"
	"testing"
	"testing/quick"
	"time"

	"github.com/cloudflare/origin-ca-issuer/internal/cfapi"
	v1 "github.com/cloudflare/origin-ca-issuer/pkgs/apis/v1"
	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp/cmpopts"
	certmanager "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	cmgen "github.com/jetstack/cert-manager/test/unit/gen"
	"gotest.tools/v3/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSign(t *testing.T) {
	type testCase struct {
		name     string
		reqType  v1.RequestType
		req      *certmanager.CertificateRequest
		signReq  *cfapi.SignRequest
		expected []byte
	}

	run := func(t *testing.T, tc testCase) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		signer := SignerFunc(func(ctx context.Context, req *cfapi.SignRequest) (*cfapi.SignResponse, error) {
			assert.DeepEqual(t, req, tc.signReq, cmpopts.IgnoreFields(cfapi.SignRequest{}, "CSR"))
			return &cfapi.SignResponse{
				Certificate: "-----BEGIN CERTIFICATE-----\n-----END CERTIFICATE-----\n",
			}, nil
		})

		provisioner, err := New(signer, tc.reqType, logr.Discard())
		assert.NilError(t, err)

		res, err := provisioner.Sign(ctx, tc.req)
		assert.NilError(t, err)
		assert.DeepEqual(t, res, tc.expected)
	}

	testCases := []testCase{
		{
			name:    "origin rsa",
			reqType: v1.RequestTypeOriginRSA,
			req: cmgen.CertificateRequest("foobar",
				cmgen.SetCertificateRequestNamespace("default"),
				cmgen.SetCertificateRequestDuration(&metav1.Duration{Duration: 7 * 24 * time.Hour}),
				cmgen.SetCertificateRequestCSR((func() []byte {
					csr, _, err := cmgen.CSR(x509.RSA, cmgen.SetCSRDNSNames("example.com"))
					assert.NilError(t, err)

					return csr
				})()),
			),
			signReq: &cfapi.SignRequest{
				Hostnames: []string{"example.com"},
				Validity:  7,
				Type:      "origin-rsa",
				CSR:       "",
			},
			expected: []byte("-----BEGIN CERTIFICATE-----\n-----END CERTIFICATE-----\n"),
		},
		{
			name:    "origin ecc",
			reqType: v1.RequestTypeOriginECC,
			req: cmgen.CertificateRequest("foobar",
				cmgen.SetCertificateRequestNamespace("default"),
				cmgen.SetCertificateRequestDuration(&metav1.Duration{Duration: 7 * 24 * time.Hour}),
				cmgen.SetCertificateRequestCSR((func() []byte {
					csr, _, err := cmgen.CSR(x509.ECDSA, cmgen.SetCSRDNSNames("example.com"))
					assert.NilError(t, err)

					return csr
				})()),
			),
			signReq: &cfapi.SignRequest{
				Hostnames: []string{"example.com"},
				Validity:  7,
				Type:      "origin-ecc",
				CSR:       "",
			},
			expected: []byte("-----BEGIN CERTIFICATE-----\n-----END CERTIFICATE-----\n"),
		},
		{
			name:    "find closest duration",
			reqType: v1.RequestTypeOriginECC,
			req: cmgen.CertificateRequest("foobar",
				cmgen.SetCertificateRequestNamespace("default"),
				cmgen.SetCertificateRequestDuration(&metav1.Duration{Duration: 10 * 365 * 24 * time.Hour}),
				cmgen.SetCertificateRequestCSR((func() []byte {
					csr, _, err := cmgen.CSR(x509.ECDSA, cmgen.SetCSRDNSNames("example.com"))
					assert.NilError(t, err)

					return csr
				})()),
			),
			signReq: &cfapi.SignRequest{
				Hostnames: []string{"example.com"},
				Validity:  5475,
				Type:      "origin-ecc",
				CSR:       "",
			},
			expected: []byte("-----BEGIN CERTIFICATE-----\n-----END CERTIFICATE-----\n"),
		},
		{
			name:    "default duration",
			reqType: v1.RequestTypeOriginECC,
			req: cmgen.CertificateRequest("foobar",
				cmgen.SetCertificateRequestNamespace("default"),
				cmgen.SetCertificateRequestCSR((func() []byte {
					csr, _, err := cmgen.CSR(x509.ECDSA, cmgen.SetCSRDNSNames("example.com"))
					assert.NilError(t, err)

					return csr
				})()),
			),
			signReq: &cfapi.SignRequest{
				Hostnames: []string{"example.com"},
				Validity:  7,
				Type:      "origin-ecc",
				CSR:       "",
			},
			expected: []byte("-----BEGIN CERTIFICATE-----\n-----END CERTIFICATE-----\n"),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func TestSign_Error(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	signer := SignerFunc(func(ctx context.Context, req *cfapi.SignRequest) (*cfapi.SignResponse, error) {
		return nil, errors.New("cfapi error")
	})

	req := cmgen.CertificateRequest("foobar",
		cmgen.SetCertificateRequestNamespace("default"),
		cmgen.SetCertificateRequestCSR((func() []byte {
			csr, _, err := cmgen.CSR(x509.ECDSA, cmgen.SetCSRDNSNames("example.com"))
			assert.NilError(t, err)

			return csr
		})()),
	)

	provisioner, err := New(signer, v1.RequestTypeOriginECC, logr.Discard())
	assert.NilError(t, err)

	_, err = provisioner.Sign(ctx, req)
	assert.Error(t, err, "unable to sign request: cfapi error")
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
		d := closest(x, allowedValidty)
		return index(d, allowedValidty) >= 0
	}

	err := quick.Check(f, nil)
	assert.NilError(t, err)
}

type SignerFunc func(ctx context.Context, req *cfapi.SignRequest) (*cfapi.SignResponse, error)

func (f SignerFunc) Sign(ctx context.Context, req *cfapi.SignRequest) (*cfapi.SignResponse, error) {
	return f(ctx, req)
}
