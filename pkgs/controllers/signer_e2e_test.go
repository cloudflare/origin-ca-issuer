package controllers

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	cmutil "github.com/cert-manager/cert-manager/pkg/api/util"
	cmapi "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	cmgen "github.com/cert-manager/cert-manager/test/unit/gen"
	"github.com/cert-manager/issuer-lib/conditions"
	"github.com/cloudflare/origin-ca-issuer/internal/cfapi"
	v1 "github.com/cloudflare/origin-ca-issuer/pkgs/apis/v1"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/golden"
	"gotest.tools/v3/poll"
	"gotest.tools/v3/skip"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/utils/clock"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

func TestOriginIssuerCertificateRequestE2E(t *testing.T) {
	ctx := t.Context()

	issuer := &v1.OriginIssuer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "default",
		},
		Spec: v1.OriginIssuerSpec{
			RequestType: v1.RequestTypeOriginECC,
			Auth: v1.OriginIssuerAuthentication{
				ServiceKeyRef: &v1.SecretKeySelector{
					Name: "issuer-service-key",
					Key:  "key",
				},
			},
		},
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "issuer-service-key",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"key": []byte("v1.0-0x00BAB10C"),
		},
	}
	request := cmgen.CertificateRequest("foobar",
		cmgen.SetCertificateRequestNamespace("default"),
		cmgen.SetCertificateRequestDuration(&metav1.Duration{Duration: 7 * 24 * time.Hour}),
		cmgen.SetCertificateRequestCSR(golden.Get(t, "csr.golden")),
		cmgen.SetCertificateRequestIssuer(cmmeta.ObjectReference{
			Name:  "foo",
			Kind:  "OriginIssuer",
			Group: "cert-manager.k8s.cloudflare.com",
		}),
	)

	skip.If(t, os.Getenv("KUBEBUILDER_ASSETS") == "", "no kubebuilder environment")

	cfg := envtestConfig(t)

	mgr, err := manager.New(cfg, manager.Options{
		Metrics: metricsserver.Options{
			BindAddress: "0",
		},
		Scheme: scheme.Scheme,
	})
	assert.NilError(t, err)

	c := mgr.GetClient()

	recorder := RecorderMust(t, "testdata/working")
	signer := &Signer{
		Reader:  mgr.GetAPIReader(),
		Builder: cfapi.NewBuilder().WithClient(recorder.GetDefaultClient()),
	}
	signer.SetupWithManager(ctx, mgr)

	StartTestManager(mgr, t)

	assert.NilError(t, c.Create(ctx, secret))
	assert.NilError(t, c.Create(ctx, issuer))

	poll.WaitOn(t, func(t poll.LogT) poll.Result {
		iss := v1.OriginIssuer{}
		namespacedName := types.NamespacedName{
			Namespace: issuer.Namespace,
			Name:      issuer.Name,
		}

		err := c.Get(ctx, namespacedName, &iss)
		if err != nil {
			return poll.Continue("issuer %q was not created", namespacedName.String())
		}

		condition := conditions.GetIssuerStatusCondition(iss.GetConditions(), string(cmapi.IssuerConditionReady))
		if condition == nil {
			return poll.Continue("issuer %q does not have Ready status", namespacedName.String())
		}

		if condition.Status == metav1.ConditionTrue {
			return poll.Success()
		}

		return poll.Continue("issuer %q is Not Ready", namespacedName.String())
	})

	assert.NilError(t, c.Create(ctx, request))
	conditions.SetCertificateRequestStatusCondition(
		clock.RealClock{},
		request.Status.Conditions,
		&request.Status.Conditions,
		cmapi.CertificateRequestConditionApproved,
		cmmeta.ConditionTrue,
		"Approved",
		"Approved by Decree",
	)
	assert.NilError(t, c.Status().Update(ctx, request))
	poll.WaitOn(t, func(t poll.LogT) poll.Result {
		cr := &cmapi.CertificateRequest{}
		namespacedName := types.NamespacedName{
			Namespace: request.Namespace,
			Name:      request.Name,
		}

		err := c.Get(ctx, namespacedName, cr)
		if err != nil {
			return poll.Error(err)
		}

		if cmutil.CertificateRequestHasCondition(cr, cmapi.CertificateRequestCondition{
			Type:   cmapi.CertificateRequestConditionReady,
			Status: cmmeta.ConditionTrue,
		}) {
			return poll.Success()
		}

		return poll.Continue("certificate request %q is Not Ready", namespacedName.String())
	})

	cr := &cmapi.CertificateRequest{}
	namespacedName := types.NamespacedName{
		Namespace: request.Namespace,
		Name:      request.Name,
	}
	assert.NilError(t, c.Get(ctx, namespacedName, cr))
	golden.AssertBytes(t, cr.Status.Certificate, "certificate.golden")
}

func envtestConfig(t *testing.T) *rest.Config {
	t.Helper()

	env := &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "..", "deploy", "crds"),
			filepath.Join("..", "..", "hack", "crds"),
		},
	}
	cmapi.AddToScheme(scheme.Scheme)
	v1.AddToScheme(scheme.Scheme)

	cfg, err := env.Start()
	assert.NilError(t, err)

	t.Cleanup(func() {
		assert.NilError(t, env.Stop())
	})

	return cfg
}

func StartTestManager(mgr manager.Manager, t *testing.T) {
	t.Helper()

	var err error
	go func() {
		err = mgr.Start(t.Context())
	}()

	t.Cleanup(func() {
		assert.NilError(t, err)
	})
}
