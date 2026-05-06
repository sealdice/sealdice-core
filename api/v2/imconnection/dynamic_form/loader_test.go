package dynamicform

import (
	"testing"
)

func mustLoad(t *testing.T) {
	t.Helper()
	if forms == nil {
		if err := LoadFromFile("forms.json"); err != nil {
			t.Fatalf("LoadFromFile failed: %v", err)
		}
	}
}

func idByField(items []*FormConfigItem, field string) uint64 {
	for _, it := range items {
		if it.FieldName == field {
			return it.ID
		}
	}
	return 0
}

func TestLoadAndGetFormsDiscord(t *testing.T) {
	mustLoad(t)
	items := GetFormConfig("discord")
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
	items := GetFormConfig("discord")
	sub := SubmitFormItems{
		&SubmitFormItem{ID: idByField(items, "token"), Data: "TKN"},
		&SubmitFormItem{ID: idByField(items, "proxyURL"), Data: "http://127.0.0.1:7890"},
	}
	params, err := BuildParamsBySubmit(items, sub)
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
	items := GetFormConfig("satori")
	sub := SubmitFormItems{
		&SubmitFormItem{ID: idByField(items, "platform"), Data: "QQ"},
		&SubmitFormItem{ID: idByField(items, "host"), Data: "localhost"},
		&SubmitFormItem{ID: idByField(items, "port"), Data: "3100"},
		&SubmitFormItem{ID: idByField(items, "token"), Data: "abc"},
	}
	params, err := BuildParamsBySubmit(items, sub)
	if err != nil {
		t.Fatalf("BuildParamsBySubmit error: %v", err)
	}
	if params["port"] != 3100 {
		t.Fatalf("port expected 3100 got %v", params["port"])
	}
}

func TestRequiredMissing(t *testing.T) {
	mustLoad(t)
	items := GetFormConfig("discord")
	sub := SubmitFormItems{
		// omit token (required)
	}
	_, err := BuildParamsBySubmit(items, sub)
	if err == nil {
		t.Fatalf("expected error for missing required")
	}
}

func TestKeysAndGetAll(t *testing.T) {
	mustLoad(t)
	ks := Keys()
	if len(ks) == 0 {
		t.Fatalf("Keys empty")
	}
	all := GetAll()
	if len(all) == 0 {
		t.Fatalf("GetAll empty")
	}
	// consistency: each key in Keys must exist in GetAll and GetFormConfig
	for _, k := range ks {
		if _, ok := all[k]; !ok {
			t.Fatalf("key %s missing in GetAll", k)
		}
		if len(GetFormConfig(k)) == 0 {
			t.Fatalf("GetFormConfig(%s) empty", k)
		}
	}
}

func TestSelectTypesSupport(t *testing.T) {
	forms := []*FormConfigItem{
		{ID: 1, FieldName: "p1", InputType: InputTypeSin, IsRequired: RequiredTrue},
		{ID: 2, FieldName: "p2", InputType: InputTypeMul, IsRequired: RequiredFalse},
	}
	sub := SubmitFormItems{
		&SubmitFormItem{ID: 1, Data: "5"},
		&SubmitFormItem{ID: 2, Data: "[1,2,3]"},
	}
	params, err := BuildParamsBySubmit(forms, sub)
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
	forms := []*FormConfigItem{
		{ID: 1, FieldName: "t1", InputType: InputTypeText, IsRequired: RequiredFalse, Default: "DEF"},
		{ID: 2, FieldName: "n1", InputType: InputTypeNum, IsRequired: RequiredFalse, Default: "42"},
		{ID: 3, FieldName: "d1", InputType: InputTypeDate, IsRequired: RequiredFalse, Default: "1700000000"},
		{ID: 4, FieldName: "r1", InputType: InputTypeDateRange, IsRequired: RequiredFalse, DefaultRange: &RangeValue{Start: 1, End: 2}},
		{ID: 5, FieldName: "b1", InputType: InputTypeBool, IsRequired: RequiredFalse, Default: "true"},
	}
	sub := SubmitFormItems{
		// empty submit, all should use defaults
	}
	params, err := BuildParamsBySubmit(forms, sub)
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
	if r, ok := params["r1"].(*RangeValue); !ok || r.Start != 1 || r.End != 2 {
		t.Fatalf("r1 default mismatch")
	}
	if params["b1"] != true {
		t.Fatalf("b1 default mismatch")
	}
}

func TestSelectWithOptions(t *testing.T) {
	forms := []*FormConfigItem{
		{ID: 1, FieldName: "color", InputType: InputTypeSelect, IsRequired: RequiredTrue, SubOption: []*Option{
			{Label: "Red", Value: "red"},
			{Label: "Blue", Value: "blue"},
		}},
	}
	sub := SubmitFormItems{
		&SubmitFormItem{ID: 1, Data: "red"},
	}
	params, err := BuildParamsBySubmit(forms, sub)
	if err != nil {
		t.Fatalf("BuildParamsBySubmit error: %v", err)
	}
	if params["color"] != "red" {
		t.Fatalf("select value mismatch")
	}
	// invalid value
	subBad := SubmitFormItems{
		&SubmitFormItem{ID: 1, Data: "green"},
	}
	_, err = BuildParamsBySubmit(forms, subBad)
	if err == nil {
		t.Fatalf("expected error for invalid option")
	}
}

func TestRadioCheckboxWithOptions(t *testing.T) {
	forms := []*FormConfigItem{
		{ID: 1, FieldName: "radio", InputType: InputTypeSin, IsRequired: RequiredTrue, SubOption: []*Option{
			{Label: "A", Value: "a"},
			{Label: "B", Value: "b"},
		}},
		{ID: 2, FieldName: "checks", InputType: InputTypeMul, IsRequired: RequiredFalse, SubOption: []*Option{
			{Label: "A", Value: "a"},
			{Label: "B", Value: "b"},
			{Label: "C", Value: "c"},
		}},
	}
	sub := SubmitFormItems{
		&SubmitFormItem{ID: 1, Data: "a"},
		&SubmitFormItem{ID: 2, Data: "[\"a\",\"c\"]"},
	}
	params, err := BuildParamsBySubmit(forms, sub)
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
	forms := []*FormConfigItem{
		{ID: 1, FieldName: "dyn", InputType: InputTypeSelect, IsRequired: RequiredTrue, OptionsURL: "mock://colors"},
	}
	RegisterOptionsProvider(func(item *FormConfigItem) ([]*Option, error) {
		if item.OptionsURL == "mock://colors" {
			return []*Option{
				{Label: "Red", Value: "red"},
				{Label: "Blue", Value: "blue"},
			}, nil
		}
		return nil, nil
	})
	sub := SubmitFormItems{
		&SubmitFormItem{ID: 1, Data: "blue"},
	}
	params, err := BuildParamsBySubmit(forms, sub)
	if err != nil {
		t.Fatalf("BuildParamsBySubmit error: %v", err)
	}
	if params["dyn"] != "blue" {
		t.Fatalf("dynamic select mismatch")
	}
}
