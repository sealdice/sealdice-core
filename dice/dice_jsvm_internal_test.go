package dice

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Masterminds/semver/v3"
)

func TestIsJSAPIVersionCompatible(t *testing.T) {
	compatibleVersions := []*semver.Version{
		semver.MustParse("1.6.1-dev"),
		semver.MustParse("1.6.0"),
		semver.MustParse("1.5.0"),
	}
	tests := []struct {
		name           string
		currentVersion string
		constraint     string
		want           bool
	}{
		{name: "prerelease satisfies older stable minimum", currentVersion: "1.6.1-dev", constraint: ">=1.6.0", want: true},
		{name: "prerelease does not satisfy its stable minimum", currentVersion: "1.6.1-dev", constraint: ">=1.6.1", want: false},
		{name: "prerelease satisfies its stable maximum", currentVersion: "1.6.1-dev", constraint: "<=1.6.1", want: true},
		{name: "prerelease exceeds older stable maximum", currentVersion: "1.6.1-dev", constraint: "<=1.6.0", want: false},
		{name: "prerelease satisfies bounded range", currentVersion: "1.6.1-dev", constraint: ">=1.6.0, <1.6.1", want: true},
		{name: "prerelease satisfies caret range", currentVersion: "1.6.1-dev", constraint: "^1.6.0", want: true},
		{name: "strict equality does not use compatibility list", currentVersion: "1.6.1-dev", constraint: "=1.6.0", want: false},
		{name: "prerelease satisfies exact prerelease", currentVersion: "1.6.1-dev", constraint: "=1.6.1-dev", want: true},
		{name: "plain version uses compatibility list", currentVersion: "1.6.1-dev", constraint: "1.6.0", want: true},
		{name: "unknown plain version is rejected", currentVersion: "1.6.1-dev", constraint: "1.6.2", want: false},
		{name: "stable version behavior is unchanged", currentVersion: "1.6.1", constraint: ">=1.6.1", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			constraint, err := semver.NewConstraint(tt.constraint)
			if err != nil {
				t.Fatalf("NewConstraint(%q) error = %v", tt.constraint, err)
			}

			got := isJSAPIVersionCompatible(
				constraint,
				tt.constraint,
				semver.MustParse(tt.currentVersion),
				compatibleVersions,
			)
			if got != tt.want {
				t.Errorf("isJSAPIVersionCompatible(%q, %q) = %v, want %v", tt.currentVersion, tt.constraint, got, tt.want)
			}
		})
	}
}

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
	for i := range a {
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
