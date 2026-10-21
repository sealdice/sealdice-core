//nolint:testpackage
package dice

import (
	"strings"
	"testing"
)

func TestTeamListUsesStoredNamesCacheAndIDFallback(t *testing.T) {
	dm := &DiceManager{}
	d := &Dice{Parent: dm}
	ctx := &MsgContext{Dice: d}
	group := &GroupInfo{
		Players:      new(SyncMap[string, *GroupPlayerInfo]),
		PlayerGroups: new(SyncMap[string, []string]),
	}
	group.PlayerGroups.Store("B组", []string{})
	group.PlayerGroups.Store("A组", []string{"QQ:1", "QQ:2", "QQ:3"})
	group.Players.Store("QQ:1", &GroupPlayerInfo{Name: "  海\n豹\t名  ", UserID: "QQ:1"})
	group.Players.Store("QQ:2", nil)
	group.Players.Store("QQ:3", nil)
	dm.UserNameCache.Store("QQ:1", &GroupNameCacheItem{Name: "不应使用的缓存名"})
	dm.UserNameCache.Store("QQ:2", &GroupNameCacheItem{Name: "缓存名"})

	got := teamListText(ctx, group)
	want := "当前群共有2个团队：\n" +
		"A组（3人）：海 豹 名（QQ:1）、缓存名（QQ:2）、QQ:3\n" +
		"B组（0人）：无成员"
	if got != want {
		t.Fatalf("team list reply = %q, want %q", got, want)
	}
	if strings.Contains(got, "[CQ:at") {
		t.Fatalf("team list unexpectedly mentions members: %q", got)
	}
}

func TestTeamListReportsNoTeams(t *testing.T) {
	group := &GroupInfo{PlayerGroups: new(SyncMap[string, []string])}
	if got := teamListText(nil, group); got != "当前群尚未创建团队" {
		t.Fatalf("team list reply = %q, want empty-team hint", got)
	}
}

func TestTeamNamedListCanStillBeCalledExplicitly(t *testing.T) {
	if !teamIsListRequest(&CmdArgs{Args: []string{"list"}}) {
		t.Fatal(".team list was not recognized as a list request")
	}
	if teamIsListRequest(&CmdArgs{Args: []string{"list", "call"}}) {
		t.Fatal(".team list call must remain an explicit call for the team named list")
	}
}

func TestTeamListPagesOnlyWhenTextIsLong(t *testing.T) {
	shortText := "当前群共有1个团队：\n调查组（1人）：Alice（QQ:1）"
	shortPages := teamListPages(shortText)
	if len(shortPages) != 1 || shortPages[0] != shortText {
		t.Fatalf("short pages = %#v, want unchanged text", shortPages)
	}

	longText := strings.Repeat("x", teamListPageMaxLength+100)
	longPages := teamListPages(longText)
	if len(longPages) < 2 {
		t.Fatalf("long page count = %d, want at least two", len(longPages))
	}
	for index, page := range longPages {
		wantPrefix := "团队列表（"
		if !strings.HasPrefix(page, wantPrefix) {
			t.Fatalf("page %d = %q, want pagination prefix", index+1, page)
		}
		if len(page) >= 15000 {
			t.Fatalf("page %d length = %d, want below reply hard limit", index+1, len(page))
		}
	}
}
