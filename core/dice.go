package main

import (
	"encoding/json"
	"errors"
	"fmt"
	wr "github.com/mroth/weightedrand"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"sealdice-core/core"
	"sealdice-core/model"
	"time"
)

var VERSION = "0.9测试版 v20220226"

type CmdItemInfo struct {
	name  string
	solve func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) struct{ success bool }
	Brief string
	texts map[string]string
	Help  string
}

type CmdMapCls map[string]*CmdItemInfo

type ExtInfo struct {
	Name    string // 名字
	version string // 版本
	// 作者
	// 更新时间
	autoActive      bool      // 是否自动开启
	cmdMap          CmdMapCls `yaml:"-"` // 指令集合
	Brief           string    `yaml:"-"`
	ActiveOnPrivate bool      `yaml:"-"`

	Author string `yaml:"-"`
	//activeInSession bool; // 在当前会话中开启

	OnCommandReceived func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) `yaml:"-"`
	OnMessageReceived func(ctx *MsgContext, msg *Message) `yaml:"-"`
	OnMessageSend func(ctx *MsgContext, messageType string, userId int64, text string, flag string) `yaml:"-"`
	GetDescText       func(i *ExtInfo) string                                  `yaml:"-"`
	IsLoaded         bool                                                     `yaml:"-"`
	OnLoad            func()                                                   `yaml:"-"`
}


type Dice struct {
	ImSession             *IMSession `yaml:"imSession"`
	cmdMap                CmdMapCls
	extList               []*ExtInfo
	RollParser            *DiceRollParser `yaml:"-"`
	CommandCompatibleMode bool            `yaml:"commandCompatibleMode"`
	lastSavedTime         *time.Time      `yaml:"lastSavedTime"`
	TextMap               map[string]*wr.Chooser `yaml:"-"`
}

func (self *Dice) init() {
	self.CommandCompatibleMode = true
	self.ImSession = &IMSession{}
	self.ImSession.parent = self
	self.ImSession.ServiceAt = make(map[int64]*ServiceAtItem)
	self.cmdMap = CmdMapCls{}

	self.registerCoreCommands()
	self.RegisterBuiltinExt()
	self.loads()

	for _, i := range self.extList {
		if i.OnLoad != nil {
			i.OnLoad()
		}
	}

	autoSave := func() {
		t := time.Tick(15 * time.Second)
		for {
			<-t
			self.save()
		}
	}
	go autoSave()

	refreshGroupInfo := func() {
		t := time.Tick(35 * time.Second)
		for {
			<-t
			for k := range self.ImSession.ServiceAt {
				GetGroupInfo(self.ImSession.Socket, k)
			}
		}
	}
	go refreshGroupInfo()
}


func (self *Dice) rebuildParser(buffer string) *DiceRollParser {
	p := &DiceRollParser{Buffer: buffer}
	_ = p.Init()
	p.RollExpression.Init(255)
	//self.RollParser = p;
	return p
}

func (self *Dice) exprEvalBase(buffer string, ctx *MsgContext, bigFailDice bool) (*vmResult, string, error) {
	parser := self.rebuildParser(buffer)
	err := parser.Parse()
	parser.RollExpression.BigFailDiceOn = bigFailDice

	if err == nil {
		parser.Execute()
		if parser.error != nil{
			return nil, "", parser.error
		}
		num, detail, _ := parser.Evaluate(self, ctx)
		ret := vmResult{}
		ret.Value = num.Value
		ret.TypeId = num.TypeId
		ret.parser = parser
		return &ret, detail, nil
	}
	return nil, "", err
}

func (self *Dice) exprEval(buffer string, ctx *MsgContext) (*vmResult, string, error) {
	return self.exprEvalBase(buffer, ctx, false)
}

func (self *Dice) exprText(buffer string, ctx *MsgContext) (string, string, error) {
	val, detail, err := self.exprEval("`" + buffer + "`", ctx)

	if err == nil && val.TypeId == VMTypeString {
		return val.Value.(string), detail, err
	}

	return "", "", errors.New("错误的表达式")
}

func (self *Dice) loads() {
	os.MkdirAll("./data", 0644)
	data, err := ioutil.ReadFile(BASE_CONFIG)

	if err == nil {
		session := self.ImSession
		d := Dice{}
		err2 := yaml.Unmarshal(data, &d)
		if err2 == nil {
			m := map[string]*ExtInfo{}
			for _, i := range self.extList {
				m[i.Name] = i
			}

			session.ServiceAt = d.ImSession.ServiceAt
			for _, v := range d.ImSession.ServiceAt {
				tmp := []*ExtInfo{}
				for _, i := range v.ActivatedExtList {
					if m[i.Name] != nil {
						tmp = append(tmp, m[i.Name])
					}
				}
				v.ActivatedExtList = tmp
			}

			// 读取新版数据
			for _, g := range d.ImSession.ServiceAt {
				// 群组数据
				data := model.AttrGroupGetAll(g.GroupId)
				err := JsonValueMapUnmarshal(data, &g.ValueMap)
				if err != nil {
					core.GetLogger().Error(err)
				}
				if g.ValueMap == nil {
					g.ValueMap = map[string]VMValue{}
				}

				// 个人群组数据
				for _, p := range g.Players {
					if p.ValueMap == nil {
						p.ValueMap = map[string]VMValue{}
					}
					if p.ValueMapTemp == nil {
						p.ValueMapTemp = map[string]VMValue{}
					}

					data := model.AttrGroupUserGetAll(g.GroupId, p.UserId)
					err := JsonValueMapUnmarshal(data, &p.ValueMap)
					if err != nil {
						core.GetLogger().Error(err)
					}
				}
			}

			// 旧版数据转写
			for _, g := range d.ImSession.ServiceAt {
				for _, b := range g.Players {
					for k, v := range b.ValueNumMap {
						b.ValueMap[k] = VMValue{VMTypeInt64, v}
					}
					for k, v := range b.ValueStrMap {
						b.ValueMap[k] = VMValue{VMTypeString, v}
					}
				}
			}

			log.Println("config.yaml loaded")
			//info, _ := yaml.Marshal(session.ServiceAt)
			//replyGroup(ctx, msg.GroupId, fmt.Sprintf("临时指令：加载配置 似乎成功\n%s", info));
		} else {
			log.Println("config.yaml parse failed")
			panic(err2)
		}
	} else {
		log.Println("config.yaml not found")
	}

	// 读取文本模板
	data, err = ioutil.ReadFile(CONFIG_TEXT_TEMPLATE_FILE)
	if err != nil {
		panic(err)
	}
	texts := TextTemplateWithWeightDict{}
	err = yaml.Unmarshal(data, &texts)
	if err != nil {
		panic(err)
	}
	self.TextMap = map[string]*wr.Chooser{}

	for category, item := range texts {
		for k, v := range item {
			choices := []wr.Choice{}
			for t, w := range v {
				choices = append(choices, wr.Choice{Item: t, Weight: w})
			}

			pool, _ := wr.NewChooser(choices...)
			self.TextMap[fmt.Sprintf("%s:%s", category, k)] = pool
		}
	}

	picker, _ := wr.NewChooser(wr.Choice{VERSION, 1})
	self.TextMap["常量:VERSION"] = picker
}

func (self *Dice) save() {
	//for _, g := range self.ImSession.ServiceAt {
	//	for _, ui := range g.Players {
	//		ui.ValueStrMap = nil
	//		ui.ValueNumMap = nil
	//	}
	//}

	a, err := yaml.Marshal(self)
	if err == nil {
		err := ioutil.WriteFile(BASE_CONFIG, a, 0644)
		if err == nil {
			now := time.Now()
			self.lastSavedTime = &now
			fmt.Println("此时的用户ID", self.ImSession.UserId)
			if self.ImSession.UserId == 0 {
				self.ImSession.GetLoginInfo()
			}
		}
	}

	userIds := map[int64]bool{}
	for _, g := range self.ImSession.ServiceAt {
		for _, b := range g.Players {
			userIds[b.UserId] = true
			data, _ := json.Marshal(b.ValueMap)
			model.AttrGroupUserSave(g.GroupId, b.UserId, data)
		}

		data, _ := json.Marshal(g.ValueMap)
		model.AttrGroupSave(g.GroupId, data)
	}

	// 保存玩家个人全局数据
	for k, v := range self.ImSession.PlayerVarsData {
		if v.Loaded {
			data, _ := json.Marshal(v.ValueMap)
			model.AttrUserSave(k, data)
		}
	}
}

func DiceRoll(dicePoints int) int {
	if dicePoints <= 0 {
		return 0
	}
	val := rand.Int() % dicePoints + 1
	return val
}

func DiceRoll64(dicePoints int64) int64 {
	if dicePoints == 0 {
		return 0
	}
	val := rand.Int63() % dicePoints + 1
	return val
}

func isCurGroupBotOn(session *IMSession, msg *Message) bool {
	return msg.MessageType == "group" && session.ServiceAt[msg.GroupId] != nil && session.ServiceAt[msg.GroupId].Active
}

func isCurGroupBotOnById(session *IMSession, messageType string, groupId int64) bool {
	return messageType == "group" && session.ServiceAt[groupId] != nil && session.ServiceAt[groupId].Active
}

/** 获取玩家群内信息，没有就创建 */
func getPlayerInfoBySender(session *IMSession, msg *Message) *PlayerInfo {
	if msg.MessageType == "group" {
		g := session.ServiceAt[msg.GroupId]
		if g == nil {
			return nil
		}
		players := g.Players
		p := players[msg.Sender.UserId]
		if p == nil {
			p = &PlayerInfo{
				Name:         msg.Sender.Nickname,
				UserId:       msg.Sender.UserId,
				ValueNumMap:  map[string]int64{},
				ValueStrMap:  map[string]string{},
				ValueMap:     map[string]VMValue{},
				ValueMapTemp: map[string]VMValue{},
			}
			players[msg.Sender.UserId] = p
		}
		if p.ValueMap == nil {
			p.ValueMap = map[string]VMValue{}
		}
		if p.ValueMapTemp == nil {
			p.ValueMapTemp = map[string]VMValue{}
		}
		return p
	}
	return nil
}

