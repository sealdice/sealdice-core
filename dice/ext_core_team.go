package dice

import (
	"cmp"
	"fmt"
	"math/rand/v2"
	"slices"
	"strconv"
	"strings"

	"github.com/samber/lo"
	ds "github.com/sealdice/dicescript"
)

type attributeContainer struct {
	UserID string
	Value  *ds.VMValue
}

var cmdTeam = &CmdItemInfo{
	Name:      "team",
	ShortHelp: ".team <团队名> add/del/clear/call/st/ra/rc [attr-expr]",
	Help: `队伍管理指令:
.team <团队名> add/del <@成员...> // 增减队伍列表，若无团队会自动新建
.team <团队名> clear // 清空队伍
.team <团队名> call // 艾特队伍
.team <团队名> draw [数量] // 随机抽取队伍成员
.team <团队名> <属性> // 列出队内成员属性`,
	DisabledInPrivate: true,
	AllowDelegate:     true,
	Solve: func(context *MsgContext, message *Message, arguments *CmdArgs) CmdExecuteResult {
		context.DelegateText = ""
		execResult := CmdExecuteResult{Matched: true, Solved: true}
		// Check is in group because DisabledInPrivate can be unreliable
		if context.IsPrivate {
			s := DiceFormatTmpl(context, "核心:提示_私聊不可用")
			ReplyToSender(context, message, s)
			return execResult
		}

		groupName := arguments.GetArgN(1)
		if groupName == "" || groupName == "help" {
			execResult.ShowHelp = true
			return execResult
		}

		group := context.Group
		if group.PlayerGroups == nil {
			group.PlayerGroups = new(SyncMap[string, []string])
		}
		playerGroup, groupExists := group.PlayerGroups.Load(groupName)

		switch subcommand := arguments.GetArgN(2); subcommand {
		case "add":
			if teamIsAtNoneOrSelf(context.EndPoint, arguments.At) {
				ReplyToSender(context, message, "必须有一名被艾特的成员，且不能是骰子自己或@全体成员")
				break
			}
			at := lo.UniqBy(slices.Clone(arguments.At), func(item *AtInfo) string { return item.UserID })
			var count int
			for _, atInfo := range at {
				userID := atInfo.UserID
				if userID == context.EndPoint.UserID {
					continue
				}
				if !slices.Contains(playerGroup, userID) {
					playerGroup = append(playerGroup, userID)
					count++
				}
			}
			group.PlayerGroups.Store(groupName, playerGroup)
			ReplyToSender(context, message, fmt.Sprintf("已经添加%d名玩家至团队%s", count, groupName))
		case "del", "rm", "delete", "remove":
			if teamIsAtNoneOrSelf(context.EndPoint, arguments.At) {
				ReplyToSender(context, message, "必须有一名被艾特的成员，且不能是骰子自己或@全体成员")
				break
			}
			if !groupExists {
				ReplyToSender(context, message, fmt.Sprintf("没有叫%s的团队或它已经被清除", groupName))
				break
			}
			at := lo.UniqBy(slices.Clone(arguments.At), func(item *AtInfo) string { return item.UserID })
			var count int
			for _, atInfo := range at {
				userID := atInfo.UserID
				if userID == context.EndPoint.UserID {
					continue
				}
				if idx := slices.Index(playerGroup, userID); idx > -1 {
					playerGroup = append(playerGroup[:idx], playerGroup[idx+1:]...)
					count++
				}
			}
			group.PlayerGroups.Store(groupName, playerGroup)
			ReplyToSender(context, message, fmt.Sprintf("已经从团队%s删除%d名玩家", groupName, count))
		case "clear":
			if !groupExists {
				ReplyToSender(context, message, fmt.Sprintf("没有叫%s的团队或它已经被清除", groupName))
				break
			}
			group.PlayerGroups.Delete(groupName)
			ReplyToSender(context, message, fmt.Sprintf("清空了团队%s", groupName))
		case "call", "":
			if !groupExists {
				ReplyToSender(context, message, fmt.Sprintf("没有名叫%s的群组", groupName))
				break
			}
			rawUserIDs := teamExtractRawIDsFromGroup(playerGroup)
			cqCodes := make([]string, 0, len(rawUserIDs))
			for _, id := range rawUserIDs {
				cqCodes = append(cqCodes, fmt.Sprintf("[CQ:at,qq=%s]", id))
			}
			ReplyToSender(context, message, fmt.Sprintf("呼叫%s：%s", groupName, strings.Join(cqCodes, " ")))
		case "draw":
			if !groupExists {
				ReplyToSender(context, message, fmt.Sprintf("没有叫%s的团队或它已经被清除", groupName))
				break
			}
			if len(playerGroup) == 0 {
				ReplyToSender(context, message, fmt.Sprintf("团队%s中没有成员", groupName))
				break
			}
			countStr := arguments.GetArgN(3)
			count := 1
			if countStr != "" {
				parsedCount, err := strconv.Atoi(countStr)
				if err != nil || parsedCount < 1 {
					ReplyToSender(context, message, "抽取数量必须是大于等于1的整数")
					break
				}
				count = parsedCount
				if count > len(playerGroup) {
					ReplyToSender(context, message, fmt.Sprintf("抽取数量不能超过团队人数(%d)", len(playerGroup)))
					break
				}
			}
			availableMembers := make([]string, len(playerGroup))
			copy(availableMembers, playerGroup)
			selectedMembers := make([]string, 0, count)
			for i := 0; i < count && len(availableMembers) > 0; i++ {
				index := rand.IntN(len(availableMembers))
				selectedUserID := availableMembers[index]
				selectedMembers = append(selectedMembers, selectedUserID)
				availableMembers = append(availableMembers[:index], availableMembers[index+1:]...)
			}
			cqCodes := make([]string, 0, len(selectedMembers))
			for _, userID := range selectedMembers {
				rawUserID := teamStripPlatformPrefix(userID)
				cqCodes = append(cqCodes, fmt.Sprintf("[CQ:at,qq=%s]", rawUserID))
			}
			if count == 1 {
				ReplyToSender(context, message, fmt.Sprintf("从团队%s中随机抽取到：%s", groupName, cqCodes[0]))
			} else {
				ReplyToSender(context, message, fmt.Sprintf("从团队%s中随机抽取%d名成员：%s", groupName, count, strings.Join(cqCodes, " ")))
			}
		default:
			if !groupExists {
				ReplyToSender(context, message, fmt.Sprintf("没有名叫%s的团队", groupName))
				break
			}

			currentGameSystem, exists := context.Dice.GameSystemMap.Load(context.Group.System)
			if !exists {
				context.Dice.Logger.Errorf("Group game system %s not found", context.Group.System)
				ReplyToSender(context, message, DiceFormatTmpl(context, "核心:骰子执行异常"))
				break
			}
			attributeName := currentGameSystem.GetAlias(subcommand)
			attributeManager := context.Dice.AttrsManager
			defaultAttributeValue := currentGameSystem.GetDefaultValueEx(context, attributeName)

			containers := make([]attributeContainer, 0, len(playerGroup))
			for _, userID := range playerGroup {
				characterAttributes, err := attributeManager.Load(group.GroupID, userID)
				if err != nil {
					// Most likely, no such user
					context.Dice.Logger.Error(err)
					tmpl := DiceFormatTmpl(context, "核心:骰子执行异常")
					ReplyToSender(context, message, tmpl)
					break
				}
				attr := characterAttributes.Load(attributeName)
				if attr == nil || ds.ValueEqual(attr, ds.NewIntVal(0), false) {
					if defaultAttributeValue != nil {
						attr = defaultAttributeValue
					}
				}
				containers = append(containers, attributeContainer{
					UserID: userID,
					Value:  attr,
				})
			}

			attributeType := ds.VMTypeNull
			if defaultAttributeValue != nil {
				attributeType = defaultAttributeValue.TypeId
			}

			switch attributeType {
			case ds.VMTypeInt:
				slices.SortFunc(containers, func(a, b attributeContainer) int {
					v1 := a.Value.MustReadInt()
					v2 := b.Value.MustReadInt()
					return int(v2) - int(v1)
				})
			case ds.VMTypeFloat:
				slices.SortFunc(containers, func(a, b attributeContainer) int {
					v1 := a.Value.MustReadFloat()
					v2 := b.Value.MustReadFloat()
					return cmp.Compare(v2, v1)
				})
			default:
				// We don't sort types that are too complex or incomparable
				// Do not delete this default branch. LINT might break.
			}

			formatList := make([]string, 0, len(containers))
			for _, c := range containers {
				// STR 50 @木落 SAN65 HP11/11 DEX50
				// This format postpones username, which can be long and irregular
				s := fmt.Sprintf("%s %s [CQ:at,qq=%s]", attributeName, c.Value.ToString(), teamStripPlatformPrefix(c.UserID)) // ToString is by no means Go-idiomatic
				formatList = append(formatList, s)
			}

			ReplyToSender(context, message, fmt.Sprintf("队伍%s的属性：\n%s", groupName, strings.Join(formatList, "\n")))
		}

		return execResult
	},
}

func teamStripPlatformPrefix(userID string) string {
	return strings.SplitN(userID, ":", 2)[1]
}

// teamExtractRawIDsFromGroup extracts user IDs from AtInfo. The platform prefix is kept.
func teamExtractRawIDsFromGroup(ids []string) []string {
	userIDs := make([]string, 0, len(ids))
	for _, userID := range ids {
		rawID := teamStripPlatformPrefix(userID)
		userIDs = append(userIDs, rawID)
	}
	return userIDs
}

// teamIsAtNoneOrSelf if there is no AtInfo or if the only AtInfo refers to the bot itself.
func teamIsAtNoneOrSelf(endpoint *EndPointInfo, at []*AtInfo) bool {
	return len(at) == 0 || (len(at) == 1 && at[0].UserID == endpoint.UserID)
}
