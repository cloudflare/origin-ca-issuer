// Package provisioners provides a mapping between CertificateRequest
// and the Cloudflare API, with credentials already bounded by an
// OriginIssuer.
package provisioners

import (
	"context"
	"fmt"
	"math"
	"sync"

	"github.com/cloudflare/origin-ca-issuer/internal/cfapi"
	v1 "github.com/cloudflare/origin-ca-issuer/pkgs/apis/v1"
	"github.com/go-logr/logr"
	certmanager "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	"github.com/jetstack/cert-manager/pkg/util/pki"
	"k8s.io/apimachinery/pkg/types"
)

const (
	// The default validity duration, if not provided.
	DefaultDurationInternval = 7
)

var allowedValidty = []int{7, 30, 90, 365, 730, 1095, 5475}

// Collection stores cached Provisioners, stored by namespaced names of the
// issuer.
type Collection struct {
	m sync.Map
}

// A CollectionItem allows for the namespaced name and provisioner to
// be stored together.
type CollectionItem struct {
	NamespacedName types.NamespacedName
	Provisioner    *Provisioner
}

// CollectionWith returns a Collection storing the provided provisioners.
func CollectionWith(items []CollectionItem) *Collection {
	c := &Collection{}

	for _, i := range items {
		c.Store(i.NamespacedName, i.Provisioner)
	}

	return c
}

// Provisioner allows for CertificateRequests to be signed using the stored
// Cloudflare API client.
type Provisioner struct {
	client Signer
	log    logr.Logger

	reqType v1.RequestType
}

// Signer implements the Origin CA signing API.
type Signer interface {
	Sign(ctx context.Context, req *cfapi.SignRequest) (*cfapi.SignResponse, error)
}

// New returns a new provisioner.
func New(client Signer, reqType v1.RequestType, log logr.Logger) (*Provisioner, error) {
	p := &Provisioner{
		client:  client,
		log:     log,
		reqType: reqType,
	}

	return p, nil
}

// Store adds a provisioner to the collection.
func (c *Collection) Store(namespacedName types.NamespacedName, provisioner *Provisioner) {
	c.m.Store(namespacedName, provisioner)
}

// Load returns the stored provisioner, or returns false if nothing is cached with
// the proved namespaced name.
func (c *Collection) Load(namespacedName types.NamespacedName) (*Provisioner, bool) {
	v, ok := c.m.Load(namespacedName)
	if !ok {
		return nil, ok
	}

	p, ok := v.(*Provisioner)

	return p, ok
}

// Sign uses the Cloduflare API to sign a CertificateRequest. The validity of the CertificateRequest is
// normalized to the closests validity allowed by the Cloudflare API, which make be significantly different
// than the validity provided.
func (p *Provisioner) Sign(ctx context.Context, cr *certmanager.CertificateRequest) (certPem []byte, err error) {
	csr, err := pki.DecodeX509CertificateRequestBytes(cr.Spec.Request)
	if err != nil {
		return nil, fmt.Errorf("failed to decode CSR for signing: %s", err)
	}

	hostnames := csr.DNSNames
	var duration int
	if cr.Spec.Duration == nil {
		duration = DefaultDurationInternval
	} else {
		duration = closest(int(cr.Spec.Duration.Duration.Hours()/24), allowedValidty)
	}

	var reqType string
	switch p.reqType {
	case v1.RequestTypeOriginECC:
		reqType = "origin-ecc"
	case v1.RequestTypeOriginRSA:
		reqType = "origin-rsa"
	}

	resp, err := p.client.Sign(ctx, &cfapi.SignRequest{
		Hostnames: hostnames,
		Validity:  duration,
		Type:      reqType,
		CSR:       string(cr.Spec.Request),
	})

	if err != nil {
		return nil, fmt.Errorf("unable to sign request: %w", err)
	}

	return []byte(resp.Certificate), nil
}

func closest(of int, valid []int) int {
	min := math.MaxFloat64
	closest := of

	for _, v := range valid {
		diff := math.Abs(float64(v - of))

		if diff < min {
			min = diff
			closest = v
		}
	}

	return closest
}
