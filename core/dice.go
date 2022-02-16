package main

import (
	"errors"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"math/big"
	"math/rand"
	"strconv"
	"time"
)

type CmdItemInfo struct {
	name  string
	solve func(session *IMSession, msg *Message, cmdArgs *CmdArgs) struct{ success bool }
	texts map[string]string
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
	CommandCompatibleMode bool `yaml:"commandCompatibleMode"`
}

func (self *Dice) init() {
	self.CommandCompatibleMode = true;
	self.ImSession = &IMSession{};
	self.ImSession.parent = self;
	self.ImSession.ServiceAt = make(map[int64]*ServiceAtItem);
	self.cmdMap = CmdMapCls{}

	self.registerCoreCommands();
	self.registerBuiltinExt()
	self.loads();
}


func (self *Dice) rebuildParser(buffer string) *DiceRollParser {
	p := &DiceRollParser{Buffer: buffer}
	_ = p.Init()
	p.RollExpression.Init(255)
	//self.RollParser = p;
	return p;
}

func (self *Dice) exprEvalBase(buffer string, p *PlayerInfo, bigFailDice bool) (*vmStack, string, error) {
	parser := self.rebuildParser(buffer)
	err := parser.Parse()
	parser.RollExpression.BigFailDiceOn = bigFailDice

	if err == nil {
		parser.Execute()
		num, detail, _ := parser.Evaluate(self, p)
		return num, detail, nil;
	}
	return nil, "", err
}

func (self *Dice) exprEval(buffer string, p *PlayerInfo) (*vmStack, string, error) {
	return self.exprEvalBase(buffer, p, false)
}

func (self *Dice) exprText(buffer string, p *PlayerInfo) (string, string, error) {
	val, detail, err := self.exprEval("`" + buffer + "`", p)

	if err == nil && val.typeId == 1 {
		return val.value.(string), detail, err
	}

	return "", "", errors.New("错误的表达式")
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
			if msg.MessageType == "group" && cmdArgs.AmIBeMentioned {
				if len(cmdArgs.Args) == 0 {
					replyGroup(session.Socket, msg.GroupId, "SealDice 测试版");
				} else if len(cmdArgs.Args) >= 1 {
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
								Active:           true,
								ActivatedExtList: extLst,
								Players:          map[int64]*PlayerInfo{},
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
				var detail string
				p := getPlayerInfoBySender(session, msg)

				if session.parent.CommandCompatibleMode {
					if cmdArgs.Command == "rd" {
						cmdArgs.Args[0] = "d" + cmdArgs.Args[0]
					}
				} else {
					return struct{ success bool }{
						success: false,
					}
				}

				forWhat := "";
				if len(cmdArgs.Args) >= 1 {
					var err error
					var r *vmStack
					r, detail, err = session.parent.exprEval(cmdArgs.Args[0], p)
					if r != nil && r.typeId == 0 {
						diceResult = r.value.(*big.Int).Int64()
						//return errors.New("错误的类型")
					}

					if err == nil {
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

				if diceResult != 0 {
					detailWrap := ""
					if detail != "" {
						detailWrap = "=" + detail
					}
					text = fmt.Sprintf("%s<%s>掷出了 %s%s=%d", prefix, p.Name, cmdArgs.Args[0], detailWrap, diceResult);
				} else {
					dicePoints := p.DiceSideNum
					if dicePoints <= 0 {
						dicePoints = 100
					}
					val := DiceRoll(dicePoints);
					text = fmt.Sprintf("%s<%s>掷出了 D%d=%d", prefix, p.Name, dicePoints, val);
				}

				if cmdArgs.Command == "rh" {
					replyGroup(session.Socket, msg.GroupId, "黑暗的角落里，传来命运转动的声音");
					replyPerson(session.Socket, msg.Sender.UserId, text);
				} else {
					replyGroup(session.Socket, msg.GroupId, text);
				}
			}

			return struct{ success bool }{
				success: true,
			}
		},
	}
	self.cmdMap["r"] = cmdRoll;
	self.cmdMap["rd"] = cmdRoll;
	self.cmdMap["roll"] = cmdRoll;
	self.cmdMap["rh"] = cmdRoll;

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

	jrrpTexts := map[string]string{
		"rp": "<%s> 的今日人品为 %d",
	}
	cmdJrrp := &CmdItemInfo{
		name: "jrrp",
		texts: jrrpTexts,
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

					replyGroup(session.Socket, msg.GroupId, fmt.Sprintf(jrrpTexts["rp"], p.Name, rp));
				}
			}

			return struct{ success bool }{
				success: true,
			}
		},
	}
	self.cmdMap["jrrp"] = cmdJrrp;

	cmdSet := &CmdItemInfo{
		name: "set",
		solve: func(session *IMSession, msg *Message, cmdArgs *CmdArgs) struct{ success bool } {
			if isCurGroupBotOn(session, msg) {
				p := getPlayerInfoBySender(session, msg)
				if len(cmdArgs.Args) >= 1 {
					num, err := strconv.Atoi(cmdArgs.Args[0])
					if err == nil {
						p.DiceSideNum = num
						replyGroup(session.Socket, msg.GroupId, fmt.Sprintf("设定默认骰子面数为 %d", num));
					} else {
						replyGroup(session.Socket, msg.GroupId, fmt.Sprintf("设定默认骰子面数: 格式错误"));
					}
				} else {
					p.DiceSideNum = 0
					replyGroup(session.Socket, msg.GroupId, fmt.Sprintf("重设默认骰子面数为初始"));
				}
			}

			return struct{ success bool }{
				success: true,
			}
		},
	}
	self.cmdMap["set"] = cmdSet;

	cmdTmpSave := &CmdItemInfo{
		name: "save",
		// help
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
	self.cmdMap["cfgsave"] = cmdTmpSave;


	cmdText := &CmdItemInfo{
		name: "text",
		// help
		solve: func(session *IMSession, msg *Message, cmdArgs *CmdArgs) struct{ success bool } {
			if msg.MessageType == "group" {
				if isCurGroupBotOn(session, msg) {
					p := getPlayerInfoBySender(session, msg)
					val, _, err := self.exprText(cmdArgs.RawArgs, p)

					if err == nil {
						replyGroup(session.Socket, msg.GroupId, val);
					} else {
						replyGroup(session.Socket, msg.GroupId, "格式错误");
					}
				}
			}

			return struct{ success bool }{
				success: true,
			}
		},
	}
	self.cmdMap["text"] = cmdText;
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
