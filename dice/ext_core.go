package dice

import (
	"fmt"

	"sealdice-core/dice/events"
	log "sealdice-core/utils/kratos"
)

func RegisterBuiltinExtCore(dice *Dice) {
	theExt := &ExtInfo{
		Name:        "core",
		Version:     "1.0.0",
		Brief:       "核心逻辑模块，该扩展即使被关闭也会依然生效",
		Author:      "SealDice-Team",
		AutoActive:  true, // 是否自动开启
		Official:    true,
		GetDescText: GetExtensionDesc,
		OnGroupLeave: func(ctx *MsgContext, event *events.GroupLeaveEvent) {
			if event.UserID == ctx.EndPoint.UserID {
				opUID := event.OperatorID
				groupName := ctx.Dice.Parent.TryGetGroupName(event.GroupID)
				userName := ctx.Dice.Parent.TryGetUserName(opUID)
				if opUID == "" {
					txt := fmt.Sprintf("退出群组<%s>(%s)，但操作者ID为空，停止继续处理", groupName, event.GroupID)
					log.Info(txt)
					ctx.Notice(txt)
					return
				}

				skip := false
				skipReason := ""
				banInfo, ok := ctx.Dice.Config.BanList.GetByID(opUID)
				if ok {
					if banInfo.Rank == 30 {
						skip = true
						skipReason = "信任用户"
					}
				}
				if ctx.Dice.IsMaster(opUID) {
					skip = true
					skipReason = "Master"
				}

				var extra string
				if skip {
					extra = fmt.Sprintf("\n取消处罚，原因为%s", skipReason)
				} else {
					ctx.Dice.Config.BanList.AddScoreByGroupKicked(opUID, event.GroupID, ctx)
				}

				txt := fmt.Sprintf("被踢出群: 在群组<%s>(%s)中被踢出，操作者:<%s>(%s)%s", groupName, event.GroupID, userName, event.OperatorID, extra)
				log.Info(txt)
				ctx.Notice(txt)
			}
		},
	}

	theExt.CmdMap = map[string]*CmdItemInfo{
		"team": cmdTeam,
	}

	dice.RegisterExtension(theExt)
}
