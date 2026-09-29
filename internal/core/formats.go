package core

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"strings"
	"unicode/utf8"
)

// Expanded names preserve namespaces without treating sender names as paths.
func xmlName(n xml.Name) string {
	if n.Space != "" {
		return "{" + n.Space + "}" + n.Local
	}
	return n.Local
}

func boundedPayload(data map[string]any) (map[string]any, error) {
	raw, err := json.Marshal(data)
	if err != nil || len(raw) > 240*1024 {
		return nil, errors.New("normalized payload too large")
	}
	return data, nil
}

func parseXML(raw []byte) (map[string]any, error) {
	if len(raw) > 262144 {
		return nil, errors.New("XML body too large")
	}
	decoder := xml.NewDecoder(bytes.NewReader(raw))
	// Strict mode, nil Entity and nil CharsetReader: no custom entities, external
	// resolver or legacy charset conversion. Reject directives before processing.
	var stack []map[string]any
	var root map[string]any
	nodes := 0
	for count := 0; ; count++ {
		if count > 2048 {
			return nil, errors.New("too many XML tokens")
		}
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		switch value := token.(type) {
		case xml.Directive:
			return nil, errors.New("XML directives forbidden")
		case xml.ProcInst:
			if value.Target != "xml" || root != nil {
				return nil, errors.New("XML processing instructions forbidden")
			}
		case xml.StartElement:
			nodes++
			if len(stack) >= 16 || nodes > 512 || len(value.Attr) > 32 {
				return nil, errors.New("XML structure limit exceeded")
			}
			attrs := map[string]any{}
			for _, attr := range value.Attr {
				key := xmlName(attr.Name)
				if _, exists := attrs[key]; exists {
					return nil, errors.New("duplicate XML attribute")
				}
				attrs[key] = attr.Value
			}
			node := map[string]any{"name": xmlName(value.Name), "attributes": attrs, "text": "", "children": []any{}}
			if len(stack) == 0 {
				if root != nil {
					return nil, errors.New("multiple XML roots")
				}
				root = node
			} else {
				parent := stack[len(stack)-1]
				parent["children"] = append(parent["children"].([]any), node)
			}
			stack = append(stack, node)
		case xml.EndElement:
			stack = stack[:len(stack)-1]
		case xml.CharData:
			if len(stack) == 0 {
				if strings.TrimSpace(string(value)) != "" {
					return nil, errors.New("text outside XML root")
				}
				continue
			}
			node := stack[len(stack)-1]
			node["text"] = node["text"].(string) + string(value)
		}
	}
	if root == nil || len(stack) != 0 {
		return nil, errors.New("incomplete XML document")
	}
	return boundedPayload(map[string]any{"xml": root})
}

func parseMultipart(raw []byte, boundary string) (map[string]any, error) {
	if len(raw) > 262144 || len(boundary) == 0 || len(boundary) > 70 {
		return nil, errors.New("invalid multipart size or boundary")
	}
	reader := multipart.NewReader(bytes.NewReader(raw), boundary)
	fields, files := map[string]any{}, map[string]any{}
	seen := map[string]bool{}
	for count := 0; ; count++ {
		part, err := reader.NextRawPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if count >= 16 || len(part.Header) > 8 {
			part.Close()
			return nil, errors.New("too many multipart parts or headers")
		}
		for _, values := range part.Header {
			if len(values) != 1 {
				part.Close()
				return nil, errors.New("duplicate multipart header")
			}
		}
		disposition, params, err := mime.ParseMediaType(part.Header.Get("Content-Disposition"))
		name := params["name"]
		if err != nil || disposition != "form-data" || name == "" || len(name) > 128 || !utf8.ValidString(name) || seen[name] {
			part.Close()
			return nil, errors.New("invalid or duplicate multipart name")
		}
		seen[name] = true
		if part.Header.Get("Content-Transfer-Encoding") != "" {
			part.Close()
			return nil, errors.New("multipart transfer encoding forbidden")
		}
		filename, isFile := params["filename"]
		limit := 16384
		if isFile {
			limit = 65536
		}
		body, err := io.ReadAll(io.LimitReader(part, int64(limit+1)))
		part.Close()
		if err != nil || len(body) > limit {
			return nil, errors.New("multipart part too large")
		}
		if isFile {
			// Metadata only: never create a file or interpret the sender's path.
			if filename == "" || filename == "." || filename == ".." || len(filename) > 255 || strings.ContainsAny(filename, "/\\\x00\r\n") || !utf8.ValidString(filename) {
				return nil, errors.New("invalid multipart filename")
			}
			media := part.Header.Get("Content-Type")
			if media == "" {
				media = "application/octet-stream"
			}
			if _, _, err = mime.ParseMediaType(media); err != nil {
				return nil, errors.New("invalid part content type")
			}
			files[name] = map[string]any{"filename": filename, "content_type": media, "size": len(body), "base64": base64.StdEncoding.EncodeToString(body)}
		} else {
			if !utf8.Valid(body) {
				return nil, errors.New("multipart text must be UTF-8")
			}
			fields[name] = string(body)
		}
	}
	if len(seen) == 0 {
		return nil, errors.New("empty multipart payload")
	}
	return boundedPayload(map[string]any{"fields": fields, "files": files})
}
