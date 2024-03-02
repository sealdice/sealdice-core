package dice

import (
	"fmt"
	"strings"
	"testing"
)

func TestSortJsScripts(t *testing.T) {
	type args struct {
		jsScripts []*JsScriptInfo
	}
	tests := []struct {
		name    string
		args    args
		want    []*JsScriptInfo
		wantErr bool
	}{
		{
			name: "test only builtins",
			args: args{
				jsScripts: []*JsScriptInfo{
					{
						Name:    "A",
						Author:  "sealdice",
						Builtin: true,
					},
					{
						Name:    "B",
						Author:  "sealdice",
						Builtin: true,
						Depends: []JsScriptDepends{
							{
								Author: "sealdice",
								Name:   "C",
							},
						},
					},
					{
						Name:    "C",
						Author:  "sealdice",
						Builtin: true,
						Depends: []JsScriptDepends{
							{
								Author: "sealdice",
								Name:   "A",
							},
						},
					},
					{
						Name:    "D",
						Author:  "sealdice",
						Builtin: true,
						Depends: []JsScriptDepends{
							{
								Author: "sealdice",
								Name:   "B",
							},
							{
								Author: "sealdice",
								Name:   "C",
							},
						},
					},
				},
			},
			want: []*JsScriptInfo{
				{
					Name:    "A",
					Author:  "sealdice",
					Builtin: true,
				},
				{
					Name:    "C",
					Author:  "sealdice",
					Builtin: true,
				},
				{
					Name:    "B",
					Author:  "sealdice",
					Builtin: true,
				},
				{
					Name:    "D",
					Author:  "sealdice",
					Builtin: true,
				},
			},
			wantErr: false,
		},
		{
			name: "test only not builtins",
			args: args{
				jsScripts: []*JsScriptInfo{
					{
						Name:    "A",
						Author:  "JustAnotherID",
						Builtin: false,
					},
					{
						Name:    "B",
						Author:  "JustAnotherID",
						Builtin: false,
						Depends: []JsScriptDepends{
							{
								Author: "JustAnotherID",
								Name:   "C",
							},
						},
					},
					{
						Name:    "C",
						Author:  "JustAnotherID",
						Builtin: false,
						Depends: []JsScriptDepends{
							{
								Author: "JustAnotherID",
								Name:   "A",
							},
						},
					},
				}},
			want: []*JsScriptInfo{
				{
					Name:    "A",
					Author:  "JustAnotherID",
					Builtin: false,
				},
				{
					Name:    "C",
					Author:  "JustAnotherID",
					Builtin: false,
				},
				{
					Name:    "B",
					Author:  "JustAnotherID",
					Builtin: false,
				},
			},
			wantErr: false,
		},
		{
			name: "test both",
			args: args{
				jsScripts: []*JsScriptInfo{
					{
						Name:    "A",
						Author:  "sealdice",
						Builtin: true,
					},
					{
						Name:    "B",
						Author:  "JustAnotherID",
						Builtin: false,
					},
					{
						Name:    "C",
						Author:  "JustAnotherID",
						Builtin: false,
						Depends: []JsScriptDepends{
							{
								Author: "sealdice",
								Name:   "A",
							},
						},
					},
					{
						Name:    "D",
						Author:  "sealdice",
						Builtin: true,
						Depends: []JsScriptDepends{
							{
								Author: "sealdice",
								Name:   "A",
							},
						},
					},
					{
						Name:    "E",
						Author:  "sealdice",
						Builtin: true,
						Depends: []JsScriptDepends{
							{
								Author: "sealdice",
								Name:   "A",
							},
							{
								Author: "sealdice",
								Name:   "D",
							},
						},
					},
				}},
			want: []*JsScriptInfo{
				{
					Name:    "A",
					Author:  "sealdice",
					Builtin: true,
				},
				{
					Name:    "D",
					Author:  "sealdice",
					Builtin: true,
				},
				{
					Name:    "E",
					Author:  "sealdice",
					Builtin: true,
				},
				{
					Name:    "B",
					Author:  "JustAnotherID",
					Builtin: false,
				},
				{
					Name:    "C",
					Author:  "JustAnotherID",
					Builtin: false,
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, errMap := sortJsScripts(tt.args.jsScripts)
			if len(errMap) != 0 && !tt.wantErr {
				t.Errorf("sortJsScripts() errMap = %v", errMap)
				return
			}
			if !sameScriptInfos(got, tt.want) {
				t.Errorf("sortJsScripts() got = %v, want %v", showScriptInfos(got), showScriptInfos(tt.want))
			}
		})
	}
}

func showScriptInfos(jsScripts []*JsScriptInfo) string {
	var result []string
	for _, jsScript := range jsScripts {
		result = append(result, fmt.Sprintf("%s::%s", jsScript.Author, jsScript.Name))
	}
	return "[" + strings.Join(result, ", ") + "]"
}

func sameScriptInfos(a []*JsScriptInfo, b []*JsScriptInfo) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if !sameScriptInfo(a[i], b[i]) {
			return false
		}
	}
	return true
}

func sameScriptInfo(a *JsScriptInfo, b *JsScriptInfo) bool {
	if a.Name != b.Name {
		return false
	}
	if a.Author != b.Author {
		return false
	}
	if a.Builtin != b.Builtin {
		return false
	}
	return true
}
