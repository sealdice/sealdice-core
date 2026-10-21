package js_test

import (
	"math"
	"reflect"
	"testing"

	. "sealdice-core/api/v2/js"
	"sealdice-core/dice"
)

func TestDataListResolvesExtensionByScriptName(t *testing.T) {
	dm := &dice.DiceManager{}
	d := &dice.Dice{Parent: dm, JsExtRegistry: new(dice.SyncMap[string, *dice.ExtInfo])}
	dm.Dice = d
	d.BaseConfig.DataDir = t.TempDir()
	script := &dice.JsScriptInfo{Name: "AI骰娘 4", Filename: "./data/default/scripts/chat.js"}
	d.JsScriptList = []*dice.JsScriptInfo{script}
	// 注册名与脚本 @name 不同，列表传入的是 @name。
	ext := &dice.ExtInfo{Name: "chat", Source: script}
	d.RegisterExtension(ext)
	d.JsExtRegistry.Store(ext.Name, ext)
	t.Cleanup(func() { _ = ext.StorageClose() })
	if err := ext.StorageSet("key", "value"); err != nil {
		t.Fatal(err)
	}

	service := NewService(dm)
	got, err := service.DataList(t.Context(), &struct {
		Name string `path:"name"`
		DataListQuery
	}{Name: script.Name, DataListQuery: DataListQuery{Page: 1, PageSize: 20}})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Body.Item.Keys) != 1 || got.Body.Item.Keys[0].Key != "key" {
		t.Fatalf("got %+v, want one key", got.Body.Item)
	}
}

func TestDataListPagination(t *testing.T) {
	dm := &dice.DiceManager{}
	d := &dice.Dice{Parent: dm, JsExtRegistry: new(dice.SyncMap[string, *dice.ExtInfo])}
	dm.Dice = d
	d.BaseConfig.DataDir = t.TempDir()
	ext := &dice.ExtInfo{Name: "plugin"}
	d.RegisterExtension(ext)
	d.JsExtRegistry.Store(ext.Name, ext)
	t.Cleanup(func() { _ = ext.StorageClose() })
	for key, value := range map[string]string{
		"a": "plain", "b": `{"value":1}`, "c": "null", "keep-1": "1", "keep-2": "text",
	} {
		if err := ext.StorageSet(key, value); err != nil {
			t.Fatal(err)
		}
	}
	service := NewService(dm)
	all := []DataKV{
		{Key: "a", Value: "plain"},
		{Key: "b", Value: `{"value":1}`, IsJSON: true},
		{Key: "c", Value: "null", IsJSON: true},
		{Key: "keep-1", Value: "1", IsJSON: true},
		{Key: "keep-2", Value: "text"},
	}
	for _, tc := range []struct {
		name           string
		page, pageSize int
		keyword        string
		want           DataListResp
	}{
		{"defaults", 0, 0, "", DataListResp{Keys: all, Total: 5, Page: 1, PageSize: 20}},
		{"first", 1, 2, "", DataListResp{Keys: all[:2], Total: 5, Page: 1, PageSize: 2}},
		{"middle", 2, 2, "", DataListResp{Keys: all[2:4], Total: 5, Page: 2, PageSize: 2}},
		{"last", 3, 2, "", DataListResp{Keys: all[4:], Total: 5, Page: 3, PageSize: 2}},
		{"past end", 4, 2, "", DataListResp{Keys: []DataKV{}, Total: 5, Page: 4, PageSize: 2}},
		{"glob", 2, 1, "keep-?", DataListResp{Keys: all[4:], Total: 2, Page: 2, PageSize: 1}},
		{"no matches", 1, 10, "missing*", DataListResp{Keys: []DataKV{}, Page: 1, PageSize: 10}},
		{"huge page", math.MaxInt, 2, "", DataListResp{Keys: []DataKV{}, Total: 5, Page: math.MaxInt, PageSize: 2}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := service.DataList(t.Context(), &struct {
				Name string `path:"name"`
				DataListQuery
			}{Name: ext.Name, DataListQuery: DataListQuery{Page: tc.page, PageSize: tc.pageSize, Keyword: tc.keyword}})
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got.Body.Item, tc.want) {
				t.Fatalf("got %+v, want %+v", got.Body.Item, tc.want)
			}
		})
	}
}
