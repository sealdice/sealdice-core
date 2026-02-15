package dice

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/robfig/cron/v3"

	"sealdice-core/dice/service"
	"sealdice-core/model"
)

type BanRankType int

const (
	BanRankBanned  BanRankType = -30
	BanRankWarn    BanRankType = -10
	BanRankNormal  BanRankType = 0
	BanRankTrusted BanRankType = 30
)

type BanListInfoItem struct {
	ID      string      `jsbind:"id"      json:"ID"`
	Name    string      `jsbind:"name"    json:"name"`
	Score   int64       `jsbind:"score"   json:"score"`   // 怒气值
	Rank    BanRankType `jsbind:"rank"    json:"rank"`    // 0 没事 -10警告 -30禁止 30信任
	Times   []int64     `jsbind:"times"   json:"times"`   // 事发时间
	Reasons []string    `jsbind:"reasons" json:"reasons"` // 拉黑原因
	Places  []string    `jsbind:"places"  json:"places"`  // 发生地点
	BanTime int64       `jsbind:"banTime" json:"banTime"` // 上黑名单时间

	BanUpdatedAt int64 `json:"-"` // 排序依据，不过可能和bantime重复？
	UpdatedAt    int64 `json:"-"` // 数据更新时间
}

var BanRankText = map[BanRankType]string{
	BanRankTrusted: "信任",
	BanRankBanned:  "禁止",
	BanRankWarn:    "警告",
	BanRankNormal:  "常规",
}

// BanScoreChangeType 怒气值变更类型
const (
	BanScoreChangeTypeCensor = "censor" // 敏感词
	BanScoreChangeTypeMuted  = "muted"  // 禁言
	BanScoreChangeTypeKicked = "kicked" // 踢出
	BanScoreChangeTypeSpam   = "spam"   // 刷屏
	BanScoreChangeTypeManual = "manual" // 手动
	BanScoreChangeTypeJoint  = "joint"  // 连带责任
)

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
	Parent                                 *Dice                              `json:"-"                                      yaml:"-"`
	Map                                    *SyncMap[string, *BanListInfoItem] `json:"-"                                      yaml:"-"`
	BanBehaviorRefuseReply                 bool                               `json:"banBehaviorRefuseReply"                 yaml:"banBehaviorRefuseReply"`                 // 拉黑行为: 拒绝回复
	BanBehaviorRefuseInvite                bool                               `json:"banBehaviorRefuseInvite"                yaml:"banBehaviorRefuseInvite"`                // 拉黑行为: 拒绝邀请
	BanBehaviorQuitLastPlace               bool                               `json:"banBehaviorQuitLastPlace"               yaml:"banBehaviorQuitLastPlace"`               // 拉黑行为: 退出事发群
	BanBehaviorQuitPlaceImmediately        bool                               `json:"banBehaviorQuitPlaceImmediately"        yaml:"banBehaviorQuitPlaceImmediately"`        // 拉黑行为: 使用时立即退出群
	BanBehaviorQuitIfAdmin                 bool                               `json:"banBehaviorQuitIfAdmin"                 yaml:"banBehaviorQuitIfAdmin"`                 // 拉黑行为: 邀请者以上权限使用时立即退群，否则发出警告信息
	BanBehaviorQuitIfAdminSilentIfNotAdmin bool                               `json:"banBehaviorQuitIfAdminSilentIfNotAdmin" yaml:"banBehaviorQuitIfAdminSilentIfNotAdmin"` // 拉黑行为: 邀请者以上权限使用时立即退群，否则仅拒绝回复
	ThresholdWarn                          int64                              `json:"thresholdWarn"                          yaml:"thresholdWarn"`                          // 警告阈值
	ThresholdBan                           int64                              `json:"thresholdBan"                           yaml:"thresholdBan"`                           // 错误阈值
	AutoBanMinutes                         int64                              `json:"autoBanMinutes"                         yaml:"autoBanMinutes"`                         // 自动禁止时长

	ScoreReducePerMinute int64 `json:"scoreReducePerMinute" yaml:"scoreReducePerMinute"` // 每分钟下降
	ScoreGroupMuted      int64 `json:"scoreGroupMuted"      yaml:"scoreGroupMuted"`      // 群组禁言
	ScoreGroupKicked     int64 `json:"scoreGroupKicked"     yaml:"scoreGroupKicked"`     // 群组踢出
	ScoreTooManyCommand  int64 `json:"scoreTooManyCommand"  yaml:"scoreTooManyCommand"`  // 刷指令

	JointScorePercentOfGroup   float64 `json:"jointScorePercentOfGroup"   yaml:"jointScorePercentOfGroup"`   // 群组连带责任
	JointScorePercentOfInviter float64 `json:"jointScorePercentOfInviter" yaml:"jointScorePercentOfInviter"` // 邀请人连带责任

	BanNotifyIntervalMinutes int64 `json:"banNotifyIntervalMinutes" yaml:"banNotifyIntervalMinutes"` // 黑名单警告间隔(分钟)，0表示每次都警告

	cronID         cron.EntryID
	banNotifyCache *SyncMap[string, int64] `json:"-" yaml:"-"` // 黑名单警告缓存 key: "groupID-userID", value: 上次警告时间戳
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
	i.BanNotifyIntervalMinutes = 0 // 0=默认(20分钟), -1=每次都警告, >0=自定义分钟数
	i.Map = new(SyncMap[string, *BanListInfoItem])
	i.banNotifyCache = new(SyncMap[string, int64])
}

func (i *BanListInfo) Loads() {
}

func (i *BanListInfo) AfterLoads() {
	// 加载完成了
	d := i.Parent
	i.cronID, _ = d.Parent.Cron.AddFunc("@every 1m", func() {
		if d.DBOperator == nil {
			return
		}
		var toDelete []string
		(&d.Config).BanList.Map.Range(func(k string, v *BanListInfoItem) bool {
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
			_ = service.BanItemDel(d.DBOperator, j)
		}

		(&d.Config).BanList.SaveChanged(d)
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
		//nolint:forbidigo // that is a todo
		fmt.Println("TODO Alert")
	}

	return v
}

// 返回连带责任人
func (i *BanListInfo) addJointScore(_ string, score int64, place string, reason string, ctx *MsgContext) (string, BanRankType) {
	d := i.Parent
	if i.JointScorePercentOfGroup > 0 {
		groupRank := i.NoticeCheckPrepare(place)
		jointScore := int64(i.JointScorePercentOfGroup * float64(score))
		groupItem := i.AddScoreBase(place, jointScore, place, reason, ctx)
		if groupItem != nil {
			i.LogScoreChange(place, groupItem.Name, place, jointScore, groupItem.Score, reason, groupRank, groupItem.Rank, &BanScoreLogInfo{
				ChangeType: BanScoreChangeTypeJoint,
			})
		}
	}
	if i.JointScorePercentOfInviter > 0 {
		groupInfo, ok := d.ImSession.ServiceAtNew.Load(place)
		if ok && groupInfo.InviteUserID != "" {
			rank := i.NoticeCheckPrepare(groupInfo.InviteUserID)
			jointScore := int64(i.JointScorePercentOfInviter * float64(score))
			inviterItem := i.AddScoreBase(groupInfo.InviteUserID, jointScore, place, reason, ctx)
			if inviterItem != nil {
				i.LogScoreChange(groupInfo.InviteUserID, inviterItem.Name, place, jointScore, inviterItem.Score, reason, rank, inviterItem.Rank, &BanScoreLogInfo{
					ChangeType: BanScoreChangeTypeJoint,
				})
			}

			// text := fmt.Sprintf("提醒: 你邀请的骰子在群组<%s>中被禁言/踢出/指令刷屏了", groupInfo.GroupName)
			// ReplyPersonRaw(ctx, &Message{Sender: SenderBase{UserId: groupInfo.InviteUserId}}, text, "")
			return groupInfo.InviteUserID, rank
		}
	}
	return "", BanRankNormal
}

func (i *BanListInfo) NoticeCheckPrepare(uid string) BanRankType {
	item, ok := i.GetByID(uid)
	if ok {
		return item.Rank
	}
	return BanRankNormal
}

func (i *BanListInfo) NoticeCheck(uid string, place string, oldRank BanRankType, ctx *MsgContext) BanRankType {
	log := i.Parent.Logger
	item, ok := i.GetByID(uid)
	if !ok {
		return 0
	}

	curRank := item.Rank
	if oldRank == curRank || (curRank != BanRankWarn && curRank != BanRankBanned) {
		return 0
	}

	txt := fmt.Sprintf("黑名单等级提升: %v", item.toText(i.Parent))
	log.Info(txt)

	// If user is banned and we should quit immediately, do it first before sending notifications
	if curRank == BanRankBanned && i.BanBehaviorQuitLastPlace && ctx != nil {
		var isWhiteGroup bool
		d := ctx.Dice
		value, exists := (&d.Config).BanList.Map.Load(place)
		if exists {
			if value.Rank == BanRankTrusted {
				isWhiteGroup = true
			}
		}

		if !isWhiteGroup {
			// Quit group immediately before sending other notifications to avoid spam
			ReplyGroupRaw(ctx,
				&Message{GroupID: place},
				DiceFormatTmpl(ctx, "核心:黑名单惩罚_退群"),
				"")
			time.Sleep(1 * time.Second)
			ctx.EndPoint.Adapter.QuitGroup(ctx, place)
			return 0
		}
		d.Logger.Infof("群<%s>触发\"退出事发群\"的拉黑惩罚，但该群是信任群所以未退群", place)
	}

	if ctx != nil {
		// 做出通知
		ctx.Notice(txt)

		if ctx.Player == nil {
			ctx.Player = &GroupPlayerInfo{} // 为了能存 $t 变量，God bless this design
		}

		VarSetValueStr(ctx, "$t黑名单事件", txt)

		// 发给当事人
		ReplyPersonRaw(ctx,
			&Message{Sender: SenderBase{UserID: uid}},
			DiceFormatTmpl(ctx, "核心:黑名单触发_当事人"),
			"")

		// 发给当事群
		time.Sleep(1 * time.Second)
		ReplyGroupRaw(ctx,
			&Message{GroupID: place},
			DiceFormatTmpl(ctx, "核心:黑名单触发_所在群"),
			"")

		// 发给邀请者
		time.Sleep(1 * time.Second)
		groupInfo, ok := i.Parent.ImSession.ServiceAtNew.Load(place)
		if ok && groupInfo.InviteUserID != "" {
			VarSetValueStr(ctx, "$t事发群名", groupInfo.GroupName)
			VarSetValueStr(ctx, "$t事发群号", groupInfo.GroupID)
			text := DiceFormatTmpl(ctx, "核心:黑名单触发_邀请人")
			ReplyPersonRaw(ctx, &Message{Sender: SenderBase{UserID: groupInfo.InviteUserID}}, text, "")
		}
	}

	return 0
}

// AddScoreByGroupMuted 群组禁言
func (i *BanListInfo) AddScoreByGroupMuted(uid string, place string, ctx *MsgContext) {
	rank := i.NoticeCheckPrepare(uid)

	item := i.AddScoreBase(uid, i.ScoreGroupMuted, place, "禁言骰子", ctx)
	if item != nil {
		i.LogScoreChange(uid, item.Name, place, i.ScoreGroupMuted, item.Score, "禁言骰子", rank, item.Rank, &BanScoreLogInfo{
			ChangeType: BanScoreChangeTypeMuted,
		})
	}
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

	item := i.AddScoreBase(uid, i.ScoreGroupKicked, place, "踢出骰子", ctx)
	if item != nil {
		i.LogScoreChange(uid, item.Name, place, i.ScoreGroupKicked, item.Score, "踢出骰子", rank, item.Rank, &BanScoreLogInfo{
			ChangeType: BanScoreChangeTypeKicked,
		})
	}
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

	item := i.AddScoreBase(uid, i.ScoreTooManyCommand, place, "指令刷屏", ctx)
	if item != nil {
		i.LogScoreChange(uid, item.Name, place, i.ScoreTooManyCommand, item.Score, "指令刷屏", rank, item.Rank, &BanScoreLogInfo{
			ChangeType: BanScoreChangeTypeSpam,
		})
	}
	inviterID, inviterRank := i.addJointScore(uid, i.ScoreTooManyCommand, place, "连带责任:指令刷屏", ctx)

	i.NoticeCheck(uid, place, rank, ctx)
	if inviterID != "" && inviterID != uid {
		// 如果连带责任人与操作者不是同一人，进行单独计算
		i.NoticeCheck(inviterID, place, inviterRank, ctx)
	}
}

// AddScoreByCensor 敏感词审查
func (i *BanListInfo) AddScoreByCensor(uid string, score int64, place string, level string, ctx *MsgContext) {
	i.AddScoreByCensorWithWords(uid, score, place, level, nil, ctx)
}

// AddScoreByCensorWithWords 敏感词审查(带违禁词列表)
func (i *BanListInfo) AddScoreByCensorWithWords(uid string, score int64, place string, level string, words []string, ctx *MsgContext) {
	rank := i.NoticeCheckPrepare(uid)

	reason := "触发<" + level + ">敏感词"
	if len(words) > 0 {
		reason = "触发<" + level + ">敏感词: " + strings.Join(words, ", ")
	}
	item := i.AddScoreBase(uid, score, place, reason, ctx)
	if item != nil {
		i.LogScoreChange(uid, item.Name, place, score, item.Score, reason, rank, item.Rank, &BanScoreLogInfo{
			ChangeType:  BanScoreChangeTypeCensor,
			CensorWords: words,
			CensorLevel: level,
		})
	}
	inviterID, inviterRank := i.addJointScore(uid, score, place, "连带责任:触发<"+level+">敏感词", ctx)

	i.NoticeCheck(uid, place, rank, ctx)
	if inviterID != "" && inviterID != uid {
		// 如果连带责任人与操作者不是同一人，进行单独计算
		i.NoticeCheck(inviterID, place, inviterRank, ctx)
	}
}

func (i *BanListInfo) GetByID(uid string) (*BanListInfoItem, bool) {
	if uid == "" {
		return nil, false
	}
	return i.Map.Load(uid)
}

func (i *BanListInfo) SetTrustByID(uid string, place string, reason string) {
	v, ok := i.GetByID(uid)
	if !ok {
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
	_ = service.BanItemList(d.DBOperator, func(id string, banUpdatedAt int64, data []byte) {
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
	(&d.Config).BanList.Map.Range(func(k string, v *BanListInfoItem) bool {
		if v.UpdatedAt != 0 {
			data, err := json.Marshal(v)
			if err == nil {
				_ = service.BanItemSave(d.DBOperator, k, v.UpdatedAt, v.BanUpdatedAt, data)
				v.UpdatedAt = 0
			}
		}
		return true
	})
}

func (i *BanListInfo) DeleteByID(d *Dice, id string) {
	i.Map.Delete(id)
	_ = service.BanItemDel(d.DBOperator, id)
}

// BanScoreLogInfo 怒气值变更日志信息
type BanScoreLogInfo struct {
	ChangeType  string   // 变更类型
	CensorWords []string // 触发的违禁词(仅敏感词类型使用)
	CensorLevel string   // 违禁词等级(仅敏感词类型使用)
}

// LogScoreChange 记录怒气值变更日志
func (i *BanListInfo) LogScoreChange(uid string, userName string, groupID string, score int64, scoreAfter int64, reason string, oldRank BanRankType, newRank BanRankType, info *BanScoreLogInfo) {
	d := i.Parent
	if d == nil || d.DBOperator == nil {
		return
	}

	log := d.Logger
	isWarning := oldRank != BanRankWarn && newRank == BanRankWarn
	isBanned := oldRank != BanRankBanned && newRank == BanRankBanned

	// 记录到日志
	if info != nil && info.ChangeType == BanScoreChangeTypeCensor && len(info.CensorWords) > 0 {
		log.Infof("怒气值变更[敏感词]: 用户<%s>(%s) 在群<%s> 触发<%s>级违禁词%v, 增加%d分, 当前%d分, 等级: %s->%s",
			userName, uid, groupID, info.CensorLevel, info.CensorWords, score, scoreAfter,
			BanRankText[oldRank], BanRankText[newRank])
	} else {
		log.Infof("怒气值变更: 用户<%s>(%s) 在群<%s> 因<%s>, 增加%d分, 当前%d分, 等级: %s->%s",
			userName, uid, groupID, reason, score, scoreAfter,
			BanRankText[oldRank], BanRankText[newRank])
	}

	if isWarning {
		log.Warnf("警告触发: 用户<%s>(%s) 怒气值达到警告阈值, 当前%d分", userName, uid, scoreAfter)
	}
	if isBanned {
		log.Warnf("黑名单触发: 用户<%s>(%s) 怒气值达到黑名单阈值, 当前%d分", userName, uid, scoreAfter)
	}

	// 构建敏感词JSON
	var censorWordsJSON string
	if info != nil && len(info.CensorWords) > 0 {
		data, err := json.Marshal(info.CensorWords)
		if err == nil {
			censorWordsJSON = string(data)
		}
	}

	// 变更类型
	changeType := BanScoreChangeTypeManual
	censorLevel := ""
	if info != nil {
		changeType = info.ChangeType
		censorLevel = info.CensorLevel
	}

	// 保存到数据库
	logEntry := &model.BanScoreLog{
		UserID:      uid,
		UserName:    userName,
		GroupID:     groupID,
		Score:       score,
		ScoreAfter:  scoreAfter,
		Reason:      reason,
		RankBefore:  int(oldRank),
		RankAfter:   int(newRank),
		ChangeType:  changeType,
		CensorWords: censorWordsJSON,
		CensorLevel: censorLevel,
		IsWarning:   isWarning,
		IsBanned:    isBanned,
	}

	if err := service.BanScoreLogAppend(d.DBOperator, logEntry); err != nil {
		log.Errorf("保存怒气值变更日志失败: %v", err)
	}
}

// ShouldNotifyBan 检查是否应该发送黑名单警告（基于冷却时间）
// 返回 true 表示应该警告，false 表示在冷却中不警告
// BanNotifyIntervalMinutes: -1=每次都警告, 0=默认(20分钟), >0=自定义分钟数
func (i *BanListInfo) ShouldNotifyBan(groupID string, userID string) bool {
	// -1 表示每次都警告（旧行为）
	if i.BanNotifyIntervalMinutes < 0 {
		return true
	}

	// 确定实际间隔：0使用默认值20分钟，否则使用配置值
	intervalMinutes := i.BanNotifyIntervalMinutes
	if intervalMinutes == 0 {
		intervalMinutes = 20 // 默认20分钟
	}

	// 确保缓存已初始化
	if i.banNotifyCache == nil {
		i.banNotifyCache = new(SyncMap[string, int64])
	}

	cacheKey := groupID + "-" + userID
	now := time.Now().Unix()
	intervalSeconds := intervalMinutes * 60

	if lastTime, exists := i.banNotifyCache.Load(cacheKey); exists {
		if now-lastTime < intervalSeconds {
			return false // 冷却中，不警告
		}
	}

	i.banNotifyCache.Store(cacheKey, now)
	return true // 需要警告
}
