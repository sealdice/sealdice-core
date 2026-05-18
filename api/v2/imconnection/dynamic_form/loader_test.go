package dynamicform_test

import (
	"testing"

	dynamicform "sealdice-core/api/v2/imconnection/dynamic_form"
)

func mustLoad(t *testing.T) {
	t.Helper()
	if dynamicform.GetAll() == nil {
		if err := dynamicform.LoadFromFile("forms.json"); err != nil {
			t.Fatalf("LoadFromFile failed: %v", err)
		}
	}
}

func idByField(items []*dynamicform.FormConfigItem, field string) uint64 {
	for _, it := range items {
		if it.FieldName == field {
			return it.ID
		}
	}
	return 0
}

func TestLoadAndGetFormsDiscord(t *testing.T) {
	mustLoad(t)
	items := dynamicform.GetFormConfig("discord")
	if len(items) == 0 {
		t.Fatalf("discord form empty")
	}
	tokenID := idByField(items, "token")
	if tokenID == 0 {
		t.Fatalf("discord token id not found")
	}
}

func TestBuildParamsBySubmitDiscord(t *testing.T) {
	mustLoad(t)
	items := dynamicform.GetFormConfig("discord")
	sub := dynamicform.SubmitFormItems{
		&dynamicform.SubmitFormItem{ID: idByField(items, "token"), Data: "TKN"},
		&dynamicform.SubmitFormItem{ID: idByField(items, "proxyURL"), Data: "http://127.0.0.1:7890"},
	}
	params, err := dynamicform.BuildParamsBySubmit(items, sub)
	if err != nil {
		t.Fatalf("BuildParamsBySubmit error: %v", err)
	}
	if params["token"] != "TKN" {
		t.Fatalf("token mismatch")
	}
	if params["proxyURL"] != "http://127.0.0.1:7890" {
		t.Fatalf("proxyURL mismatch")
	}
}

func TestBuildParamsNumericSatoriPort(t *testing.T) {
	mustLoad(t)
	items := dynamicform.GetFormConfig("satori")
	sub := dynamicform.SubmitFormItems{
		&dynamicform.SubmitFormItem{ID: idByField(items, "platform"), Data: "QQ"},
		&dynamicform.SubmitFormItem{ID: idByField(items, "host"), Data: "localhost"},
		&dynamicform.SubmitFormItem{ID: idByField(items, "port"), Data: "3100"},
		&dynamicform.SubmitFormItem{ID: idByField(items, "token"), Data: "abc"},
	}
	params, err := dynamicform.BuildParamsBySubmit(items, sub)
	if err != nil {
		t.Fatalf("BuildParamsBySubmit error: %v", err)
	}
	if params["port"] != 3100 {
		t.Fatalf("port expected 3100 got %v", params["port"])
	}
}

func TestRequiredMissing(t *testing.T) {
	mustLoad(t)
	items := dynamicform.GetFormConfig("discord")
	sub := dynamicform.SubmitFormItems{
		// omit token (required)
	}
	_, err := dynamicform.BuildParamsBySubmit(items, sub)
	if err == nil {
		t.Fatalf("expected error for missing required")
	}
}

func TestKeysAndGetAll(t *testing.T) {
	mustLoad(t)
	ks := dynamicform.Keys()
	if len(ks) == 0 {
		t.Fatalf("Keys empty")
	}
	all := dynamicform.GetAll()
	if len(all) == 0 {
		t.Fatalf("GetAll empty")
	}
	// consistency: each key in Keys must exist in GetAll and GetFormConfig
	for _, k := range ks {
		if _, ok := all[k]; !ok {
			t.Fatalf("key %s missing in GetAll", k)
		}
		if len(dynamicform.GetFormConfig(k)) == 0 {
			t.Fatalf("GetFormConfig(%s) empty", k)
		}
	}
}

func TestSelectTypesSupport(t *testing.T) {
	forms := []*dynamicform.FormConfigItem{
		{ID: 1, FieldName: "p1", InputType: dynamicform.InputTypeSin, IsRequired: dynamicform.RequiredTrue},
		{ID: 2, FieldName: "p2", InputType: dynamicform.InputTypeMul, IsRequired: dynamicform.RequiredFalse},
	}
	sub := dynamicform.SubmitFormItems{
		&dynamicform.SubmitFormItem{ID: 1, Data: "5"},
		&dynamicform.SubmitFormItem{ID: 2, Data: "[1,2,3]"},
	}
	params, err := dynamicform.BuildParamsBySubmit(forms, sub)
	if err != nil {
		t.Fatalf("BuildParamsBySubmit error: %v", err)
	}
	if params["p1"] != 5 {
		t.Fatalf("p1 expected 5 got %v", params["p1"])
	}
	if arr, ok := params["p2"].([]int); !ok || len(arr) != 3 || arr[0] != 1 || arr[2] != 3 {
		t.Fatalf("p2 mismatch")
	}
}

func TestDefaultsApplied(t *testing.T) {
	forms := []*dynamicform.FormConfigItem{
		{ID: 1, FieldName: "t1", InputType: dynamicform.InputTypeText, IsRequired: dynamicform.RequiredFalse, Default: "DEF"},
		{ID: 2, FieldName: "n1", InputType: dynamicform.InputTypeNum, IsRequired: dynamicform.RequiredFalse, Default: "42"},
		{ID: 3, FieldName: "d1", InputType: dynamicform.InputTypeDate, IsRequired: dynamicform.RequiredFalse, Default: "1700000000"},
		{ID: 4, FieldName: "r1", InputType: dynamicform.InputTypeDateRange, IsRequired: dynamicform.RequiredFalse, DefaultRange: &dynamicform.RangeValue{Start: 1, End: 2}},
		{ID: 5, FieldName: "b1", InputType: dynamicform.InputTypeBool, IsRequired: dynamicform.RequiredFalse, Default: "true"},
	}
	sub := dynamicform.SubmitFormItems{
		// empty submit, all should use defaults
	}
	params, err := dynamicform.BuildParamsBySubmit(forms, sub)
	if err != nil {
		t.Fatalf("BuildParamsBySubmit error: %v", err)
	}
	if params["t1"] != "DEF" {
		t.Fatalf("t1 default mismatch")
	}
	if params["n1"] != 42 {
		t.Fatalf("n1 default mismatch")
	}
	if params["d1"] != int64(1700000000) {
		t.Fatalf("d1 default mismatch got %v", params["d1"])
	}
	if r, ok := params["r1"].(*dynamicform.RangeValue); !ok || r.Start != 1 || r.End != 2 {
		t.Fatalf("r1 default mismatch")
	}
	if params["b1"] != true {
		t.Fatalf("b1 default mismatch")
	}
}

func TestSelectWithOptions(t *testing.T) {
	forms := []*dynamicform.FormConfigItem{
		{ID: 1, FieldName: "color", InputType: dynamicform.InputTypeSelect, IsRequired: dynamicform.RequiredTrue, SubOption: []*dynamicform.Option{
			{Label: "Red", Value: "red"},
			{Label: "Blue", Value: "blue"},
		}},
	}
	sub := dynamicform.SubmitFormItems{
		&dynamicform.SubmitFormItem{ID: 1, Data: "red"},
	}
	params, err := dynamicform.BuildParamsBySubmit(forms, sub)
	if err != nil {
		t.Fatalf("BuildParamsBySubmit error: %v", err)
	}
	if params["color"] != "red" {
		t.Fatalf("select value mismatch")
	}
	// invalid value
	subBad := dynamicform.SubmitFormItems{
		&dynamicform.SubmitFormItem{ID: 1, Data: "green"},
	}
	_, err = dynamicform.BuildParamsBySubmit(forms, subBad)
	if err == nil {
		t.Fatalf("expected error for invalid option")
	}
}

func TestRadioCheckboxWithOptions(t *testing.T) {
	forms := []*dynamicform.FormConfigItem{
		{ID: 1, FieldName: "radio", InputType: dynamicform.InputTypeSin, IsRequired: dynamicform.RequiredTrue, SubOption: []*dynamicform.Option{
			{Label: "A", Value: "a"},
			{Label: "B", Value: "b"},
		}},
		{ID: 2, FieldName: "checks", InputType: dynamicform.InputTypeMul, IsRequired: dynamicform.RequiredFalse, SubOption: []*dynamicform.Option{
			{Label: "A", Value: "a"},
			{Label: "B", Value: "b"},
			{Label: "C", Value: "c"},
		}},
	}
	sub := dynamicform.SubmitFormItems{
		&dynamicform.SubmitFormItem{ID: 1, Data: "a"},
		&dynamicform.SubmitFormItem{ID: 2, Data: "[\"a\",\"c\"]"},
	}
	params, err := dynamicform.BuildParamsBySubmit(forms, sub)
	if err != nil {
		t.Fatalf("BuildParamsBySubmit error: %v", err)
	}
	if params["radio"] != "a" {
		t.Fatalf("radio mismatch")
	}
	values, ok := params["checks"].([]string)
	if !ok || len(values) != 2 || values[0] != "a" || values[1] != "c" {
		t.Fatalf("checkbox mismatch")
	}
}

func TestDynamicOptionsProvider(t *testing.T) {
	forms := []*dynamicform.FormConfigItem{
		{ID: 1, FieldName: "dyn", InputType: dynamicform.InputTypeSelect, IsRequired: dynamicform.RequiredTrue, OptionsURL: "mock://colors"},
	}
	dynamicform.RegisterOptionsProvider(func(item *dynamicform.FormConfigItem) ([]*dynamicform.Option, error) {
		if item.OptionsURL == "mock://colors" {
			return []*dynamicform.Option{
				{Label: "Red", Value: "red"},
				{Label: "Blue", Value: "blue"},
			}, nil
		}
		return nil, nil
	})
	sub := dynamicform.SubmitFormItems{
		&dynamicform.SubmitFormItem{ID: 1, Data: "blue"},
	}
	params, err := dynamicform.BuildParamsBySubmit(forms, sub)
	if err != nil {
		t.Fatalf("BuildParamsBySubmit error: %v", err)
	}
	if params["dyn"] != "blue" {
		t.Fatalf("dynamic select mismatch")
	}
}

func TestBuildParamsByConfigUsesFieldNamesAndNativeValues(t *testing.T) {
	forms := []*dynamicform.FormConfigItem{
		{ID: 1, FieldName: "token", InputType: dynamicform.InputTypeText, IsRequired: dynamicform.RequiredTrue},
		{ID: 2, FieldName: "port", InputType: dynamicform.InputTypeNum, IsRequired: dynamicform.RequiredTrue},
		{ID: 3, FieldName: "enabled", InputType: dynamicform.InputTypeBool, IsRequired: dynamicform.RequiredFalse, Default: "true"},
		{ID: 4, FieldName: "mode", InputType: dynamicform.InputTypeSelect, IsRequired: dynamicform.RequiredTrue, SubOption: []*dynamicform.Option{
			{Label: "Yogurt", Value: "yogurt"},
			{Label: "LagrangeV2", Value: "lagrangeV2"},
		}},
	}

	params, err := dynamicform.BuildParamsByConfig(forms, map[string]interface{}{
		"token": "abc",
		"port":  5500,
		"mode":  "yogurt",
	})
	if err != nil {
		t.Fatalf("BuildParamsByConfig error: %v", err)
	}
	if params["token"] != "abc" {
		t.Fatalf("token mismatch")
	}
	if params["port"] != 5500 {
		t.Fatalf("port expected 5500 got %v", params["port"])
	}
	if params["enabled"] != true {
		t.Fatalf("enabled default expected true got %v", params["enabled"])
	}
	if params["mode"] != "yogurt" {
		t.Fatalf("mode mismatch")
	}
}

func TestFormConfigItemSensitiveMetadataLoads(t *testing.T) {
	mustLoad(t)
	items := dynamicform.GetFormConfig("discord")
	for _, it := range items {
		if it.FieldName == "token" {
			if !it.Sensitive {
				t.Fatalf("discord token should be marked sensitive")
			}
			return
		}
	}
	t.Fatalf("discord token field not found")
}
