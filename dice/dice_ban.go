package dice

import (
	"encoding/json"
	"fmt"
	"sealdice-core/dice/model"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
)

type BanRankType int

const (
	BanRankBanned  BanRankType = -30
	BanRankWarn    BanRankType = -10
	BanRankNormal  BanRankType = 0
	BanRankTrusted BanRankType = 30
)

type BanListInfoItem struct {
	ID      string      `json:"ID" jsbind:"id"`
	Name    string      `json:"name" jsbind:"name"`
	Score   int64       `json:"score" jsbind:"score"`     // 怒气值
	Rank    BanRankType `json:"rank" jsbind:"rank"`       // 0 没事 -10警告 -30禁止 30信任
	Times   []int64     `json:"times" jsbind:"times"`     // 事发时间
	Reasons []string    `json:"reasons" jsbind:"reasons"` // 拉黑原因
	Places  []string    `json:"places" jsbind:"places"`   // 发生地点
	BanTime int64       `json:"banTime" jsbind:"banTime"` // 上黑名单时间

	BanUpdatedAt int64 `json:"-"` // 排序依据，不过可能和bantime重复？
	UpdatedAt    int64 `json:"-"` // 数据更新时间
}

var BanRankText = map[BanRankType]string{
	BanRankTrusted: "信任",
	BanRankBanned:  "禁止",
	BanRankWarn:    "警告",
	BanRankNormal:  "常规",
}

func (i *BanListInfoItem) toText(_ *Dice) string {
	prefix := BanRankText[i.Rank]
	if i.Rank == -10 || i.Rank == -30 {
		return fmt.Sprintf("[%s] <%s>(%s) 原因: %s", prefix, i.Name, i.ID, strings.Join(i.Reasons, ","))
	}
	if i.Rank == 30 {
		return fmt.Sprintf("[%s] <%s> <%s> 原因: %s", prefix, i.Name, i.ID, strings.Join(i.Reasons, ","))
	}
	return ""
}

type BanListInfo struct {
	Parent                          *Dice                              `yaml:"-" json:"-"`
	Map                             *SyncMap[string, *BanListInfoItem] `yaml:"-" json:"-"`
	BanBehaviorRefuseReply          bool                               `yaml:"banBehaviorRefuseReply" json:"banBehaviorRefuseReply"`                   // 拉黑行为: 拒绝回复
	BanBehaviorRefuseInvite         bool                               `yaml:"banBehaviorRefuseInvite" json:"banBehaviorRefuseInvite"`                 // 拉黑行为: 拒绝邀请
	BanBehaviorQuitLastPlace        bool                               `yaml:"banBehaviorQuitLastPlace" json:"banBehaviorQuitLastPlace"`               // 拉黑行为: 退出事发群
	BanBehaviorQuitPlaceImmediately bool                               `yaml:"banBehaviorQuitPlaceImmediately" json:"banBehaviorQuitPlaceImmediately"` // 拉黑行为: 使用时立即退出群
	BanBehaviorQuitIfAdmin          bool                               `yaml:"banBehaviorQuitIfAdmin" json:"banBehaviorQuitIfAdmin"`                   // 拉黑行为: 邀请者以上权限使用时立即退群，否则发出警告信息
	ThresholdWarn                   int64                              `yaml:"thresholdWarn" json:"thresholdWarn"`                                     // 警告阈值
	ThresholdBan                    int64                              `yaml:"thresholdBan" json:"thresholdBan"`                                       // 错误阈值
	AutoBanMinutes                  int64                              `yaml:"autoBanMinutes" json:"autoBanMinutes"`                                   // 自动禁止时长

	ScoreReducePerMinute int64 `yaml:"scoreReducePerMinute" json:"scoreReducePerMinute"` // 每分钟下降
	ScoreGroupMuted      int64 `yaml:"scoreGroupMuted" json:"scoreGroupMuted"`           // 群组禁言
	ScoreGroupKicked     int64 `yaml:"scoreGroupKicked" json:"scoreGroupKicked"`         // 群组踢出
	ScoreTooManyCommand  int64 `yaml:"scoreTooManyCommand" json:"scoreTooManyCommand"`   // 刷指令

	JointScorePercentOfGroup   float64 `yaml:"jointScorePercentOfGroup" json:"jointScorePercentOfGroup"`     // 群组连带责任
	JointScorePercentOfInviter float64 `yaml:"jointScorePercentOfInviter" json:"jointScorePercentOfInviter"` // 邀请人连带责任

	cronID cron.EntryID
}

func (i *BanListInfo) Init() {
	// 此为配置装载前的默认设置
	i.BanBehaviorRefuseReply = true
	i.BanBehaviorRefuseInvite = true
	i.BanBehaviorQuitLastPlace = false
	i.ThresholdWarn = 100
	i.ThresholdBan = 200
	i.AutoBanMinutes = 60 * 12 // 12小时

	i.ScoreReducePerMinute = 1
	i.ScoreGroupMuted = 100
	i.ScoreGroupKicked = 200
	i.ScoreTooManyCommand = 100

	i.JointScorePercentOfGroup = 0.5
	i.JointScorePercentOfInviter = 0.3
	i.Map = new(SyncMap[string, *BanListInfoItem])
}

func (i *BanListInfo) Loads() {
}

func (i *BanListInfo) AfterLoads() {
	// 加载完成了
	d := i.Parent
	i.cronID, _ = d.Parent.Cron.AddFunc("@every 1m", func() {
		if d.DBData == nil {
			return
		}
		var toDelete []string
		d.BanList.Map.Range(func(k string, v *BanListInfoItem) bool {
			if v.Rank == BanRankNormal || v.Rank == BanRankWarn {
				v.Score -= i.ScoreReducePerMinute
				if v.Score <= 0 {
					// 小于0之后就移除掉
					toDelete = append(toDelete, k)
				}
				v.UpdatedAt = time.Now().Unix()
			}
			return true
		})
		for _, j := range toDelete {
			i.Map.Delete(j)
			_ = model.BanItemDel(d.DBData, j)
		}

		d.BanList.SaveChanged(d)
	})
}

// AddScoreBase
// 这一份ctx有endpoint就行
func (i *BanListInfo) AddScoreBase(uid string, score int64, place string, reason string, ctx *MsgContext) *BanListInfoItem {
	log := i.Parent.Logger
	v, _ := i.Map.Load(uid)
	if v == nil {
		v = &BanListInfoItem{
			ID:      uid,
			Reasons: []string{},
			Places:  []string{},
		}
	}

	v.Score += score
	v.Name = i.Parent.Parent.TryGetUserName(uid)
	if strings.Contains(uid, "-Group:") {
		v.Name = i.Parent.Parent.TryGetGroupName(uid)
	}
	v.Places = append(v.Places, place)
	v.Reasons = append(v.Reasons, reason)
	v.Times = append(v.Times, time.Now().Unix())
	oldRank := v.Rank

	switch v.Rank {
	case BanRankTrusted, BanRankBanned: /* no-op */
	case BanRankNormal, BanRankWarn:
		if v.Score < i.ThresholdWarn {
			v.Rank = BanRankNormal
		}

		if v.Score >= i.ThresholdWarn {
			v.Rank = BanRankWarn
		}
		if v.Score >= i.ThresholdBan {
			v.Rank = BanRankBanned
			v.BanTime = time.Now().Unix()

			if ctx.EndPoint.Platform == "QQ" {
				switch adapter := ctx.EndPoint.Adapter.(type) {
				case *PlatformAdapterGocq:
					adapter.DeleteFriend(ctx, place)
				case *PlatformAdapterWalleQ:
					adapter.DeleteFriend(ctx, place)
				case *PlatformAdapterRed:
					log.Warn("qq red 适配器不支持删除好友")
				case *PlatformAdapterOfficialQQ:
					log.Warn("official qq 适配器不支持删除好友")
				default:
					log.Error("unknown qq adapter")
				}
			}
		}

		if oldRank != v.Rank {
			v.BanUpdatedAt = time.Now().Unix()
		}
	}

	v.UpdatedAt = time.Now().Unix()
	i.Map.Store(uid, v)

	// 发送通知
	if ctx != nil {
		// 警告: XXX 因为等行为，进入警告列表
		// 黑名单: XXX 因为等行为，进入黑名单。将作出以下惩罚：拒绝回复、拒绝邀请、退出事发群
		// TODO
		fmt.Println("TODO Alert")
	}

	return v
}

// 返回连带责任人
func (i *BanListInfo) addJointScore(_ string, score int64, place string, reason string, ctx *MsgContext) (string, BanRankType) {
	d := i.Parent
	if i.JointScorePercentOfGroup > 0 {
		score := i.JointScorePercentOfGroup * float64(score)
		i.AddScoreBase(place, int64(score), place, reason, ctx)
	}
	if i.JointScorePercentOfInviter > 0 {
		group := d.ImSession.ServiceAtNew[place]
		if group != nil && group.InviteUserID != "" {
			rank := i.NoticeCheckPrepare(group.InviteUserID)
			score := i.JointScorePercentOfInviter * float64(score)
			i.AddScoreBase(group.InviteUserID, int64(score), place, reason, ctx)

			// text := fmt.Sprintf("提醒: 你邀请的骰子在群组<%s>中被禁言/踢出/指令刷屏了", group.GroupName)
			// ReplyPersonRaw(ctx, &Message{Sender: SenderBase{UserId: group.InviteUserId}}, text, "")
			return group.InviteUserID, rank
		}
	}
	return "", BanRankNormal
}

func (i *BanListInfo) NoticeCheckPrepare(uid string) BanRankType {
	item := i.GetByID(uid)
	if item != nil {
		return item.Rank
	}
	return BanRankNormal
}

func (i *BanListInfo) NoticeCheck(uid string, place string, oldRank BanRankType, ctx *MsgContext) BanRankType {
	log := i.Parent.Logger
	item := i.GetByID(uid)
	if item == nil {
		return 0
	}

	curRank := item.Rank
	if oldRank == curRank || !(curRank == BanRankWarn || curRank == BanRankBanned) {
		return 0
	}

	txt := fmt.Sprintf("黑名单等级提升: %v", item.toText(i.Parent))
	log.Info(txt)

	if ctx != nil {
		// 做出通知
		ctx.Notice(txt)

		// 发给当事人
		ReplyPersonRaw(ctx, &Message{Sender: SenderBase{UserID: uid}}, "提醒：你引发了黑名单事件:\n"+txt, "")

		// 发给当事群
		time.Sleep(1 * time.Second)
		ReplyGroupRaw(ctx, &Message{GroupID: place}, "提醒: 当前群组发生黑名单事件\n"+txt, "")

		// 发给邀请者
		time.Sleep(1 * time.Second)
		group := i.Parent.ImSession.ServiceAtNew[place]
		if group != nil && group.InviteUserID != "" {
			text := fmt.Sprintf("提醒: 你邀请的骰子在群组<%s>(%s)中遭遇黑名单事件:\n%v", group.GroupName, group.GroupID, txt)
			ReplyPersonRaw(ctx, &Message{Sender: SenderBase{UserID: group.InviteUserID}}, text, "")
		}
	}

	if curRank == BanRankBanned {
		if i.BanBehaviorQuitLastPlace {
			if ctx != nil {
				var isWhiteGroup bool
				d := ctx.Dice
				value, exists := d.BanList.Map.Load(place)
				if exists {
					if value.Rank == BanRankTrusted {
						isWhiteGroup = true
					}
				}

				if isWhiteGroup {
					d.Logger.Infof("群<%s>触发“退出事发群”的拉黑惩罚，但该群是信任群所以未退群", place)
				} else {
					ReplyGroupRaw(ctx, &Message{GroupID: place}, "因拉黑惩罚选项中有“退出事发群”，即将自动退群。", "")
					time.Sleep(1 * time.Second)
					ctx.EndPoint.Adapter.QuitGroup(ctx, place)
				}
			}
		}
	}
	return 0
}

// AddScoreByGroupMuted 群组禁言
func (i *BanListInfo) AddScoreByGroupMuted(uid string, place string, ctx *MsgContext) {
	rank := i.NoticeCheckPrepare(uid)

	i.AddScoreBase(uid, i.ScoreGroupMuted, place, "禁言骰子", ctx)
	inviterID, inviterRank := i.addJointScore(uid, i.ScoreGroupMuted, place, "连带责任:禁言骰子", ctx)

	i.NoticeCheck(uid, place, rank, ctx)
	if inviterID != "" && inviterID != uid {
		// 如果连带责任人与操作者不是同一人，进行单独计算
		i.NoticeCheck(inviterID, place, inviterRank, ctx)
	}
}

// AddScoreByGroupKicked 群组踢出
func (i *BanListInfo) AddScoreByGroupKicked(uid string, place string, ctx *MsgContext) {
	rank := i.NoticeCheckPrepare(uid)

	i.AddScoreBase(uid, i.ScoreGroupKicked, place, "踢出骰子", ctx)
	inviterID, inviterRank := i.addJointScore(uid, i.ScoreGroupKicked, place, "连带责任:踢出骰子", ctx)

	i.NoticeCheck(uid, place, rank, ctx)
	if inviterID != "" && inviterID != uid {
		// 如果连带责任人与操作者不是同一人，进行单独计算
		i.NoticeCheck(inviterID, place, inviterRank, ctx)
	}
}

// AddScoreByCommandSpam 指令刷屏
func (i *BanListInfo) AddScoreByCommandSpam(uid string, place string, ctx *MsgContext) {
	rank := i.NoticeCheckPrepare(uid)

	i.AddScoreBase(uid, i.ScoreTooManyCommand, place, "指令刷屏", ctx)
	inviterID, inviterRank := i.addJointScore(uid, i.ScoreTooManyCommand, place, "连带责任:指令刷屏", ctx)

	i.NoticeCheck(uid, place, rank, ctx)
	if inviterID != "" && inviterID != uid {
		// 如果连带责任人与操作者不是同一人，进行单独计算
		i.NoticeCheck(inviterID, place, inviterRank, ctx)
	}
}

// AddScoreByCensor 敏感词审查
func (i *BanListInfo) AddScoreByCensor(uid string, score int64, place string, level string, ctx *MsgContext) {
	rank := i.NoticeCheckPrepare(uid)

	i.AddScoreBase(uid, score, place, "触发<"+level+">敏感词", ctx)
	inviterID, inviterRank := i.addJointScore(uid, score, place, "连带责任:触发<"+level+">敏感词", ctx)

	i.NoticeCheck(uid, place, rank, ctx)
	if inviterID != "" && inviterID != uid {
		// 如果连带责任人与操作者不是同一人，进行单独计算
		i.NoticeCheck(inviterID, place, inviterRank, ctx)
	}
}

func (i *BanListInfo) GetByID(uid string) *BanListInfoItem {
	v, _ := i.Map.Load(uid)
	return v
}

func (i *BanListInfo) SetTrustByID(uid string, place string, reason string) {
	v := i.GetByID(uid)
	if v == nil {
		v = &BanListInfoItem{
			ID:      uid,
			Reasons: []string{},
			Places:  []string{},
		}
	}
	v.Rank = BanRankTrusted
	v.Name = i.Parent.Parent.TryGetUserName(uid)
	if strings.Contains(uid, "-Group:") {
		v.Name = i.Parent.Parent.TryGetGroupName(uid)
	}
	v.Places = append(v.Places, place)
	v.Reasons = append(v.Reasons, reason)
	v.Times = append(v.Times, time.Now().Unix())

	v.UpdatedAt = time.Now().Unix()
	i.Map.Store(uid, v)
}

func (d *Dice) GetBanList() []*BanListInfoItem {
	var lst []*BanListInfoItem
	_ = model.BanItemList(d.DBData, func(id string, banUpdatedAt int64, data []byte) {
		var v BanListInfoItem
		err := json.Unmarshal(data, &v)
		if err != nil {
			v.BanUpdatedAt = banUpdatedAt
		}
		lst = append(lst, &v)
	})
	return lst
}

func (i *BanListInfo) SaveChanged(d *Dice) {
	d.BanList.Map.Range(func(k string, v *BanListInfoItem) bool {
		if v.UpdatedAt != 0 {
			data, err := json.Marshal(v)
			if err == nil {
				_ = model.BanItemSave(d.DBData, k, v.UpdatedAt, v.BanUpdatedAt, data)
				v.UpdatedAt = 0
			}
		}
		return true
	})
}

func (i *BanListInfo) DeleteByID(d *Dice, id string) {
	i.Map.Delete(id)
	_ = model.BanItemDel(d.DBData, id)
}
