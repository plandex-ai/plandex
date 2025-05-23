package schema

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"strings"

	shared "plandex-shared"

	gojsonreference "github.com/xeipuuv/gojsonreference"
	"github.com/xeipuuv/gojsonschema"
)

const scheme = "embed://"

type SchemaPath string

const (
	SchemaPathInputConfig SchemaPath = "json-schemas/models-input.schema.json"
	SchemaPathPlanConfig  SchemaPath = "json-schemas/plan-config.schema.json"
)

//go:embed json-schemas/*.schema.json json-schemas/definitions/*.schema.json
var schemaFS embed.FS

type embeddedSchemaLoader struct {
	source string
	fs     embed.FS
}

func ValidateModelsInputJSON(jsonData []byte) (shared.ModelsInput, error) {
	return validateJSON[shared.ModelsInput](jsonData, SchemaPathInputConfig)
}

func validateJSON[T any](jsonData []byte, schemaPath SchemaPath) (T, error) {
	var zero T

	schemaLoader := newEmbeddedSchemaLoader(schemaPath)
	documentLoader := gojsonschema.NewBytesLoader(jsonData)

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return zero, err
	}
	if !result.Valid() {
		var msgs []string
		for _, d := range result.Errors() {
			msgs = append(msgs, "• "+d.String())
		}
		return zero, errors.New(strings.Join(msgs, "\n"))
	}

	var v T
	if err := json.Unmarshal(jsonData, &v); err != nil {
		return zero, fmt.Errorf("unmarshal error: %w", err)
	}
	return v, nil
}

func newEmbeddedSchemaLoader(source SchemaPath) *embeddedSchemaLoader {
	return &embeddedSchemaLoader{
		source: string(source),
		fs:     schemaFS,
	}
}

func (l *embeddedSchemaLoader) JsonSource() interface{} {
	return l.source
}

func (l *embeddedSchemaLoader) LoadJSON() (interface{}, error) {
	// remove both "./" and the scheme prefix
	source := strings.TrimPrefix(l.source, "./")
	source = strings.TrimPrefix(source, scheme)

	// convert absolute Plandex URLs to our embed path
	const webPrefix = "https://plandex.ai/schemas/"
	if strings.HasPrefix(source, webPrefix) {
		source = path.Join("json-schemas", strings.TrimPrefix(source, webPrefix))
	}

	if strings.HasSuffix(source, ".schema.json") {
		schemaPath := source
		// for schemas with relative path references, add the json-schemas prefix
		if !strings.HasPrefix(schemaPath, "json-schemas/") {
			schemaPath = path.Join("json-schemas", schemaPath)
		}
		data, err := l.fs.ReadFile(schemaPath)
		if err != nil {
			return nil, fmt.Errorf("error reading embedded schema %s: %v", schemaPath, err)
		}
		var v interface{}
		dec := json.NewDecoder(bytes.NewReader(data))
		dec.UseNumber() // use numbers instead of floats
		if err := dec.Decode(&v); err != nil {
			return nil, fmt.Errorf("error parsing embedded schema %s: %v", source, err)
		}
		return v, nil
	}

	var v interface{}
	dec := json.NewDecoder(bytes.NewReader([]byte(source)))
	dec.UseNumber() // use numbers instead of floats
	if err := dec.Decode(&v); err != nil {
		return nil, fmt.Errorf("error parsing schema JSON: %v", err)
	}
	return v, nil
}

func (l *embeddedSchemaLoader) JsonReference() (gojsonreference.JsonReference, error) {
	return gojsonreference.NewJsonReference(scheme + l.source)
}

type embeddedLoaderFactory struct{}

func (embeddedLoaderFactory) New(source string) gojsonschema.JSONLoader {
	source = strings.TrimPrefix(source, scheme)
	return newEmbeddedSchemaLoader(SchemaPath(source))
}

func (l *embeddedSchemaLoader) LoaderFactory() gojsonschema.JSONLoaderFactory {
	return embeddedLoaderFactory{}
}
