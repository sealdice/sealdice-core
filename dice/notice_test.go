//nolint:testpackage
package dice

import (
	"reflect"
	"testing"
)

func normalizeForCompare(target NoticeTarget) NoticeTarget {
	if target.NoticeTypes == nil {
		target.NoticeTypes = []NoticeType{}
	}
	return target
}

func TestParseNoticeTarget(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want NoticeTarget
	}{
		{
			name: "legacy target allows all",
			raw:  "QQ:12345",
			want: NoticeTarget{ID: "QQ:12345"},
		},
		{
			name: "disabled target",
			raw:  "QQ-Group:67890:disable",
			want: NoticeTarget{ID: "QQ-Group:67890", Disabled: true},
		},
		{
			name: "filtered target uses stable order",
			raw:  "QQ:12345:only=ban,group,ban",
			want: NoticeTarget{
				ID:                  "QQ:12345",
				NoticeTypes:         []NoticeType{NoticeTypeGroup, NoticeTypeBan},
				HasNoticeTypeFilter: true,
			},
		},
		{
			name: "metadata can be reversed",
			raw:  "QQ:12345:only=send:disable",
			want: NoticeTarget{
				ID:                  "QQ:12345",
				Disabled:            true,
				NoticeTypes:         []NoticeType{NoticeTypeSend},
				HasNoticeTypeFilter: true,
			},
		},
		{
			name: "embedded colon is retained",
			raw:  "OpenQQ-Group:100-abc-OpenQQ:100-user:disable:only=group",
			want: NoticeTarget{
				ID:                  "OpenQQ-Group:100-abc-OpenQQ:100-user",
				Disabled:            true,
				NoticeTypes:         []NoticeType{NoticeTypeGroup},
				HasNoticeTypeFilter: true,
			},
		},
		{
			name: "empty filter allows no categories",
			raw:  "Mail:user@example.com:only=",
			want: NoticeTarget{
				ID:                  "Mail:user@example.com",
				NoticeTypes:         []NoticeType{},
				HasNoticeTypeFilter: true,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := normalizeForCompare(ParseNoticeTarget(test.raw))
			want := normalizeForCompare(test.want)
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("ParseNoticeTarget(%q) = %#v, want %#v", test.raw, got, want)
			}
		})
	}
}

func TestNoticeTargetAllows(t *testing.T) {
	legacy := ParseNoticeTarget("QQ:1")
	if !legacy.Allows(NoticeTypeGroup) || !legacy.Allows(NoticeTypeSend) {
		t.Fatal("legacy target should allow every category")
	}

	filtered := ParseNoticeTarget("QQ:1:only=group,ban")
	if !filtered.Allows(NoticeTypeGroup) || !filtered.Allows(NoticeTypeBan) {
		t.Fatal("filtered target should allow selected categories")
	}
	if filtered.Allows(NoticeTypeSend) {
		t.Fatal("filtered target should reject unselected categories")
	}

	disabled := ParseNoticeTarget("QQ:1:disable:only=group")
	if disabled.Allows(NoticeTypeGroup) {
		t.Fatal("disabled target should reject every category")
	}
}

func TestNoticeTargetString(t *testing.T) {
	tests := map[string]string{
		"QQ:1":                "QQ:1",
		"QQ:1:disable":        "QQ:1:disable",
		"QQ:1:only=ban,group": "QQ:1:only=group,ban",
		"QQ:1:only=system,send,inactive,censor,ban,invite,group": "QQ:1",
		"QQ:1:disable:only=": "QQ:1:disable:only=",
	}

	for raw, want := range tests {
		if got := ParseNoticeTarget(raw).String(); got != want {
			t.Errorf("ParseNoticeTarget(%q).String() = %q, want %q", raw, got, want)
		}
	}
}

func TestNoticeTargetAddressing(t *testing.T) {
	tests := []struct {
		raw      string
		platform string
		group    bool
	}{
		{raw: "QQ:1:disable", platform: "QQ", group: false},
		{raw: "QQ-Group:1:only=group", platform: "QQ", group: true},
		{raw: "DISCORD-CH-Channel:1", platform: "DISCORD", group: true},
		{raw: "Mail:user@example.com", platform: "Mail", group: false},
	}

	for _, test := range tests {
		target := ParseNoticeTarget(test.raw)
		platform, ok := target.Platform()
		if !ok || platform != test.platform {
			t.Errorf("%q platform = %q, %t; want %q, true", test.raw, platform, ok, test.platform)
		}
		if got := target.IsGroup(); got != test.group {
			t.Errorf("%q IsGroup() = %t, want %t", test.raw, got, test.group)
		}
	}
}
