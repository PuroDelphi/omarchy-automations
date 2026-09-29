package core

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDocumentedPayloadFormats(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "payload-formats.json"))
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Format      string `json:"format"`
		ContentType string `json:"content_type"`
		Body        string `json:"body"`
		Field       string `json:"field"`
		Expected    any    `json:"expected"`
	}
	if err := decode(raw, &cases); err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for _, tc := range cases {
		counts[tc.Format]++
		data, err := parsePayload(tc.Format, tc.ContentType, []byte(tc.Body))
		if err != nil {
			t.Fatalf("%s: %v", tc.Format, err)
		}
		got, ok := field(data, tc.Field)
		if !ok || !reflect.DeepEqual(got, tc.Expected) {
			t.Fatalf("%s %s: got %#v, want %#v", tc.Format, tc.Field, got, tc.Expected)
		}
	}
	for _, format := range []string{"json", "form", "raw", "xml", "multipart"} {
		if counts[format] != 2 {
			t.Fatalf("%s needs two examples", format)
		}
	}
}
