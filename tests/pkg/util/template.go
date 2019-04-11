package util

import (
	"bytes"
	"fmt"
	"html/template"
)

// ParseTemplate validates and parses passed as argument template
func ParseTemplate(strtmpl string, obj interface{}) ([]byte, error) {
	var buf bytes.Buffer
	tmpl, err := template.New("template").Parse(strtmpl)
	if err != nil {
		return nil, fmt.Errorf("error when parsing template: %v", err)
	}
	err = tmpl.Execute(&buf, obj)
	if err != nil {
		return nil, fmt.Errorf("error when executing template: %v", err)
	}
	return buf.Bytes(), nil
}
