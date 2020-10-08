package testingcfapi

import (
	"context"

	"github.com/cloudflare/origin-ca-issuer/internal/cfapi"
)

type FakeClient struct {
	Response *cfapi.SignResponse
}

func (f *FakeClient) Sign(context.Context, *cfapi.SignRequest) (*cfapi.SignResponse, error) {
	return f.Response, nil
}
