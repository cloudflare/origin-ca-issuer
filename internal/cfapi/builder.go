package cfapi

import (
	"net/http"
)

type Builder struct {
	hc    *http.Client
	token []byte
}

func NewBuilder() *Builder {
	return &Builder{}
}

func (b *Builder) WithToken(token []byte) *Builder {
	b.token = token
	return b
}

func (b *Builder) WithClient(hc *http.Client) *Builder {
	b.hc = hc
	return b
}

func (b *Builder) Clone() *Builder {
	return &Builder{
		hc:    b.hc,
		token: b.token,
	}
}

func (b *Builder) Build() *Client {
	if b.token != nil {
		return New(WithToken(b.token), WithClient(b.hc))
	}
	return nil
}
