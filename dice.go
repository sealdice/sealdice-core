package main

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"math/rand"
	"time"
)

type CmdItemInfo struct {
	name string;
	solve func(session *IMSession, msg *Message, cmdArgs *CmdArgs) struct { success bool };
}

type CmdMapCls map[string]*CmdItemInfo;

type ExtInfo struct {
	Name       string     // 名字
	version    string     // 版本
	autoActive bool       // 是否自动开启
	cmdMap     CmdMapCls `yaml:"-"`  // 指令集合
	//activeInSession bool; // 在当前会话中开启
}

type Dice struct {
	ImSession *IMSession `yaml:"imSession"`;
	cmdMap    CmdMapCls;
	extList   []*ExtInfo;
	RollParser *DiceRollParser `yaml:"-"`;
}

func (self *Dice) init() {
	self.ImSession = &IMSession{};
	self.ImSession.parent = self;
	self.ImSession.ServiceAt = make(map[int64]*ServiceAtItem);
	self.cmdMap = CmdMapCls{}

	self.registerCoreCommands();
	self.registerBuiltinExt()
	self.loads();
}


func (self *Dice) rebuildParser(buffer string) {
	p := &DiceRollParser{Buffer: buffer}
	_ = p.Init()
	p.RollExpression.Init(255)
	self.RollParser = p;
}

func (self *Dice) loads() {
	data, err := ioutil.ReadFile("save.yaml")

	if err == nil {
		session := self.ImSession
		d := Dice{}
		err2 := yaml.Unmarshal(data, &d)
		if err2 == nil {
			m := map[string]*ExtInfo{};
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
				v.ActivatedExtList = tmp;
			}

			log.Println("save.yaml loaded")
			//info, _ := yaml.Marshal(session.ServiceAt)
			//replyGroup(session.Socket, msg.GroupId, fmt.Sprintf("临时指令：加载配置 似乎成功\n%s", info));
		} else {
			log.Println("save.yaml parse failed")
			panic(err2)
		}
	} else {
		log.Println("save.yaml not found")
	}
}

func (self Dice) save() {
	
}

func DiceRoll(dicePoints int) int {
	if dicePoints <= 0 {
		return 0
	}
	val := rand.Int() % dicePoints + 1;
	return val
}

func DiceRoll64(dicePoints int64) int64 {
	if dicePoints == 0 {
		return 0
	}
	val := rand.Int63() % dicePoints + 1;
	return val
}

/** 这几条指令不能移除 */
func (self *Dice) registerCoreCommands() {
	cmdBot := &CmdItemInfo{
		name: "bot",
		solve: func(session *IMSession, msg *Message, cmdArgs *CmdArgs) struct{ success bool } {
			if msg.MessageType == "group" && cmdArgs.AmIBeMentioned && len(cmdArgs.Args) >= 1 {
				if cmdArgs.Args[0] == "on" {
					if session.ServiceAt[msg.GroupId] != nil {
						session.ServiceAt[msg.GroupId].Active = true;
					} else {
						extLst := []*ExtInfo{};
						for _, i := range self.extList {
							if i.autoActive {
								extLst = append(extLst, i)
							}
						}
						session.ServiceAt[msg.GroupId] = &ServiceAtItem{
							Active: true,
							ActivatedExtList: extLst,
							Players: map[int64]*PlayerInfo{},
						};
					}
					replyGroup(session.Socket, msg.GroupId, "SealDice 已启用(开发中) v20220210");
				} else if cmdArgs.Args[0] == "off" {
					if len(session.ServiceAt[msg.GroupId].ActivatedExtList) == 0 {
						delete(session.ServiceAt, msg.GroupId);
					} else {
						session.ServiceAt[msg.GroupId].Active = false;
					}
					replyGroup(session.Socket, msg.GroupId, "停止服务");
				}
			}
			return struct{ success bool }{
				success: true,
			}
		},
	}
	self.cmdMap["bot"] = cmdBot;

	cmdRoll := &CmdItemInfo{
		name: "roll",
		solve: func(session *IMSession, msg *Message, cmdArgs *CmdArgs) struct{ success bool } {
			if isCurGroupBotOn(session, msg) {
				var text string;
				var prefix string;
				var diceResult int64
				p := getPlayerInfoBySender(session, msg)

				forWhat := "";
				if len(cmdArgs.Args) >= 1 {
					session.parent.rebuildParser(cmdArgs.Args[0])
					p := session.parent.RollParser;

					if err := p.Parse(); err != nil {
						fmt.Println("???", err)
						forWhat = cmdArgs.Args[0];
					} else {
						p.Execute()
						diceResult = p.Evaluate().Int64();

						if len(cmdArgs.Args) >= 2 {
							forWhat = cmdArgs.Args[1]
						}
					}
				}

				if forWhat != "" {
					prefix = "为了" + forWhat + "，";
				}

				if diceResult != 0 {
					text = fmt.Sprintf("%s<%s>掷出了 %s=%d", prefix, p.Name, cmdArgs.Args[0], diceResult);
				} else {
					dicePoints := 100
					val := DiceRoll(dicePoints);
					text = fmt.Sprintf("%s<%s>掷出了 D%d=%d", prefix, p.Name, dicePoints, val);
				}

				replyGroup(session.Socket, msg.GroupId, text);
			}

			return struct{ success bool }{
				success: true,
			}
		},
	}
	self.cmdMap["r"] = cmdRoll;
	self.cmdMap["roll"] = cmdRoll;

	cmdExt := &CmdItemInfo{
		name: "ext",
		solve: func(session *IMSession, msg *Message, cmdArgs *CmdArgs) struct{ success bool } {
			if msg.MessageType == "group" && len(cmdArgs.Args) >= 1 {
				if cmdArgs.Args[0] == "list" {
					text := "检测到以下扩展：\n"
					for index, i := range session.parent.extList {
						text += fmt.Sprintf("%d. [%s] version %s\n", index + 1, i.Name, i.version)
					}
					text += "使用命令 ”.ext on/off 扩展名“ 可以在当前群开启或关闭某扩展。"
					replyGroup(session.Socket, msg.GroupId, text);
				} else if cmdArgs.Args[0] == "on" && session.ServiceAt[msg.GroupId] != nil && session.ServiceAt[msg.GroupId].Active {
					if len(cmdArgs.Args) >= 2 {
						extName := cmdArgs.Args[1];
						for _, i := range self.extList {
							if i.Name == extName {
								session.ServiceAt[msg.GroupId].ActivatedExtList = append(session.ServiceAt[msg.GroupId].ActivatedExtList, i);
								replyGroup(session.Socket, msg.GroupId, fmt.Sprintf("打开扩展 %s", extName));
								break;
							}
						}
					}
				} else if cmdArgs.Args[0] == "off" && session.ServiceAt[msg.GroupId] != nil && session.ServiceAt[msg.GroupId].Active {
					if len(cmdArgs.Args) >= 2 {
						gInfo := session.ServiceAt[msg.GroupId]
						extName := cmdArgs.Args[1];
						for index, i := range gInfo.ActivatedExtList {
							if i.Name == extName {
								gInfo.ActivatedExtList = append(gInfo.ActivatedExtList[:index], gInfo.ActivatedExtList[index+1:]...)
								replyGroup(session.Socket, msg.GroupId, fmt.Sprintf("关闭扩展 %s", extName));
								break;
							}
						}
					}
				}
			}
			return struct{ success bool }{
				success: true,
			}
		},
	}
	self.cmdMap["ext"] = cmdExt;

	cmdNN := &CmdItemInfo{
		name: "nn",
		solve: func(session *IMSession, msg *Message, cmdArgs *CmdArgs) struct{ success bool } {
			if msg.MessageType == "group" {
				if isCurGroupBotOn(session, msg) {
					if len(cmdArgs.Args) == 0 {
						p := getPlayerInfoBySender(session, msg)
						p.Name = msg.Sender.Nickname;
						replyGroup(session.Socket, msg.GroupId, fmt.Sprintf("%s(%d) 的昵称已重置为<%s>", msg.Sender.Nickname, msg.Sender.UserId, p.Name));
					}
					if len(cmdArgs.Args) >= 1 {
						p := getPlayerInfoBySender(session, msg)
						p.Name = cmdArgs.Args[0]
						replyGroup(session.Socket, msg.GroupId, fmt.Sprintf("%s(%d) 的昵称被设定为<%s>", msg.Sender.Nickname, msg.Sender.UserId, p.Name));
					}
				}
			}

			return struct{ success bool }{
				success: true,
			}
		},
	}
	self.cmdMap["nn"] = cmdNN;

	cmdJrrp := &CmdItemInfo{
		name: "jrrp",
		solve: func(session *IMSession, msg *Message, cmdArgs *CmdArgs) struct{ success bool } {
			if msg.MessageType == "group" {
				if isCurGroupBotOn(session, msg) {
					p := getPlayerInfoBySender(session, msg)
					todayTime := time.Now().Format("2006-01-02")

					rp := 0
					if p.RpTime == todayTime {
						rp = p.RpToday
					} else {
						rp = DiceRoll(100)
						p.RpTime = todayTime
						p.RpToday = rp
					}

					replyGroup(session.Socket, msg.GroupId, fmt.Sprintf("<%s> 的今日人品为 %d", p.Name, rp));
				}
			}

			return struct{ success bool }{
				success: true,
			}
		},
	}
	self.cmdMap["jrrp"] = cmdJrrp;

	cmdTmpLoad := &CmdItemInfo{
		name: "load",
		solve: func(session *IMSession, msg *Message, cmdArgs *CmdArgs) struct{ success bool } {
			if msg.MessageType == "group" {

				//a, err := yaml.Marshal(session.parent)
				//if err == nil {
				//}
				//replyGroup(session.Socket, msg.GroupId, fmt.Sprintf("临时指令：试图存档 \n%s \n%s", string(a), err));
			}

			return struct{ success bool }{
				success: true,
			}
		},
	}
	self.cmdMap["load"] = cmdTmpLoad;

	cmdTmpSave := &CmdItemInfo{
		name: "save",
		solve: func(session *IMSession, msg *Message, cmdArgs *CmdArgs) struct{ success bool } {
			if msg.MessageType == "group" {
				if isCurGroupBotOn(session, msg) {
					a, err := yaml.Marshal(session.parent)
					if err == nil {
						ioutil.WriteFile("save.yaml", a, 0666)
					}

					replyGroup(session.Socket, msg.GroupId, fmt.Sprintf("临时指令：试图存档"));
				}
			}

			return struct{ success bool }{
				success: true,
			}
		},
	}
	self.cmdMap["save"] = cmdTmpSave;
}

func isCurGroupBotOn(session *IMSession, msg *Message) bool {
	return msg.MessageType == "group" && session.ServiceAt[msg.GroupId] != nil && session.ServiceAt[msg.GroupId].Active
}

/** 获取玩家群内信息，没有就创建 */
func getPlayerInfoBySender(session *IMSession, msg *Message) *PlayerInfo {
	players := session.ServiceAt[msg.GroupId].Players;
	p := players[msg.Sender.UserId]
	if p == nil {
		p = &PlayerInfo{
			Name: msg.Sender.Nickname,
			UserId: msg.Sender.UserId,
			ValueNumMap: map[string]int64{},
			ValueStrMap: map[string]string{},
		}
		players[msg.Sender.UserId] = p;
	}
	return p;
}
