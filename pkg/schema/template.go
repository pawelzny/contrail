package schema

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/flosch/pongo2"
	"github.com/pkg/errors"

	"github.com/Juniper/contrail/pkg/fileutil"
)

const (
	dictGetJSONSchemaByStringKeyFilter = "dict_get_JSONSchema_by_string_key"
)

// TemplateConfig contains configuration for template.
type TemplateConfig struct {
	TemplateType string `yaml:"type"`
	TemplatePath string `yaml:"template_path"`
	OutputPath   string `yaml:"output_path"`
}

// ApplyTemplates writes files with content generated from templates.
func ApplyTemplates(api *API, config []*TemplateConfig) error {
	if err := registerCustomFilters(); err != nil {
		return err
	}

	for _, tc := range config {
		if err := tc.resolveOutputPath(); err != nil {
			return err
		}
		if !tc.isOutdated(api) {
			continue
		}
		err := tc.apply(api)
		if err != nil {
			return err
		}
	}
	return nil
}

func (tc *TemplateConfig) resolveOutputPath() error {
	if tc.OutputPath != "" {
		return nil
	}

	tc.OutputPath = generatedFilePath(tc.TemplatePath)
	return nil
}

func (tc *TemplateConfig) isOutdated(api *API) bool {
	sourceInfo, err := os.Stat(tc.TemplatePath)
	if err != nil {
		return true
	}
	targetInfo, err := os.Stat(tc.OutputPath)
	if err != nil {
		return true
	}
	return sourceInfo.ModTime().After(targetInfo.ModTime()) || api.Timestamp.After(targetInfo.ModTime())
}

// nolint: gocyclo
func (tc *TemplateConfig) apply(api *API) error {
	tpl, err := tc.load()
	if err != nil {
		return err
	}
	if err = ensureDir(tc.OutputPath); err != nil {
		return err
	}
	if tc.TemplateType == "all" {
		output, err := tpl.Execute(pongo2.Context{
			"schemas": api.Schemas,
			"types":   api.Types,
		})
		if err != nil {
			return err
		}

		if err = writeGeneratedFile(tc.OutputPath, output, tc.TemplatePath); err != nil {
			return err
		}
	} else if tc.TemplateType == "alltype" {
		var schemas []*Schema
		for typeName, typeJSONSchema := range api.Types {
			typeJSONSchema.GoName = typeName
			schemas = append(schemas, &Schema{
				JSONSchema:     typeJSONSchema,
				Children:       map[string]*BackReference{},
				BackReferences: map[string]*BackReference{},
			})
		}
		for _, schema := range api.Schemas {
			if schema.Type == AbstractType || schema.ID == "" {
				continue
			}
			schemas = append(schemas, schema)
		}
		output, err := tpl.Execute(pongo2.Context{
			"schemas": schemas,
		})
		if err != nil {
			return err
		}

		if err = writeGeneratedFile(tc.OutputPath, output, tc.TemplatePath); err != nil {
			return err
		}
	}
	return nil
}

func (tc *TemplateConfig) load() (*pongo2.Template, error) {
	templateCode, err := fileutil.GetContent(tc.TemplatePath)
	if err != nil {
		return nil, err
	}
	return pongo2.FromString(string(templateCode))
}

// LoadTemplates loads template configurations from given path.
func LoadTemplates(path string) ([]*TemplateConfig, error) {
	var config []*TemplateConfig
	err := fileutil.LoadFile(path, &config)
	return config, err
}

func generatedFilePath(tmplFilePath string) string {
	dir, file := filepath.Split(tmplFilePath)
	return filepath.Join(dir, generatedFileName(file))
}

func generatedFileName(tmplFile string) string {
	return "gen_" + strings.TrimSuffix(tmplFile, ".tmpl")
}

func ensureDir(path string) error {
	return os.MkdirAll(filepath.Dir(path), os.ModePerm)
}

func registerCustomFilters() error {
	/* When called like this: {{ dict_value|dict_get_JSONSchema_by_string_key:key_var }}
	then: dict_value is here as `in' variable and key_var is here as `param'
	This is needed to obtain value from map with a key in variable (not as a hardcoded string)
	*/
	if !pongo2.FilterExists(dictGetJSONSchemaByStringKeyFilter) {
		if err := pongo2.RegisterFilter(
			dictGetJSONSchemaByStringKeyFilter,
			func(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
				m, _ := in.Interface().(map[string]*JSONSchema) //nolint: errcheck
				return pongo2.AsValue(m[param.String()]), nil
			},
		); err != nil {
			return err
		}
	}

	return nil
}

func writeGeneratedFile(path, data, template string) error {
	if err := ioutil.WriteFile(path, []byte(generationPrefix(path, template)+data), 0644); err != nil {
		return errors.Wrapf(err, "failed to write generate file to path %q", path)
	}
	return nil
}

func generationPrefix(path, template string) string {
	prefix := "# "
	if strings.HasSuffix(path, ".go") || strings.HasSuffix(path, ".proto") {
		prefix = "// "
	} else if strings.HasSuffix(path, ".sql") {
		prefix = "-- "
	}
	return prefix + fmt.Sprintf("Code generated by contrailschema tool from template %s; DO NOT EDIT.\n\n", template)
}
