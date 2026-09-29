package core

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"
)

func TestXMLNormalizationAndLimits(t *testing.T) {
	data, err := parsePayload("xml", "application/example+xml", []byte(`<?xml version="1.0"?><event xmlns="urn:example" status="failed">a&lt;b<item>one</item><item>two</item></event>`))
	if err != nil {
		t.Fatal(err)
	}
	root := data["xml"].(map[string]any)
	if root["name"] != "{urn:example}event" || root["text"] != "a<b" || len(root["children"].([]any)) != 2 {
		t.Fatalf("lost XML data: %#v", root)
	}
	value, ok := field(data, "xml.children.1.text")
	if !ok || value != "two" {
		t.Fatal("XML child not accessible to rules", value)
	}
	for _, path := range []string{"xml.children.-1", "xml.children.2", "xml.children.+1", "xml.children.01"} {
		if _, ok := field(data, path); ok {
			t.Fatal("invalid array index accepted", path)
		}
	}
	cases := []string{
		`<!DOCTYPE e [<!ENTITY x SYSTEM "file:///etc/passwd">]><e>&x;</e>`,
		`<!DOCTYPE e [<!ENTITY x "boom">]><e>&x;</e>`,
		`<e>&unknown;</e>`, `<e>`, `<a/><b/>`, `outside<a/>`, `<a/>outside`,
		`<?execute command?><a/>`, `<a x="1" x="2"/>`,
		strings.Repeat("<a>", 17) + strings.Repeat("</a>", 17),
		"<root>" + strings.Repeat("<a/>", 512) + "</root>",
		"<root>" + strings.Repeat("<!--c-->", 2050) + "</root>",
		"<root>" + strings.Repeat("a", 262144) + "</root>",
	}
	for i, raw := range cases {
		if _, err := parseXML([]byte(raw)); err == nil {
			t.Errorf("accepted XML attack %d", i)
		}
	}
	if _, err = parsePayload("xml", "text/plain", []byte("<a/>")); err == nil {
		t.Fatal("accepted mismatched type")
	}
}

type multipartFixture struct {
	name, filename, body string
	file                 bool
}

func makeMultipart(t *testing.T, parts []multipartFixture) ([]byte, string) {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for _, p := range parts {
		header := textproto.MIMEHeader{}
		disposition := fmt.Sprintf("form-data; name=%q", p.name)
		if p.file {
			disposition += fmt.Sprintf("; filename=%q", p.filename)
		}
		header.Set("Content-Disposition", disposition)
		part, err := writer.CreatePart(header)
		if err != nil {
			t.Fatal(err)
		}
		part.Write([]byte(p.body))
	}
	writer.Close()
	return body.Bytes(), writer.FormDataContentType()
}

func TestMultipartNormalizationAndLimits(t *testing.T) {
	raw, media := makeMultipart(t, []multipartFixture{{name: "status", body: "failed"}, {name: "attachment", filename: "payload.bin", body: "\x00\xff", file: true}})
	data, err := parsePayload("multipart", media, raw)
	if err != nil {
		t.Fatal(err)
	}
	if data["fields"].(map[string]any)["status"] != "failed" {
		t.Fatal(data)
	}
	file := data["files"].(map[string]any)["attachment"].(map[string]any)
	decoded, _ := base64.StdEncoding.DecodeString(file["base64"].(string))
	if !bytes.Equal(decoded, []byte{0, 255}) || file["size"] != 2 {
		t.Fatal(file)
	}
	cases := [][]multipartFixture{
		{{name: "x", body: strings.Repeat("x", 16385)}},
		{{name: "x", filename: "x", file: true, body: strings.Repeat("x", 65537)}},
		{{name: "x"}, {name: "x"}},
		{{name: "x", filename: "../escape", file: true}},
		{{name: "x", filename: `C:\escape`, file: true}},
		{{name: "x", body: "\xff"}},
	}
	for _, parts := range cases {
		raw, media := makeMultipart(t, parts)
		if _, err := parsePayload("multipart", media, raw); err == nil {
			t.Fatalf("accepted invalid multipart: %+v", parts[0].name)
		}
	}
	many := []multipartFixture{}
	for i := 0; i < 17; i++ {
		many = append(many, multipartFixture{name: fmt.Sprint(i)})
	}
	raw, media = makeMultipart(t, many)
	if _, err := parsePayload("multipart", media, raw); err == nil {
		t.Fatal("accepted excessive parts")
	}
	raw, media = makeMultipart(t, []multipartFixture{{name: "x", body: "value"}})
	if _, err := parsePayload("multipart", media, raw[:len(raw)-12]); err == nil {
		t.Fatal("accepted truncated multipart")
	}
	if _, err := parsePayload("multipart", "multipart/form-data", raw); err == nil {
		t.Fatal("accepted absent boundary")
	}
}

func TestStructuredFormatsPersistOnlyAuthenticatedInput(t *testing.T) {
	for _, format := range []string{"xml", "multipart"} {
		t.Run(format, func(t *testing.T) {
			e := testEngine(t)
			secret := "format-test-secret-long-enough"
			credential, _ := json.Marshal(map[string]string{"id": "incoming", "backend": "file", "value": secret})
			if _, err := e.putSecret(context.Background(), credential); err != nil {
				t.Fatal(err)
			}
			c := notificationConfig()
			c.Entries = []Entry{{ID: "input", Auth: "bearer", Secret: "incoming", Format: format, Enabled: true}}
			c.Flows[0].Source = "entry:input"
			activateTest(t, e, c)
			raw, media := []byte(`<event status="failed"/>`), "application/xml"
			if format == "multipart" {
				raw, media = makeMultipart(t, []multipartFixture{{name: "status", body: "failed"}})
			}
			handler := e.IngressHandler()
			for _, authorized := range []bool{false, true} {
				request := httptest.NewRequest("POST", "/hooks/input", bytes.NewReader(raw))
				request.Header.Set("Content-Type", media)
				if authorized {
					request.Header.Set("Authorization", "Bearer "+secret)
				}
				response := httptest.NewRecorder()
				handler.ServeHTTP(response, request)
				want := 401
				if authorized {
					want = 202
				}
				if response.Code != want {
					t.Fatal(response.Code, response.Body.String())
				}
			}
			var count int
			if err := e.db.QueryRow("SELECT count(*) FROM events").Scan(&count); err != nil || count != 1 {
				t.Fatal(count, err)
			}
		})
	}
}
