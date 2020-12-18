package cfapi

import (
	"encoding/json"
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
