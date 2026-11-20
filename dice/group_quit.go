package dice

import (
	"fmt"
	"time"

	"go.uber.org/zap"
)

// GroupQuitConfirmation describes why a manual quit has not been allowed yet.
// ConfirmCode is only issued after the bot has been positively identified as the owner.
type GroupQuitConfirmation struct {
	GroupID     string `json:"groupId"`
	DiceID      string `json:"diceId"`
	Message     string `json:"message"`
	ConfirmCode string `json:"confirmCode,omitempty"`
}

// GroupQuitPermit binds an explicit owner confirmation to one account and group.
// A permit is consumed immediately before sending the platform's quit request.
type GroupQuitPermit struct {
	endpoint *EndPointInfo
	groupID  string
	botID    string
	key      string
	code     string
}

func groupQuitNeedsRoleCheck(ctx *MsgContext) bool {
	return ctx != nil && ctx.EndPoint != nil && (ctx.EndPoint.Platform == "QQ" || ctx.EndPoint.ProtocolType == "onebot")
}

// PrepareGroupQuit checks the current identity or validates a previously issued
// owner confirmation. QuitGroupSafely checks the identity again before quitting.
func PrepareGroupQuit(ctx *MsgContext, groupID, operatorID, inputCode string) (*GroupQuitPermit, *GroupQuitConfirmation) {
	if ctx == nil || ctx.EndPoint == nil || ctx.EndPoint.Adapter == nil {
		return nil, &GroupQuitConfirmation{GroupID: groupID, Message: "当前账号不可用，已取消退群。"}
	}
	if !groupQuitNeedsRoleCheck(ctx) {
		return nil, nil
	}

	result := &GroupQuitConfirmation{GroupID: groupID, DiceID: ctx.EndPoint.UserID}
	key := getDismissConfirmKeyForGroup(ctx, operatorID, groupID)
	if inputCode != "" {
		code, exists := loadDismissConfirmCode(key)
		if !exists || code != inputCode {
			result.Message = "退群确认码无效或已过期，请重新发起退群。"
			return nil, result
		}
		return &GroupQuitPermit{endpoint: ctx.EndPoint, groupID: groupID, botID: ctx.EndPoint.UserID, key: key, code: code}, nil
	}

	needConfirm, ok, detail := shouldDismissRequireOwnerConfirm(ctx, groupID)
	if !ok {
		result.Message = "无法确认骰子账号在目标群中的身份，已取消退群，请稍后重试。"
		logGroupQuitBlocked(ctx, groupID, detail)
		return nil, result
	}
	if !needConfirm {
		return nil, nil
	}

	result.ConfirmCode = generateFourDigitCode()
	result.Message = "骰子账号在目标群中是群主，继续退出将直接解散群聊。请再次确认是否解散该群。"
	storeDismissConfirmCode(key, result.ConfirmCode)
	return nil, result
}

func logGroupQuitBlocked(ctx *MsgContext, groupID, detail string) {
	log := zap.S()
	botID := ""
	if ctx != nil {
		if ctx.Dice != nil && ctx.Dice.Logger != nil {
			log = ctx.Dice.Logger
		}
		if ctx.EndPoint != nil {
			botID = ctx.EndPoint.UserID
		}
	}
	log.Warnf("退群已取消: 群组<%s>，账号<%s>，身份检查结果: %s", groupID, botID, detail)
}

func canAutoQuitGroup(ctx *MsgContext, groupID string) bool {
	if ctx == nil || ctx.EndPoint == nil || ctx.EndPoint.Adapter == nil {
		return false
	}
	if !groupQuitNeedsRoleCheck(ctx) {
		return true
	}
	role, ok := checkBotGroupRole(ctx, groupID)
	if !ok || role == "owner" {
		logGroupQuitBlocked(ctx, groupID, role)
		return false
	}
	return true
}

// QuitGroupSafely refreshes the bot's identity immediately before quitting.
// Automated callers pass nil, so they can never dissolve a group they own.
func QuitGroupSafely(ctx *MsgContext, groupID string, permit *GroupQuitPermit) bool {
	if ctx == nil || ctx.EndPoint == nil || ctx.EndPoint.Adapter == nil {
		return false
	}
	if groupQuitNeedsRoleCheck(ctx) {
		role, ok := checkBotGroupRole(ctx, groupID)
		if !ok {
			logGroupQuitBlocked(ctx, groupID, role)
			return false
		}
		if role == "owner" && permit == nil {
			logGroupQuitBlocked(ctx, groupID, "当前账号为群主，尚未完成二次确认")
			return false
		}
		if permit != nil {
			if permit.endpoint != ctx.EndPoint || permit.groupID != groupID || permit.botID != ctx.EndPoint.UserID {
				logGroupQuitBlocked(ctx, groupID, "群主退群确认与目标群或当前账号不匹配")
				return false
			}
			state, exists := dismissConfirmCodes.LoadAndDelete(permit.key)
			if !exists || state == nil || state.Code != permit.code || time.Now().Unix() > state.ExpiresAt {
				logGroupQuitBlocked(ctx, groupID, "群主退群确认码无效或已过期")
				return false
			}
		}
	}
	ctx.EndPoint.Adapter.QuitGroup(ctx, groupID)
	return true
}

func quitGroupForBlacklist(ctx *MsgContext, groupID string, beforeQuit func()) bool {
	if !canAutoQuitGroup(ctx, groupID) {
		return false
	}
	if beforeQuit != nil {
		beforeQuit()
	}
	return QuitGroupSafely(ctx, groupID, nil)
}

// Identity API replies must remain receivable while the quit task is waiting.
func quitGroupForBlacklistAsync(ctx *MsgContext, groupID string, beforeQuit func(*MsgContext)) {
	quitCtx := ctx.ShallowCopy()
	ctx.Dice.runMessageTask(func() {
		defer ErrorLogAndContinue(quitCtx.Dice)
		quitGroupForBlacklist(quitCtx, groupID, func() {
			if beforeQuit != nil {
				beforeQuit(quitCtx)
			}
		})
	})
}

func groupQuitCancelledMessage(groupID string) string {
	return fmt.Sprintf("群组 %s 的身份检查未通过或身份发生变化，已取消退群，请重新发起操作。", groupID)
}
