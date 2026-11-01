//nolint:testpackage
package dice

import (
	"strings"
	"testing"
	"time"

	"sealdice-core/dice/service"
	"sealdice-core/model"
	"sealdice-core/utils/constant"
)

func TestOfficialQQLogNicknameUsesNameCard(t *testing.T) {
	ctx, _ := newNameCardTestContext(t)
	ctx.Player.AutoSetNameTemplate = "{$t玩家_RAW} HP10/10"

	got := officialQQLogNickname(ctx, "原始昵称")
	if got != "茶叶 HP10/10" {
		t.Fatalf("officialQQLogNickname = %q, want %q", got, "茶叶 HP10/10")
	}
}

func TestOfficialQQLogNicknameOnlyForOfficialQQ(t *testing.T) {
	ctx, _ := newNameCardTestContext(t)
	ctx.Player.AutoSetNameTemplate = "{$t玩家_RAW} HP10/10"
	ctx.EndPoint.ProtocolType = "onebot"

	if got := officialQQLogNickname(ctx, "原始昵称"); got != "" {
		t.Fatalf("officialQQLogNickname = %q, want empty for non-official endpoint", got)
	}
}

func TestReplyCarriesDisplayOnlyNameCardContext(t *testing.T) {
	ctx, spy := newNameCardTestContext(t)
	ctx.Player.AutoSetNameTemplate = "茶叶 HP10/10"
	msg := &Message{
		MessageType: "group",
		GroupID:     ctx.Group.GroupID,
		Sender: SenderBase{
			UserID:   ctx.Player.UserID,
			Nickname: ctx.Player.Name,
		},
	}

	ReplyGroup(ctx, msg, "D100=50")

	line := buildOfficialQQNameCardLine(ctx)
	if line == "" {
		t.Fatal("name card line should not be empty")
	}
	if len(spy.sendMessageText) != 1 {
		t.Fatalf("send count = %d, want 1", len(spy.sendMessageText))
	}
	if spy.sendMessageText[0] != line+"D100=50" {
		t.Fatalf("sent text = %q, want %q", spy.sendMessageText[0], line+"D100=50")
	}
	if len(spy.sendCardLines) != 1 || spy.sendCardLines[0] != line {
		t.Fatalf("send ctx name card line = %#v, want [%q]", spy.sendCardLines, line)
	}
	if got := stripOfficialQQNameCard(spy.sendCardLines[0], spy.sendMessageText[0]); got != "D100=50" {
		t.Fatalf("stripOfficialQQNameCard = %q, want %q", got, "D100=50")
	}
}

func TestOfficialQQLogRecordingUsesNameCardAndDropsDisplayPrefix(t *testing.T) {
	ctx, _ := newNameCardTestContext(t)
	ctx.Player.AutoSetNameTemplate = "{$t玩家_RAW} HP10/10"

	logDB := ctx.Dice.DBOperator.GetLogDB(constant.WRITE)
	if err := logDB.AutoMigrate(&model.LogInfo{}, &model.LogOneItem{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}

	group := SetBotOnAtGroup(ctx, "QQ-Group:10001")
	group.LogCurName = "测试"
	group.LogOn = true
	ctx.Group = group

	ext, ok := ctx.Dice.ExtRegistry.Load("log")
	if !ok || ext == nil || ext.OnMessageReceived == nil || ext.OnMessageSend == nil {
		t.Fatal("log extension is not registered")
	}

	playerMsg := &Message{
		Platform:    "QQ",
		MessageType: "group",
		GroupID:     group.GroupID,
		Time:        time.Now().Unix(),
		Message:     "hello",
		RawID:       "player-1",
		Sender: SenderBase{
			UserID:   ctx.Player.UserID,
			Nickname: "原始昵称",
		},
	}
	ext.OnMessageReceived(ctx, playerMsg)

	cardLine := buildOfficialQQNameCardLine(ctx)
	if cardLine == "" {
		t.Fatal("name card line should not be empty")
	}
	sendCtx := ctx.ShallowCopy()
	sendCtx.officialQQNameCardLine = cardLine

	diceMsg := &Message{
		Platform:    "QQ",
		MessageType: "group",
		GroupID:     group.GroupID,
		Time:        time.Now().Unix(),
		Message:     cardLine + "D100=50",
		RawID:       "dice-1",
		Sender: SenderBase{
			UserID:   ctx.EndPoint.UserID,
			Nickname: ctx.EndPoint.Nickname,
		},
	}
	ext.OnMessageSend(sendCtx, diceMsg, "")

	lines, err := service.LogGetAllLines(ctx.Dice.DBOperator, group.GroupID, "测试")
	if err != nil {
		t.Fatalf("LogGetAllLines: %v", err)
	}
	if len(lines) != 2 {
		t.Fatalf("log lines = %d, want 2", len(lines))
	}
	if lines[0].Nickname != "茶叶 HP10/10" {
		t.Fatalf("player log nickname = %q, want %q", lines[0].Nickname, "茶叶 HP10/10")
	}
	if lines[1].Message != "D100=50" {
		t.Fatalf("dice log message = %q, want %q", lines[1].Message, "D100=50")
	}
	if strings.Contains(lines[1].Message, "<font") {
		t.Fatalf("dice log message still contains name card: %q", lines[1].Message)
	}
}
