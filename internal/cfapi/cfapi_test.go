package cfapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestSignResponse_Unmarshal(t *testing.T) {
	expectedTime := time.Date(2020, time.December, 25, 6, 27, 0, 0, time.UTC)
	expected := SignResponse{
		Id:          "9001",
		Certificate: "-----BEGIN CERTIFICATE-----\n-----END CERTIFICATE-----\n",
		Hostnames:   []string{"example.com"},
		Expiration:  expectedTime,
		Type:        "origin-ecc",
		Validity:    7,
		CSR:         "-----BEGIN CERTIFICATE REQUEST-----\n-----END CERTIFICATE REQUEST-----",
	}

	tests := []struct {
		name    string
		payload []byte
	}{
		{
			name: "time.String",
			payload: []byte(`{
        "id":"9001",
        "certificate":"-----BEGIN CERTIFICATE-----\n-----END CERTIFICATE-----\n",
        "expires_on":"2020-12-25 06:27:00 +0000 UTC",
        "request_type":"origin-ecc",
        "hostnames":["example.com"],
        "csr":"-----BEGIN CERTIFICATE REQUEST-----\n-----END CERTIFICATE REQUEST-----",
        "requested_validity":7
      }`),
		},
		{
			name: "RFC3339",
			payload: []byte(`{
        "id":"9001",
        "certificate":"-----BEGIN CERTIFICATE-----\n-----END CERTIFICATE-----\n",
        "expires_on":"2020-12-25T06:27:00Z",
        "request_type":"origin-ecc",
        "hostnames":["example.com"],
        "csr":"-----BEGIN CERTIFICATE REQUEST-----\n-----END CERTIFICATE REQUEST-----",
        "requested_validity":7
      }`),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			var resp SignResponse

			if err := json.Unmarshal(tt.payload, &resp); err != nil {
				t.Fatalf("unable to unmarsahl: %s", err)
			}

			if diff := cmp.Diff(resp, expected); diff != "" {
				t.Fatalf("diff: (-want +got)\n%s", diff)
			}
		})
	}
}

func TestSign(t *testing.T) {
	expectedTime := time.Date(2020, time.December, 25, 6, 27, 0, 0, time.UTC)
	tests := []struct {
		name     string
		handler  http.Handler
		response *SignResponse
		error    string
	}{
		{name: "API success",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Add("cf-ray", "0123456789abcdef-ABC")
				fmt.Fprintln(w, `{
	"success": true,
	"errors": [],
	"message": [],
	"result": {
		"id":"9001",
		"certificate":"-----BEGIN CERTIFICATE-----\n-----END CERTIFICATE-----\n",
		"expires_on":"2020-12-25T06:27:00Z",
		"request_type":"origin-ecc",
		"hostnames":["example.com"],
		"csr":"-----BEGIN CERTIFICATE REQUEST-----\n-----END CERTIFICATE REQUEST-----",
		"requested_validity":7
	}
}`)
			}),
			response: &SignResponse{
				Id:          "9001",
				Certificate: "-----BEGIN CERTIFICATE-----\n-----END CERTIFICATE-----\n",
				Hostnames:   []string{"example.com"},
				Expiration:  expectedTime,
				Type:        "origin-ecc",
				Validity:    7,
				CSR:         "-----BEGIN CERTIFICATE REQUEST-----\n-----END CERTIFICATE REQUEST-----",
			},
			error: "",
		},
		{
			name: "API error",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Add("cf-ray", "0123456789abcdef-ABC")
				fmt.Fprintln(w, `{
	"success": false,
	"errors": [{"code": 9001, "message": "Over Nine Thousand!"}],
	"message": [],
	"result": {}
}`)
			}),
			response: nil,
			error:    "Cloudflare API Error code=9001 message=Over Nine Thousand! ray_id=0123456789abcdef-ABC",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewTLSServer(tt.handler)
			defer ts.Close()

			client := New([]byte("v1.0-FFFF-FFFF"),
				WithClient(ts.Client()),
				Must(WithEndpoint(ts.URL)),
			)
			resp, err := client.Sign(context.Background(), &SignRequest{
				Hostnames: []string{"example.com"},
				Validity:  3600,
				Type:      "MD4",
				CSR:       "Lorem ipsum dolor sit amet, consectetur adipiscing elit.",
			})

			if diff := cmp.Diff(resp, tt.response); diff != "" {
				t.Fatalf("diff: (-want +got)\n%s", diff)
			}

			if tt.error != "" {
				if diff := cmp.Diff(err.Error(), tt.error); diff != "" {
					t.Fatalf("diff: (-want +got)\n%s", diff)
				}
			}
		})
	}

}

func Must(opt Options, err error) Options {
	if err != nil {
		panic("option constructo returned error " + err.Error())
	}

	return opt
}
