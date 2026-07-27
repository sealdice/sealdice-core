//nolint:testpackage
package dice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/sealdice/botgo/dto"
	ds "github.com/sealdice/dicescript"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm"

	sealdiceLogger "sealdice-core/logger"
	"sealdice-core/model"
)

type officialQQTransportFunc func(ctx context.Context, method, url string, body interface{}) ([]byte, error)

func (f officialQQTransportFunc) Transport(ctx context.Context, method, url string, body interface{}) ([]byte, error) {
	return f(ctx, method, url, body)
}

func TestServerOfficialQQSkipsRunningSessionBeforeStateChange(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conn := &PlatformAdapterOfficialQQ{Ctx: ctx}
	ep := &EndPointInfo{EndPointInfoBase: EndPointInfoBase{
		State:  StateConnected,
		Enable: true,
	}}

	serverOfficialQQ(&Dice{}, ep, conn)

	if conn.DiceServing {
		t.Fatal("running session was marked as a new server")
	}
	if ep.State != StateConnected || !ep.Enable {
		t.Fatalf("endpoint state changed for running session: state=%d enable=%t", ep.State, ep.Enable)
	}
}

func TestExtractOfficialQQBotUIN(t *testing.T) {
	t.Parallel()

	uin, err := extractOfficialQQBotUIN("https://qun.qq.com/qqweb/qunpro/share?_wv=3&robot_uin=1&robot_appid=123456789")
	if err != nil {
		t.Fatalf("extractOfficialQQBotUIN returned error: %v", err)
	}
	if uin != "1" {
		t.Fatalf("uin = %q, want %q", uin, "1")
	}

	for _, link := range []string{
		"https://example.com/share",
		"https://example.com/share?robot_uin=not-a-number",
		"https://example.com/share?robot_uin=0",
	} {
		if _, err := extractOfficialQQBotUIN(link); err == nil {
			t.Fatalf("extractOfficialQQBotUIN(%q) unexpectedly succeeded", link)
		}
	}
}

func TestExtractOfficialQQBotUINFromResponse(t *testing.T) {
	t.Parallel()

	for _, body := range []string{
		`{"url":"https://qun.qq.com/qqweb/qunpro/share?robot_uin=1&robot_appid=123456789"}`,
		`{"url_link":"https://qun.qq.com/qqweb/qunpro/share?robot_uin=1&robot_appid=123456789"}`,
	} {
		uin, err := extractOfficialQQBotUINFromResponse([]byte(body))
		if err != nil {
			t.Fatalf("extract response returned error: %v", err)
		}
		if uin != "1" {
			t.Fatalf("uin = %q, want %q", uin, "1")
		}
	}

	if _, err := extractOfficialQQBotUINFromResponse([]byte(`{}`)); err == nil {
		t.Fatal("response without url unexpectedly succeeded")
	}

	// The documented url_link is authoritative when compatibility fields coexist.
	uin, err := extractOfficialQQBotUINFromResponse([]byte(`{
		"url":"https://example.com/short-link",
		"url_link":"https://qun.qq.com/qunpro/robot/qunshare?robot_uin=2"
	}`))
	if err != nil || uin != "2" {
		t.Fatalf("preferred url_link result = %q, %v", uin, err)
	}
}

func TestParseOfficialQQBotInfo(t *testing.T) {
	t.Parallel()

	botInfo, err := parseOfficialQQBotInfo([]byte(`{
		"id":"bot-open-id",
		"username":"test bot",
		"share_url":"https://qun.qq.com/qunpro/robot/qunshare?robot_uin=3&robot_appid=123456789"
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if botInfo.ID != "bot-open-id" || botInfo.Username != "test bot" {
		t.Fatalf("unexpected bot info: %#v", botInfo)
	}
	uin, err := extractOfficialQQBotUIN(botInfo.ShareURL)
	if err != nil || uin != "3" {
		t.Fatalf("share_url UIN = %q, %v", uin, err)
	}

	if _, err := parseOfficialQQBotInfo([]byte(`{"username":"missing id"}`)); err == nil {
		t.Fatal("bot info without id unexpectedly succeeded")
	}
}

func TestGetOfficialQQBotInfoUsesShareURL(t *testing.T) {
	t.Parallel()

	calls := 0
	api := officialQQTransportFunc(func(_ context.Context, method, requestURL string, _ interface{}) ([]byte, error) {
		calls++
		if method != http.MethodGet || !strings.HasSuffix(requestURL, "/users/@me") {
			t.Fatalf("unexpected request: %s %s", method, requestURL)
		}
		return []byte(`{
			"id":"bot-open-id",
			"username":"test bot",
			"share_url":"https://qun.qq.com/qunpro/robot/qunshare?robot_uin=3"
		}`), nil
	})

	botInfo, err := getOfficialQQBotInfo(context.Background(), api)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 || botInfo.UIN != "3" || botInfo.BotID != "bot-open-id" {
		t.Fatalf("calls = %d, bot info = %#v", calls, botInfo)
	}
}

func TestGetOfficialQQBotInfoGeneratedLinkFallback(t *testing.T) {
	t.Parallel()

	calls := 0
	api := officialQQTransportFunc(func(_ context.Context, method, requestURL string, _ interface{}) ([]byte, error) {
		calls++
		switch {
		case method == http.MethodGet && strings.HasSuffix(requestURL, "/users/@me"):
			return []byte(`{"id":"bot-open-id","username":"test bot"}`), nil
		case method == http.MethodPost && strings.HasSuffix(requestURL, "/v2/generate_url_link"):
			return []byte(`{"url_link":"https://qun.qq.com/qunpro/robot/qunshare?robot_uin=4"}`), nil
		default:
			t.Fatalf("unexpected request: %s %s", method, requestURL)
			return nil, nil
		}
	})

	botInfo, err := getOfficialQQBotInfo(context.Background(), api)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 || botInfo.UIN != "4" || botInfo.BotID != "bot-open-id" {
		t.Fatalf("calls = %d, bot info = %#v", calls, botInfo)
	}
}

func TestOfficialQQIDRoundTrip(t *testing.T) {
	t.Parallel()

	const (
		uin         = "1"
		groupOpenID = "group-open-id-with-hyphens"
		userOpenID  = "user-open-id-with-hyphens"
	)
	pa := &PlatformAdapterOfficialQQ{UIN: uin}

	groupID := formatDiceIDOfficialQQGroupOpenID(uin, groupOpenID)
	if groupID != "OpenQQ-Group:1-"+groupOpenID {
		t.Fatalf("group ID = %q", groupID)
	}
	if raw, kind := pa.mustExtractID(groupID); raw != groupOpenID || kind != OpenQQGroupOpenid {
		t.Fatalf("group parse = (%q, %d)", raw, kind)
	}

	memberID := formatDiceIDOfficialQQMemberOpenID(uin, groupOpenID, userOpenID)
	userID := formatDiceIDOfficialQQUserOpenID(uin, userOpenID)
	if memberID != userID || userID != "OpenQQ:1-"+userOpenID {
		t.Fatalf("member ID = %q, user ID = %q", memberID, userID)
	}
	if raw, kind := pa.mustExtractID(userID); raw != userOpenID || kind != OpenQQUserOpenid {
		t.Fatalf("user parse = (%q, %d)", raw, kind)
	}
	if raw, kind := pa.mustExtractID(formatDiceIDOfficialQQ(uin)); raw != uin || kind != OpenQQUser {
		t.Fatalf("bot parse = (%q, %d)", raw, kind)
	}

	for _, unsupported := range []string{
		"OpenQQ-Group:" + groupOpenID,
		"OpenQQ:" + userOpenID,
		"OpenQQ-Group:other-" + groupOpenID,
		"QQ:" + uin,
	} {
		if _, kind := pa.mustExtractID(unsupported); kind != OpenQQUnknown {
			t.Fatalf("unsupported ID %q parsed as kind %d", unsupported, kind)
		}
	}
}

func TestOfficialQQLegacyEmptyGroupIdentity(t *testing.T) {
	t.Parallel()

	const (
		appID        = "123456789"
		uin          = "1"
		memberOpenID = "member-open-id"
	)
	migration := &officialQQIdentityMigration{
		appID:  appID,
		uin:    uin,
		groups: map[string]string{},
		users:  map[string]string{},
	}
	oldGroupID := "OpenQQ-Group-T:" + appID + "-"
	oldMemberID := "OpenQQ-Member-T:" + appID + "--" + memberOpenID

	if !migration.addGroup(oldGroupID) {
		t.Fatal("legacy empty group was not collected")
	}
	if !migration.addMember(oldGroupID, oldMemberID) {
		t.Fatal("legacy empty-group member was not collected")
	}
	if got, want := migration.migrateGroupID(oldGroupID), "OpenQQ-Group:"+uin+"-legacy-empty-"+appID; got != want {
		t.Fatalf("group ID = %q, want %q", got, want)
	}
	if got, want := migration.migrateUserID(oldMemberID), "OpenQQ:"+uin+"-"+memberOpenID; got != want {
		t.Fatalf("member ID = %q, want %q", got, want)
	}
}

func TestOfficialQQAtFallbackUsesBotOpenID(t *testing.T) {
	t.Parallel()

	pa := &PlatformAdapterOfficialQQ{UIN: "1", botID: "bot-open-id"}
	msg := pa.groupMsgToStdMsg(nil, &dto.WSGroupATMessageData{Content: "hello", GroupOpenID: "group-open-id"})
	if msg.TmpUID != "OpenQQ:bot-open-id" {
		t.Fatalf("TmpUID = %q, want bot OpenID", msg.TmpUID)
	}
}

func TestOfficialQQGroupMessagesPreserveMemberRole(t *testing.T) {
	t.Parallel()

	pa := &PlatformAdapterOfficialQQ{UIN: "1"}
	author := &dto.User{MemberOpenID: "member-open-id", MemberRole: "admin"}

	atMsg := pa.groupMsgToStdMsg(nil, &dto.WSGroupATMessageData{GroupOpenID: "group-open-id", Author: author})
	normalMsg := pa.groupNormalMsgToStdMsg(nil, &dto.WSGroupMessageData{GroupOpenID: "group-open-id", Author: author})
	for name, msg := range map[string]*Message{"at": atMsg, "normal": normalMsg} {
		if msg.Sender.UserID != "OpenQQ:1-member-open-id" {
			t.Errorf("%s message user ID = %q, want UIN-based ID", name, msg.Sender.UserID)
		}
		if msg.Sender.GroupRole != "admin" {
			t.Errorf("%s message group role = %q, want admin", name, msg.Sender.GroupRole)
		}
	}
}

func TestMigratedCharacterNameLimit(t *testing.T) {
	t.Parallel()

	name := migratedCharacterName(strings.Repeat("角", 30), 2)
	if len(name) > 90 || !strings.HasSuffix(name, " (2)") || !utf8.ValidString(name) {
		t.Fatalf("migrated character name is invalid: %q (%d bytes)", name, len(name))
	}
}

func TestOfficialQQAdapterSerializationDoesNotExposeSecrets(t *testing.T) {
	t.Parallel()

	adapter := &PlatformAdapterOfficialQQ{
		AppID:     "123456789",
		AppSecret: "secret-value",
		Token:     "legacy-token-value",
		UIN:       "1",
	}
	rawJSON, err := json.Marshal(adapter)
	if err != nil {
		t.Fatal(err)
	}
	jsonText := string(rawJSON)
	if strings.Contains(jsonText, "secret-value") || strings.Contains(strings.ToLower(jsonText), "token") {
		t.Fatalf("JSON exposed credentials: %s", jsonText)
	}
	if !strings.Contains(jsonText, `"appID":"123456789"`) || !strings.Contains(jsonText, `"uin":"1"`) {
		t.Fatalf("JSON omitted public account identity: %s", jsonText)
	}

	conn := NewOfficialQQConnItem(adapter.AppID, adapter.AppSecret, adapter.UIN, false)
	conn.Adapter.(*PlatformAdapterOfficialQQ).Token = adapter.Token
	rawYAML, err := yaml.Marshal(conn)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(rawYAML), "token: legacy-token-value") {
		t.Fatalf("YAML did not preserve the deprecated token: %s", rawYAML)
	}
	if !strings.Contains(string(rawYAML), "uin: \"1\"") {
		t.Fatalf("YAML omitted UIN: %s", rawYAML)
	}
}

func TestReplaceOfficialQQCredentials(t *testing.T) {
	t.Parallel()

	existing := NewOfficialQQConnItem("old-app", "old-secret", "123", false)
	existing.UserID = "OpenQQ:123"
	existing.Nickname = "old bot"
	source := &PlatformAdapterOfficialQQ{
		AppID:     "new-app",
		AppSecret: "new-secret",
	}
	probe := &OfficialQQAccountProbeResult{
		UIN:      "123",
		BotID:    "new-bot-id",
		Nickname: "new bot",
	}

	adapter, err := replaceOfficialQQCredentials(existing, source, probe)
	if err != nil {
		t.Fatal(err)
	}
	if adapter != existing.Adapter {
		t.Fatal("replacement returned a different adapter")
	}
	if adapter.AppID != "new-app" || adapter.AppSecret != "new-secret" || adapter.UIN != "123" {
		t.Fatalf("credentials were not replaced: %+v", adapter)
	}
	if existing.UserID != "OpenQQ:123" || existing.Nickname != "new bot" {
		t.Fatalf("endpoint identity was not updated: userID=%s nickname=%s", existing.UserID, existing.Nickname)
	}
}

func TestReplaceOfficialQQCredentialsRejectsUINMismatch(t *testing.T) {
	t.Parallel()

	existing := NewOfficialQQConnItem("old-app", "old-secret", "123", false)
	existing.UserID = "OpenQQ:123"
	_, err := replaceOfficialQQCredentials(
		existing,
		&PlatformAdapterOfficialQQ{AppID: "new-app", AppSecret: "new-secret"},
		&OfficialQQAccountProbeResult{UIN: "456"},
	)
	if err == nil {
		t.Fatal("UIN mismatch was accepted")
	}
}

func TestPublicDiceEndpointAppID(t *testing.T) {
	t.Parallel()

	official := &EndPointInfo{
		EndPointInfoBase: EndPointInfoBase{Platform: "QQ", ProtocolType: "official"},
		Adapter:          &PlatformAdapterOfficialQQ{AppID: "123456789"},
	}
	if appID := publicDiceEndpointAppID(official); appID != "123456789" {
		t.Fatalf("official appID = %q", appID)
	}

	ordinary := &EndPointInfo{EndPointInfoBase: EndPointInfoBase{Platform: "QQ", ProtocolType: "onebot"}}
	if appID := publicDiceEndpointAppID(ordinary); appID != "" {
		t.Fatalf("ordinary QQ appID = %q, want empty", appID)
	}
}

func TestOfficialQQIdentityMigrationConfigRequiresOptIn(t *testing.T) {
	config := DefaultConfig
	if config.OfficialQQMigrationEnable {
		t.Fatal("official QQ identity migration is enabled by default")
	}
	if err := yaml.Unmarshal([]byte("officialQQEnableIdentityMigration: true\n"), &config); err != nil {
		t.Fatal(err)
	}
	if !config.OfficialQQMigrationEnable {
		t.Fatal("serve.yaml did not enable official QQ identity migration")
	}
}

func TestOfficialQQIdentityMigrationLegacyDataProbe(t *testing.T) {
	const appID = "123456789"

	dbOperator, openErr := newMockDatabaseOperator(filepath.Join(t.TempDir(), "official-qq-probe.db"))
	if openErr != nil {
		t.Fatal(openErr)
	}
	t.Cleanup(dbOperator.Close)
	if err := dbOperator.db.AutoMigrate(&model.GroupInfo{}, &model.LogInfo{}, &model.LogOneItem{}); err != nil {
		t.Fatal(err)
	}

	migration := &officialQQIdentityMigration{appID: appID}
	prefix := migration.oldGroupPrefix()
	_, upperBound := migration.oldGroupBounds()
	timestamp := int64(1)
	for _, id := range []string{
		"OpenQQ-Group:1-new-group",
		"OpenQQ-Group-T:123456788-other-app",
		upperBound,
	} {
		if err := dbOperator.db.Create(&model.GroupInfo{ID: id, CreatedAt: timestamp, UpdatedAt: &timestamp}).Error; err != nil {
			t.Fatal(err)
		}
	}

	shouldMigrate, probeErr := migration.shouldRunIdentityMigration(dbOperator)
	if probeErr != nil {
		t.Fatal(probeErr)
	}
	if shouldMigrate {
		t.Fatal("legacy group probe matched an unrelated group")
	}

	legacyID := prefix + "group-open-id"
	if err := dbOperator.db.Create(&model.GroupInfo{ID: legacyID, CreatedAt: timestamp, UpdatedAt: &timestamp}).Error; err != nil {
		t.Fatal(err)
	}
	shouldMigrate, probeErr = migration.shouldRunIdentityMigration(dbOperator)
	if probeErr != nil {
		t.Fatal(probeErr)
	}
	if !shouldMigrate {
		t.Fatal("legacy group probe missed an old official QQ group")
	}
	if err := dbOperator.db.Delete(&model.GroupInfo{}, "id = ?", legacyID).Error; err != nil {
		t.Fatal(err)
	}
	if err := dbOperator.db.Create(&model.LogOneItem{GroupID: legacyID, Time: 1}).Error; err != nil {
		t.Fatal(err)
	}
	shouldMigrate, probeErr = migration.shouldRunIdentityMigration(dbOperator)
	if probeErr != nil {
		t.Fatal(probeErr)
	}
	if !shouldMigrate {
		t.Fatal("legacy data probe missed residual official QQ log items")
	}

	type queryPlanRow struct {
		Detail string `gorm:"column:detail"`
	}
	var plan []queryPlanRow
	if err := dbOperator.db.Raw(
		"EXPLAIN QUERY PLAN SELECT id FROM group_info WHERE id >= ? AND id < ? LIMIT 1",
		prefix,
		upperBound,
	).Scan(&plan).Error; err != nil {
		t.Fatal(err)
	}
	usesIndexedSearch := false
	for _, row := range plan {
		detail := strings.ToUpper(row.Detail)
		if !strings.Contains(detail, "GROUP_INFO") {
			continue
		}
		if strings.Contains(detail, "SCAN") {
			t.Fatalf("legacy group probe uses a table scan: %#v", plan)
		}
		usesIndexedSearch = usesIndexedSearch || strings.Contains(detail, "SEARCH")
	}
	if !usesIndexedSearch {
		t.Fatalf("legacy group probe does not use an indexed search: %#v", plan)
	}
}

func TestOfficialQQIdentityMigration(t *testing.T) {
	const (
		appID        = "123456789"
		uin          = "1"
		botID        = "bot-open-id"
		groupOpenID  = "group-open-id-with-hyphens"
		groupOpenID2 = "second-group-open-id"
		memberID     = "member-open-id-with-hyphens"
	)

	dataDir := t.TempDir()
	dbOperator, openErr := newMockDatabaseOperator(filepath.Join(dataDir, "official-qq-migration.db"))
	if openErr != nil {
		t.Fatal(openErr)
	}
	t.Cleanup(dbOperator.Close)
	db := dbOperator.db
	if err := db.AutoMigrate(
		&model.GroupInfo{},
		&model.GroupPlayerInfoBase{},
		&model.AttributesItemModel{},
		&model.BanInfo{},
		&model.EndpointInfo{},
		&model.LogInfo{},
		&model.LogOneItem{},
		&model.CensorLog{},
	); err != nil {
		t.Fatal(err)
	}

	oldEndpoint := "OpenQQ:" + botID
	newEndpoint := "OpenQQ:" + uin
	oldGroupID := "OpenQQ-Group-T:" + appID + "-" + groupOpenID
	newGroupID := "OpenQQ-Group:" + uin + "-" + groupOpenID
	oldUserID := "OpenQQ-Member-T:" + appID + "-" + groupOpenID + "-" + memberID
	newUserID := "OpenQQ:" + uin + "-" + memberID
	oldAttrsID := oldGroupID + "-" + oldUserID
	newAttrsID := newGroupID + "-" + newUserID
	oldGroupID2 := "OpenQQ-Group-T:" + appID + "-" + groupOpenID2
	newGroupID2 := "OpenQQ-Group:" + uin + "-" + groupOpenID2
	oldUserID2 := "OpenQQ-Member-T:" + appID + "-" + groupOpenID2 + "-" + memberID
	oldAttrsID2 := oldGroupID2 + "-" + oldUserID2
	newAttrsID2 := newGroupID2 + "-" + newUserID
	timestamp := int64(1)
	if groupPart, userPart, ok := UnpackGroupUserId(newAttrsID); !ok || groupPart != newGroupID || userPart != newUserID {
		t.Fatalf("UnpackGroupUserId(%q) = (%q, %q, %t)", newAttrsID, groupPart, userPart, ok)
	}

	persistedGroup := newOfficialQQMigrationTestGroup(oldGroupID, oldEndpoint, oldUserID)
	groupData, marshalGroupErr := json.Marshal(persistedGroup)
	if marshalGroupErr != nil {
		t.Fatal(marshalGroupErr)
	}
	if err := db.Create(&model.GroupInfo{ID: oldGroupID, CreatedAt: timestamp, UpdatedAt: &timestamp, Data: groupData}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.GroupPlayerInfoBase{GroupID: oldGroupID, UserID: oldUserID, Name: "member", CreatedAt: 1, UpdatedAt: 1}).Error; err != nil {
		t.Fatal(err)
	}
	secondGroupData, marshalSecondGroupErr := json.Marshal(newOfficialQQMigrationTestGroup(oldGroupID2, oldEndpoint, oldUserID2))
	if marshalSecondGroupErr != nil {
		t.Fatal(marshalSecondGroupErr)
	}
	if err := db.Create(&model.GroupInfo{ID: oldGroupID2, CreatedAt: timestamp, UpdatedAt: &timestamp, Data: secondGroupData}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.GroupPlayerInfoBase{GroupID: oldGroupID2, UserID: oldUserID2, Name: "member", CreatedAt: 1, UpdatedAt: 1}).Error; err != nil {
		t.Fatal(err)
	}
	characters := []model.AttributesItemModel{
		{Id: "character-a", AttrsType: "character", Name: "调查员", OwnerId: oldUserID, CreatedAt: 1, UpdatedAt: 1},
		{Id: "character-b", AttrsType: "character", Name: "调查员", OwnerId: oldUserID2, CreatedAt: 2, UpdatedAt: 2},
		{Id: "character-c", AttrsType: "character", Name: "调查员 (2)", OwnerId: oldUserID2, CreatedAt: 3, UpdatedAt: 3},
	}
	if err := db.Create(&characters).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.AttributesItemModel{Id: oldAttrsID, AttrsType: "group_user", OwnerId: oldUserID, BindingSheetId: "character-a", CreatedAt: timestamp, UpdatedAt: timestamp}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.AttributesItemModel{Id: oldAttrsID2, OwnerId: oldUserID2, BindingSheetId: "character-b", CreatedAt: timestamp, UpdatedAt: timestamp}).Error; err != nil {
		t.Fatal(err)
	}
	olderUserAttrs := &ds.ValueMap{}
	olderUserAttrs.Store("older-only", ds.NewIntVal(1))
	olderUserAttrs.Store("shared", ds.NewIntVal(1))
	olderUserData, encodeOlderErr := ds.NewDictVal(olderUserAttrs).V().ToJSON()
	if encodeOlderErr != nil {
		t.Fatal(encodeOlderErr)
	}
	newerUserAttrs := &ds.ValueMap{}
	newerUserAttrs.Store("newer-only", ds.NewIntVal(2))
	newerUserAttrs.Store("shared", ds.NewIntVal(2))
	newerUserData, encodeNewerErr := ds.NewDictVal(newerUserAttrs).V().ToJSON()
	if encodeNewerErr != nil {
		t.Fatal(encodeNewerErr)
	}
	if err := db.Create(&model.AttributesItemModel{Id: oldUserID, AttrsType: "user", Data: olderUserData, CreatedAt: 1, UpdatedAt: 1}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.AttributesItemModel{Id: oldUserID2, AttrsType: "user", Data: newerUserData, CreatedAt: 2, UpdatedAt: 2}).Error; err != nil {
		t.Fatal(err)
	}
	banItem := &BanListInfoItem{ID: oldUserID, Places: []string{oldGroupID}, Reasons: []string{"test"}}
	banData, marshalBanErr := json.Marshal(banItem)
	if marshalBanErr != nil {
		t.Fatal(marshalBanErr)
	}
	if err := db.Create(&model.BanInfo{ID: oldUserID, UpdatedAt: 1, Data: banData}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.EndpointInfo{UserID: oldEndpoint, CmdNum: 12, OnlineTime: 34, UpdatedAt: timestamp}).Error; err != nil {
		t.Fatal(err)
	}
	logInfo := &model.LogInfo{Name: "session", GroupID: oldGroupID, CreatedAt: timestamp, UpdatedAt: timestamp}
	if err := db.Create(logInfo).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.LogOneItem{LogID: logInfo.ID, GroupID: oldGroupID, IMUserID: oldUserID, UniformID: oldUserID, Time: timestamp}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.CensorLog{GroupID: oldGroupID, UserID: oldUserID, Content: "test", CreatedAt: 1}).Error; err != nil {
		t.Fatal(err)
	}

	// The short-lived simplified format from the previous change is intentionally untouched.
	const simplifiedGroupID = "OpenQQ-Group:simplified-group"
	simplifiedData, marshalSimplifiedErr := json.Marshal(newOfficialQQMigrationTestGroup(simplifiedGroupID, oldEndpoint, "OpenQQ:simplified-user"))
	if marshalSimplifiedErr != nil {
		t.Fatal(marshalSimplifiedErr)
	}
	if err := db.Create(&model.GroupInfo{ID: simplifiedGroupID, CreatedAt: timestamp, UpdatedAt: &timestamp, Data: simplifiedData}).Error; err != nil {
		t.Fatal(err)
	}

	d := &Dice{
		BaseConfig: BaseConfig{DataDir: dataDir, Name: "default"},
		DBOperator: dbOperator,
		Logger:     sealdiceLogger.M(),
		ImSession: &IMSession{
			ServiceAtNew: new(SyncMap[string, *GroupInfo]),
			PendingQuits: new(SyncMap[string, *PendingQuitInfo]),
		},
		DirtyGroups: new(SyncMap[string, int64]),
		Parent:      &DiceManager{},
	}
	d.Config = NewConfig(d)
	d.AttrsManager = &AttrsManager{db: dbOperator}
	d.AttrsManager.m.Store("character-b", &AttributesItem{ID: "character-b", Name: "调查员", IsSaved: true})
	memoryGroup := newOfficialQQMigrationTestGroup(oldGroupID, oldEndpoint, oldUserID)
	memoryGroup.Players = new(SyncMap[string, *GroupPlayerInfo])
	memoryGroup.Players.Store(oldUserID, &GroupPlayerInfo{UserID: oldUserID, GroupID: oldGroupID})
	d.ImSession.ServiceAtNew.Store(oldGroupID, memoryGroup)
	d.Config.BanList.Map.Store(oldUserID, banItem)
	d.DiceMasters = []string{oldEndpoint, oldUserID}
	d.Config.NoticeIDs = []string{oldUserID}
	d.Config.UpgradeWindowID = oldGroupID
	d.ImSession.MarkPendingQuit(oldGroupID, oldEndpoint, QuitOriginAutoInactive, 0)

	endpoint := &EndPointInfo{
		EndPointInfoBase: EndPointInfoBase{
			ID:           "re-added-official-qq",
			Platform:     "QQ",
			ProtocolType: "official",
			UserID:       newEndpoint,
		},
	}
	adapter := &PlatformAdapterOfficialQQ{EndPoint: endpoint, AppID: appID, UIN: uin}
	endpoint.Adapter = adapter
	endpoint.BindRuntime(d.ImSession)
	d.ImSession.EndPoints = []*EndPointInfo{endpoint}

	if err := ensureOfficialQQIdentity(d, adapter, botID, uin); err != nil {
		t.Fatalf("identity initialization with migration disabled failed: %v", err)
	}
	assertOfficialQQMigrationRow(t, db, &model.GroupInfo{}, "id = ?", oldGroupID)
	assertOfficialQQMigrationMissing(t, db, &model.GroupInfo{}, "id = ?", newGroupID)

	d.Config.OfficialQQMigrationEnable = true
	if err := ensureOfficialQQIdentity(d, adapter, botID, uin); err != nil {
		t.Fatalf("first migration failed: %v", err)
	}
	// Simulate stale memory after the data phase succeeded but a later phase failed.
	if cached, ok := d.AttrsManager.m.Load("character-b"); ok {
		cached.Name = "调查员"
	}
	if err := ensureOfficialQQIdentity(d, adapter, botID, uin); err != nil {
		t.Fatalf("idempotent migration failed: %v", err)
	}

	assertOfficialQQMigrationRow(t, db, &model.GroupInfo{}, "id = ?", newGroupID)
	assertOfficialQQMigrationMissing(t, db, &model.GroupInfo{}, "id = ?", oldGroupID)
	assertOfficialQQMigrationRow(t, db, &model.GroupPlayerInfoBase{}, "group_id = ? AND user_id = ?", newGroupID, newUserID)
	assertOfficialQQMigrationRow(t, db, &model.AttributesItemModel{}, "id = ? AND owner_id = ? AND binding_sheet_id = ?", newAttrsID, newUserID, "character-a")
	assertOfficialQQMigrationRow(t, db, &model.AttributesItemModel{}, "id = ? AND owner_id = ? AND binding_sheet_id = ?", newAttrsID2, newUserID, "character-b")
	assertOfficialQQMigrationRow(t, db, &model.AttributesItemModel{}, "id = ? AND owner_id = ? AND name = ?", "character-a", newUserID, "调查员")
	assertOfficialQQMigrationRow(t, db, &model.AttributesItemModel{}, "id = ? AND owner_id = ? AND name = ?", "character-b", newUserID, "调查员 (3)")
	assertOfficialQQMigrationRow(t, db, &model.AttributesItemModel{}, "id = ? AND owner_id = ? AND name = ?", "character-c", newUserID, "调查员 (2)")
	var mergedUserAttrs model.AttributesItemModel
	if err := db.Where("id = ?", newUserID).First(&mergedUserAttrs).Error; err != nil {
		t.Fatal(err)
	}
	mergedValue, decodeErr := ds.VMValueFromJSON(mergedUserAttrs.Data)
	if decodeErr != nil {
		t.Fatal(decodeErr)
	}
	mergedDict, ok := mergedValue.ReadDictData()
	if !ok || mergedDict.Dict == nil {
		t.Fatal("merged QQ user attrs are not a dictionary")
	}
	for key, want := range map[string]*ds.VMValue{
		"older-only": ds.NewIntVal(1),
		"newer-only": ds.NewIntVal(2),
		"shared":     ds.NewIntVal(2),
	} {
		got, exists := mergedDict.Dict.Load(key)
		if !exists || !ds.ValueEqual(got, want, false) {
			t.Fatalf("merged QQ user attr %q = %#v, want %#v", key, got, want)
		}
	}
	assertOfficialQQMigrationRow(t, db, &model.EndpointInfo{}, "user_id = ?", newEndpoint)
	if endpoint.CmdExecutedNum != 12 {
		t.Fatalf("re-added endpoint command count = %d, want 12", endpoint.CmdExecutedNum)
	}
	assertOfficialQQMigrationRow(t, db, &model.LogInfo{}, "group_id = ?", newGroupID)
	assertOfficialQQMigrationRow(t, db, &model.LogOneItem{}, "group_id = ? AND im_userid = ?", newGroupID, newUserID)
	assertOfficialQQMigrationRow(t, db, &model.CensorLog{}, "group_id = ? AND user_id = ?", newGroupID, newUserID)
	assertOfficialQQMigrationRow(t, db, &model.BanInfo{}, "id = ?", newUserID)
	assertOfficialQQMigrationRow(t, db, &model.GroupInfo{}, "id = ?", simplifiedGroupID)

	group, ok := d.ImSession.ServiceAtNew.Load(newGroupID)
	if !ok || group.GroupID != newGroupID || group.InviteUserID != newUserID {
		t.Fatalf("memory group was not migrated: %#v", group)
	}
	if !group.DiceIDExistsMap.Exists(newEndpoint) || group.DiceIDExistsMap.Exists(oldEndpoint) {
		t.Fatalf("endpoint membership map was not migrated")
	}
	if !group.Players.Exists(newUserID) || group.Players.Exists(oldUserID) {
		t.Fatalf("memory players were not migrated")
	}
	if _, ok := d.Config.BanList.Map.Load(newUserID); !ok {
		t.Fatalf("memory ban list was not migrated")
	}
	if d.DiceMasters[0] != newEndpoint || d.DiceMasters[1] != newUserID || d.Config.NoticeIDs[0] != newUserID || d.Config.UpgradeWindowID != newGroupID {
		t.Fatalf("configuration references were not migrated")
	}
	if !d.ImSession.PendingQuits.Exists(makePendingQuitKey(newGroupID, newEndpoint)) {
		t.Fatalf("pending quit reference was not migrated")
	}
	if cached, ok := d.AttrsManager.m.Load("character-b"); !ok || cached.Name != "调查员 (3)" {
		t.Fatalf("cached character name was not migrated: %#v", cached)
	}
}

func TestOfficialQQIdentityMigrationProcessesMultipleBatches(t *testing.T) {
	const (
		appID = "123456789"
		uin   = "1"
	)
	rowCount := officialQQIdentityMigrationBatchSize*2 + 17

	dbOperator, openErr := newMockDatabaseOperator(filepath.Join(t.TempDir(), "official-qq-batches.db"))
	if openErr != nil {
		t.Fatal(openErr)
	}
	t.Cleanup(dbOperator.Close)
	db := dbOperator.db
	if err := db.AutoMigrate(&model.GroupInfo{}, &model.LogOneItem{}); err != nil {
		t.Fatal(err)
	}

	groups := make([]model.GroupInfo, 0, rowCount)
	logItems := make([]model.LogOneItem, 0, rowCount)
	for index := range rowCount {
		groupOpenID := fmt.Sprintf("group-%04d", index)
		memberOpenID := fmt.Sprintf("member-%04d", index)
		oldGroupID := "OpenQQ-Group-T:" + appID + "-" + groupOpenID
		oldUserID := "OpenQQ-Member-T:" + appID + "-" + groupOpenID + "-" + memberOpenID
		groupData, err := json.Marshal(&GroupInfo{GroupID: oldGroupID, InviteUserID: oldUserID})
		if err != nil {
			t.Fatal(err)
		}
		timestamp := int64(index + 1)
		groups = append(groups, model.GroupInfo{ID: oldGroupID, CreatedAt: timestamp, UpdatedAt: &timestamp, Data: groupData})
		logItems = append(logItems, model.LogOneItem{
			GroupID:   oldGroupID,
			IMUserID:  oldUserID,
			UniformID: oldUserID,
			Time:      timestamp,
			Message:   "batch migration payload",
		})
	}
	if err := db.CreateInBatches(&groups, officialQQIdentityMigrationBatchSize).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.CreateInBatches(&logItems, officialQQIdentityMigrationBatchSize).Error; err != nil {
		t.Fatal(err)
	}

	migration := &officialQQIdentityMigration{
		appID:       appID,
		uin:         uin,
		oldEndpoint: "OpenQQ:legacy-bot",
		newEndpoint: formatDiceIDOfficialQQ(uin),
		groups:      map[string]string{},
		users:       map[string]string{},
	}
	if err := migration.collectDatabase(dbOperator); err != nil {
		t.Fatalf("collecting migration identities failed: %v", err)
	}
	if len(migration.groups) != rowCount || len(migration.users) != rowCount {
		t.Fatalf("collected groups/users = %d/%d, want %d/%d", len(migration.groups), len(migration.users), rowCount, rowCount)
	}
	if err := migration.migrateLogDB(dbOperator); err != nil {
		t.Fatalf("batched log migration failed: %v", err)
	}
	if err := migration.migrateDataDB(dbOperator); err != nil {
		t.Fatalf("batched data migration failed: %v", err)
	}

	var oldGroupCount int64
	if err := db.Model(&model.GroupInfo{}).Where("id LIKE ?", migration.oldGroupPrefix()+"%").Count(&oldGroupCount).Error; err != nil {
		t.Fatal(err)
	}
	var oldLogCount int64
	if err := db.Model(&model.LogOneItem{}).Where("group_id LIKE ?", migration.oldGroupPrefix()+"%").Count(&oldLogCount).Error; err != nil {
		t.Fatal(err)
	}
	var newGroupCount int64
	if err := db.Model(&model.GroupInfo{}).Where("id LIKE ?", "OpenQQ-Group:"+uin+"-%").Count(&newGroupCount).Error; err != nil {
		t.Fatal(err)
	}
	var newLogCount int64
	if err := db.Model(&model.LogOneItem{}).Where("group_id LIKE ?", "OpenQQ-Group:"+uin+"-%").Count(&newLogCount).Error; err != nil {
		t.Fatal(err)
	}
	if oldGroupCount != 0 || oldLogCount != 0 || newGroupCount != int64(rowCount) || newLogCount != int64(rowCount) {
		t.Fatalf(
			"migration counts old groups/logs=%d/%d new groups/logs=%d/%d, want 0/0/%d/%d",
			oldGroupCount,
			oldLogCount,
			newGroupCount,
			newLogCount,
			rowCount,
			rowCount,
		)
	}
	var migratedLogItem model.LogOneItem
	if err := db.Where("group_id LIKE ?", "OpenQQ-Group:"+uin+"-%").First(&migratedLogItem).Error; err != nil {
		t.Fatal(err)
	}
	if migratedLogItem.Message != "batch migration payload" {
		t.Fatalf("migrated log message = %q", migratedLogItem.Message)
	}
}

func TestOfficialQQIdentityMigrationRetriesAfterFailure(t *testing.T) {
	const (
		appID    = "123456789"
		uin      = "1"
		botID    = "bot-open-id"
		memberID = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	)

	dataDir := t.TempDir()
	dbOperator, openErr := newMockDatabaseOperator(filepath.Join(dataDir, "official-qq-retry.db"))
	if openErr != nil {
		t.Fatal(openErr)
	}
	t.Cleanup(dbOperator.Close)
	db := dbOperator.db
	if err := db.AutoMigrate(&model.AttributesItemModel{}, &model.EndpointInfo{}); err != nil {
		t.Fatal(err)
	}

	oldUserID1 := "OpenQQ-Member-T:" + appID + "-BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB-" + memberID
	oldUserID2 := "OpenQQ-Member-T:" + appID + "-CCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCC-" + memberID
	newUserID := "OpenQQ:" + uin + "-" + memberID
	for _, id := range []string{oldUserID1, oldUserID2} {
		if err := db.Create(&model.AttributesItemModel{Id: id, AttrsType: "user", Data: []byte("{")}).Error; err != nil {
			t.Fatal(err)
		}
	}

	endpoint := &EndPointInfo{
		EndPointInfoBase: EndPointInfoBase{
			ID:           "re-added-official-qq",
			Platform:     "QQ",
			ProtocolType: "official",
			UserID:       formatDiceIDOfficialQQ(uin),
		},
	}
	adapter := &PlatformAdapterOfficialQQ{EndPoint: endpoint, AppID: appID, UIN: uin}
	endpoint.Adapter = adapter
	d := &Dice{
		BaseConfig: BaseConfig{DataDir: dataDir, Name: "default"},
		DBOperator: dbOperator,
		Logger:     sealdiceLogger.M(),
		ImSession: &IMSession{
			EndPoints:    []*EndPointInfo{endpoint},
			ServiceAtNew: new(SyncMap[string, *GroupInfo]),
			PendingQuits: new(SyncMap[string, *PendingQuitInfo]),
		},
		DirtyGroups: new(SyncMap[string, int64]),
		Parent:      &DiceManager{},
	}
	d.ImSession.Parent = d
	d.Config = NewConfig(d)
	d.Config.OfficialQQMigrationEnable = true
	d.AttrsManager = &AttrsManager{db: dbOperator}
	endpoint.BindRuntime(d.ImSession)

	migrationErr := ensureOfficialQQIdentity(d, adapter, botID, uin)
	if migrationErr == nil {
		t.Fatal("migration with malformed attrs unexpectedly succeeded")
	}
	var migrationFailure *officialQQIdentityMigrationError
	if !errors.As(migrationErr, &migrationFailure) {
		t.Fatalf("migration error has wrong type: %T", migrationErr)
	}
	if adapter.UIN != uin || endpoint.UserID != formatDiceIDOfficialQQ(uin) {
		t.Fatalf("resolved identity was discarded after migration failure: UIN=%q userID=%q", adapter.UIN, endpoint.UserID)
	}

	emptyData, encodeErr := ds.NewDictVal(&ds.ValueMap{}).V().ToJSON()
	if encodeErr != nil {
		t.Fatal(encodeErr)
	}
	if err := db.Model(&model.AttributesItemModel{}).Where("id IN ?", []string{oldUserID1, oldUserID2}).Update("data", emptyData).Error; err != nil {
		t.Fatal(err)
	}
	if err := ensureOfficialQQIdentity(d, adapter, botID, uin); err != nil {
		t.Fatalf("migration retry failed: %v", err)
	}
	assertOfficialQQMigrationRow(t, db, &model.AttributesItemModel{}, "id = ?", newUserID)
	assertOfficialQQMigrationMissing(t, db, &model.AttributesItemModel{}, "id IN ?", []string{oldUserID1, oldUserID2})
}

func newOfficialQQMigrationTestGroup(groupID, endpointID, userID string) *GroupInfo {
	group := &GroupInfo{
		GroupID:         groupID,
		InviteUserID:    userID,
		DiceIDActiveMap: new(SyncMap[string, bool]),
		DiceIDExistsMap: new(SyncMap[string, bool]),
		BotList:         new(SyncMap[string, bool]),
		PlayerGroups:    new(SyncMap[string, []string]),
	}
	group.DiceIDActiveMap.Store(endpointID, true)
	group.DiceIDExistsMap.Store(endpointID, true)
	group.PlayerGroups.Store(userID, []string{groupID})
	return group
}

func assertOfficialQQMigrationRow(t *testing.T, db interface {
	Model(value any) *gorm.DB
}, value any, query string, args ...any) {
	t.Helper()
	var count int64
	if err := db.Model(value).Where(query, args...).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("query %q %v returned %d rows, want 1", query, args, count)
	}
}

func assertOfficialQQMigrationMissing(t *testing.T, db interface {
	Model(value any) *gorm.DB
}, value any, query string, args ...any) {
	t.Helper()
	var count int64
	if err := db.Model(value).Where(query, args...).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("query %q %v returned %d rows, want 0", query, args, count)
	}
}
