package dice

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/rand"
	"regexp"
	"runtime/debug"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"sealdice-core/dice/events"
	"sealdice-core/dice/service"
	"sealdice-core/logger"
	"sealdice-core/message"
	"sealdice-core/model"
	"sealdice-core/utils/dboperator/engine"

	"github.com/golang-module/carbon"
	ds "github.com/sealdice/dicescript"
	rand2 "golang.org/x/exp/rand" //nolint:staticcheck // against my better judgment, but this was mandated due to a strongly held opinion from you know who

	"github.com/dop251/goja"
	"golang.org/x/time/rate"
	"gopkg.in/yaml.v3"
)

type SenderBase struct {
	Nickname  string `jsbind:"nickname" json:"nickname"`
	UserID    string `jsbind:"userId"   json:"userId"`
	GroupRole string `json:"-"` // 缇ゅ唴瑙掕壊 admin绠＄悊鍛?owner缇や富
}

// Message 娑堟伅鐨勯噸瑕佷俊鎭?
// 鏃堕棿
// 鍙戦€佸湴鐐?缇よ亰/绉佽亰)
// 浜虹墿(鏄皝鍙戠殑)
// 鍐呭
type Message struct {
	Time        int64       `jsbind:"time"        json:"time"`        // 鍙戦€佹椂闂?
	MessageType string      `jsbind:"messageType" json:"messageType"` // group private
	GroupID     string      `jsbind:"groupId"     json:"groupId"`     // 缇ゅ彿锛屽鏋滄槸缇よ亰娑堟伅
	GuildID     string      `jsbind:"guildId"     json:"guildId"`     // 鏈嶅姟鍣ㄧ兢缁勫彿锛屼細鍦╠iscord,kook,dodo绛夊钩鍙拌鍒?
	ChannelID   string      `jsbind:"channelId"   json:"channelId"`
	Sender      SenderBase  `jsbind:"sender"      json:"sender"`   // 鍙戦€佽€?
	Message     string      `jsbind:"message"     json:"message"`  // 娑堟伅鍐呭
	RawID       interface{} `jsbind:"rawId"       json:"rawId"`    // 鍘熷淇℃伅ID锛岀敤浜庡鐞嗘挙鍥炵瓑
	Platform    string      `jsbind:"platform"    json:"platform"` // 褰撳墠骞冲彴
	GroupName   string      `json:"groupName"`
	TmpUID      string      `json:"-"             yaml:"-"`
	// Note(Szzrain): 杩欓噷鏄秷鎭锛屼负浜嗘敮鎸佸绉嶆秷鎭被鍨嬶紝鐩墠鍙湁 Milky 鏀寔锛屽叾浠栧钩鍙颁篃搴旇灏藉揩杩佺Щ鏀寔锛屽苟浣跨敤 Session.ExecuteNew 鏂规硶
	Segment []message.IMessageElement `jsbind:"segment" json:"-" yaml:"-"`
}

// GroupPlayerInfo 杩欐槸涓€涓猋amlWrapper锛屾病鏈夊疄闄呬綔鐢?
// 鍘熷洜瑙?https://github.com/go-yaml/yaml/issues/712
// type GroupPlayerInfo struct {
// 	GroupPlayerInfoBase `yaml:",inline,flow"`
// }

type GroupPlayerInfo model.GroupPlayerInfoBase

type GroupInfo struct {
	Active    bool                               `jsbind:"active" json:"active" yaml:"active"` // 鏄惁鍦ㄧ兢鍐呭紑鍚?- 杩囨浮涓鸿薄寰佹剰涔?
	extInitMu sync.Mutex                         `json:"-" yaml:"-"`                           // 寤惰繜鍒濆鍖栭攣
	Players   *SyncMap[string, *GroupPlayerInfo] `json:"-" yaml:"-"`                           // 缇ゅ憳瑙掕壊鏁版嵁

	activatedExtList  []*ExtInfo // 褰撳墠缇ゅ紑鍚殑鎵╁睍鍒楄〃锛堢鏈夛紝閫氳繃 Getter 璁块棶锛岀敱 MarshalJSON/UnmarshalJSON 澶勭悊搴忓垪鍖栵級
	InactivatedExtSet StringSet  `json:"inactivatedExtSet" yaml:"inactivatedExtSet,flow"` // 鎵嬪姩鍏抽棴鎴栧皻鏈惎鐢ㄧ殑鎵╁睍

	GroupID         string                 `jsbind:"groupId"       json:"groupId"      yaml:"groupId"`
	GuildID         string                 `jsbind:"guildId"       json:"guildId"      yaml:"guildId"`
	ChannelID       string                 `jsbind:"channelId"     json:"channelId"    yaml:"channelId"`
	GroupName       string                 `jsbind:"groupName"     json:"groupName"    yaml:"groupName"`
	DiceIDActiveMap *SyncMap[string, bool] `json:"diceIdActiveMap" yaml:"diceIds,flow"` // 瀵瑰簲鐨勯瀛怚D(鏍煎紡 骞冲彴:ID)锛屽搴斿崟楠板鍙锋儏鍐碉紝渚嬪楠癆 B閮藉姞浜嗙兢Z锛孉閫€缇や笉浼氬奖鍝岯鍦ㄧ兢鍐呮湇鍔?
	DiceIDExistsMap *SyncMap[string, bool] `json:"diceIdExistsMap" yaml:"-"`            // 瀵瑰簲鐨勯瀛怚D(鏍煎紡 骞冲彴:ID)鏄惁瀛樺湪浜庣兢鍐?
	BotList         *SyncMap[string, bool] `json:"botList"         yaml:"botList,flow"` // 鍏朵粬楠板瓙鍒楄〃
	DiceSideNum     int64                  `json:"diceSideNum"     yaml:"diceSideNum"`  // 浠ュ悗鍙兘浼氭敮鎸?1d4 杩欑榛樿闈㈡暟锛屾殏涓嶅紑鏀剧粰js
	DiceSideExpr    string                 `json:"diceSideExpr"    yaml:"diceSideExpr"` //
	System          string                 `json:"system"          yaml:"system"`       // 瑙勫垯绯荤粺锛屾蹇靛悓bcdice鐨刧amesystem锛岃窛绂诲dnd5e coc7

	HelpPackages []string `json:"helpPackages"   yaml:"-"`
	CocRuleIndex int      `jsbind:"cocRuleIndex" json:"cocRuleIndex" yaml:"cocRuleIndex"`
	LogCurName   string   `jsbind:"logCurName"   json:"logCurName"   yaml:"logCurFile"`
	LogOn        bool     `jsbind:"logOn"        json:"logOn"        yaml:"logOn"`

	QuitMarkAutoClean   bool   `json:"-"                     yaml:"-"` // 鑷姩娓呯兢 - 鎾姤锛屽嵆灏嗚嚜鍔ㄩ€€鍑虹兢缁?
	QuitMarkMaster      bool   `json:"-"                     yaml:"-"` // 楠颁富鍛戒护閫€缇?- 鎾姤锛屽嵆灏嗚嚜鍔ㄩ€€鍑虹兢缁?
	RecentDiceSendTime  int64  `jsbind:"recentDiceSendTime"  json:"recentDiceSendTime"`
	ShowGroupWelcome    bool   `jsbind:"showGroupWelcome"    json:"showGroupWelcome"    yaml:"showGroupWelcome"` // 鏄惁杩庢柊
	GroupWelcomeMessage string `jsbind:"groupWelcomeMessage" json:"groupWelcomeMessage" yaml:"groupWelcomeMessage"`
	// FirstSpeechMade     bool   `yaml:"firstSpeechMade"` // 鏄惁鍋氳繃杩涚兢鍙戣█
	LastCustomReplyTime float64 `json:"-" yaml:"-"` // 涓婃鑷畾涔夊洖澶嶆椂闂?

	RateLimiter     *rate.Limiter `json:"-" yaml:"-"`
	RateLimitWarned bool          `json:"-" yaml:"-"`

	EnteredTime  int64  `jsbind:"enteredTime"  json:"enteredTime"  yaml:"enteredTime"`  // 鍏ョ兢鏃堕棿
	InviteUserID string `jsbind:"inviteUserId" json:"inviteUserId" yaml:"inviteUserId"` // 閭€璇蜂汉
	// 浠呯敤浜巋ttp鎺ュ彛
	TmpPlayerNum int64    `json:"tmpPlayerNum" yaml:"-"`
	TmpExtList   []string `json:"tmpExtList"   yaml:"-"`

	UpdatedAtTime int64 `json:"-" yaml:"-"`

	DefaultHelpGroup string `json:"defaultHelpGroup" yaml:"defaultHelpGroup"` // 褰撳墠缇ら粯璁ょ殑甯姪鏉＄洰

	PlayerGroups      *SyncMap[string, []string] `json:"playerGroups"      yaml:"playerGroups"` // 缁檛eam鎸囦护浣跨敤锛屽拰鐜╁銆佺兢绛変俊鎭竴鏍凤紝閮芥潵鑷狿layers锛屼笉浼氶噸澶嶅瓨鍌?
	ExtAppliedVersion int64                      `json:"extAppliedVersion" yaml:"extAppliedVersion"`

	/* Wrapper 鏋舵瀯 */
	ExtAppliedTime int64 `json:"-" yaml:"-"` // 缇ょ粍搴旂敤鎵╁睍鐨勬椂闂存埑锛岃繍琛屾椂浣跨敤锛屼笉搴忓垪鍖栵紙寮哄埗姣忔鍚姩閲嶆柊鍒濆鍖栵級
}

// GetActivatedExtList 鑾峰彇婵€娲荤殑鎵╁睍鍒楄〃锛岃嚜鍔ㄥ鐞嗗欢杩熷垵濮嬪寲
// 閫氳繃 ExtAppliedTime == 0 鍒ゆ柇鏄惁闇€瑕佸垵濮嬪寲
// 鍚屾椂澶勭悊鏂版墿灞曠殑寤惰繜婵€娲?
func (g *GroupInfo) GetActivatedExtList(d *Dice) []*ExtInfo {
	// 蹇€熻矾寰勶細宸插垵濮嬪寲
	if atomic.LoadInt64(&g.ExtAppliedTime) != 0 {
		g.extInitMu.Lock()
		list := g.activatedExtList
		g.extInitMu.Unlock()
		return list
	}
	g.extInitMu.Lock()
	defer g.extInitMu.Unlock()
	if atomic.LoadInt64(&g.ExtAppliedTime) != 0 {
		return g.activatedExtList // double-check
	}

	// 寤惰繜鍒濆鍖栵細鐢ㄥ叏灞€ ExtList 鏇挎崲鍙嶅簭鍒楀寲鐨勫崰浣嶅璞?
	extMap := make(map[string]*ExtInfo)
	for _, ext := range d.ExtList {
		extMap[ext.Name] = ext
	}

	oldCount := len(g.activatedExtList)
	var newList []*ExtInfo
	activated := make(map[string]bool)
	for _, item := range g.activatedExtList {
		if item != nil && extMap[item.Name] != nil {
			newList = append(newList, extMap[item.Name])
			activated[item.Name] = true
		}
	}

	// 寤惰繜婵€娲绘柊鎵╁睍锛氭鏌?ExtList 涓槸鍚︽湁鏂版墿灞曢渶瑕佹縺娲?
	// 鏂版墿灞?= 涓嶅湪 activatedExtList 涓紝涔熶笉鍦?InactivatedExtSet 涓?
	g.ensureInactivatedSet()
	newExtCount := 0
	for _, ext := range d.ExtList {
		if ext == nil {
			continue
		}
		// 璺宠繃宸叉縺娲荤殑鎵╁睍
		if activated[ext.Name] {
			continue
		}
		// 璺宠繃琚敤鎴锋墜鍔ㄥ叧闂殑鎵╁睍
		if g.IsExtInactivated(ext.Name) {
			continue
		}
		// 鏂版墿灞曪細鏍规嵁 AutoActive 鍐冲畾鏄惁婵€娲?
		if ext.AutoActive || (ext.DefaultSetting != nil && ext.DefaultSetting.AutoActive) {
			newList = append([]*ExtInfo{ext}, newList...) // 鎻掑叆澶撮儴
			activated[ext.Name] = true
			newExtCount++
		} else {
			g.AddToInactivated(ext.Name)
		}
	}

	g.activatedExtList = newList
	// 鏍囪宸插垵濮嬪寲锛岀‘淇濆€间笉涓?0锛堝惁鍒欎笅娆℃鏌ヤ細鍐嶆杩涘叆鍒濆鍖栵級
	appliedTime := d.ExtUpdateTime
	if appliedTime == 0 {
		appliedTime = 1
	}
	atomic.StoreInt64(&g.ExtAppliedTime, appliedTime)

	// 濡傛灉婵€娲讳簡鏂版墿灞曪紝鏍囪缇ょ粍涓?dirty
	if newExtCount > 0 {
		g.MarkDirty(d)
	}

	// 鎵撳嵃鍒濆鍖栨棩蹇?
	d.Logger.Infof("缇ょ粍鎵╁睍鍒濆鍖? %s, 鎵╁睍鏁?%d -> %d (鏂版縺娲?%d)", g.GroupID, oldCount, len(newList), newExtCount)
	return g.activatedExtList
}

// TriggerExtHook 閬嶅巻宸叉縺娲荤殑鎵╁睍骞惰Е鍙戦挬瀛?
// getHook 杩斿洖瑕佹墽琛岀殑鍑芥暟锛岃嫢杩斿洖 nil 鍒欒烦杩囪鎵╁睍
func (g *GroupInfo) TriggerExtHook(d *Dice, getHook func(*ExtInfo) func()) {
	for _, wrapper := range g.GetActivatedExtList(d) {
		ext := wrapper.GetRealExt()
		if ext == nil {
			continue
		}
		if hook := getHook(ext); hook != nil {
			ext.callWithJsCheck(d, hook)
		}
	}
}

// GetActivatedExtListRaw 鐩存帴璁块棶鎵╁睍鍒楄〃锛堢敤浜庡簭鍒楀寲銆佸唴閮ㄤ慨鏀圭瓑鍦烘櫙锛?
func (g *GroupInfo) GetActivatedExtListRaw() []*ExtInfo {
	g.extInitMu.Lock()
	defer g.extInitMu.Unlock()
	return g.activatedExtList
}

// SetActivatedExtList 璁剧疆鎵╁睍鍒楄〃锛堢敤浜庢柊缇ょ粍鍒涘缓绛夊満鏅級
func (g *GroupInfo) SetActivatedExtList(list []*ExtInfo, d *Dice) {
	g.extInitMu.Lock()
	defer g.extInitMu.Unlock()
	g.activatedExtList = list
	if d != nil {
		atomic.StoreInt64(&g.ExtAppliedTime, d.ExtUpdateTime) // 鏍囪宸插垵濮嬪寲
	} else {
		atomic.StoreInt64(&g.ExtAppliedTime, 1) // 娌℃湁 Dice 鏃惰缃潪闆跺€兼爣璁板凡鍒濆鍖?
	}
}

// groupInfoAlias 鐢ㄤ簬閬垮厤 MarshalJSON 閫掑綊璋冪敤
type groupInfoAlias GroupInfo

// groupInfoJSON 鐢ㄤ簬搴忓垪鍖?鍙嶅簭鍒楀寲 GroupInfo
// 鐢变簬 activatedExtList 鏄鏈夊瓧娈碉紝闇€瑕侀€氳繃姝ょ粨鏋勪綋澶勭悊
type groupInfoJSON struct {
	*groupInfoAlias
	ActivatedExtList []*ExtInfo `json:"activatedExtList"`
}

// MarshalJSON 鑷畾涔夊簭鍒楀寲锛屽鐞嗙鏈夊瓧娈?activatedExtList
// 鍚屾椂杩囨护鎺夊凡鍒犻櫎鐨?wrapper锛圛sDeleted=true锛?
func (g *GroupInfo) MarshalJSON() ([]byte, error) {
	g.extInitMu.Lock()
	// 杩囨护鎺夊凡鍒犻櫎鐨?wrapper
	var filteredList []*ExtInfo
	for _, ext := range g.activatedExtList {
		if ext != nil && !ext.IsDeleted {
			filteredList = append(filteredList, ext)
		}
	}
	g.extInitMu.Unlock()

	return json.Marshal(&groupInfoJSON{
		groupInfoAlias:   (*groupInfoAlias)(g),
		ActivatedExtList: filteredList,
	})
}

// UnmarshalJSON 鑷畾涔夊弽搴忓垪鍖栵紝澶勭悊绉佹湁瀛楁 activatedExtList
func (g *GroupInfo) UnmarshalJSON(data []byte) error {
	temp := &groupInfoJSON{
		groupInfoAlias: (*groupInfoAlias)(g),
	}
	if err := json.Unmarshal(data, temp); err != nil {
		return err
	}
	g.extInitMu.Lock()
	g.activatedExtList = temp.ActivatedExtList
	g.extInitMu.Unlock()
	return nil
}

// MarkDirty 鏍囪缇ょ粍涓鸿剰鏁版嵁锛岄渶瑕佷繚瀛樺埌鏁版嵁搴?
// 鍚屾椂灏嗙兢缁?ID 鍔犲叆鑴忓垪琛紝Save 鏃跺彧閬嶅巻鑴忓垪琛?
func (g *GroupInfo) MarkDirty(d *Dice) {
	now := time.Now().Unix()
	atomic.StoreInt64(&g.UpdatedAtTime, now)
	if d != nil && d.DirtyGroups != nil {
		d.DirtyGroups.Store(g.GroupID, now)
	}
}

func (group *GroupInfo) IsActive(ctx *MsgContext) bool {
	if strings.HasPrefix(group.GroupID, "UI-Group:") {
		return true
	}
	firstCheck := group.Active && group.DiceIDActiveMap.Len() >= 1
	if firstCheck {
		v, _ := group.DiceIDActiveMap.Load(ctx.EndPoint.UserID)
		return v
	}
	return false
}

func (group *GroupInfo) PlayerGet(operator engine.DatabaseOperator, id string) *GroupPlayerInfo {
	if group.Players == nil {
		group.Players = new(SyncMap[string, *GroupPlayerInfo])
	}
	p, exists := group.Players.Load(id)
	if !exists {
		basePtr := service.GroupPlayerInfoGet(operator, group.GroupID, id)
		p = (*GroupPlayerInfo)(basePtr)
		if p != nil {
			group.Players.Store(id, p)
		}
	}
	return p
}

// GetCharTemplate 杩欎釜鍑芥暟鏈€濂界粰ctx锛屽湪group涓嬩笉鍚堢悊锛屼紶鍏ice灏卞緢婊戠ń浜?
func (group *GroupInfo) GetCharTemplate(dice *Dice) *GameSystemTemplate {
	// 鏈塻ystem浼樺厛system
	if group.System != "" {
		v, _ := dice.GameSystemMap.Load(group.System)
		if v != nil {
			return v
		}
		// 杩斿洖杩欎釜鍗曠函鏄负浜嗕笉璁﹕t灏嗗叾瑕嗙洊
		// 杩欑鎯呭喌灞炰簬鍗＄墖鐨勮鍒欐ā鏉胯鍒犻櫎浜?
		tmpl := &GameSystemTemplate{
			GameSystemTemplateV2: &GameSystemTemplateV2{
				Name:     group.System,
				FullName: "绌虹櫧妯℃澘",
			},
		}
		tmpl.Init()
		return tmpl
	}
	// 娌℃湁system锛屾煡鐪嬫墿灞曠殑鍚姩鎯呭喌
	if group.ExtGetActive("coc7") != nil {
		v, _ := dice.GameSystemMap.Load("coc7")
		return v
	}

	if group.ExtGetActive("dnd5e") != nil {
		v, _ := dice.GameSystemMap.Load("dnd5e")
		return v
	}

	// 鍟ラ兘娌℃湁锛岃繑鍥炵┖锛岃繕鏄櫧鍗★紵
	// 杩斿洖涓┖鐧芥ā鏉垮ソ浜?
	blankTmpl := &GameSystemTemplate{
		GameSystemTemplateV2: &GameSystemTemplateV2{
			Name:     "绌虹櫧妯℃澘",
			FullName: "绌虹櫧妯℃澘",
		},
	}
	blankTmpl.Init()
	return blankTmpl
}

type EndpointState int

type EndPointInfoBase struct {
	ID                  string        `jsbind:"id"                  json:"id"                  yaml:"id"` // uuid
	Nickname            string        `jsbind:"nickname"            json:"nickname"            yaml:"nickname"`
	State               EndpointState `jsbind:"state"               json:"state"               yaml:"state"` // 鐘舵€?0鏂紑 1宸茶繛鎺?2杩炴帴涓?3杩炴帴澶辫触
	UserID              string        `jsbind:"userId"              json:"userId"              yaml:"userId"`
	GroupNum            int64         `jsbind:"groupNum"            json:"groupNum"            yaml:"groupNum"`            // 鎷ユ湁缇ゆ暟
	CmdExecutedNum      int64         `jsbind:"cmdExecutedNum"      json:"cmdExecutedNum"      yaml:"cmdExecutedNum"`      // 鎸囦护鎵ц娆℃暟
	CmdExecutedLastTime int64         `jsbind:"cmdExecutedLastTime" json:"cmdExecutedLastTime" yaml:"cmdExecutedLastTime"` // 鎸囦护鎵ц娆℃暟
	OnlineTotalTime     int64         `jsbind:"onlineTotalTime"     json:"onlineTotalTime"     yaml:"onlineTotalTime"`     // 鍦ㄧ嚎鏃堕暱

	Platform     string `jsbind:"platform"   json:"platform"     yaml:"platform"` // 骞冲彴锛屽QQ绛?
	RelWorkDir   string `json:"relWorkDir"   yaml:"relWorkDir"`                   // 宸ヤ綔鐩綍
	Enable       bool   `jsbind:"enable"     json:"enable"       yaml:"enable"`   // 鏄惁鍚敤
	ProtocolType string `json:"protocolType" yaml:"protocolType"`                 // 鍗忚绫诲瀷锛屽onebot銆乲oishi绛?

	IsPublic bool       `json:"isPublic" yaml:"isPublic"`
	Session  *IMSession `json:"-"        yaml:"-"`
}

const (
	StateDisconnected     EndpointState = iota // 0: 鏂紑
	StateConnected                             // 1: 宸茶繛鎺?
	StateConnecting                            // 2: 杩炴帴涓?
	StateConnectionFailed                      // 3: 杩炴帴澶辫触
)

type EndPointInfo struct {
	EndPointInfoBase `jsbind:"baseInfo" yaml:"baseInfo"`

	Adapter PlatformAdapter `json:"adapter" yaml:"adapter"`
}

func (ep *EndPointInfo) UnmarshalYAML(value *yaml.Node) error {
	if ep.Adapter != nil {
		return value.Decode(ep)
	}

	var val struct {
		EndPointInfoBase `yaml:"baseInfo"`
	}
	err := value.Decode(&val)
	if err != nil {
		return err
	}
	ep.EndPointInfoBase = val.EndPointInfoBase

	switch val.Platform {
	case "QQ":
		switch ep.ProtocolType {
		case "onebot":
			var val struct {
				Adapter *PlatformAdapterGocq `yaml:"adapter"`
			}

			err = value.Decode(&val)
			if err != nil {
				return err
			}
			ep.Adapter = val.Adapter
		case "walle-q":
			var val struct {
				Adapter *PlatformAdapterWalleQ `yaml:"adapter"`
			}

			err = value.Decode(&val)
			if err != nil {
				return err
			}
			ep.Adapter = val.Adapter
		case "red":
			var val struct {
				Adapter *PlatformAdapterRed `yaml:"adapter"`
			}
			err = value.Decode(&val)
			if err != nil {
				return err
			}
			ep.Adapter = val.Adapter
		case "official":
			var val struct {
				Adapter *PlatformAdapterOfficialQQ `yaml:"adapter"`
			}
			err = value.Decode(&val)
			if err != nil {
				return err
			}
			ep.Adapter = val.Adapter
		case "satori":
			var val struct {
				Adapter *PlatformAdapterSatori `yaml:"adapter"`
			}
			err = value.Decode(&val)
			if err != nil {
				return err
			}
			ep.Adapter = val.Adapter
		// case "LagrangeGo":
		//	var val struct {
		//		Adapter *PlatformAdapterLagrangeGo `yaml:"adapter"`
		//	}
		//	err = value.Decode(&val)
		//	if err != nil {
		//		return err
		//	}
		//	ep.Adapter = val.Adapter
		case "milky":
			var val struct {
				Adapter *PlatformAdapterMilky `yaml:"adapter"`
			}
			err = value.Decode(&val)
			if err != nil {
				return err
			}
			ep.Adapter = val.Adapter
		case "pureonebot":
			var val struct {
				Adapter *PlatformAdapterOnebot `yaml:"adapter"`
			}
			err = value.Decode(&val)
			if err != nil {
				return err
			}
			ep.Adapter = val.Adapter
		}
	case "DISCORD":
		var val struct {
			Adapter *PlatformAdapterDiscord `yaml:"adapter"`
		}
		err = value.Decode(&val)
		if err != nil {
			return err
		}
		ep.Adapter = val.Adapter
	case "KOOK":
		var val struct {
			Adapter *PlatformAdapterKook `yaml:"adapter"`
		}
		err = value.Decode(&val)
		if err != nil {
			return err
		}
		ep.Adapter = val.Adapter
	case "TG":
		var val struct {
			Adapter *PlatformAdapterTelegram `yaml:"adapter"`
		}
		err = value.Decode(&val)
		if err != nil {
			return err
		}
		ep.Adapter = val.Adapter
	case "MC":
		var val struct {
			Adapter *PlatformAdapterMinecraft `yaml:"adapter"`
		}
		err = value.Decode(&val)
		if err != nil {
			return err
		}
		ep.Adapter = val.Adapter
	case "DODO":
		var val struct {
			Adapter *PlatformAdapterDodo `yaml:"adapter"`
		}
		err = value.Decode(&val)
		if err != nil {
			return err
		}
		ep.Adapter = val.Adapter
	case "DINGTALK":
		var val struct {
			Adapter *PlatformAdapterDingTalk `yaml:"adapter"`
		}
		err = value.Decode(&val)
		if err != nil {
			return err
		}
		ep.Adapter = val.Adapter
	case "SLACK":
		var val struct {
			Adapter *PlatformAdapterSlack `yaml:"adapter"`
		}
		err = value.Decode(&val)
		if err != nil {
			return err
		}
		ep.Adapter = val.Adapter
	case "SEALCHAT":
		var val struct {
			Adapter *PlatformAdapterSealChat `yaml:"adapter"`
		}
		err = value.Decode(&val)
		if err != nil {
			return err
		}
		ep.Adapter = val.Adapter
	}
	return err
}

// StatsRestore 灏濊瘯浠庢暟鎹簱涓仮澶岴P鐨勭粺璁℃暟鎹?
func (ep *EndPointInfo) StatsRestore(d *Dice) {
	if len(ep.UserID) == 0 {
		return // 灏氭湭杩炴帴瀹屾垚鐨勬柊璐﹀彿娌℃湁UserId, 璺宠繃
	}

	m := model.EndpointInfo{UserID: ep.UserID}
	err := service.Query(d.DBOperator, &m)
	if err != nil {
		d.Logger.Errorf("鎭㈠endpoint缁熻鏁版嵁澶辫触 %v : %v", ep.UserID, err)
		return
	}

	if m.UpdatedAt <= ep.CmdExecutedLastTime {
		// 鍙湪鏁版嵁搴撲腑淇濆瓨鐨勬暟鎹瘮褰撳墠鏁版嵁鏂版椂鎵嶆浛鎹? 閬垮厤涓婃Dump涔嬪悗鏂扮殑鎸囦护缁熻琚鐩?
		return
	}

	// 铏界劧瑙夊緱涓嶈嚦浜? 杩樻槸鍒ゆ柇涓€涓? 鍙繘琛屽闀挎柟鍚戠殑鏇存柊
	if ep.CmdExecutedNum < m.CmdNum {
		ep.CmdExecutedNum = m.CmdNum
	}
	if ep.CmdExecutedLastTime < m.CmdLastTime {
		ep.CmdExecutedLastTime = m.CmdLastTime
	}
	if ep.OnlineTotalTime < m.OnlineTime {
		ep.OnlineTotalTime = m.OnlineTime
	}
}

// StatsDump EP缁熻鏁版嵁钀藉簱
func (ep *EndPointInfo) StatsDump(d *Dice) {
	if len(ep.UserID) == 0 {
		return // 灏氭湭杩炴帴瀹屾垚鐨勬柊璐﹀彿娌℃湁UserId, 璺宠繃
	}

	m := model.EndpointInfo{UserID: ep.UserID, CmdNum: ep.CmdExecutedNum, CmdLastTime: ep.CmdExecutedLastTime, OnlineTime: ep.OnlineTotalTime}
	err := service.Save(d.DBOperator, &m)
	if err != nil {
		d.Logger.Errorf("淇濆瓨endpoint鏁版嵁鍒版暟鎹簱澶辫触 %v : %v", ep.UserID, err)
	}
}

type IMSession struct {
	Parent       *Dice                              `yaml:"-"`
	EndPoints    []*EndPointInfo                    `yaml:"endPoints"`
	ServiceAtNew *SyncMap[string, *GroupInfo]       `json:"servicesAt" yaml:"-"`
	PendingQuits *SyncMap[string, *PendingQuitInfo] `json:"-" yaml:"-"`
}

type PendingQuitInfo struct {
	Origin    string
	CreatedAt time.Time
	ExpireAt  time.Time
}

const (
	QuitOriginAutoInactive = "auto_inactive"
)

func makePendingQuitKey(groupID string, endpointID string) string {
	return groupID + "\x00" + endpointID
}

func (s *IMSession) MarkPendingQuit(groupID string, endpointID string, origin string, ttl time.Duration) {
	if s == nil {
		return
	}
	if s.PendingQuits == nil {
		s.PendingQuits = new(SyncMap[string, *PendingQuitInfo])
	}
	now := time.Now()
	s.PendingQuits.Store(makePendingQuitKey(groupID, endpointID), &PendingQuitInfo{
		Origin:    origin,
		CreatedAt: now,
		ExpireAt:  now.Add(ttl),
	})
}

func (s *IMSession) ConsumePendingQuit(groupID string, endpointID string) *PendingQuitInfo {
	if s == nil || s.PendingQuits == nil {
		return nil
	}
	info, ok := s.PendingQuits.LoadAndDelete(makePendingQuitKey(groupID, endpointID))
	if !ok || info == nil {
		return nil
	}
	if time.Now().After(info.ExpireAt) {
		return nil
	}
	return info
}

type MsgContext struct {
	MessageType string
	Group       *GroupInfo       `jsbind:"group"`  // 褰撳墠缇や俊鎭?
	Player      *GroupPlayerInfo `jsbind:"player"` // 褰撳墠缇ょ殑鐜╁鏁版嵁

	IsCompatibilityTest bool // 鏄惁涓哄吋瀹规€ф祴璇曠幆澧冿紝鐢ㄤ簬璺宠繃涓嶅繀瑕佺殑鏁版嵁搴撴煡璇?

	EndPoint        *EndPointInfo `jsbind:"endPoint"` // 瀵瑰簲鐨凟ndpoint
	Session         *IMSession    // 瀵瑰簲鐨処MSession
	Dice            *Dice         // 瀵瑰簲鐨?Dice
	IsCurGroupBotOn bool          `jsbind:"isCurGroupBotOn"` // 鍦ㄧ兢鍐呮槸鍚ot on

	IsPrivate       bool        `jsbind:"isPrivate"` // 鏄惁绉佽亰
	CommandID       int64       // 鎸囦护ID
	CommandHideFlag string      `jsbind:"commandHideFlag"` // 鏆楅鏉ユ簮缇ゅ彿
	CommandInfo     interface{} // 鍛戒护淇℃伅
	PrivilegeLevel  int         `jsbind:"privilegeLevel"` // 鏉冮檺绛夌骇 -30ban 40閭€璇疯€?50绠＄悊 60缇や富 70淇′换 100master
	GroupRoleLevel  int         // 缇ゅ唴鏉冮檺 40閭€璇疯€?50绠＄悊 60缇や富 70淇′换 100master锛岀浉褰撲簬涓嶈€冭檻ban鐨勬潈闄愮瓑绾?
	DelegateText    string      `jsbind:"delegateText"`  // 浠ｉ闄勫姞鏂囨湰
	AliasPrefixText string      `json:"aliasPrefixText"` // 蹇嵎鎸囦护鍥炲鍓嶇紑鏂囨湰

	deckDepth         int                                         // 鎶界墝閫掑綊娣卞害
	DeckPools         map[*DeckInfo]map[string]*ShuffleRandomPool // 涓嶆斁鍥炴娊鍙栫殑缂撳瓨
	diceExprOverwrite string                                      // 榛樿楠拌〃杈惧紡瑕嗙洊
	SystemTemplate    *GameSystemTemplate
	Censored          bool // 宸叉鏌ヨ繃鏁忔劅璇?
	SpamCheckedGroup  bool
	SpamCheckedPerson bool

	splitKey      string
	vm            *ds.Context
	AttrsCurCache *AttributesItem
	_v1Rand       *rand2.PCGSource
}

// fillPrivilege 濉啓MsgContext涓殑鏉冮檺瀛楁, 骞惰繑鍥炲～鍐欑殑鏉冮檺绛夌骇
//   - msg 浣跨敤鍏朵腑鐨刴sg.Sender.GroupRole
//
// MsgContext.Dice闇€瑕佹寚鍚戜竴涓湁鏁堢殑Dice瀵硅薄
func (ctx *MsgContext) fillPrivilege(msg *Message) int {
	switch {
	case msg.Sender.GroupRole == "owner":
		ctx.PrivilegeLevel = 60 // 缇や富
	case ctx.IsPrivate || msg.Sender.GroupRole == "admin":
		ctx.PrivilegeLevel = 50 // 缇ょ鐞?
	case ctx.Group != nil && msg.Sender.UserID == ctx.Group.InviteUserID:
		ctx.PrivilegeLevel = 40 // 閭€璇疯€?
	default: /* no-op */
	}

	ctx.GroupRoleLevel = ctx.PrivilegeLevel

	if ctx.Dice == nil || ctx.Player == nil {
		return ctx.PrivilegeLevel
	}

	// 鍔犲叆榛戝悕鍗曠浉鍏虫潈闄?
	if val, exists := ctx.Dice.Config.BanList.GetByID(ctx.Player.UserID); exists {
		switch val.Rank {
		case BanRankBanned:
			ctx.PrivilegeLevel = -30
		case BanRankTrusted:
			ctx.PrivilegeLevel = 70
		default: /* no-op */
		}
	}

	grpID := ""
	if ctx.Group != nil {
		grpID = ctx.Group.GroupID
	}
	// master 鏉冮檺澶т簬榛戝悕鍗曟潈闄?
	if ctx.Dice.MasterCheck(grpID, ctx.Player.UserID) {
		ctx.PrivilegeLevel = 100
	}

	return ctx.PrivilegeLevel
}

func (s *IMSession) Execute(ep *EndPointInfo, msg *Message, runInSync bool) {
	d := s.Parent

	mctx := &MsgContext{}
	mctx.Dice = d
	mctx.MessageType = msg.MessageType
	mctx.IsPrivate = mctx.MessageType == "private"
	mctx.Session = s
	mctx.EndPoint = ep
	log := d.Logger

	// 澶勭悊鍛戒护
	if msg.MessageType == "group" || msg.MessageType == "private" { //nolint:nestif
		// GroupEnableCheck TODO: 鍚庣画鐪嬬湅鏄惁闇€瑕?
		groupInfo, ok := s.ServiceAtNew.Load(msg.GroupID)
		if !ok && msg.GroupID != "" {
			// 娉ㄦ剰: 姝ゅ蹇呴』寮€鍚紝涓嶇劧涓嬮潰mctx.player鍙栦笉鍒?
			autoOn := true
			if msg.Platform == "QQ-CH" {
				autoOn = d.Config.QQChannelAutoOn
			}
			groupInfo = SetBotOnAtGroup(mctx, msg.GroupID)
			groupInfo.Active = autoOn
			groupInfo.DiceIDExistsMap.Store(ep.UserID, true)
			if msg.GroupName != "" {
				groupInfo.GroupName = msg.GroupName
			}
			groupInfo.MarkDirty(d) // SetBotOnAtGroup 宸茶皟鐢ㄨ繃涓€娆★紝杩欓噷纭繚鍚庣画淇敼涔熻鏍囪

			dm := d.Parent
			groupName := dm.TryGetGroupName(groupInfo.GroupID)

			txt := fmt.Sprintf("自动激活: 发现无记录群组 %s(%s)，因为已是群成员，所以自动激活，开启状态: %t", groupName, groupInfo.GroupID, autoOn)
			if dm.ShouldRefreshGroupInfo(msg.GroupID) {
				ep.Adapter.GetGroupInfoAsync(msg.GroupID)
			}
			log.Info(txt)
			mctx.Notice(txt)

			if msg.Platform == "QQ" || msg.Platform == "TG" {
				// ServiceAtNew changed
				// Pinenutn:杩欎釜i涓嶇煡閬撴槸鍟ワ紝鏀句綘涓€椹紙
				activatedList, _ := mctx.Session.ServiceAtNew.Load(msg.GroupID)
				if ok {
					for _, wrapper := range activatedList.GetActivatedExtList(mctx.Dice) {
						ext := wrapper.GetRealExt()
						if ext == nil {
							continue
						}
						if ext.OnGroupJoined != nil {
							ext.callWithJsCheck(mctx.Dice, func() {
								ext.OnGroupJoined(mctx, msg)
							})
						}
					}
				}
			}
		}

		// 褰撴枃鏈彲鑳芥槸鍦ㄥ彂閫佸懡浠ゆ椂锛屽繀椤诲姞杞戒俊鎭?
		maybeCommand := CommandCheckPrefix(msg.Message, d.CommandPrefix, msg.Platform)

		amIBeMentioned := false
		if true {
			// 琚獲鏃讹紝蹇呴』鍔犺浇淇℃伅
			// 杩欐浠ｇ爜閲嶅浜嗭紝浠ュ悗閲嶆瀯
			_, ats := AtParse(msg.Message, msg.Platform)
			tmpUID := ep.UserID
			if msg.TmpUID != "" {
				tmpUID = msg.TmpUID
			}
			for _, i := range ats {
				// 鐗规畩澶勭悊 OpenQQ 鍜?OpenQQCH
				if i.UserID == tmpUID {
					amIBeMentioned = true
					break
				} else if strings.HasPrefix(i.UserID, "OpenQQ:") ||
					strings.HasPrefix(i.UserID, "OpenQQCH:") {
					uid := strings.TrimPrefix(tmpUID, "OpenQQ:")
					if i.UserID == "OpenQQ:"+uid || i.UserID == "OpenQQCH:"+uid {
						amIBeMentioned = true
						break
					}
				}
			}
		}

		mctx.Group, mctx.Player = GetPlayerInfoBySender(mctx, msg)
		mctx.IsCurGroupBotOn = msg.MessageType == "group" && mctx.Group.IsActive(mctx)

		if mctx.Group != nil && mctx.Group.System != "" {
			mctx.SystemTemplate = mctx.Group.GetCharTemplate(d)
			// tmpl, _ := d.GameSystemMap.Load(group.System)
			// mctx.SystemTemplate = tmpl
		}

		if groupInfo != nil && !strings.HasPrefix(groupInfo.GroupID, "UI-Group:") {
			// 鑷姩婵€娲诲瓨鍦ㄧ姸鎬?
			if _, exists := groupInfo.DiceIDExistsMap.Load(ep.UserID); !exists {
				groupInfo.DiceIDExistsMap.Store(ep.UserID, true)
				groupInfo.MarkDirty(d)
			}
		}

		// 鏉冮檺鍙疯缃?
		_ = mctx.fillPrivilege(msg)

		if mctx.Group != nil && mctx.Group.IsActive(mctx) {
			if mctx.PrivilegeLevel != -30 {
				for _, wrapper := range mctx.Group.GetActivatedExtList(mctx.Dice) {
					ext := wrapper.GetRealExt()
					if ext == nil {
						continue
					}
					if ext.OnMessageReceived != nil {
						ext.callWithJsCheck(mctx.Dice, func() {
							ext.OnMessageReceived(mctx, msg)
						})
					}
				}
			}
		}

		var cmdLst []string
		if maybeCommand {
			// 鍏煎妯″紡妫€鏌ュ凡缁忕Щ闄?
			for k := range d.CmdMap {
				cmdLst = append(cmdLst, k)
			}
			// 杩欓噷涓嶇敤group鏄负浜嗙鑱?
			g := mctx.Group
			if g != nil {
				for _, wrapper := range g.GetActivatedExtList(d) {
					for k := range wrapper.GetCmdMap() {
						cmdLst = append(cmdLst, k)
					}
				}
			}
			sort.Sort(ByLength(cmdLst))
		}

		if notReply := checkBan(mctx, msg); notReply {
			return
		}

		platformPrefix := msg.Platform
		cmdArgs := CommandParse(msg.Message, cmdLst, d.CommandPrefix, platformPrefix, false)
		if cmdArgs != nil {
			mctx.CommandID = getNextCommandID()

			var tmpUID string
			if platformPrefix == "OpenQQCH" {
				// 鐗规畩澶勭悊 OpenQQ棰戦亾
				uid := strings.TrimPrefix(ep.UserID, "OpenQQ:")
				tmpUID = "OpenQQCH:" + uid
			} else {
				tmpUID = ep.UserID
			}
			if msg.TmpUID != "" {
				tmpUID = msg.TmpUID
			}

			// 璁剧疆at淇℃伅
			cmdArgs.SetupAtInfo(tmpUID)
		}

		// 鏀跺埌缇?test(1111) 鍐?XX(222) 鐨勬秷鎭? 濂界湅 (1232611291)
		if msg.MessageType == "group" {
			if mctx.CommandID != 0 {
				// 鍏抽棴鐘舵€佷笅锛屽鏋滆@锛屼笖鏄涓€涓@鐨勶紝閭ｄ箞瑙嗕负寮€鍚?
				if !mctx.IsCurGroupBotOn && cmdArgs.AmIBeMentionedFirst {
					mctx.IsCurGroupBotOn = true
				}

				log.Infof("鏀跺埌缇?%s)鍐?%s>(%s)鐨勬寚浠? %s", msg.GroupID, msg.Sender.Nickname, msg.Sender.UserID, msg.Message)
			} else {
				doLog := true
				if d.Config.OnlyLogCommandInGroup {
					// 妫€鏌ヤ笂绾ч€夐」
					doLog = false
				}
				if doLog {
					// 妫€鏌Q棰戦亾鐨勭嫭绔嬮€夐」
					if msg.Platform == "QQ-CH" && (!d.Config.QQChannelLogMessage) {
						doLog = false
					}
				}
				if doLog {
					log.Infof("鏀跺埌缇?%s)鍐?%s>(%s)鐨勬秷鎭? %s", msg.GroupID, msg.Sender.Nickname, msg.Sender.UserID, msg.Message)
					// fmt.Printf("娑堟伅闀垮害 %v 鍐呭 %v \n", len(msg.Message), []byte(msg.Message))
				}
			}
		}

		// 鏁忔劅璇嶆嫤鎴細鍏ㄩ儴杈撳叆
		if mctx.IsCurGroupBotOn && d.Config.EnableCensor && d.Config.CensorMode == AllInput {
			hit, words, needToTerminate, _ := d.CensorMsg(mctx, msg, msg.Message, "")
			if needToTerminate {
				return
			}
			if hit {
				text := DiceFormatTmpl(mctx, "核心:拦截_完全拦截_收到的所有消息")
				if text != "" {
					ReplyToSender(mctx, msg, text)
				}
				if msg.MessageType == "group" {
					log.Infof(
						"鎷掔粷澶勭悊鍛戒腑鏁忔劅璇嶃€?s銆嶇殑鍐呭銆?s銆? 鏉ヨ嚜缇?%s)鍐?%s>(%s)",
						strings.Join(words, "|"),
						msg.Message, msg.GroupID, msg.Sender.Nickname, msg.Sender.UserID,
					)
				} else {
					log.Infof(
						"鎷掔粷澶勭悊鍛戒腑鏁忔劅璇嶃€?s銆嶇殑鍐呭銆?s銆? 鏉ヨ嚜<%s>(%s)",
						strings.Join(words, "|"),
						msg.Message,
						msg.Sender.Nickname,
						msg.Sender.UserID,
					)
				}
				return
			}
		}

		if msg.MessageType == "private" {
			if mctx.CommandID != 0 {
				log.Infof("鏀跺埌<%s>(%s)鐨勭鑱婃寚浠? %s", msg.Sender.Nickname, msg.Sender.UserID, msg.Message)
			} else if !d.Config.OnlyLogCommandInPrivate {
				log.Infof("鏀跺埌<%s>(%s)鐨勭鑱婃秷鎭? %s", msg.Sender.Nickname, msg.Sender.UserID, msg.Message)
			}
		}
		// Note(Szzrain): 璧嬪€间复鏃跺彉閲忥紝涓嶇劧鏈変簺鍦版柟娌℃硶鐢?
		SetTempVars(mctx, msg.Sender.Nickname)
		if cmdArgs != nil {
			// 鏀跺埌淇℃伅鍥炶皟
			f := func() {
				defer func() {
					if r := recover(); r != nil {
						//  + fmt.Sprintf("%s", r)
						log.Errorf("寮傚父: %v 鍫嗘爤: %v", r, string(debug.Stack()))
						ReplyToSender(mctx, msg, DiceFormatTmpl(mctx, "鏍稿績:楠板瓙鎵ц寮傚父"))
					}
				}()

				// 鏁忔劅璇嶆嫤鎴細鍛戒护杈撳叆
				if (msg.MessageType == "private" || mctx.IsCurGroupBotOn) && d.Config.EnableCensor && d.Config.CensorMode == OnlyInputCommand {
					hit, words, needToTerminate, _ := d.CensorMsg(mctx, msg, msg.Message, "")
					if needToTerminate {
						return
					}
					if hit {
						text := DiceFormatTmpl(mctx, "核心:拦截_完全拦截_收到的指令")
						if text != "" {
							ReplyToSender(mctx, msg, text)
						}
						if msg.MessageType == "group" {
							log.Infof(
								"鎷掔粷澶勭悊鍛戒腑鏁忔劅璇嶃€?s銆嶇殑鎸囦护銆?s銆? 鏉ヨ嚜缇?%s)鍐?%s>(%s)",
								strings.Join(words, "|"),
								msg.Message,
								msg.GroupID,
								msg.Sender.Nickname,
								msg.Sender.UserID,
							)
						} else {
							log.Infof(
								"鎷掔粷澶勭悊鍛戒腑鏁忔劅璇嶃€?s銆嶇殑鎸囦护銆?s銆? 鏉ヨ嚜<%s>(%s)",
								strings.Join(words, "|"),
								msg.Message,
								msg.Sender.Nickname,
								msg.Sender.UserID,
							)
						}
						return
					}
				}

				if cmdArgs.Command != "botlist" && !cmdArgs.AmIBeMentioned {
					myuid := ep.UserID
					// 灞忚斀鏈哄櫒浜哄彂閫佺殑娑堟伅
					if mctx.MessageType == "group" {
						// fmt.Println("YYYYYYYYY", myuid, mctx.Group != nil)
						if mctx.Group.BotList.Exists(msg.Sender.UserID) {
							log.Infof("蹇界暐鎸囦护(鏈哄櫒浜?: 鏉ヨ嚜缇?%s)鍐?%s>(%s): %s", msg.GroupID, msg.Sender.Nickname, msg.Sender.UserID, msg.Message)
							return
						}
						// 褰撳叾浠栨満鍣ㄤ汉琚獲锛屼笉鍥炲簲
						for _, i := range cmdArgs.At {
							uid := i.UserID
							if uid == myuid {
								// 蹇界暐鑷繁
								continue
							}
							if mctx.Group.BotList.Exists(uid) {
								return
							}
						}
					}
				}

				ep.TriggerCommand(mctx, msg, cmdArgs)
			}
			if runInSync {
				f()
			} else {
				go f()
			}
		} else {
			if mctx.PrivilegeLevel == -30 {
				// 榛戝悕鍗曠敤鎴?
				return
			}

			// 璇曞浘鍖归厤鑷畾涔夊洖澶?
			isSenderBot := false
			if mctx.MessageType == "group" {
				if mctx.Group != nil && mctx.Group.BotList.Exists(msg.Sender.UserID) {
					isSenderBot = true
				}
			}

			if !isSenderBot {
				if mctx.Group != nil && (mctx.Group.IsActive(mctx) || amIBeMentioned) {
					for _, wrapper := range mctx.Group.GetActivatedExtList(mctx.Dice) {
						ext := wrapper.GetRealExt()
						if ext == nil {
							continue
						}
						i := ext // 淇濈暀寮曠敤
						if i.OnNotCommandReceived != nil {
							notCommandReceiveCall := func() {
								if i.IsJsExt {
									// 鍏堝垽鏂繍琛岀幆澧?
									loop, err := d.ExtLoopManager.GetLoop(i.JSLoopVersion)
									if err != nil {
										// 鎵撲釜DEBUG鏃ュ織锛?
										mctx.Dice.Logger.Errorf("鎵╁睍<%s>杩愯鐜宸茬粡杩囨湡: %v", i.Name, err)
										return
									}
									waitRun := make(chan int, 1)
									loop.RunOnLoop(func(runtime *goja.Runtime) {
										defer func() {
											if r := recover(); r != nil {
												mctx.Dice.Logger.Errorf("鎵╁睍<%s>澶勭悊闈炴寚浠ゆ秷鎭紓甯? %v 鍫嗘爤: %v", i.Name, r, string(debug.Stack()))
											}
											waitRun <- 1
										}()

										i.OnNotCommandReceived(mctx, msg)
									})
									<-waitRun
								} else {
									i.OnNotCommandReceived(mctx, msg)
								}
							}

							if runInSync {
								notCommandReceiveCall()
							} else {
								go notCommandReceiveCall()
							}
						}
					}
				}
			}
		}
	}
}

// ExecuteNew Note(Szzrain): 鏃笉鐮村潖鍏煎鎬ц繕瑕佹敮鎸佹柊 feature 鎴戠湡鏄崏浜嗭紝杩欓噷鏄?copy paste 鐨勪唬鐮佺◢寰敼浜嗕竴涓嬶紝鎴戠煡閬撹繖鏄湪灞庡北涓婂缓鎴垮瓙锛屼絾鏄病鍔炴硶
// 鍙湁鍦?Adapter 鍐呴儴瀹炵幇浜嗘柊鐨勬秷鎭瑙ｆ瀽鎵嶈兘浣跨敤杩欎釜鏂规硶锛屽嵆 Message.Segment 鏈夊€?
// 涓轰簡閬垮厤鐮村潖鍏煎鎬э紝Message.Message 涓殑鍐呭涓嶄細琚В鏋愪絾浠嶇劧浼氳祴鍊?
// 杩欎釜 ExcuteNew 鏂规硶浼樺寲浜嗗娑堟伅娈电殑瑙ｆ瀽锛屽叾浠栧钩鍙板簲褰撳敖蹇疄鐜版秷鎭瑙ｆ瀽骞朵娇鐢ㄨ繖涓柟娉?
func (s *IMSession) ExecuteNew(ep *EndPointInfo, msg *Message) {
	d := s.Parent

	mctx := &MsgContext{}
	mctx.Dice = d
	mctx.MessageType = msg.MessageType
	mctx.IsPrivate = mctx.MessageType == "private"
	mctx.Session = s
	mctx.EndPoint = ep
	log := d.Logger

	// 澶勭悊娑堟伅娈碉紝濡傛灉 2.0 瑕佸畬鍏ㄦ姏寮冧緷璧?Message.Message 鐨勫瓧绗︿覆瑙ｆ瀽锛屾妸杩欓噷鍒犳帀
	if msg.Message == "" {
		for _, elem := range msg.Segment {
			// 绫诲瀷鏂█
			if e, ok := elem.(*message.TextElement); ok {
				msg.Message += e.Content
			}
		}
	}

	if msg.MessageType != "group" && msg.MessageType != "private" {
		return
	}

	// 澶勭悊鍛戒护
	groupInfo, ok := s.ServiceAtNew.Load(msg.GroupID)
	if !ok && msg.GroupID != "" {
		// 娉ㄦ剰: 姝ゅ蹇呴』寮€鍚紝涓嶇劧涓嬮潰mctx.player鍙栦笉鍒?
		autoOn := true
		if msg.Platform == "QQ-CH" {
			autoOn = d.Config.QQChannelAutoOn
		}
		groupInfo = SetBotOnAtGroup(mctx, msg.GroupID)
		groupInfo.Active = autoOn
		groupInfo.DiceIDExistsMap.Store(ep.UserID, true)
		if msg.GroupName != "" {
			groupInfo.GroupName = msg.GroupName
		}
		groupInfo.MarkDirty(d) // SetBotOnAtGroup 宸茶皟鐢ㄨ繃涓€娆★紝杩欓噷纭繚鍚庣画淇敼涔熻鏍囪

		// dm := d.Parent
		// 鎰氳牏璋冪敤锛屾敼浜?
		// groupName := dm.TryGetGroupName(group.GroupID)
		groupName := msg.GroupName

		txt := fmt.Sprintf("自动激活: 发现无记录群组 %s(%s)，因为已是群成员，所以自动激活，开启状态: %t", groupName, groupInfo.GroupID, autoOn)
		// 鎰忎箟涓嶆槑锛屽垹鎺?
		// 鐤戜技鏄负浜嗚幏鍙栫兢淇℃伅鐒跺悗濉炲埌濂囨€殑鍦版柟
		// ep.Adapter.GetGroupInfoAsync(msg.GroupID)
		log.Info(txt)
		mctx.Notice(txt)

		if msg.Platform == "QQ" || msg.Platform == "TG" {
			groupInfo, ok = mctx.Session.ServiceAtNew.Load(msg.GroupID)
			if ok {
				for _, wrapper := range groupInfo.GetActivatedExtList(mctx.Dice) {
					ext := wrapper.GetRealExt()
					if ext == nil {
						continue
					}
					if ext.OnGroupJoined != nil {
						ext.callWithJsCheck(mctx.Dice, func() {
							ext.OnGroupJoined(mctx, msg)
						})
					}
				}
			}
		}
	}
	// 閲嶆柊璧嬪€?
	if groupInfo != nil {
		groupInfo.GroupName = msg.GroupName
	}

	// Note(Szzrain): 鍒ゆ柇鏄惁琚獲
	amIBeMentioned := false
	for _, elem := range msg.Segment {
		// 绫诲瀷鏂█
		if e, ok := elem.(*message.AtElement); ok {
			if msg.Platform+":"+e.Target == ep.UserID {
				amIBeMentioned = true
				break
			}
		}
	}

	mctx.Group, mctx.Player = GetPlayerInfoBySender(mctx, msg)
	mctx.IsCurGroupBotOn = msg.MessageType == "group" && mctx.Group.IsActive(mctx)

	if mctx.Group != nil && mctx.Group.System != "" {
		mctx.SystemTemplate = mctx.Group.GetCharTemplate(d)
		// tmpl, _ := d.GameSystemMap.Load(group.System)
		// mctx.SystemTemplate = tmpl
	}

	if groupInfo != nil {
		// 鑷姩婵€娲诲瓨鍦ㄧ姸鎬?
		if _, exists := groupInfo.DiceIDExistsMap.Load(ep.UserID); !exists {
			groupInfo.DiceIDExistsMap.Store(ep.UserID, true)
			groupInfo.MarkDirty(mctx.Dice)
		}
	}

	// 鏉冮檺璁剧疆
	_ = mctx.fillPrivilege(msg)

	if mctx.Group != nil && mctx.Group.IsActive(mctx) {
		if mctx.PrivilegeLevel != -30 {
			for _, wrapper := range mctx.Group.GetActivatedExtList(mctx.Dice) {
				ext := wrapper.GetRealExt()
				if ext == nil {
					continue
				}
				if ext.OnMessageReceived != nil {
					ext.callWithJsCheck(mctx.Dice, func() {
						ext.OnMessageReceived(mctx, msg)
					})
				}
			}
		}
	}

	// Note(Szzrain): 鍏煎妯″紡鐩稿叧鐨勪唬鐮佽鎸埌浜?cmdArgs.commandParseNew 閲岄潰

	if notReply := checkBan(mctx, msg); notReply {
		return
	}

	// Note(Szzrain): platformPrefix 寮冪敤
	// platformPrefix := msg.Platform
	cmdArgs := CommandParseNew(mctx, msg)
	if cmdArgs != nil {
		mctx.CommandID = getNextCommandID()
		// var tmpUID string
		// if platformPrefix == "OpenQQCH" {
		//	// 鐗规畩澶勭悊 OpenQQ棰戦亾
		//	uid := strings.TrimPrefix(ep.UserID, "OpenQQ:")
		//	tmpUID = "OpenQQCH:" + uid
		// } else {
		//	tmpUID = ep.UserID
		// }
		// if msg.TmpUID != "" {
		//	tmpUID = msg.TmpUID
		// }

		// 璁剧疆at淇℃伅锛岃繖閲屼笉鍐嶉渶瑕侊紝鍥犱负宸茬粡鍦?CommandParseNew 閲岄潰璁剧疆浜?
		// cmdArgs.SetupAtInfo(tmpUID)
	}

	// 鏀跺埌缇?test(1111) 鍐?XX(222) 鐨勬秷鎭? 濂界湅 (1232611291)
	if msg.MessageType == "group" {
		// TODO(Szzrain):  闇€瑕佷紭鍖栫殑鍐欐硶锛屼笉搴旀牴鎹?CommandID 鏉ュ垽鏂槸鍚︽槸鎸囦护锛岃€屽簲璇ユ牴鎹?cmdArgs 鏄惁 match 鍒版寚浠ゆ潵鍒ゆ柇
		if mctx.CommandID != 0 {
			// 鍏抽棴鐘舵€佷笅锛屽鏋滆@锛屼笖鏄涓€涓@鐨勶紝閭ｄ箞瑙嗕负寮€鍚?
			if !mctx.IsCurGroupBotOn && cmdArgs.AmIBeMentionedFirst {
				mctx.IsCurGroupBotOn = true
			}

			log.Infof("鏀跺埌缇?%s)鍐?%s>(%s)鐨勬寚浠? %s", msg.GroupID, msg.Sender.Nickname, msg.Sender.UserID, msg.Message)
		} else {
			doLog := true
			if d.Config.OnlyLogCommandInGroup {
				// 妫€鏌ヤ笂绾ч€夐」
				doLog = false
			}
			if doLog {
				// 妫€鏌Q棰戦亾鐨勭嫭绔嬮€夐」
				if msg.Platform == "QQ-CH" && (!d.Config.QQChannelLogMessage) {
					doLog = false
				}
			}
			if doLog {
				log.Infof("鏀跺埌缇?%s)鍐?%s>(%s)鐨勬秷鎭? %s", msg.GroupID, msg.Sender.Nickname, msg.Sender.UserID, msg.Message)
				// fmt.Printf("娑堟伅闀垮害 %v 鍐呭 %v \n", len(msg.Message), []byte(msg.Message))
			}
		}
	}

	// Note(Szzrain): 杩欓噷鐨勪唬鐮佹湰鏉ュ湪鏁忔劅璇嶆娴嬩笅闈紝浼氫骇鐢熼鏈熶箣澶栫殑琛屼负锛屾墍浠ユ尓鍒拌繖閲?
	if msg.MessageType == "private" {
		// TODO(Szzrain): 闇€瑕佷紭鍖栫殑鍐欐硶锛屼笉搴旀牴鎹?CommandID 鏉ュ垽鏂槸鍚︽槸鎸囦护锛岃€屽簲璇ユ牴鎹?cmdArgs 鏄惁 match 鍒版寚浠ゆ潵鍒ゆ柇锛屽悓涓?
		if mctx.CommandID != 0 {
			log.Infof("鏀跺埌<%s>(%s)鐨勭鑱婃寚浠? %s", msg.Sender.Nickname, msg.Sender.UserID, msg.Message)
		} else if !d.Config.OnlyLogCommandInPrivate {
			log.Infof("鏀跺埌<%s>(%s)鐨勭鑱婃秷鎭? %s", msg.Sender.Nickname, msg.Sender.UserID, msg.Message)
		}
	}

	// 鏁忔劅璇嶆嫤鎴細鍏ㄩ儴杈撳叆
	if mctx.IsCurGroupBotOn && d.Config.EnableCensor && d.Config.CensorMode == AllInput {
		hit, words, needToTerminate, _ := d.CensorMsg(mctx, msg, msg.Message, "")
		if needToTerminate {
			return
		}
		if hit {
			text := DiceFormatTmpl(mctx, "核心:拦截_完全拦截_收到的所有消息")
			if text != "" {
				ReplyToSender(mctx, msg, text)
			}
			if msg.MessageType == "group" {
				log.Infof(
					"鎷掔粷澶勭悊鍛戒腑鏁忔劅璇嶃€?s銆嶇殑鍐呭銆?s銆? 鏉ヨ嚜缇?%s)鍐?%s>(%s)",
					strings.Join(words, "|"),
					msg.Message, msg.GroupID, msg.Sender.Nickname, msg.Sender.UserID,
				)
			} else {
				log.Infof(
					"鎷掔粷澶勭悊鍛戒腑鏁忔劅璇嶃€?s銆嶇殑鍐呭銆?s銆? 鏉ヨ嚜<%s>(%s)",
					strings.Join(words, "|"),
					msg.Message,
					msg.Sender.Nickname,
					msg.Sender.UserID,
				)
			}
			return
		}
	}
	// Note(Szzrain): 璧嬪€间复鏃跺彉閲忥紝涓嶇劧鏈変簺鍦版柟娌℃硶鐢?
	SetTempVars(mctx, msg.Sender.Nickname)
	if cmdArgs != nil {
		go s.PreTriggerCommand(mctx, msg, cmdArgs)
	} else {
		// if cmdArgs == nil will execute this block
		if mctx.PrivilegeLevel == -30 {
			// 榛戝悕鍗曠敤鎴?
			return
		}

		// 璇曞浘鍖归厤鑷畾涔夊洖澶?
		isSenderBot := false
		if mctx.MessageType == "group" {
			if mctx.Group != nil && mctx.Group.BotList.Exists(msg.Sender.UserID) {
				isSenderBot = true
			}
		}

		if !isSenderBot {
			if mctx.Group != nil && (mctx.Group.IsActive(mctx) || amIBeMentioned) {
				for _, wrapper := range mctx.Group.GetActivatedExtList(mctx.Dice) {
					ext := wrapper.GetRealExt()
					if ext == nil {
						continue
					}
					i := ext // 淇濈暀寮曠敤
					if i.OnNotCommandReceived != nil {
						notCommandReceiveCall := func() {
							if i.IsJsExt {
								loop, err := d.ExtLoopManager.GetLoop(i.JSLoopVersion)
								if err != nil {
									// 鎵撲釜DEBUG鏃ュ織锛?
									i.dice.Logger.Errorf("鎵╁睍<%s>杩愯鐜宸茬粡杩囨湡: %v", i.Name, err)
									return
								}
								waitRun := make(chan int, 1)
								loop.RunOnLoop(func(runtime *goja.Runtime) {
									defer func() {
										if r := recover(); r != nil {
											mctx.Dice.Logger.Errorf("鎵╁睍<%s>澶勭悊闈炴寚浠ゆ秷鎭紓甯? %v 鍫嗘爤: %v", i.Name, r, string(debug.Stack()))
										}
										waitRun <- 1
									}()
									i.OnNotCommandReceived(mctx, msg)
								})
								<-waitRun
							} else {
								i.OnNotCommandReceived(mctx, msg)
							}
						}

						go notCommandReceiveCall()
					}
				}
			}
		}
	}
}

func (s *IMSession) PreTriggerCommand(mctx *MsgContext, msg *Message, cmdArgs *CmdArgs) {
	d := s.Parent
	ep := mctx.EndPoint
	log := d.Logger
	defer func() {
		if r := recover(); r != nil {
			//  + fmt.Sprintf("%s", r)
			log.Errorf("寮傚父: %v 鍫嗘爤: %v", r, string(debug.Stack()))
			ReplyToSender(mctx, msg, DiceFormatTmpl(mctx, "鏍稿績:楠板瓙鎵ц寮傚父"))
		}
	}()

	// 鏁忔劅璇嶆嫤鎴細鍛戒护杈撳叆
	if (msg.MessageType == "private" || mctx.IsCurGroupBotOn) && d.Config.EnableCensor && d.Config.CensorMode == OnlyInputCommand {
		hit, words, needToTerminate, _ := d.CensorMsg(mctx, msg, msg.Message, "")
		if needToTerminate {
			return
		}
		if hit {
			text := DiceFormatTmpl(mctx, "核心:拦截_完全拦截_收到的指令")
			if text != "" {
				ReplyToSender(mctx, msg, text)
			}
			if msg.MessageType == "group" {
				log.Infof(
					"鎷掔粷澶勭悊鍛戒腑鏁忔劅璇嶃€?s銆嶇殑鎸囦护銆?s銆? 鏉ヨ嚜缇?%s)鍐?%s>(%s)",
					strings.Join(words, "|"),
					msg.Message,
					msg.GroupID,
					msg.Sender.Nickname,
					msg.Sender.UserID,
				)
			} else {
				log.Infof(
					"鎷掔粷澶勭悊鍛戒腑鏁忔劅璇嶃€?s銆嶇殑鎸囦护銆?s銆? 鏉ヨ嚜<%s>(%s)",
					strings.Join(words, "|"),
					msg.Message,
					msg.Sender.Nickname,
					msg.Sender.UserID,
				)
			}
			return
		}
	}

	if cmdArgs.Command != "botlist" && !cmdArgs.AmIBeMentioned {
		myuid := ep.UserID
		// 灞忚斀鏈哄櫒浜哄彂閫佺殑娑堟伅
		if mctx.MessageType == "group" {
			// fmt.Println("YYYYYYYYY", myuid, mctx.Group != nil)
			if mctx.Group.BotList.Exists(msg.Sender.UserID) {
				log.Infof("蹇界暐鎸囦护(鏈哄櫒浜?: 鏉ヨ嚜缇?%s)鍐?%s>(%s): %s", msg.GroupID, msg.Sender.Nickname, msg.Sender.UserID, msg.Message)
				return
			}
			// 褰撳叾浠栨満鍣ㄤ汉琚獲锛屼笉鍥炲簲
			for _, i := range cmdArgs.At {
				uid := i.UserID
				if uid == myuid {
					// 蹇界暐鑷繁
					continue
				}
				if mctx.Group.BotList.Exists(uid) {
					return
				}
			}
		}
	}
	ep.TriggerCommand(mctx, msg, cmdArgs)
}

func (ep *EndPointInfo) TriggerCommand(mctx *MsgContext, msg *Message, cmdArgs *CmdArgs) bool {
	s := mctx.Session
	d := mctx.Dice
	log := d.Logger

	var ret bool
	// 璇曞浘鍖归厤鑷畾涔夋寚浠?
	if mctx.Group != nil && mctx.Group.IsActive(mctx) {
		for _, wrapper := range mctx.Group.GetActivatedExtList(mctx.Dice) {
			ext := wrapper.GetRealExt()
			if ext == nil {
				continue
			}
			if ext.OnCommandOverride != nil {
				ret = ext.OnCommandOverride(mctx, msg, cmdArgs)
				if ret {
					break
				}
			}
		}
	}

	if !ret {
		// 鑻ヨ嚜瀹氫箟鎸囦护鏈尮閰嶏紝鍖归厤鏍囧噯鎸囦护
		ret = s.commandSolve(mctx, msg, cmdArgs)
	}

	if ret {
		// 鍒峰睆妫€娴嬪凡缁忚縼绉诲埌 im_helpers.go锛屾澶勪笉鍐嶅鐞?
		ep.CmdExecutedNum++
		ep.CmdExecutedLastTime = time.Now().Unix()
		mctx.Player.LastCommandTime = ep.CmdExecutedLastTime
		mctx.Player.UpdatedAtTime = time.Now().Unix()
		if mctx.Group != nil {
			mctx.Group.MarkDirty(mctx.Dice)
		}
	} else {
		if msg.MessageType == "group" {
			log.Infof("蹇界暐鎸囦护(楠板瓙鍏抽棴/鎵╁睍鍏抽棴/鏈煡鎸囦护): 鏉ヨ嚜缇?%s)鍐?%s>(%s): %s", msg.GroupID, msg.Sender.Nickname, msg.Sender.UserID, msg.Message)
		}

		if msg.MessageType == "private" {
			log.Infof("蹇界暐鎸囦护(楠板瓙鍏抽棴/鎵╁睍鍏抽棴/鏈煡鎸囦护): 鏉ヨ嚜<%s>(%s)鐨勭鑱? %s", msg.Sender.Nickname, msg.Sender.UserID, msg.Message)
		}
	}
	return ret
}

// OnGroupJoined 缇ょ粍杩涚兢浜嬩欢澶勭悊锛屽叾浠?Adapter 搴斿綋灏藉揩杩佺Щ鑷虫鏂规硶瀹炵幇
func (s *IMSession) OnGroupJoined(ctx *MsgContext, msg *Message) {
	d := ctx.Dice
	log := d.Logger
	ep := ctx.EndPoint
	dm := d.Parent
	// 鍒ゆ柇杩涚兢鐨勪汉鏄嚜宸憋紝鑷姩鍚姩
	group := SetBotOnAtGroup(ctx, msg.GroupID)
	// Ensure context has group set for formatting and attrs access
	ctx.Group = group
	// 鑾峰彇閭€璇蜂汉ID锛孉dapter 搴斿綋鎸夌収缁熶竴鏍煎紡灏嗛個璇蜂汉 ID 鏀惧叆 Sender 瀛楁
	group.InviteUserID = msg.Sender.UserID
	group.DiceIDExistsMap.Store(ep.UserID, true)
	group.EnteredTime = time.Now().Unix() // 璁剧疆鍏ョ兢鏃堕棿
	group.MarkDirty(ctx.Dice)
	if dm.ShouldRefreshGroupInfo(msg.GroupID) {
		ep.Adapter.GetGroupInfoAsync(msg.GroupID)
	}
	time.Sleep(2 * time.Second)
	groupName := dm.TryGetGroupName(msg.GroupID)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Errorf("鍏ョ兢鑷磋緸寮傚父: %v 鍫嗘爤: %v", r, string(debug.Stack()))
			}
		}()

		// 绋嶄綔绛夊緟鍚庡彂閫佸叆缇よ嚧璇?
		time.Sleep(2 * time.Second)

		ctx.Player = &GroupPlayerInfo{}
		log.Infof("鍙戦€佸叆缇よ嚧杈烇紝缇? <%s>(%d)", groupName, msg.GroupID)
		text := DiceFormatTmpl(ctx, "鏍稿績:楠板瓙杩涚兢")
		for _, i := range ctx.SplitText(text) {
			doSleepQQ(ctx)
			ReplyGroup(ctx, msg, strings.TrimSpace(i))
		}
	}()
	txt := fmt.Sprintf("鍔犲叆缇ょ粍: <%s>(%s)", groupName, msg.GroupID)
	log.Info(txt)
	ctx.Notice(txt)
	for _, wrapper := range group.GetActivatedExtList(ctx.Dice) {
		ext := wrapper.GetRealExt()
		if ext == nil {
			continue
		}
		if ext.OnGroupJoined != nil {
			ext.callWithJsCheck(d, func() {
				ext.OnGroupJoined(ctx, msg)
			})
		}
	}
}

var lastWelcome *LastWelcomeInfo

// OnGroupMemberJoined 缇ゆ垚鍛樿繘缇や簨浠跺鐞嗭紝闄や簡 bot 鑷繁浠ュ鐨勭兢鎴愬憳鍏ョ兢鏃惰皟鐢ㄣ€傚叾浠?Adapter 搴斿綋灏藉揩杩佺Щ鑷虫鏂规硶瀹炵幇
func (s *IMSession) OnGroupMemberJoined(ctx *MsgContext, msg *Message) {
	log := s.Parent.Logger

	groupInfo, ok := s.ServiceAtNew.Load(msg.GroupID)
	// 杩涚兢鐨勬槸鍒汉锛屾槸鍚﹁繋鏂帮紵
	// 杩欓噷寰堣寮傦紝褰撴墜鏈篞Q瀹㈡埛绔鎵硅繘缇ゆ椂锛屽叆缇ゅ悗浼氭湁涓€鍙ラ粯璁ゅ彂瑷€
	// 姝ゆ椂浼氭敹鍒颁袱娆″畬鍏ㄤ竴鏍风殑鏌愮敤鎴峰叆缇や俊鎭紝瀵艰嚧鍙戜袱娆℃杩庤瘝
	if ok && groupInfo.ShowGroupWelcome {
		isDouble := false
		if lastWelcome != nil {
			isDouble = msg.GroupID == lastWelcome.GroupID &&
				msg.Sender.UserID == lastWelcome.UserID &&
				msg.Time == lastWelcome.Time
		}
		lastWelcome = &LastWelcomeInfo{
			GroupID: msg.GroupID,
			UserID:  msg.Sender.UserID,
			Time:    msg.Time,
		}

		if !isDouble {
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Errorf("杩庢柊鑷磋緸寮傚父: %v 鍫嗘爤: %v", r, string(debug.Stack()))
					}
				}()

				// Ensure context has group set for formatting and attrs access
				ctx.Group = groupInfo
				ctx.Player = &GroupPlayerInfo{}
				// VarSetValueStr(ctx, "$t鏂颁汉鏄电О", "<"+msgQQ.Sender.Nickname+">")
				uidRaw := UserIDExtract(msg.Sender.UserID)
				VarSetValueStr(ctx, "$t甯愬彿ID_RAW", uidRaw)
				VarSetValueStr(ctx, "$t璐﹀彿ID_RAW", uidRaw)
				stdID := msg.Sender.UserID
				VarSetValueStr(ctx, "$t甯愬彿ID", stdID)
				VarSetValueStr(ctx, "$t璐﹀彿ID", stdID)
				text := DiceFormat(ctx, groupInfo.GroupWelcomeMessage)
				for _, i := range ctx.SplitText(text) {
					doSleepQQ(ctx)
					ReplyGroup(ctx, msg, strings.TrimSpace(i))
				}
			}()
		}
	}
}

var platformRE = regexp.MustCompile(`^(.*)-Group:`)

// LongTimeQuitInactiveGroupReborn
// 瀹屽叏鎶涘純褰撳垵涓嶆噦Go鐨勬椂鍊欑殑鏂规锛屾敼鎴愬涓嬫柟妗堬細
// 姣忔灏濊瘯鎵惧埌n涓鍚堣姹傜殑缇わ紝鐒跺悗鍚竴涓嚎绋嬶紝灏嗙兢缁熶竴骞叉帀
// 杩欐牱瀛愮壓鐗蹭簡鍙樉绀虹殑鎬荤兢鏁帮紝浣嗗ぇ澶у寮轰簡绋冲畾鎬э紝鑰屼笖鎬荤兢鏁扮殑鍙傝€冨苟鏃犳剰涔夛紝鍥犱负宸茬粡鍦ㄧ殑缇ゅ緢鍙兘绐佺劧娲讳簡鑰屼笉绗﹀悎鍒ゅ畾
// 褰撳墠鐗堟湰鐨勯棶棰橈細濡傛灉鐢ㄦ埛璁剧疆浜嗗緢鐭殑鏃堕棿锛岄偅鍙兘涔嬪墠鐨勭兢杩樻病閫€瀹岋紝灏卞張閫€閭ｉ儴鍒嗙殑缇わ紝閫犳垚涓€浜涘鎬殑闂锛屼絾搴旇姒傜巼涓嶅ぇ + 璞归敊浼氳鎹曡幏
func (s *IMSession) LongTimeQuitInactiveGroupReborn(threshold time.Time, groupsPerRound int) {
	s.Parent.Logger.Infof("开始清理不活跃群聊. 判定线: %s, 本次退群数: %d", threshold.Format(time.RFC3339), groupsPerRound)
	type GroupEndpointPair struct {
		Group    *GroupInfo
		Endpoint *EndPointInfo
		Last     time.Time
	}
	isAutoQuitEndpointReady := func(grp *GroupInfo, ep *EndPointInfo, platform string, phase string) bool {
		groupID := "<nil>"
		if grp != nil {
			groupID = grp.GroupID
		}
		if grp == nil {
			s.Parent.Logger.Debugf("自动退群已跳过: 找不到群信息，暂时无法处理该群。phase=%s group=%s platform=%s", phase, groupID, platform)
			return false
		}
		if ep == nil {
			s.Parent.Logger.Debugf("自动退群已跳过: 找不到对应账号连接，暂时无法处理该群。phase=%s group=%s platform=%s", phase, groupID, platform)
			return false
		}
		if ep.Adapter == nil || ep.Session == nil {
			s.Parent.Logger.Debugf(
				"自动退群已跳过: 账号连接尚未准备完成，暂时无法处理该群。phase=%s group=%s platform=%s endpoint=%s adapter_nil=%t session_nil=%t",
				phase,
				groupID,
				platform,
				ep.UserID,
				ep.Adapter == nil,
				ep.Session == nil,
			)
			return false
		}
		if grp.DiceIDExistsMap == nil {
			s.Parent.Logger.Debugf("自动退群已跳过: 群内账号记录缺失，暂时无法确认是否可退群。phase=%s group=%s platform=%s endpoint=%s", phase, groupID, platform, ep.UserID)
			return false
		}
		if ep.Platform != platform || !grp.DiceIDExistsMap.Exists(ep.UserID) {
			return false
		}
		if !ep.Enable || ep.State != StateConnected {
			return false
		}
		return true
	}
	selectedGroupEndpoints := make([]*GroupEndpointPair, 0)
	groupCount := 0
	s.ServiceAtNew.Range(func(key string, grp *GroupInfo) bool {
		if strings.HasPrefix(grp.GroupID, "PG-") {
			return true
		}
		if s.Parent.Config.BanList != nil {
			info, ok := s.Parent.Config.BanList.GetByID(grp.GroupID)
			if ok && info.Rank > BanRankNormal {
				return true
			}
		}
		match := platformRE.FindStringSubmatch(grp.GroupID)
		if len(match) != 2 {
			return true
		}
		platform := match[1]
		if platform != "QQ" {
			return true
		}
		last := time.Unix(atomic.LoadInt64(&grp.RecentDiceSendTime), 0)
		if enter := time.Unix(grp.EnteredTime, 0); enter.After(last) {
			last = enter
		}
		if last.Unix() <= 0 {
			return true
		}
		if last.Before(threshold) {
			for _, ep := range s.EndPoints {
				if !isAutoQuitEndpointReady(grp, ep, platform, "select") {
					continue
				}
				selectedGroupEndpoints = append(selectedGroupEndpoints, &GroupEndpointPair{Group: grp, Endpoint: ep, Last: last})
				groupCount++
				if groupCount > groupsPerRound {
					return false
				}
			}
		}
		return true
	})
	if len(selectedGroupEndpoints) == 0 {
		return
	}

	summaryMode := s.Parent.Config.QuitInactiveNoticeSummaryMode
	var noticeCtx *MsgContext
	if summaryMode {
		noticeCtx = &MsgContext{EndPoint: selectedGroupEndpoints[0].Endpoint, Session: s, Dice: s.Parent}
		noticeCtx.Notice(fmt.Sprintf("自动退群任务开始：本轮预计处理 %d 个群。", len(selectedGroupEndpoints)))
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log := zap.S().Named(logger.LogKeyAdapter)
				log.Errorf("自动退群异常: %v 堆栈: %v", r, string(debug.Stack()))
			}
		}()

		processed := 0
		skipped := 0
		cancelled := 0
		quitStarted := 0
		for i, pair := range selectedGroupEndpoints {
			grp := pair.Group
			ep := pair.Endpoint
			if !isAutoQuitEndpointReady(grp, ep, "QQ", "send") {
				s.Parent.Logger.Infof("自动退群已跳过: 当前账号已离线或不可用，暂不对该群执行退群。group=%s endpoint=%s", grp.GroupID, ep.UserID)
				skipped++
				continue
			}
			processed++
			last := pair.Last
			hint := fmt.Sprintf("检测到群%s上次活跃时间为%s，尝试退出，当前为本轮第 %d 个", grp.GroupID, last.Format(time.RFC3339), i+1)
			s.Parent.Logger.Info(hint)
			msgCtx := CreateTempCtx(ep, &Message{
				MessageType: "group",
				Sender:      SenderBase{UserID: ep.UserID},
				GroupID:     grp.GroupID,
			})
			msgText := DiceFormatTmpl(msgCtx, "核心:骰子自动退群告别语")
			ep.Adapter.SendToGroup(msgCtx, grp.GroupID, msgText, "")
			time.Sleep(2 * time.Second)
			if !isAutoQuitEndpointReady(grp, ep, "QQ", "quit") {
				s.Parent.Logger.Infof("自动退群已取消: 当前账号状态发生变化，本次不再继续退群。group=%s endpoint=%s", grp.GroupID, ep.UserID)
				cancelled++
				continue
			}
			grp.DiceIDExistsMap.Delete(ep.UserID)
			grp.MarkDirty(msgCtx.Dice)
			s.MarkPendingQuit(grp.GroupID, ep.UserID, QuitOriginAutoInactive, 5*time.Minute)
			ep.Adapter.QuitGroup(msgCtx, grp.GroupID)
			quitStarted++
			if !summaryMode {
				msgCtx.Notice(hint)
			}
			randomSleep := time.Duration(rand.Intn(3000)+8000) * time.Millisecond
			logger.M().Infof("退群等待，等待 %f 秒后继续", randomSleep.Seconds())
			time.Sleep(randomSleep)
		}
		if summaryMode && noticeCtx != nil {
			noticeCtx.Notice(fmt.Sprintf("自动退群任务结束：候选 %d 个，开始处理 %d 个，已发起退群 %d 个，跳过 %d 个，取消 %d 个。", len(selectedGroupEndpoints), processed, quitStarted, skipped, cancelled))
		}
	}()
}
func FormatBlacklistReasons(v *BanListInfoItem) string {
	var sb strings.Builder
	sb.WriteString("黑名单原因：")
	if v == nil {
		sb.WriteString("\n")
		sb.WriteString("原因未知，请联系开发者获取进一步信息")
		return sb.String()
	}
	for i, reason := range v.Reasons {
		sb.WriteString("\n")
		sb.WriteString(carbon.CreateFromTimestamp(v.Times[i]).ToDateTimeString())
		sb.WriteString("在“")
		sb.WriteString(v.Places[i])
		sb.WriteString("”，原因：")
		sb.WriteString(reason)
	}
	reasontext := sb.String()
	return reasontext
}

// checkBan 榛戝悕鍗曟嫤鎴?
func checkBan(ctx *MsgContext, msg *Message) (notReply bool) {
	d := ctx.Dice
	log := d.Logger
	var isBanGroup, isWhiteGroup bool
	// log.Info("check ban ", msg.MessageType, " ", msg.GroupID, " ", ctx.PrivilegeLevel)
	if msg.MessageType == "group" {
		value, exists := d.Config.BanList.GetByID(msg.GroupID)
		if exists {
			if value.Rank == BanRankBanned {
				isBanGroup = true
			}
			if value.Rank == BanRankTrusted {
				isWhiteGroup = true
			}
		}
	}

	banQuitGroup := func() {
		banListInfoItem, _ := ctx.Dice.Config.BanList.GetByID(msg.Sender.UserID)
		reasontext := FormatBlacklistReasons(banListInfoItem)
		groupID := msg.GroupID
		noticeMsg := fmt.Sprintf("检测到群(%s)内黑名单用户<%s>(%s)，自动退群\n%s", groupID, msg.Sender.Nickname, msg.Sender.UserID, reasontext)
		log.Info(noticeMsg)

		text := fmt.Sprintf("因<%s>(%s)是黑名单用户，将自动退群。", msg.Sender.Nickname, msg.Sender.UserID)
		ReplyGroupRaw(ctx, &Message{GroupID: groupID}, text, "")

		ctx.Notice(noticeMsg)

		time.Sleep(1 * time.Second)
		ctx.EndPoint.Adapter.QuitGroup(ctx, groupID)
	}

	if ctx.PrivilegeLevel == -30 {
		groupLevel := ctx.GroupRoleLevel
		if (d.Config.BanList.BanBehaviorQuitIfAdmin || d.Config.BanList.BanBehaviorQuitIfAdminSilentIfNotAdmin) && msg.MessageType == "group" {
			// 榛戝悕鍗曠敤鎴?- 绔嬪嵆閫€鍑烘墍鍦ㄧ兢
			banListInfoItem, _ := ctx.Dice.Config.BanList.GetByID(msg.Sender.UserID)
			reasontext := FormatBlacklistReasons(banListInfoItem)
			groupID := msg.GroupID
			notReply = true
			if groupLevel >= 40 {
				if isWhiteGroup {
					log.Infof("收到群(%s)内邀请者以上权限黑名单用户<%s>(%s)的消息，但在信任群所以不尝试退群", groupID, msg.Sender.Nickname, msg.Sender.UserID)
				} else {
					text := fmt.Sprintf("警告: <%s>(%s)是黑名单用户，将对骰主进行通知并退群。", msg.Sender.Nickname, msg.Sender.UserID)
					ReplyGroupRaw(ctx, &Message{GroupID: groupID}, text, "")

					noticeMsg := fmt.Sprintf("检测到群(%s)内黑名单用户<%s>(%s)，因是管理以上权限，执行通告后自动退群\n%s", groupID, msg.Sender.Nickname, msg.Sender.UserID, reasontext)
					log.Info(noticeMsg)
					ctx.Notice(noticeMsg)
					banQuitGroup()
				}
			} else {
				if isWhiteGroup {
					log.Infof("收到群(%s)内普通群员黑名单用户<%s>(%s)的消息，但在信任群所以不做其他操作", groupID, msg.Sender.Nickname, msg.Sender.UserID)
				} else {
					notReply = true
					if d.Config.BanList.BanBehaviorQuitIfAdmin {
						noticeMsg := fmt.Sprintf("检测到群(%s)内黑名单用户<%s>(%s)，因是普通群员，进行群内通告\n%s", groupID, msg.Sender.Nickname, msg.Sender.UserID, reasontext)
						log.Info(noticeMsg)

						text := fmt.Sprintf("警告: <%s>(%s)是黑名单用户，将对骰主进行通知。", msg.Sender.Nickname, msg.Sender.UserID)
						ReplyGroupRaw(ctx, &Message{GroupID: groupID}, text, "")

						ctx.Notice(noticeMsg)
					} else {
						noticeMsg := fmt.Sprintf("检测到群(%s)内黑名单用户<%s>(%s)，因是普通群员，忽略黑名单用户信息，不做其他操作\n%s", groupID, msg.Sender.Nickname, msg.Sender.UserID, reasontext)
						log.Info(noticeMsg)
					}
				}
			}
		} else if d.Config.BanList.BanBehaviorQuitPlaceImmediately && msg.MessageType == "group" {
			notReply = true
			// 榛戝悕鍗曠敤鎴?- 绔嬪嵆閫€鍑烘墍鍦ㄧ兢
			groupID := msg.GroupID
			if isWhiteGroup {
				log.Infof("收到群(%s)内黑名单用户<%s>(%s)的消息，但在信任群所以不尝试退群", groupID, msg.Sender.Nickname, msg.Sender.UserID)
			} else {
				banQuitGroup()
			}
		} else if d.Config.BanList.BanBehaviorRefuseReply {
			notReply = true
			// 榛戝悕鍗曠敤鎴?- 鎷掔粷鍥炲
			log.Infof("忽略黑名单用户信息，来自群(%s)内<%s>(%s): %s", msg.GroupID, msg.Sender.Nickname, msg.Sender.UserID, msg.Message)
		}
	} else if isBanGroup {
		if d.Config.BanList.BanBehaviorQuitPlaceImmediately && !isWhiteGroup {
			notReply = true
			// 榛戝悕鍗曠兢 - 绔嬪嵆閫€鍑?
			// 閫€缇や娇鐢℅roupID杩涜鍒ゆ柇
			banListInfoItem, _ := ctx.Dice.Config.BanList.GetByID(msg.GroupID)
			reasontext := FormatBlacklistReasons(banListInfoItem)
			groupID := msg.GroupID
			if isWhiteGroup {
				log.Infof("群(%s)处于黑名单中，但在信任群所以不尝试退群", groupID)
			} else {
				noticeMsg := fmt.Sprintf("群(%s)处于黑名单中，自动退群\n%s", groupID, reasontext)
				log.Info(noticeMsg)

				ReplyGroupRaw(ctx, &Message{GroupID: groupID}, "因本群处于黑名单中，将自动退群。", "")

				ctx.Notice(noticeMsg)

				time.Sleep(1 * time.Second)
				ctx.EndPoint.Adapter.QuitGroup(ctx, groupID)
			}
		} else if d.Config.BanList.BanBehaviorRefuseReply {
			notReply = true
			// 榛戝悕鍗曠兢 - 鎷掔粷鍥炲
			log.Infof("蹇界暐榛戝悕鍗曠兢娑堟伅: 鏉ヨ嚜缇?%s)鍐?%s>(%s): %s", msg.GroupID, msg.Sender.Nickname, msg.Sender.UserID, msg.Message)
		}
	}
	return notReply
}

func (s *IMSession) commandSolve(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) bool {
	// 璁剧疆涓存椂鍙橀噺
	if ctx.Player != nil {
		SetTempVars(ctx, msg.Sender.Nickname)
		VarSetValueStr(ctx, "$tMsgID", fmt.Sprintf("%v", msg.RawID))
		VarSetValueInt64(ctx, "$t杞暟", int64(cmdArgs.SpecialExecuteTimes))
	}

	tryItemSolve := func(ext *ExtInfo, item *CmdItemInfo) bool {
		if item == nil {
			return false
		}

		if item.Raw { //nolint:nestif
			if item.CheckCurrentBotOn {
				if !ctx.IsCurGroupBotOn && !ctx.IsPrivate {
					return false
				}
			}

			if item.CheckMentionOthers {
				if cmdArgs.SomeoneBeMentionedButNotMe {
					return false
				}
			}
		} else { //nolint:gocritic
			// 榛樿妯″紡琛屼负锛氶渶瑕佸湪褰撳墠缇?绉佽亰寮€鍚紝鎴朄鑷繁鏃剁敓鏁?闇€瑕佷负绗竴涓狜鐩爣)
			if !ctx.IsCurGroupBotOn && !ctx.IsPrivate {
				return false
			}
		}

		if ext != nil && ext.DefaultSetting.DisabledCommand[item.Name] {
			ReplyToSender(ctx, msg, fmt.Sprintf("姝ゆ寚浠ゅ凡琚涓荤鐢? %s:%s", ext.Name, item.Name))
			return true
		}

		// Note(Szzrain): TODO: 鎰忎箟涓嶆槑锛岄渶瑕佹兂鍔炴硶骞叉帀
		if item.EnableExecuteTimesParse {
			cmdArgs.RevokeExecuteTimesParse(ctx, msg)
		}

		if ctx.Player != nil {
			VarSetValueInt64(ctx, "$t杞暟", int64(cmdArgs.SpecialExecuteTimes))
		}

		if !item.Raw {
			if item.DisabledInPrivate && ctx.IsPrivate {
				ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:提示_私聊不可用"))
				return false
			}

			if item.AllowDelegate {
				// 鍏佽浠ｉ鏃讹紝鍙戜竴鍙ヨ瘽
				cur := -1
				for index, i := range cmdArgs.At {
					if i.UserID == ctx.EndPoint.UserID {
						continue
					} else if strings.HasPrefix(ctx.EndPoint.UserID, "OpenQQ:") {
						// 鐗规畩澶勭悊 OpenQQ棰戦亾
						uid := strings.TrimPrefix(i.UserID, "OpenQQCH:")
						diceId := strings.TrimPrefix(ctx.EndPoint.UserID, "OpenQQ:")
						if uid == diceId {
							continue
						}
					}
					cur = index
				}

				if cur != -1 {
					if ctx.Dice.Config.PlayerNameWrapEnable {
						ctx.DelegateText = fmt.Sprintf("鐢?%s>浠ｉ:\n", ctx.Player.Name)
					} else {
						ctx.DelegateText = fmt.Sprintf("由%s代骰:\n", ctx.Player.Name)
					}
				}
			} else if cmdArgs.SomeoneBeMentionedButNotMe {
				// 濡傛灉鍏朵粬浜鸿@浜嗗氨涓嶇
				// 娉? 濡傛灉琚獲鐨勫璞″湪botlist鍒楄〃锛岄偅涔堜笉浼氳蛋鍒拌繖涓€姝?
				return false
			}
		}

		// 鍔犺浇瑙勫垯妯℃澘
		// TODO: 娉ㄦ剰涓€涓嬭繖閲屼娇鐢ㄧ兢妯℃澘杩樻槸涓汉鍗℃ā鏉匡紝鐩墠缇ゆā鏉匡紝鍙湁鎯呭喌鐗规畩锛?
		tmpl := ctx.SystemTemplate
		if tmpl != nil {
			ctx.Eval(tmpl.InitScript, nil)
			if tmpl.Name == "dnd5e" {
				// 杩欓噷闈㈡湁buff鏈哄埗鐨勪唬鐮侊紝鎵€浠ラ渶瑕佸姞杞?
				ctx.setDndReadForVM(false)
			}
		}

		var ret CmdExecuteResult
		// 濡傛灉鏄痡s鍛戒护锛岄偅涔堝姞閿?
		if item.IsJsSolveFunc {
			loop, err := s.Parent.ExtLoopManager.GetLoop(item.JSLoopVersion)
			if err != nil {
				// 鎵撲釜DEBUG鏃ュ織锛?
				s.Parent.Logger.Errorf("鎵╁睍娉ㄥ唽鐨勬寚浠?%s>杩愯鐜宸茬粡杩囨湡: %v", item.Name, err)
				return false
			}
			waitRun := make(chan int, 1)
			loop.RunOnLoop(func(vm *goja.Runtime) {
				defer func() {
					if r := recover(); r != nil {
						// log.Errorf("寮傚父: %v 鍫嗘爤: %v", r, string(debug.Stack()))
						ReplyToSender(ctx, msg, fmt.Sprintf("JS鎵ц寮傚父锛岃鍙嶉缁欒鎵╁睍鐨勪綔鑰咃細\n%v", r))
					}
					waitRun <- 1
				}()

				ret = item.Solve(ctx, msg, cmdArgs)
			})
			<-waitRun
		} else {
			ret = item.Solve(ctx, msg, cmdArgs)
		}

		if ret.Solved {
			if ret.ShowHelp {
				help := ""
				// 浼樺厛鑰冭檻鍑芥暟
				if item.HelpFunc != nil {
					help = item.HelpFunc(false)
				}
				// 鍏舵鑰冭檻help
				if help == "" {
					help = item.Help
				}
				// 鏈€鍚庣敤鐭環elp鎷?
				if help == "" {
					// 杩欐槸涓轰簡闃叉鍒殑楠板瓙璇Е鍙?
					help = item.Name + ":\n" + item.ShortHelp
				}
				ReplyToSender(ctx, msg, help)
			}

			return true
		}
		return false
	}

	group := ctx.Group
	builtinSolve := func() bool {
		item := ctx.Session.Parent.CmdMap[cmdArgs.Command]
		if tryItemSolve(nil, item) {
			return true
		}

		if group != nil && (group.Active || ctx.IsCurGroupBotOn) {
			for _, wrapper := range group.GetActivatedExtList(ctx.Dice) {
				cmdMap := wrapper.GetCmdMap()
				item := cmdMap[cmdArgs.Command]
				if tryItemSolve(wrapper, item) {
					return true
				}
			}
		}
		return false
	}

	solved := builtinSolve()
	if group.Active || ctx.IsCurGroupBotOn {
		for _, wrapper := range group.GetActivatedExtList(ctx.Dice) {
			ext := wrapper.GetRealExt()
			if ext == nil {
				continue
			}
			if ext.OnCommandReceived != nil {
				ext.callWithJsCheck(ctx.Dice, func() {
					ext.OnCommandReceived(ctx, msg, cmdArgs)
				})
			}
		}
	}

	return solved
}

func (s *IMSession) OnMessageDeleted(mctx *MsgContext, msg *Message) {
	d := mctx.Dice
	mctx.MessageType = msg.MessageType
	mctx.IsPrivate = mctx.MessageType == "private"
	group, ok := s.ServiceAtNew.Load(msg.GroupID)
	if !ok {
		return
	}
	mctx.Group = group
	mctx.Group, mctx.Player = GetPlayerInfoBySender(mctx, msg)

	mctx.IsCurGroupBotOn = msg.MessageType == "group" && mctx.Group.IsActive(mctx)
	if mctx.Group != nil && mctx.Group.System != "" {
		mctx.SystemTemplate = mctx.Group.GetCharTemplate(d)
		// tmpl, _ := d.GameSystemMap.Load(group.System)
		// mctx.SystemTemplate = tmpl
	}

	_ = mctx.fillPrivilege(msg)

	for _, i := range s.Parent.ExtList {
		i.CallOnMessageDeleted(mctx.Dice, mctx, msg)
	}
}

func (s *IMSession) OnMessageSend(ctx *MsgContext, msg *Message, flag string) {
	for _, i := range s.Parent.ExtList {
		i.CallOnMessageSend(ctx.Dice, ctx, msg, flag)
	}
}

func (s *IMSession) OnPoke(ctx *MsgContext, event *events.PokeEvent) {
	if ctx == nil || event == nil {
		return
	}
	// Poke 浜嬩欢鍙兘缂哄皯缇?鎴愬憳淇℃伅锛堜緥濡?OneBot 鑾峰彇缇ゆ垚鍛樹俊鎭け璐ユ椂锛夛紝閬垮厤绌烘寚閽堝鑷村穿婧冦€?
	if ctx.Group == nil && event.GroupID != "" {
		if group, ok := s.ServiceAtNew.Load(event.GroupID); ok {
			ctx.Group = group
		} else {
			// 纭繚缇や俊鎭嚦灏戣鍒濆鍖栧埌鍏ㄥ眬鍒楄〃锛屼究浜庡悗缁墿灞曡鍙?鍐欏叆
			ctx.Group = SetBotOnAtGroup(ctx, event.GroupID)
		}
	}
	if ctx.Group == nil {
		return
	}
	if ctx.MessageType == "group" && !ctx.Group.IsActive(ctx) {
		return
	}
	for _, wrapper := range ctx.Group.GetActivatedExtList(ctx.Dice) {
		ext := wrapper.GetRealExt()
		if ext == nil {
			continue
		}
		if ext.OnPoke != nil {
			ext.callWithJsCheck(ctx.Dice, func() {
				ext.OnPoke(ctx, event)
			})
		}
	}
}

func (s *IMSession) OnGroupLeave(ctx *MsgContext, event *events.GroupLeaveEvent) {
	for _, i := range s.Parent.ExtList {
		i.CallOnGroupLeave(ctx.Dice, ctx, event)
	}
}

// OnMessageEdit 娑堟伅缂栬緫浜嬩欢
//
// msg.Message 搴斾负鏇存柊鍚庣殑娑堟伅, msg.Time 搴斾负鏇存柊鏃堕棿鑰岄潪鍙戦€佹椂闂达紝鍚屾椂
// msg.RawID 搴旂‘淇濅负鍘熸秷鎭殑 ID (涓€浜?API 鍚屾椂浼氭湁绯荤粺浜嬩欢 ID锛屽嬁娣锋穯)
//
// 渚濇嵁 API锛孲ender 涓嶄竴瀹氬瓨鍦紝ctx 淇℃伅浜︿笉涓€瀹氭湁鏁?
func (s *IMSession) OnMessageEdit(ctx *MsgContext, msg *Message) {
	m := fmt.Sprintf("鏉ヨ嚜%s鐨勬秷鎭慨鏀逛簨浠? %s",
		msg.GroupID,
		msg.Message,
	)
	s.Parent.Logger.Info(m)

	if group, ok := s.ServiceAtNew.Load(msg.GroupID); ok {
		ctx.Group = group
	} else {
		return
	}

	group := ctx.Group
	if group.Active || ctx.IsCurGroupBotOn {
		for _, wrapper := range group.GetActivatedExtList(ctx.Dice) {
			ext := wrapper.GetRealExt()
			if ext == nil {
				continue
			}
			if ext.OnMessageEdit != nil {
				ext.callWithJsCheck(ctx.Dice, func() {
					ext.OnMessageEdit(ctx, msg)
				})
			}
		}
	}
}

// GetEpByPlatform
// 鍦?EndPoints 涓壘鍒扮涓€涓鍚堝钩鍙?p 涓斿惎鐢ㄧ殑
func (s *IMSession) GetEpByPlatform(p string) *EndPointInfo {
	for _, ep := range s.EndPoints {
		if ep.Enable && ep.Platform == p {
			return ep
		}
	}
	return nil
}

// SetEnable
/* 濡傛灉宸茶繛鎺ワ紝灏嗘柇寮€杩炴帴锛屽鏋滃紑鐫€GCQ灏嗚嚜鍔ㄧ粨鏉熴€傚鏋滃惎鐢ㄧ殑璇濓紝鍒欏弽杩囨潵  */
func (ep *EndPointInfo) SetEnable(_ *Dice, enable bool) {
	if ep.Enable != enable {
		ep.Adapter.SetEnable(enable)
	}
}

func (ep *EndPointInfo) AdapterSetup() {
	switch ep.Platform {
	case "QQ":
		switch ep.ProtocolType {
		case "onebot":
			pa := ep.Adapter.(*PlatformAdapterGocq)
			pa.Session = ep.Session
			pa.EndPoint = ep
		case "walle-q":
			pa := ep.Adapter.(*PlatformAdapterWalleQ)
			pa.Session = ep.Session
			pa.EndPoint = ep
		case "red":
			pa := ep.Adapter.(*PlatformAdapterRed)
			pa.Session = ep.Session
			pa.EndPoint = ep
		case "official":
			pa := ep.Adapter.(*PlatformAdapterOfficialQQ)
			pa.Session = ep.Session
			pa.EndPoint = ep
		case "satori":
			pa := ep.Adapter.(*PlatformAdapterSatori)
			pa.Session = ep.Session
			pa.EndPoint = ep
		case "milky":
			pa := ep.Adapter.(*PlatformAdapterMilky)
			pa.Session = ep.Session
			pa.EndPoint = ep
		case "pureonebot":
			pa := ep.Adapter.(*PlatformAdapterOnebot)
			log := zap.S().Named(logger.LogKeyAdapter)
			pa.Session = ep.Session
			pa.EndPoint = ep
			pa.logger = log
			pa.desiredEnabled = ep.Enable
			// case "LagrangeGo":
			//	pa := ep.Adapter.(*PlatformAdapterLagrangeGo)
			//	pa.Session = ep.Session
			//	pa.EndPoint = ep
		}
	case "DISCORD":
		pa := ep.Adapter.(*PlatformAdapterDiscord)
		pa.Session = ep.Session
		pa.EndPoint = ep
	case "KOOK":
		pa := ep.Adapter.(*PlatformAdapterKook)
		pa.Session = ep.Session
		pa.EndPoint = ep
	case "TG":
		pa := ep.Adapter.(*PlatformAdapterTelegram)
		pa.Session = ep.Session
		pa.EndPoint = ep
	case "MC":
		pa := ep.Adapter.(*PlatformAdapterMinecraft)
		pa.Session = ep.Session
		pa.EndPoint = ep
	case "DODO":
		pa := ep.Adapter.(*PlatformAdapterDodo)
		pa.Session = ep.Session
		pa.EndPoint = ep
	case "DINGTALK":
		pa := ep.Adapter.(*PlatformAdapterDingTalk)
		pa.Session = ep.Session
		pa.EndPoint = ep
	case "SLACK":
		pa := ep.Adapter.(*PlatformAdapterSlack)
		pa.Session = ep.Session
		pa.EndPoint = ep
	case "SEALCHAT":
		pa := ep.Adapter.(*PlatformAdapterSealChat)
		pa.Session = ep.Session
		pa.EndPoint = ep
	}
}

func (ep *EndPointInfo) RefreshGroupNum() {
	serveCount := 0
	session := ep.Session
	if session != nil && session.ServiceAtNew != nil {
		// Pinenutn: Range妯℃澘 ServiceAtNew閲嶆瀯浠ｇ爜
		session.ServiceAtNew.Range(func(key string, groupInfo *GroupInfo) bool {
			// Pinenutn: ServiceAtNew閲嶆瀯
			if groupInfo.GroupID != "" {
				if strings.HasPrefix(groupInfo.GroupID, "PG-") {
					return true
				}
				if groupInfo.DiceIDExistsMap.Exists(ep.UserID) {
					serveCount++
					// 鍦ㄧ兢鍐呯殑寮€鍚暟閲忔墠琚绠楋紝铏界劧涔熸湁琚涪鍑虹殑
					// if groupInfo.DiceIdActiveMap.Exists(ep.UserId) {
					// activeCount += 1
					// }
				}
			}
			return true
		})
		ep.GroupNum = int64(serveCount)
	}
}

func (d *Dice) NoticeForEveryEndpoint(txt string, allowCrossPlatform bool) {
	_ = allowCrossPlatform
	// 閫氱煡绉嶇被涔嬩竴锛氭瘡涓猲oticeId  *  姣忎釜骞冲彴鍖归厤鐨別p锛氬瓨娲?
	// TODO: 鍏堝鍒跺嚑娆″疄鐜帮紝鍚庨潰閲嶆瀯
	// Pinenutn: 鍟ユ椂鍊欓噸鏋勫晩.jpg
	foo := func() {
		defer func() {
			if r := recover(); r != nil {
				d.Logger.Errorf("鍙戦€侀€氱煡寮傚父: %v 鍫嗘爤: %v", r, string(debug.Stack()))
			}
		}()

		if d.Config.MailEnable {
			_ = d.SendMail(txt, MailTypeNotice)
			return
		}

		for _, ep := range d.ImSession.EndPoints {
			for _, i := range d.Config.NoticeIDs {
				n := strings.Split(i, ":")
				// 濡傛灉鏂囨湰涓病鏈?锛屽垯浼氬彇鍒版暣涓瓧绗︿覆
				// 浣嗗ソ鍍忎笉涓ヨ皑锛屾瘮濡俀Q-CH-Group
				prefix := strings.Split(n[0], "-")[0]

				if len(n) >= 2 && prefix == ep.Platform && ep.Enable && ep.State == 1 {
					if ep.Session == nil {
						ep.Session = d.ImSession
					}
					if strings.HasSuffix(n[0], "-Group") {
						msg := &Message{GroupID: i, MessageType: "private", Sender: SenderBase{UserID: i}}
						ctx := CreateTempCtx(ep, msg)
						ReplyGroup(ctx, msg, txt)
					} else {
						msg := &Message{GroupID: i, MessageType: "group", Sender: SenderBase{UserID: i}}
						ctx := CreateTempCtx(ep, msg)
						ReplyPerson(ctx, &Message{Sender: SenderBase{UserID: i}}, txt)
					}
				}
				time.Sleep(1 * time.Second)
			}
		}
	}
	go foo()
}

func (ctx *MsgContext) NoticeCrossPlatform(txt string) {
	// 閫氱煡绉嶇被涔嬩簩锛氭瘡涓猲oticeID  *  绗竴涓钩鍙板尮閰嶇殑ep锛氳法骞冲彴閫氱煡
	// TODO: 鍏堝鍒跺嚑娆″疄鐜帮紝鍚庨潰閲嶆瀯
	foo := func() {
		defer func() {
			if r := recover(); r != nil {
				ctx.Dice.Logger.Errorf("鍙戦€侀€氱煡寮傚父: %v 鍫嗘爤: %v", r, string(debug.Stack()))
			}
		}()

		if ctx.Dice.Config.MailEnable {
			_ = ctx.Dice.SendMail(txt, MailTypeNotice)
			return
		}

		sent := false

		for _, i := range ctx.Dice.Config.NoticeIDs {
			n := strings.Split(i, ":")
			if len(n) < 2 {
				continue
			}

			seg := strings.Split(n[0], "-")[0]

			messageType := "private"
			if strings.HasSuffix(n[0], "-Group") {
				messageType = "group"
			}

			if ctx.EndPoint.Platform == seg {
				if messageType == "group" {
					ReplyGroup(ctx, &Message{GroupID: i}, txt)
				} else {
					ReplyPerson(ctx, &Message{Sender: SenderBase{UserID: i}}, txt)
				}
				time.Sleep(1 * time.Second)
				sent = true
				continue // 鎵惧埌瀵瑰簲骞冲彴銆佽皟鐢ㄤ簡鍙戦€佺殑鍦ㄦ鍗冲垏鍑哄惊鐜?
			}

			// 濡傛灉璧板埌杩欓噷锛岃鏄庡綋鍓峞p涓嶆槸noticeID瀵瑰簲鐨勫钩鍙?
			if done := CrossMsgBySearch(ctx.Session, seg, i, txt, messageType == "private"); !done {
				ctx.Dice.Logger.Errorf("灏濊瘯璺ㄥ钩鍙板悗浠嶆湭鑳藉悜 %s 鍙戦€侀€氱煡锛?s", i, txt)
			} else {
				sent = true
				time.Sleep(1 * time.Second)
			}
		}

		if !sent {
			ctx.Dice.Logger.Errorf("鏈兘鍙戦€佹潵鑷?s鐨勯€氱煡锛?s", ctx.EndPoint.Platform, txt)
		}
	}
	go foo()
}

func (ctx *MsgContext) Notice(txt string) {
	// Notice
	// 閫氱煡绉嶇被涔嬩笁锛氭瘡涓猲oticeID  * 褰撳墠mctx鐨別p锛氫笉璺ㄥ钩鍙伴€氱煡
	// TODO: 鍏堝鍒跺嚑娆″疄鐜帮紝鍚庨潰閲嶆瀯
	foo := func() {
		defer func() {
			if r := recover(); r != nil {
				ctx.Dice.Logger.Errorf("鍙戦€侀€氱煡寮傚父: %v 鍫嗘爤: %v", r, string(debug.Stack()))
			}
		}()

		if ctx.Dice.Config.MailEnable {
			_ = ctx.Dice.SendMail(txt, MailTypeNotice)
			return
		}

		sent := false
		if ctx.EndPoint.Enable {
			for _, i := range ctx.Dice.Config.NoticeIDs {
				n := strings.Split(i, ":")
				if len(n) >= 2 {
					if strings.HasSuffix(n[0], "-Group") {
						ReplyGroup(ctx, &Message{GroupID: i}, txt)
					} else {
						ReplyPerson(ctx, &Message{Sender: SenderBase{UserID: i}}, txt)
					}
					sent = true
				}
				time.Sleep(1 * time.Second)
			}
		}

		if !sent {
			if len(ctx.Dice.Config.NoticeIDs) != 0 {
				ctx.Dice.Logger.Errorf("鏈兘鍙戦€佹潵鑷?s鐨勯€氱煡锛?s", ctx.EndPoint.Platform, txt)
			} else {
				ctx.Dice.Logger.Warnf("鍥犱负娌℃湁閰嶇疆閫氱煡鍒楄〃锛屾棤娉曞彂閫佹潵鑷?s鐨勯€氱煡锛?s", ctx.EndPoint.Platform, txt)
			}
		}
	}
	go foo()
}

var randSourceSplitKey = rand2.NewSource(uint64(time.Now().Unix()))

func (ctx *MsgContext) InitSplitKey() {
	if len(ctx.splitKey) > 0 {
		return
	}
	r := randSourceSplitKey.Uint64()
	bArray := make([]byte, 12)
	binary.LittleEndian.PutUint64(bArray[:8], r)
	r = randSourceSplitKey.Uint64()
	binary.LittleEndian.PutUint32(bArray[8:], uint32(r))

	s := base64.StdEncoding.EncodeToString(bArray)
	ctx.splitKey = "###" + s + "###"
}

func (ctx *MsgContext) TranslateSplit(s string) string {
	if len(ctx.splitKey) == 0 {
		ctx.InitSplitKey()
	}
	s = strings.ReplaceAll(s, "#{SPLIT}", ctx.splitKey)
	s = strings.ReplaceAll(s, "{FormFeed}", ctx.splitKey)
	s = strings.ReplaceAll(s, "{formfeed}", ctx.splitKey)
	s = strings.ReplaceAll(s, "\f", ctx.splitKey)
	s = strings.ReplaceAll(s, "\\f", ctx.splitKey)
	return s
}

func (ctx *MsgContext) SplitText(text string) []string {
	if len(ctx.splitKey) == 0 {
		return []string{text}
	}
	return strings.Split(text, ctx.splitKey)
}

var curCommandID int64 = 0

func getNextCommandID() int64 {
	curCommandID++
	return curCommandID
}
