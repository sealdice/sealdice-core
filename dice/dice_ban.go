package dice

import (
	"encoding/json"
	"fmt"
	"github.com/fy0/lockfree"
	"github.com/robfig/cron/v3"
	"strings"
	"time"
)

type BanRankType int

const (
	BanRankBanned  BanRankType = -30
	BanRankWarn    BanRankType = -10
	BanRankNormal  BanRankType = 0
	BanRankTrusted BanRankType = 30
)

type BanListInfoItem struct {
	ID      string      `json:"ID"`
	Name    string      `json:"name"`
	Score   int64       `json:"score"`
	Rank    BanRankType `json:"rank"`    // 0 没事 -10警告 -30禁止 30信任
	Times   []int64     `json:"times"`   // 事发时间
	Reasons []string    `json:"reasons"` // 拉黑原因
	Places  []string    `json:"places"`  // 发生地点
}

var BanRankText = map[BanRankType]string{
	BanRankTrusted: "信任",
	BanRankBanned:  "禁止",
	BanRankWarn:    "警告",
	BanRankNormal:  "常规",
}

func (i *BanListInfoItem) toText(d *Dice) string {
	prefix := BanRankText[i.Rank]
	if i.Rank == -10 || i.Rank == -30 {
		return fmt.Sprintf("[%s]%s <%s> 原因: %s", prefix, i.ID, i.Name, strings.Join(i.Reasons, ","))
	}
	if i.Rank == 30 {
		return fmt.Sprintf("[%s]%s <%s> 原因: %s", prefix, i.ID, i.Name, strings.Join(i.Reasons, ","))
	}
	return ""
}

type BanListInfo struct {
	Parent                   *Dice            `yaml:"-" json:"-"`
	Map                      lockfree.HashMap `yaml:"-" json:"-"`
	BanBehaviorRefuseReply   bool             `yaml:"banBehaviorRefuseReply" json:"banBehaviorRefuseReply"`     //拉黑行为: 拒绝回复
	BanBehaviorRefuseInvite  bool             `yaml:"banBehaviorRefuseInvite" json:"banBehaviorRefuseInvite"`   // 拉黑行为: 拒绝邀请
	BanBehaviorQuitLastPlace bool             `yaml:"banBehaviorQuitLastPlace" json:"banBehaviorQuitLastPlace"` // 拉黑行为: 退出事发群
	ReducePerMinute          int              `yaml:"reducePerMinute" json:"reducePerMinute"`                   // 每分钟下降
	ThresholdWarn            int64            `yaml:"thresholdWarn" json:"thresholdWarn"`                       // 警告阈值
	ThresholdBan             int64            `yaml:"thresholdBan" json:"thresholdBan"`                         // 错误阈值
	cronId                   cron.EntryID
}

func (i *BanListInfo) Init() {
	// 此为配置装载前的默认设置
	i.BanBehaviorRefuseReply = true
	i.BanBehaviorRefuseInvite = true
	i.BanBehaviorQuitLastPlace = false
	i.ReducePerMinute = 1
	i.ThresholdWarn = 100
	i.ThresholdBan = 200
	i.Map = lockfree.NewHashMap()
}

func (i *BanListInfo) AfterLoads() {
	// 加载完成了
	d := i.Parent
	i.cronId, _ = d.Parent.Cron.AddFunc("@every 1m", func() {
		toDelete := []interface{}{}
		_ = i.Map.Iterate(func(_k interface{}, _v interface{}) error {
			v, ok := _v.(*BanListInfoItem)
			if ok {
				if v.Rank == BanRankNormal || v.Rank == BanRankWarn {
					v.Score -= 1
					if v.Score <= 0 {
						// 小于0之后就移除掉
						toDelete = append(toDelete, _k)
					}
				}
			}
			return nil
		})
		for _, j := range toDelete {
			i.Map.Del(j)
		}
	})
}

func (i *BanListInfo) AddScoreBase(uid string, score int64, place string, reason string, ctx *MsgContext) {
	var v *BanListInfoItem
	_v, exists := i.Map.Get(uid)
	if exists {
		v, _ = _v.(*BanListInfoItem)
	}
	if v == nil {
		v = &BanListInfoItem{
			ID:      uid,
			Reasons: []string{},
			Places:  []string{},
		}
	}

	v.Score += score
	v.Name = i.Parent.Parent.TryGetUserName(uid)
	v.Places = append(v.Places, place)
	v.Reasons = append(v.Reasons, reason)
	v.Times = append(v.Times, time.Now().Unix())

	switch v.Rank {
	case BanRankTrusted:
	// 信任用户，啥都不做
	case BanRankNormal, BanRankWarn:
		if v.Score < i.ThresholdWarn {
			v.Rank = BanRankNormal
		}

		if v.Score >= i.ThresholdWarn {
			v.Rank = BanRankWarn
		}
		if v.Score >= i.ThresholdBan {
			v.Rank = BanRankBanned
		}
	}

	i.Map.Set(uid, v)

	// 发送通知
	if ctx != nil {
		// 警告: XXX 因为等行为，进入警告列表
		// 黑名单: XXX 因为等行为，进入黑名单。将作出以下惩罚：拒绝回复、拒绝邀请、退出事发群
	}
}

func (i *BanListInfo) toJSON() []byte {
	data, err := json.Marshal(i)
	if err != nil {
		return nil
	}

	tmp := map[string]interface{}{}
	err = json.Unmarshal(data, &tmp)
	if err != nil {
		return nil
	}

	dict := map[string]*BanListInfoItem{}
	err = i.Map.Iterate(func(_k interface{}, _v interface{}) error {
		k, ok1 := _k.(string)
		v, ok2 := _v.(*BanListInfoItem)
		if ok1 && ok2 {
			dict[k] = v
		}
		return nil
	})
	if err != nil {
		return nil
	}

	tmp["map"] = dict
	marshal, err := json.Marshal(tmp)
	if err != nil {
		return nil
	}
	return marshal
}

func (i *BanListInfo) loadJSON(data []byte) {
	tmp := map[string]interface{}{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return
	}

	// 进行常规转换
	err = json.Unmarshal(data, &i)
	if err != nil {
		return
	}

	// 如果存在map进行转换
	if val, ok := tmp["map"]; ok {
		data2, err := json.Marshal(val)
		if err != nil {
			return
		}
		dict := map[string]*BanListInfoItem{}
		err = json.Unmarshal(data2, &dict)
		if err != nil {
			return
		}
		realDict := lockfree.NewHashMap()
		for k, v := range dict {
			realDict.Set(k, v)
		}
	}
}

func (i *BanListInfo) mapToJSON() []byte {
	dict := map[string]*BanListInfoItem{}
	err := i.Map.Iterate(func(_k interface{}, _v interface{}) error {
		k, ok1 := _k.(string)
		v, ok2 := _v.(*BanListInfoItem)
		if ok1 && ok2 {
			dict[k] = v
		}
		return nil
	})
	if err != nil {
		return nil
	}

	marshal, err := json.Marshal(dict)
	if err != nil {
		return nil
	}
	return marshal
}

func (i *BanListInfo) loadMapFromJSON(data []byte) {
	// 如果存在map进行转换
	dict := map[string]*BanListInfoItem{}
	err := json.Unmarshal(data, &dict)
	if err != nil {
		return
	}
	realDict := lockfree.NewHashMap()
	for k, v := range dict {
		realDict.Set(k, v)
	}
	i.Map = realDict
}

func (i *BanListInfo) GetById(uid string) *BanListInfoItem {
	var v *BanListInfoItem
	_v, exists := i.Map.Get(uid)
	if exists {
		v, _ = _v.(*BanListInfoItem)
	}
	return v
}

func (i *BanListInfo) SetTrustById(uid string) {
	v := i.GetById(uid)
	if v == nil {
		v = &BanListInfoItem{
			ID:      uid,
			Reasons: []string{},
			Places:  []string{},
		}
	}
	v.Rank = BanRankTrusted
	i.Map.Set(uid, v)
}
