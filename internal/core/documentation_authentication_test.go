package core

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDocumentedAuthenticationVectors(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "authentication-vectors.json"))
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Provider string            `json:"provider"`
		Secret   string            `json:"secret"`
		Now      int64             `json:"now"`
		Body     string            `json:"body"`
		Headers  map[string]string `json:"headers"`
	}
	if err := decode(raw, &cases); err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for _, tc := range cases {
		counts[tc.Provider]++
		r := httptest.NewRequest("POST", "http://localhost/hooks/example", strings.NewReader(tc.Body))
		for k, v := range tc.Headers {
			r.Header.Set(k, v)
		}
		entry := Entry{Auth: tc.Provider}
		if _, err := authenticate(entry, r, []byte(tc.Body), tc.Secret, time.Unix(tc.Now, 0)); err != nil {
			t.Fatalf("%s valid vector: %v", tc.Provider, err)
		}
		if tc.Provider == "bearer" {
			r.Header.Set("Authorization", "Bearer wrong-token")
		}
		if _, err := authenticate(entry, r, []byte(tc.Body+" "), tc.Secret, time.Unix(tc.Now, 0)); err == nil {
			t.Fatalf("%s accepted invalid authentication", tc.Provider)
		}
	}
	for _, provider := range []string{"hmac", "slack", "github", "bearer"} {
		if counts[provider] != 2 {
			t.Fatalf("%s needs two vectors", provider)
		}
	}
}
