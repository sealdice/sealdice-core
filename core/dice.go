package main

import (
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
)

type CmdItemInfo struct{
	name string;
	solve func(session *IMSession, msg *Message, cmdArgs *CmdArgs) struct { success bool };
}

type CmdMapCls map[string]*CmdItemInfo;

type ExtInfo struct {
	name string  // 名字
	version string  // 版本
	autoActive bool // 是否自动开启
	cmdMap CmdMapCls; // 指令集合
	//activeInSession bool; // 在当前会话中开启
}

type Dice struct {
	imSession *IMSession;
	cmdMap CmdMapCls;
	extList []*ExtInfo;
}

func (self *Dice) init() {
	self.imSession = &IMSession{};
	self.imSession.parent = self;
	self.imSession.ServiceAt = make(map[int64]*ServiceAtItem);
	self.cmdMap = CmdMapCls{}
	self.loads();
	self.registerCoreCommands();
	self.registerBuiltinExt()
}

func (self Dice) loads() {
	
}

func (self Dice) save() {
	
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
					replyGroup(session.Socket, msg.GroupId, "SealDice 已启用(开发中) v20220208");
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
				diceTime := 0;
				dicePoints := 100;
				p := getPlayerInfoBySender(session, msg)

				forWhat := "";
				if len(cmdArgs.Args) >= 1 {
					re := regexp.MustCompile(`(\d*)d(\d+)`);
					m := re.FindStringSubmatch(cmdArgs.Args[0])
					if len(m) > 0 {
						diceTime, _ = strconv.Atoi(m[1])
						dicePoints, _ = strconv.Atoi(m[2])
						if len(cmdArgs.Args) >= 2 {
							forWhat = cmdArgs.Args[1]
						}
					} else {
						forWhat = cmdArgs.Args[0];
					}
				}

				if forWhat != "" {
					prefix = "为了" + forWhat + "，";
				}

				if diceTime > 0 {
					points := make([]int, 0)
					valAll := 0

					for i := 0; i < diceTime; i++ {
						val := rand.Int() % dicePoints;
						valAll += val;
						points = append(points, val)
					}

					text1 := ""
					for i := 0; i < diceTime; i++ {
						text1 = text1 + strconv.Itoa(points[i])
						if i != diceTime -1 {
							text1 += "+"
						}
					}
					text = fmt.Sprintf("%s<%s> 掷出了 %dD%d=%s=%d", prefix, p.Name, diceTime, dicePoints, text1, valAll);
				} else {
					val := rand.Int() % dicePoints;
					text = fmt.Sprintf("%s<%s> 掷出了 D%d=%d", prefix, p.Name, dicePoints, val);
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
						text += fmt.Sprintf("%d. [%s] version %s\n", index + 1, i.name, i.version)
					}
					text += "使用命令 ”.ext on/off 扩展名“ 可以在当前群开启或关闭某扩展。"
					replyGroup(session.Socket, msg.GroupId, text);
				} else if cmdArgs.Args[0] == "on" && session.ServiceAt[msg.GroupId] != nil && session.ServiceAt[msg.GroupId].Active {
					if len(cmdArgs.Args) >= 2 {
						extName := cmdArgs.Args[1];
						for _, i := range self.extList {
							if i.name == extName {
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
							if i.name == extName {
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
