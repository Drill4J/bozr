package main

import (
	"errors"
	"fmt"
	"mime"
	"strings"

	"github.com/xeipuuv/gojsonschema"
)

// ResponseExpectation is an interface to any validation
// that needs to be performed on response.
type ResponseExpectation interface {
	check(resp Response) error
}

// StatusCodeExpectation validates response HTTP code.
type StatusCodeExpectation struct {
	statusCode int
}

func (e StatusCodeExpectation) check(resp Response) error {
	if resp.http.StatusCode != e.statusCode {
		msg := fmt.Sprintf("Unexpected Status Code. Expected: %d, Actual: %d\n", e.statusCode, resp.http.StatusCode)
		return errors.New(msg)
	}
	return nil
}

// BodySchemaExpectation validates response body against schema.
// Content-Type header is used to identify either json schema or xsd is applied.
type BodySchemaExpectation struct {
	schemaURI string
}

func (e BodySchemaExpectation) check(resp Response) error {
	contentType, _, _ := mime.ParseMediaType(resp.http.Header.Get("content-type"))

	if contentType == "application/json" {
		return e.checkJSON(resp)
	}

	return fmt.Errorf("Unsupported content type: %s", contentType)
}

var jsonSchemaCache = map[string]interface{}{}

func (e BodySchemaExpectation) checkJSON(resp Response) error {
	if jsonSchemaCache[e.schemaURI] == nil {
		debug.Printf("Loading schema %s", e.schemaURI)

		schemaLoader := gojsonschema.NewReferenceLoader(e.schemaURI)
		schema, err := schemaLoader.LoadJSON()

		if err != nil {
			return fmt.Errorf("failed to load schema: %s", err)
		}

		jsonSchemaCache[e.schemaURI] = schema
	}

	schemaLoader := gojsonschema.NewGoLoader(jsonSchemaCache[e.schemaURI])
	documentLoader := gojsonschema.NewStringLoader(string(resp.body))

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return fmt.Errorf("failed to load schema file: %s", err)
	}

	if !result.Valid() {
		msg := "Unexpected Body Schema:\n"
		for _, desc := range result.Errors() {
			msg = fmt.Sprintf(msg+"%s\n", desc)
		}
		return errors.New(msg)
	}

	return nil
}

// BodyExpectation validates values under a certain path in a body.
// Applies to json and xml.
type BodyExpectation struct {
	pathExpectations map[string]interface{}
}

func (e BodyExpectation) check(resp Response) error {

	errs := []string{}
	for path, expectedValue := range e.pathExpectations {

		matcherFunc := ChooseMatcher(path)
		path := strings.Replace(path, "~", "", -1)

		splitPath := strings.Split(path, ".")

		// TODO need rememberedMap here:  expectedValue = putRememberedVars(expectedValue, rememberedMap)
		m, err := resp.parseBody()
		if err != nil {
			str := "Can't parse response body to Map." // TODO specific message for functions
			str += " " + err.Error()
			errs = append(errs, str)
		}

		ok, err := matcherFunc(m, expectedValue, splitPath...)
		if !ok {
			str := fmt.Sprintf("Expected value [%v] on path [%s] does not match.", expectedValue, path)
			errs = append(errs, str)
		}
		if err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		var msg string
		for _, err := range errs {
			msg += err + "\n"
		}
		return errors.New(msg)
	}

	return nil
}

// HeaderExpectation validates one header in a response.
type HeaderExpectation struct {
	Name        string
	Value       string
	ValueParser func(string) string
}

func (e HeaderExpectation) check(resp Response) error {
	value := resp.http.Header.Get(e.Name)
	if e.ValueParser != nil {
		value = e.ValueParser(value)
	}

	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("Missing header. Expected \"%s: %s\"\n", e.Name, e.Value)
	}
	if e.Value != "" && e.Value != value {
		msg := "Unexpected header. Expected \"%s: %s\". Actual \"%s: %s\"\n"
		return fmt.Errorf(msg, e.Name, e.Value, e.Name, value)
	}
	return nil
}

// ContentTypeExpectation validates media type returned in the Content-Type header.
// Encoding information is excluded from matching value.
// E.g. "application/json;charset=utf-8" header transformed to "application/json" media type.
type ContentTypeExpectation struct {
	Value string
}

func (e ContentTypeExpectation) check(resp Response) error {
	parser := func(value string) string {
		contentType, _, _ := mime.ParseMediaType(value)
		return contentType
	}

	headerCheck := HeaderExpectation{"content-type", e.Value, parser}
	return headerCheck.check(resp)
}
