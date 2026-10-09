package wechat

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
)

// ParseXML parses XML bytes into a map
func ParseXML(data []byte) (map[string]string, error) {
	result := make(map[string]string)
	decoder := xml.NewDecoder(bytes.NewReader(data))

	var currentKey string
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to parse XML: %w", err)
		}

		switch t := token.(type) {
		case xml.StartElement:
			currentKey = t.Name.Local
		case xml.CharData:
			if currentKey != "" && currentKey != "xml" {
				value := string(t)
				if value != "" {
					result[currentKey] = value
				}
			}
		case xml.EndElement:
			currentKey = ""
		}
	}

	return result, nil
}

// BuildXML builds XML from a map
func BuildXML(params map[string]string) ([]byte, error) {
	var buffer bytes.Buffer
	buffer.WriteString("<xml>")

	for key, value := range params {
		buffer.WriteString("<")
		buffer.WriteString(key)
		buffer.WriteString(">")
		buffer.WriteString(xmlEscape(value))
		buffer.WriteString("</")
		buffer.WriteString(key)
		buffer.WriteString(">")
	}

	buffer.WriteString("</xml>")
	return buffer.Bytes(), nil
}

// BuildSuccessXML builds a success response XML for webhook acknowledgment
func BuildSuccessXML() []byte {
	return []byte("<xml><return_code><![CDATA[SUCCESS]]></return_code><return_msg><![CDATA[OK]]></return_msg></xml>")
}

// BuildFailXML builds a failure response XML for webhook acknowledgment
func BuildFailXML(msg string) []byte {
	return []byte(fmt.Sprintf("<xml><return_code><![CDATA[FAIL]]></return_code><return_msg><![CDATA[%s]]></return_msg></xml>", xmlEscape(msg)))
}

// xmlEscape escapes special XML characters
func xmlEscape(s string) string {
	// Basic XML escaping
	s = replaceAll(s, "&", "&amp;")
	s = replaceAll(s, "<", "&lt;")
	s = replaceAll(s, ">", "&gt;")
	s = replaceAll(s, "\"", "&quot;")
	s = replaceAll(s, "'", "&apos;")
	return s
}

// replaceAll is a helper for string replacement
func replaceAll(s, old, new string) string {
	result := ""
	for {
		index := indexOf(s, old)
		if index == -1 {
			result += s
			break
		}
		result += s[:index] + new
		s = s[index+len(old):]
	}
	return result
}

// indexOf finds the first occurrence of substring in string
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
