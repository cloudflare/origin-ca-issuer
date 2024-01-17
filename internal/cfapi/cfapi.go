package cfapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type Interface interface {
	Sign(context.Context, *SignRequest) (*SignResponse, error)
}

type Client struct {
	serviceKey []byte
	client     *http.Client
	endpoint   string
}

func New(serviceKey []byte, options ...Options) *Client {
	c := &Client{
		serviceKey: serviceKey,
		client:     http.DefaultClient,
		endpoint:   "https://api.cloudflare.com/client/v4/certificates",
	}

	for _, opt := range options {
		opt(c)
	}

	return c
}

type Options func(c *Client)

func WithClient(client *http.Client) Options {
	return func(c *Client) {
		c.client = client
	}
}

func WithEndpoint(endpoint string) (Options, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}

	u.Path = "/client/v4/certificates"

	return func(c *Client) {
		c.endpoint = u.String()
	}, nil
}

type SignRequest struct {
	Hostnames []string `json:"hostnames"`
	Validity  int      `json:"requested_validity"`
	Type      string   `json:"request_type"`
	CSR       string   `json:"csr"`
}

type SignResponse struct {
	Id          string    `json:"id"`
	Certificate string    `json:"certificate"`
	Hostnames   []string  `json:"hostnames"`
	Expiration  time.Time `json:"expires_on"`
	Type        string    `json:"request_type"`
	Validity    int       `json:"requested_validity"`
	CSR         string    `json:"csr"`
}

type APIResponse struct {
	Success  bool            `json:"success"`
	Errors   []APIError      `json:"errors"`
	Messages []string        `json:"messages"`
	Result   json.RawMessage `json:"result"`
}

type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	RayID   string `json:"-"`
}

func (a *APIError) Error() string {
	return fmt.Sprintf("Cloudflare API Error code=%d message=%s ray_id=%s", a.Code, a.Message, a.RayID)
}

func (c *Client) Sign(ctx context.Context, req *SignRequest) (*SignResponse, error) {
	p, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	r, err := http.NewRequestWithContext(ctx, "POST", c.endpoint, bytes.NewBuffer(p))
	if err != nil {
		return nil, err
	}

	r.Header.Add("User-Agent", "github.com/cloudflare/origin-ca-issuer")
	r.Header.Add("X-Auth-User-Service-Key", string(c.serviceKey))

	resp, err := c.client.Do(r)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	rayID := resp.Header.Get("CF-Ray")

	api := APIResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&api); err != nil {
		return nil, err
	}

	if !api.Success {
		err := &api.Errors[0]
		err.RayID = rayID
		return nil, err
	}

	signResp := SignResponse{}
	if err := json.Unmarshal(api.Result, &signResp); err != nil {
		return nil, err
	}

	return &signResp, nil
}

// adapted from http://choly.ca/post/go-json-marshalling/
func (r *SignResponse) UnmarshalJSON(p []byte) error {
	type resp SignResponse

	tmp := &struct {
		Expiration string `json:"expires_on"`
		*resp
	}{
		resp: (*resp)(r),
	}

	if err := json.Unmarshal(p, &tmp); err != nil {
		return err
	}

	var err error
	r.Expiration, err = time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", tmp.Expiration)

	if err != nil {
		r.Expiration, err = time.Parse(time.RFC3339, tmp.Expiration)
	}

	if err != nil {
		return err
	}

	return nil
}
