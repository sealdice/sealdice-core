package main

import (
	"errors"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"math/rand"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type CmdItemInfo struct {
	name  string
	solve func(session *IMSession, msg *Message, cmdArgs *CmdArgs) struct{ success bool }
	Brief string
	texts map[string]string
}

type CmdMapCls map[string]*CmdItemInfo;

type ExtInfo struct {
	Name    string // 名字
	version string // 版本
	// 作者
	// 更新时间
	autoActive      bool      // 是否自动开启
	cmdMap          CmdMapCls `yaml:"-"` // 指令集合
	Brief           string    `yaml:"-"`
	ActiveOnPrivate bool      `yaml:"-"`

	Author      string `yaml:"-"`
	//activeInSession bool; // 在当前会话中开启

	OnPrepare   func(session *IMSession, msg *Message, cmdArgs *CmdArgs) `yaml:"-"`
	GetDescText func(i *ExtInfo) string                                  `yaml:"-"`
}


type Dice struct {
	ImSession             *IMSession `yaml:"imSession"`
	cmdMap                CmdMapCls
	extList               []*ExtInfo
	RollParser            *DiceRollParser `yaml:"-"`
	CommandCompatibleMode bool            `yaml:"commandCompatibleMode"`
	lastSavedTime         *time.Time       `yaml:"lastSavedTime"`
}

func (self *Dice) init() {
	self.CommandCompatibleMode = true;
	self.ImSession = &IMSession{};
	self.ImSession.parent = self;
	self.ImSession.ServiceAt = make(map[int64]*ServiceAtItem);
	self.cmdMap = CmdMapCls{}

	self.registerCoreCommands();
	self.RegisterBuiltinExt()
	self.loads();

	autoSave := func() {
		t := time.Tick(15 * time.Second)
		for {
			<-t
			self.save()
		}
	}
	go autoSave()
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
		if parser.error != nil{
			return nil, "", parser.error
		}
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

func (self *Dice) save() {
	a, err := yaml.Marshal(self)
	if err == nil {
		err := ioutil.WriteFile("save.yaml", a, 0666)
		if err == nil {
			now := time.Now()
			self.lastSavedTime = &now
			fmt.Println("此时的用户ID", self.ImSession.UserId)
			if self.ImSession.UserId == 0 {
				self.ImSession.GetLoginInfo()
			}
		}
	}
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

var VERSION = "v20220217"

/** 这几条指令不能移除 */
func (self *Dice) registerCoreCommands() {
	cmdHelp := &CmdItemInfo{
		name: "help",
		Brief: "查看本帮助",
		solve: func(session *IMSession, msg *Message, cmdArgs *CmdArgs) struct{ success bool } {
			// inGroup := msg.MessageType == "group"
			if isCurGroupBotOn(session, msg) {
				text := "核心指令列表如下:\n"

				used := map[*CmdItemInfo]bool{}
				keys := make([]string, 0, len(self.cmdMap))
				for k, v := range self.cmdMap {
					if used[v] {
						continue
					}
					keys = append(keys, k)
					used[v] = true
				}
				sort.Strings(keys)

				for _, i := range keys {
					i := self.cmdMap[i]
					brief := i.Brief
					if brief != "" {
						brief = "   // " + brief
					}
					text += "." + i.name + brief + "\n"
				}

				text += "注意：由于篇幅此处仅列出核心指令。\n"
				text += "扩展指令请输入 .ext 和 .ext <扩展名> 进行查看\n"
				text += "-----------------------------------------------\n"
				text += "SealDice 目前 7*24h 运行于一块陈年OrangePi卡片电脑上，随时可能因为软硬件故障停机（例如过热、被猫打翻）。届时可以来Q群524364253询问。"
				replyToSender(session.Socket, msg, text)
			}
			return struct{ success bool }{
				success: true,
			}
		},
	}
	self.cmdMap["help"] = cmdHelp;

	cmdBot := &CmdItemInfo{
		name: "bot on/off",
		Brief: "开启、关闭、查看信息",
		solve: func(session *IMSession, msg *Message, cmdArgs *CmdArgs) struct{ success bool } {
			inGroup := msg.MessageType == "group"

			if len(cmdArgs.Args) == 0 {
				count := 0
				for _, i := range self.ImSession.ServiceAt {
					if i.Active {
						count += 1
					}
				}
				lastSavedTimeText := "从未"
				if self.lastSavedTime != nil {
					lastSavedTimeText = self.lastSavedTime.Format("2006-01-02 15:04:05") + " UTC"
				}
				text := fmt.Sprintf("SealDice 0.9测试版 %s\n兼容模式: 已开启\n供职于%d个群，其中%d个处于开启状态\n上次自动保存时间: %s", VERSION, len(self.ImSession.ServiceAt), count, lastSavedTimeText)

				if inGroup {
					if cmdArgs.AmIBeMentioned {
						replyGroup(session.Socket, msg.GroupId, text)
					}
				} else {
					replyPerson(session.Socket, msg.Sender.UserId, text)
				}
			} else {
				fmt.Println("?????????????", inGroup, cmdArgs.At, cmdArgs.AmIBeMentioned, session.UserId)
				if inGroup && cmdArgs.AmIBeMentioned {
					if len(cmdArgs.Args) >= 1 {
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
							replyGroup(session.Socket, msg.GroupId, "SealDice 已启用(开发中) " + VERSION);
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
			}

			return struct{ success bool }{
				success: true,
			}
		},
	}
	self.cmdMap["bot"] = cmdBot;

	cmdRoll := &CmdItemInfo{
		name: "r <表达式> <原因>",
		Brief: "骰点指令，案例:“.r d16” “.r 3d10*2+3” “.r d10+力量” “.r 2d(力量+1d3)” “.rh d16 (暗骰)” ",
		solve: func(session *IMSession, msg *Message, cmdArgs *CmdArgs) struct{ success bool } {
			if isCurGroupBotOn(session, msg) {
				var text string;
				var prefix string;
				var diceResult int64
				var diceResultExists bool
				var detail string
				p := getPlayerInfoBySender(session, msg)

				if session.parent.CommandCompatibleMode {
					if cmdArgs.Command == "rd" && len(cmdArgs.Args) >= 1 {
						if m, _ := regexp.MatchString(`^\d`, cmdArgs.Args[0]); m {
							cmdArgs.Args[0] = "d" + cmdArgs.Args[0]
						}
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
						diceResult = r.value.(int64)
						diceResultExists = true
						//return errors.New("错误的类型")
					}

					if err == nil {
						if len(cmdArgs.Args) >= 2 {
							forWhat = cmdArgs.Args[1]
						}
					} else {
						errs := string(err.Error())
						if strings.HasPrefix(errs, "E1:") {
							replyGroup(session.Socket, msg.GroupId, errs);

							return struct{ success bool }{
								success: true,
							}
						}
						forWhat = cmdArgs.Args[0];
					}
				}

				if forWhat != "" {
					prefix = "为了" + forWhat + "，";
				}

				if diceResultExists {
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
		Brief: "查看扩展列表",
		solve: func(session *IMSession, msg *Message, cmdArgs *CmdArgs) struct{ success bool } {
			if isCurGroupBotOn(session, msg) {
				showList := func () {
					text := "检测到以下扩展：\n"
					for index, i := range session.parent.extList {
						state := "关"
						for _, j := range session.ServiceAt[msg.GroupId].ActivatedExtList {
							if i.Name == j.Name {
								state = "开"
								break
							}
						}
						author := i.Author
						if author == "" {
							author = "<未注明>"
						}
						text += fmt.Sprintf("%d. [%s]%s - 版本:%s 作者:%s\n", index + 1, state, i.Name, i.version, author)
					}
					text += "使用命令: .ext <扩展名> on/off 可以在当前群开启或关闭某扩展。\n"
					text += "命令: .ext <扩展名> 可以查看扩展介绍及帮助"
					replyGroup(session.Socket, msg.GroupId, text);
				}

				if len(cmdArgs.Args) == 0 {
					showList()
				}

				if len(cmdArgs.Args) >= 1 {
					if cmdArgs.isArgEqual(1, "list") {
						showList()
					} else if cmdArgs.isArgEqual(2, "on") {
						extName := cmdArgs.Args[0];
						for _, i := range self.extList {
							if i.Name == extName {
								session.ServiceAt[msg.GroupId].ActivatedExtList = append(session.ServiceAt[msg.GroupId].ActivatedExtList, i);
								replyGroup(session.Socket, msg.GroupId, fmt.Sprintf("打开扩展 %s", extName));
								break;
							}
						}
					} else if cmdArgs.isArgEqual(2, "off") {
						gInfo := session.ServiceAt[msg.GroupId]
						extName := cmdArgs.Args[0];
						for index, i := range gInfo.ActivatedExtList {
							if i.Name == extName {
								gInfo.ActivatedExtList = append(gInfo.ActivatedExtList[:index], gInfo.ActivatedExtList[index+1:]...)
								replyGroup(session.Socket, msg.GroupId, fmt.Sprintf("关闭扩展 %s", extName));
							}
						}
					} else {
						extName := cmdArgs.Args[0]
						for _, i := range self.extList {
							if i.Name == extName {
								text := fmt.Sprintf("> [%s] 版本%s 作者%s\n", i.Name, i.version, i.Author)
								replyToSender(session.Socket, msg, text + i.GetDescText(i))
								break
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
		name: "nn <角色名>",
		Brief: ".nn后跟角色名则改角色名，不带则重置角色名",
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
		Brief: "获得一个D100随机值，一天内不会变化",
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
		name: "set <面数>",
		Brief: "设置默认骰子面数，只对自己有效",
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
		name: "cfgsave",
		// help
		solve: func(session *IMSession, msg *Message, cmdArgs *CmdArgs) struct{ success bool } {
			if msg.MessageType == "group" {
				if isCurGroupBotOn(session, msg) {
					self.save()
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
		name: "text <文本模板>",
		Brief: "文本指令(测试)，举例: .text 1D16={ 1d16 }，属性计算: 攻击 - 防御 = {攻击} - {防御} = {攻击 - 防御}",
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
