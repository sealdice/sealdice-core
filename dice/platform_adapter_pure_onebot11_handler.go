package dice

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/PaienNate/SealSocketIO/socketio"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bytedance/sonic"
	"github.com/google/uuid"
	"github.com/orbs-network/govnr"
	"github.com/tidwall/gjson"
)

// handle函数是在收到事件后，根据事件进行处理的函数们
// 阶层标识： serveOnebot-> onXXXEvent->handleXXXAction->messageQueueOnxxxxx

// 这部分代码只是做了封装，什么时候会调用这部分逻辑我现在仍然蒙在鼓里
func (p *PlatformAdapterPureOnebot11) handleCustomGroupInfoAction(req gjson.Result, _ *socketio.EventPayload) error {
	groupID := FormatDiceIDQQGroup(req.Get("data.group_id").String())
	groupInfo, ok := p.Session.ServiceAtNew.Load(groupID)
	if !ok {
		// 我也不知道木落要创建个什么，保持一致罢。
		return nil
	}
	if req.Get("data.max_member_count").Int() == 0 {
		diceID := p.EndPoint.UserID
		if _, exists := groupInfo.DiceIDExistsMap.Load(diceID); exists {
			// 不在群里了，更新信息
			groupInfo.DiceIDExistsMap.Delete(diceID)
			groupInfo.UpdatedAtTime = time.Now().Unix()
		}
	} else if req.Get("data.group_name").String() != groupInfo.GroupName {
		// 更新群名
		groupInfo.GroupName = req.Get("data.group_name").String()
		groupInfo.UpdatedAtTime = time.Now().Unix()
	}
	// 我有点搞不懂这边在干什么，先贴过来
	// 处理被强制拉群的情况
	uid := groupInfo.InviteUserID
	// 这里的Parent拿的是Dice，是否应该把Dice抽出去呢
	banInfo, ok := p.Session.Parent.Config.BanList.GetByID(uid)
	// 创建名义上是Context的玩意 为了适配im_session的Context
	ctx := &MsgContext{EndPoint: p.EndPoint, Session: p.Session, Dice: p.Session.Parent}
	if ok {
		if banInfo.Rank == BanRankBanned && p.Session.Parent.Config.BanList.BanBehaviorRefuseInvite {
			// 如果是被ban之后拉群，判定为强制拉群
			if groupInfo.EnteredTime > 0 && groupInfo.EnteredTime > banInfo.BanTime {
				text := fmt.Sprintf("本次入群为遭遇强制邀请，即将主动退群，因为邀请人%s正处于黑名单上。打扰各位还请见谅。感谢使用海豹核心。", groupInfo.InviteUserID)
				ReplyGroupRaw(ctx, &Message{GroupID: groupID}, text, "")
				time.Sleep(1 * time.Second)
				p.QuitGroup(ctx, groupID)
			}
			return nil
		}
	}

	// 强制拉群情况2 - 群在黑名单
	banInfo, ok = ctx.Dice.Config.BanList.GetByID(groupID)
	if ok {
		if banInfo.Rank == BanRankBanned {
			// 如果是被ban之后拉群，判定为强制拉群
			if groupInfo.EnteredTime > 0 && groupInfo.EnteredTime > banInfo.BanTime {
				text := fmt.Sprintf("被群已被拉黑，即将自动退出，解封请联系骰主。打扰各位还请见谅。感谢使用海豹核心:\n当前情况: %s", banInfo.toText(ctx.Dice))
				ReplyGroupRaw(ctx, &Message{GroupID: groupID}, text, "")
				time.Sleep(1 * time.Second)
				p.QuitGroup(ctx, groupID)
			}
			return nil
		}
	}
	return nil
}

// 加群逻辑里比较复杂，列在这里
// 加群：被好友邀请-> 获取群信息 -> 根据获取的群信息，判断是否应该加群
func (p *PlatformAdapterPureOnebot11) handleReqGroupAction(req gjson.Result, _ *socketio.EventPayload) error {
	// 创建虚拟Context
	ctx := &MsgContext{EndPoint: p.EndPoint, Session: p.Session, Dice: p.Session.Parent}
	switch req.Get("sub_type").String() {
	case "invite":
		// 先判断是否需要加群
		ok, reason := p.checkPassBlackListGroup(req.Get("user_id").String(), req.Get("group_id").String(), ctx)
		// 从缓存里取数据
		groupCache, exist := p.GroupInfoCache.Get(req.Get("group_id").String())
		// 缓存里有，直接处理
		if exist {
			handleResp := GroupRequestResponse{
				Flag:      req.Get("flag").String(),
				Approve:   ok,
				Reason:    reason,
				GroupName: groupCache.Get("data.group_name").String(),
				GroupID:   req.Get("group_id").String(),
				UserID:    req.Get("user_id").String(),
				SubType:   req.Get("sub_type").String(),
			}
			newUUID, _ := uuid.NewUUID()
			marshal, _ := sonic.Marshal(handleResp)
			err := p.Publisher.Publish(TopicHandleAddNewGroup, message.NewMessage(newUUID.String(), marshal))
			if err != nil {
				p.Logger.Errorf("收到邀请，但处理到加群请求时失败：%v", err)
				return err
			}
			return nil
		}
		// 缓存里没有，创建goroutine 等缓存上数据！
		// TODO: ctx估计得全用一个衍生
		cancelContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		p.GetGroupInfoAsync(req.Get("group_id").String() + "@" + TopicHandleInviteToGroup)
		runner, err := p.Subscriber.Subscribe(cancelContext, fmt.Sprintf("%s/%s", TopicHandleInviteToGroup, req.Get("group_id").String()))
		if err != nil {
			cancel()
			return err
		}
		// TODO: 有不少重复冗余代码，考虑以后优化 -> 真的有人优化吗
		govnr.Once(p.GoVNRErrorLogger, func() {
			defer cancel()
			select {
			case msg := <-runner:
				// 此时获取到信息了 直接从缓存里拿数据就得了 别再等着传一份了 没必要
				// 这里获取的信息是：群信息 将群信息拼接一下，丢给加群队列
				result := gjson.ParseBytes(msg.Payload)
				// 需要拼接的字段： 群组名 群组ID 邀请人ID 是否同意 原因 flag 信息subType
				handleResp := GroupRequestResponse{
					Flag:      req.Get("flag").String(),
					Approve:   ok,
					Reason:    reason,
					GroupName: result.Get("data.group_name").String(),
					GroupID:   result.Get("data.group_id").String(),
					UserID:    req.Get("user_id").String(),
					SubType:   req.Get("sub_type").String(),
				}
				newUUID, _ := uuid.NewUUID()
				marshal, _ := sonic.Marshal(handleResp)
				err = p.Publisher.Publish(TopicHandleAddNewGroup, message.NewMessage(newUUID.String(), marshal))
				if err != nil {
					p.Logger.Errorf("收到邀请，但处理到加群请求时失败：%v", err)
					return
				}
				return
			case <-cancelContext.Done():
				// 能执行到这里，说明超时过了 5 秒 都没获取到信息 填写俺不知道.jpg
				handleResp := GroupRequestResponse{
					Flag:      req.Get("flag").String(),
					Approve:   ok,
					Reason:    reason,
					GroupName: "%未知群聊%",
					GroupID:   req.Get("group_id").String(),
					UserID:    req.Get("user_id").String(),
					SubType:   req.Get("sub_type").String(),
				}
				newUUID, _ := uuid.NewUUID()
				marshal, _ := sonic.Marshal(handleResp)
				err = p.Publisher.Publish(TopicHandleAddNewGroup, message.NewMessage(newUUID.String(), marshal))
				if err != nil {
					p.Logger.Errorf("收到邀请，但处理到加群请求时失败：%v", err)
					return
				}
			default:
				// DO NOTHING
			}
		})
	default:
		// DO NOTHING NOW
	}
	return nil
}

func (p *PlatformAdapterPureOnebot11) handleReqFriendAction(req gjson.Result, _ *socketio.EventPayload) error {
	// 只有一种情况 就是好友添加
	// 获取请求详情
	var comment string
	if req.Get("comment").Exists() {
		comment = strings.TrimSpace(req.Get("comment").String())
		comment = strings.ReplaceAll(comment, "\u00a0", "")
	}
	// 将匹配的验证问题
	toMatch := strings.TrimSpace(p.Session.Parent.Config.FriendAddComment)
	// 创建虚构MsgContext
	ctx := &MsgContext{EndPoint: p.EndPoint, Session: p.Session, Dice: p.Session.Parent}
	var extra string
	// 匹配验证问题检查
	var passQuestion bool
	var passblackList bool
	if comment != DiceFormat(ctx, toMatch) {
		passQuestion = p.checkMultiFriendAddVerify(comment, toMatch)
	}
	// 匹配黑名单检查
	passblackList = p.checkPassBlackList(req.Get("user_id").String(), ctx)
	// 格式化请求的数据
	comment = strconv.Quote(comment)
	if comment == "" {
		comment = "(无)"
	}
	if !passQuestion {
		extra = "。回答错误"
	} else {
		extra = "。回答正确"
	}
	if !passblackList {
		extra += "。（被禁止用户）"
	}
	if p.IgnoreFriendRequest {
		extra += "。由于设置了忽略邀请，此信息仅为通报"
	}

	txt := fmt.Sprintf("收到QQ好友邀请: 邀请人:%s, 验证信息: %s, 是否自动同意: %t%s", req.Get("user_id").String(), comment, passQuestion && passblackList, extra)
	p.Logger.Info(txt)
	ctx.Notice(txt)
	result := map[string]interface{}{
		"flag":    req.Get("flag").String(),
		"approve": passQuestion && passblackList,
		"remark":  "",
		"reason":  extra,
	}
	newUUID, err := uuid.NewUUID()
	if err != nil {
		return err
	}
	marshal, _ := sonic.Marshal(result)
	msg := message.NewMessage(newUUID.String(), marshal)
	err = p.Publisher.Publish(TopicHandleAddNewFriends, msg)
	if err != nil {
		return err
	}
	return nil
}
