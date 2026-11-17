package dynamicform_test

import (
	"reflect"
	"testing"

	dynamicform "sealdice-core/api/v2/imconnection/dynamic_form"
)

func TestEmbeddedProtocolSchemas(t *testing.T) {
	t.Chdir(t.TempDir())
	all := dynamicform.GetAll()
	if len(all) == 0 {
		t.Fatal("protocol schemas must be available without source files")
	}
	for protocol, items := range all {
		t.Run(protocol, func(t *testing.T) {
			if len(items) == 0 {
				t.Fatal("empty protocol schema")
			}
			// Exercise every shipped field and its default through the same entry
			// point as the HTTP create/edit handlers.
			cfg := map[string]interface{}{}
			for _, item := range items {
				if item.FieldName == "" {
					t.Fatal("schema field has no name")
				}
				if item.Default != "" {
					continue
				}
				switch item.InputType {
				case dynamicform.InputTypeText:
					cfg[item.FieldName] = "12345"
				case dynamicform.InputTypeNum:
					cfg[item.FieldName] = 5500
				case dynamicform.InputTypeBool:
					cfg[item.FieldName] = false
				case dynamicform.InputTypeSelect:
					if len(item.SubOption) == 0 {
						t.Fatal("select field has no options")
					}
					cfg[item.FieldName] = item.SubOption[0].Value
				default:
					t.Fatalf("unused field type %d", item.InputType)
				}
			}
			if _, err := dynamicform.BuildParamsByConfig(items, cfg); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestBuildParamsByConfig(t *testing.T) {
	minimum, maximum := 1, 65535
	items := []*dynamicform.FormConfigItem{
		{FieldName: "token", InputType: dynamicform.InputTypeText, IsRequired: dynamicform.RequiredTrue},
		{FieldName: "port", InputType: dynamicform.InputTypeNum, Default: "8099", Minimum: &minimum, Maximum: &maximum},
		{FieldName: "enabled", InputType: dynamicform.InputTypeBool, Default: "true"},
		{FieldName: "mode", InputType: dynamicform.InputTypeSelect, Default: "yogurt",
			SubOption: []*dynamicform.Option{{Label: "Yogurt", Value: "yogurt"}}},
		{FieldName: "account", InputType: dynamicform.InputTypeText, CheckType: dynamicform.CheckTypeNum},
	}
	tests := []struct {
		name string
		cfg  map[string]interface{}
		want map[string]interface{}
	}{
		{"defaults", map[string]interface{}{"token": "abc", "ignored": "value"}, map[string]interface{}{
			"token": "abc", "port": 8099, "enabled": true, "mode": "yogurt",
		}},
		{"native values", map[string]interface{}{"token": "abc", "port": float64(5500), "enabled": false, "account": "00123"}, map[string]interface{}{
			"token": "abc", "port": 5500, "enabled": false, "mode": "yogurt", "account": "00123",
		}},
		{"string values", map[string]interface{}{"token": "abc", "port": "5500", "enabled": "0"}, map[string]interface{}{
			"token": "abc", "port": 5500, "enabled": false, "mode": "yogurt",
		}},
		{"required missing", map[string]interface{}{}, nil},
		{"required empty", map[string]interface{}{"token": ""}, nil},
		{"invalid option", map[string]interface{}{"token": "abc", "mode": "unknown"}, nil},
		{"invalid default", map[string]interface{}{"token": "abc", "port": "not-a-number"}, nil},
		{"zero port", map[string]interface{}{"token": "abc", "port": 0}, nil},
		{"oversize port", map[string]interface{}{"token": "abc", "port": 65536}, nil},
		{"fractional integer", map[string]interface{}{"token": "abc", "port": 5500.5}, nil},
		{"fractional boolean", map[string]interface{}{"token": "abc", "enabled": 0.5}, nil},
		{"invalid boolean", map[string]interface{}{"token": "abc", "enabled": 2}, nil},
		{"invalid numeric text", map[string]interface{}{"token": "abc", "account": "x"}, nil},
		{"compound text", map[string]interface{}{"token": []string{"abc"}}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := dynamicform.BuildParamsByConfig(items, tt.cfg)
			if tt.want == nil {
				if err == nil {
					t.Fatalf("invalid input accepted: %#v", got)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %#v, want %#v", got, tt.want)
			}
		})
	}
}
