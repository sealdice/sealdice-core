package dice

import (
	"hash/fnv"
	"math/rand"
	"strings"
	"time"
)

var guguText = `
{$t玩家}为了拯救公主前往了巨龙的巢穴，还没赶回来！|鹊鹊结合实际经历创作
{$t玩家}在来开团的路上被巨龙叼走了！|鹊鹊结合实际经历创作
来的路上出现了哥布林劫匪！{$t玩家}大概是赶不过来了！|鹊鹊结合实际经历创作
咕咕咕~广场上的鸽子把{$t玩家}叼回了巢穴~|鹊鹊结合实际经历创作
为了拯救不慎滑落下水道的一元硬币，{$t玩家}化身搜救队英勇赶赴！|鹊鹊结合实际经历创作
{$t玩家}睡着了——zzzzzzzz......|鹊鹊结合实际经历创作
在互联网上约到可爱美少女不惜搁置跑团前去约会的{$t玩家}，还不知道这个叫奈亚的妹子隐藏着什么...|鹊鹊结合实际经历创作
在聚会上完全喝断片的{$t玩家}被半兽人三兄弟抬走咯~！♡|鹊鹊结合实际经历创作
{$t玩家}在地铁上睡着了，不断前行的车厢逐渐带他来到了最终站点...mogeko~！|鹊鹊结合实际经历创作
今天绿色章鱼俱乐部有活动，来不了了呢——by{$t玩家}|鹊鹊结合实际经历创作
“喂？跑团？啊，抱歉可能有点事情来不了了”你听着{$t玩家}电话背景音里一阵阵未知语言咏唱的声音，开始明白他现在很忙。|鹊鹊结合实际经历创作
给{$t玩家}打电话的时候，自己关注的vtb的电话也正好响了起来...|鹊鹊结合实际经历创作
因为被长发龙男逼到了小巷子，{$t玩家}大概没心思思考别的事情了。|鹊鹊结合实际经历创作
在海边散步的时候，突然被触手拉入海底的{$t玩家}！|鹊鹊结合实际经历创作
“来不了了，对不起...”电话对面的{$t玩家}房间里隐约传来阵阵喘息。|鹊鹊结合实际经历创作
黄色雨衣团真是赛高~！综上所述今天要去参加活动，来不了了哦~！——by{$t玩家}|鹊鹊结合实际经历创作
{$t玩家}正在看书，啊！不好！他被知识的巨浪冲走了！搜救队——！！！|鹊鹊结合实际经历创作
为了帮助突然晕倒的程序员木落，{$t玩家}错过了开团时间，撑住啊木落！！！|鹊鹊结合实际经历创作
由于尝试邪神召唤而来到异界的{$t玩家}，好了，这下该怎么回去呢？距离开团还有5...3...1...|鹊鹊结合实际经历创作
不慎穿越的{$t玩家}！但是接下来还有团！这一切该如何是好？《心跳！穿越到异世界了这下不得不咕咕掉跑团了呢~！》好评发售~！|鹊鹊结合实际经历创作
因为海豹一直缠着{$t玩家}，所以只好先陪他玩啦——|鹊鹊结合实际经历创作
开开心心准备开团的时候，几只大蜘蛛破窗而入！啊！{$t玩家}被他们劫走了！！！|鹊鹊结合实际经历创作
“没想到食尸鬼俱乐部的大家不是化妆特效...以后可能再也没法儿一起玩了...”{$t玩家}发来了这种意义不明的短信。|鹊鹊结合实际经历创作
“走在马路上被突如其来的龙娘威胁了，现在在小巷子里！！！请大家带一万金币救我！！！”{$t玩家}在电话里这样说。|鹊鹊结合实际经历创作
前往了一个以前捕鲸的小岛度假~这里人很亲切！但是吃了这里的鱼肉料理之后有点晕晕的诶...想到前几天{$t玩家}的短信，还是别追究他为什么不在了。|鹊鹊结合实际经历创作
因为沉迷vtb而完全忘记开团的{$t玩家}，毕竟太可爱了所以原谅他吧~！|鹊鹊结合实际经历创作
观看海豹顶球的时候站的太近被溅了一身水，换衣服的功夫{$t玩家}发现开团时间已经错过了。|鹊鹊结合实际经历创作
不知为什么平坦的路面上会躺着一只海豹，就那样玩着手机没注意就被绊倒昏过去了！可怜的{$t玩家}！|鹊鹊结合实际经历创作



{$t玩家}去依盖队大本营给大家抢香蕉了。|yumeno结合实际经历创作
“我家金鱼淹死了，要去处理一下，晚点再来”原来如此，节哀{$t玩家}！|yumeno结合实际经历创作
“我家狗在学校被老师请家长，今天不来了”这条{$t玩家}的短信让你打开手机开始搜索狗学校。|yumeno结合实际经历创作
“钱不够坐车回家，待我走回去先”{$t玩家}你其实知道手机可以支付车费的吧？|yumeno结合实际经历创作
救命！我变成鸽子了！——by{$t玩家}的短信。|yumeno结合实际经历创作
咕咕，咕咕咕咕咕，咕咕咕！——by{$t玩家}的短信。|yumeno结合实际经历创作
老板让我现在回去加班，我正在写辞呈。{$t玩家}一边内卷一边对着电话这样说。|yumeno结合实际经历创作
键盘坏了，快递还没送到，今晚不开——by{$t玩家}的短信。|yumeno结合实际经历创作
要肝活动，晚点来！——by{$t玩家}的短信。|yumeno结合实际经历创作
社区通知我现在去做核酸！by{$t玩家}的短信。|yumeno结合实际经历创作
今晚妈妈买了十斤小龙虾，可能来不了了——by{$t玩家}的短信。小龙虾是无辜的！|yumeno结合实际经历创作
“有个小孩的玩具掉轨道里了，高铁晚点了，我晚点来...是真的啦！”{$t玩家}对着手机吼道。|yumeno结合实际经历创作
“飞机没油了，我去加点油，晚点来。”——by{$t玩家}的短信。|yumeno结合实际经历创作
“寂静岭出新作了，今晚没空，咕咕咕”你看到{$t玩家}的对话框跳出这样一条内容。|yumeno结合实际经历创作
老头环中...你看着Steam好友里{$t玩家}的状态，感觉也不是不能理解。|yumeno结合实际经历创作
你打开狒狒，看见了{$t玩家}在线中，看来原因找到了。|yumeno结合实际经历创作
|yumeno结合实际经历创作


哎呀，身份证丢了，要去补办——！这条信息by{$t玩家}|秦祚轩结合实际经历创作
亲戚结婚了，我喝个喜酒就来！{$t玩家}留下这样一段话。|秦祚轩结合实际经历创作
疫情期间，主动核酸！我辈义不容辞！这样说着，{$t玩家}冲出去了。|秦祚轩结合实际经历创作
学校突然加课，大家！对不起！就算没有我你们也要从邪神手中拯救这个世界！！！{$t玩家}绝笔。|秦祚轩结合实际经历创作
滴滴滴250°c——测温枪发出这种警报，“我还有团啊——！”不理会{$t玩家}的反抗，医护人员拖走了他。|秦祚轩结合实际经历创作
钱包！我的钱包！！！不见了！！！！！！{$t玩家}一边报警一边离开了大家的视线。|秦祚轩结合实际经历创作
即使不是学雷锋日也要学雷锋！路上的老爷爷老奶奶们需要我！对不起大家！{$t玩家}在一边扶着老奶奶一边艰难的解释。|秦祚轩结合实际经历创作



你不知道今天是什么日子吗？今天是周四！你不知道周四会发生什么吗？周四有疯狂星期四！不说了，我去吃KFC了。by{$t玩家}|月森优姬结合实际经历创作



我有点事，你们先开|木落好像在结合实际经历创作
今天忽然加班了，可能来不了了|木落好像在结合实际经历创作
今天发版本，领导说发不完不让走|木落好像在结合实际经历创作
我家猫生病了，带他去看病|木落好像在结合实际经历创作
医生说今天疫苗到了，带猫打疫苗|木落好像在结合实际经历创作
我鸽某人今天就是要咕口牙！|木落好像在结合实际经历创作
当你们都觉得{$t玩家}要咕的时候，{$t玩家}咕了，这其实是一种不咕|木落好像在结合实际经历创作
`

var gugu = []string{
	"我尽量8点半就到  但是明天有人要约我吃饭……",
	"先说下，晚上先去看丈母娘，然后回来",
	"正常吧，我的工作想不咕是不可能的啊，只能说提前打个招呼。",
	"以后都要上夜班了，突然安排我的，本来也不知道",
	"卡还没车好，要不我下周来吧",
	"你们先玩，我躺着眯一会儿，有点累，没事儿你们先玩，我躺着插两句就行",
	"开的话，你们先玩，我办好事，就来，先咕为敬",
	"今天就咕了，明天考试，要背题",
	"……今天去医院查出肝虚，不能熬夜了。只能围观了。",
	"我在外面吃饭",
	"我感冒加重了，应该是中药吃反了",
	"今晚要咕咕咕了吗，不如先去打几局游戏",
	"我今天应该不行了~晚上有事~",
	"咕咕咕，这段时间眼睛患了结膜炎，回归日待定",
	"加班，回来迟",
	"233今天似乎有点事情，可能会晚",
	"早上发生了地震，要去包的村里查看灾情",
	"十分钟到家。不吃红灯的话",
	"和同事吃饭，应该不会晚太多吧，我争取8.30",
	"饭局，你们先开走剧情，我就到了",
	"DM上班累不累啊？晚上需要休息嘛？我就问问，关心下DM的健康，没其他意思，真的，咕咕咕。",
	"你们的dm在外出取材，今晚咕了",
	"今天可能会鸽一下，一直在外面学习，天天学到九点多",
	"晚上要加班很晚，来不了",
	"我今天要晚些，工作有变动",
	"今天我可能晚上gg了，在山里值班，信号不好",
	"我这边要晚一点到，突然被同事叫去吃饭了",
	"我这路上有点堵车不过马上",
	"同志们对不住，工作原因开团要暂停两周",
	"星期一集体旅游咕咕咕。。",
	"算我算我，忘记了",
	"明天我加班，不知道能不能来",
	"出去浪了，可能回来晚点",
	"今晚大家是不是都要咕咕咕呢？",
	"电脑才装好，晚上要不要咕咕咕",
	"那么，我和XX预计咕一个也有4个人",
	"一个问题，，出去批冰棒忘带钥匙了",
	"这么早[斜眼笑]不吃个饭，睡一会",
	"家里有人，语音的话还是太羞耻了",
	"晚上几点，还是直接下周",
	"这个月婚庆临近可能咕咕咕",
	"明天我也不知道能不能跑，要加班",
	"眼镜掉了，配个眼镜。可能要晚点",
	"我明天应该不能来，星期一有个考试，需要背下题",
	"我今天可能会晚~充电线或者手机坏了昨天晚上没充上电",
	"我还在外面，估计还要1小时",
	"我今天准时到，你给什么奖励",
	"这周不跑吧，我被晒伤了，休息一周",
	"应该轮到我加班啊，我要看情况才知道能来不",
	"我和同事去吃个饭，可能会晚点。",
	"要准备考试，星期一来不了",
	"我的工作你别问，不能说，我们干工作需要向你解释么？",
	"下周我可能临时加一节课",
	"那我去买点东西，来得及赶回来，也许",
	"在地铁上，说话实在是不太方便~",
	"今晚看球，请假，咕咕咕",
	"今天可能会迟到 你们先开着。",
	"我看看吧，可能不能说话",
	"本周三陪老婆出去逛吃，预计两天",

	"我有点事，你们先开",
	"今天忽然加班了，可能来不了了",
	"今天发版本，领导说发不完不让走",
	"我家猫生病了，带他去看病",
	"医生说今天疫苗到了，带猫打疫苗",
	"我鸽某人今天就是要咕口牙！",
	"当你们都觉得我要咕的时候，我咕了，这其实是一种不咕",

	"说咕就咕怎么能算是咕咕了呢！",
}

func RegisterBuiltinExtFun(self *Dice) {
	//choices := []wr.Choice{}
	//for _, i := range gugu {
	//	choices = append(choices, wr.Choice{Item: i, Weight: 1})
	//}
	//guguRandomPool, _ := wr.NewChooser(choices...)
	// guguRandomPool.Pick().(string)

	cmdGugu := CmdItemInfo{
		Name:     "gugu",
		Help:     ".gugu 来源 // 获取一个随机的咕咕理由，带上来源可以看作者",
		LongHelp: "人工智能鸽子:\n.gugu 来源 // 获取一个随机的咕咕理由，带上来源可以看作者\n.text // 文本指令",
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			if ctx.IsCurGroupBotOn || msg.MessageType == "private" {
				if cmdArgs.SomeoneBeMentionedButNotMe {
					return CmdExecuteResult{Matched: false, Solved: false}
				}

				//p := getPlayerInfoBySender(session, msg)
				isShowFrom := cmdArgs.IsArgEqual(1, "from", "showfrom", "来源", "作者")
				rand.Seed(time.Now().UTC().UnixNano()) // always seed random!

				reason := DiceFormatTmpl(ctx, "娱乐:鸽子理由")
				reasonInfo := strings.SplitN(reason, "|", 2)

				text := "🕊️: " + reasonInfo[0]
				if isShowFrom && len(reasonInfo) == 2 {
					text += "\n    ——" + reasonInfo[1]
				}
				ReplyToSender(ctx, msg, text)
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			return CmdExecuteResult{Matched: true, Solved: false}
		},
	}

	cmdJrrp := CmdItemInfo{
		Name:     "jrrp",
		Help:     ".jrrp 获得一个D100随机值，一天内不会变化",
		LongHelp: "今日人品:\n.jrrp 获得一个D100随机值，一天内不会变化",
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			if ctx.IsCurGroupBotOn || ctx.IsPrivate {
				if cmdArgs.SomeoneBeMentionedButNotMe {
					return CmdExecuteResult{Matched: false, Solved: false}
				}

				rpSeed := (time.Now().Unix() + (8 * 60 * 60)) / (24 * 60 * 60)
				rpSeed += int64(fingerprint(ctx.EndPoint.UserId))
				rpSeed += int64(fingerprint(ctx.Player.UserId))
				rand.Seed(rpSeed)
				rp := rand.Int63()%100 + 1

				VarSetValue(ctx, "$t人品", &VMValue{VMTypeInt64, int64(rp)})
				ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "娱乐:今日人品"))
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			return CmdExecuteResult{Matched: true, Solved: false}
		},
	}

	textHelp := ".text <文本模板> // 文本指令，例: .text 看看手气: {1d16}"
	cmdText := CmdItemInfo{
		Name:     "text",
		Help:     textHelp,
		LongHelp: "文本模板指令:\n" + textHelp,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			if ctx.IsCurGroupBotOn || ctx.IsPrivate {
				if cmdArgs.SomeoneBeMentionedButNotMe {
					return CmdExecuteResult{Matched: false, Solved: false}
				}

				_, exists := cmdArgs.GetArgN(1)
				if exists {
					r, _, err := ctx.Dice.ExprTextBase(cmdArgs.CleanArgs, ctx)

					if err == nil && (r.TypeId == VMTypeString || r.TypeId == VMTypeNone) {
						text := r.Value.(string)

						if kw := cmdArgs.GetKwarg("asm"); r != nil && kw != nil {
							asm := r.Parser.GetAsmText()
							text += "\n" + asm
						}

						seemsCommand := false
						if strings.HasPrefix(text, ".") || strings.HasPrefix(text, "。") || strings.HasPrefix(text, "!") {
							seemsCommand = true
							if strings.HasPrefix(text, "..") || strings.HasPrefix(text, "。。") || strings.HasPrefix(text, "!!") {
								seemsCommand = false
							}
						}

						if seemsCommand {
							ReplyToSender(ctx, msg, "你可能在利用text让骰子发出指令文本，这被视为恶意行为并已经记录")
						} else {
							ReplyToSender(ctx, msg, text)
						}
					} else {
						ReplyToSender(ctx, msg, "格式错误")
					}
					return CmdExecuteResult{Matched: true, Solved: true}
				} else {
					return CmdExecuteResult{Matched: true, Solved: true, ShowLongHelp: true}
				}
			}
			return CmdExecuteResult{Matched: true, Solved: false}
		},
	}

	self.ExtList = append(self.ExtList, &ExtInfo{
		Name:            "fun", // 扩展的名称，需要用于指令中，写简短点
		Version:         "1.0.0",
		Brief:           "娱乐扩展，目前提供今日人品和智能鸽子",
		AutoActive:      true, // 是否自动开启
		ActiveOnPrivate: true,
		Author:          "木落",
		OnCommandReceived: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) {
			//p := getPlayerInfoBySender(session, msg)
			//p.TempValueAlias = &ac.Alias;
		},
		OnLoad: func() {
		},
		GetDescText: func(i *ExtInfo) string {
			return "> " + i.Brief + "\n" + "提供命令:\n.gugu / .咕咕  // 获取一个随机的咕咕理由, .gugu来源可以看到作者\n.jrrp 今日人品"
		},
		CmdMap: CmdMapCls{
			"gugu": &cmdGugu,
			"咕咕":   &cmdGugu,
			"jrrp": &cmdJrrp,
			"text": &cmdText,
		},
	})
}

func fingerprint(b string) uint64 {
	hash := fnv.New64a()
	hash.Write([]byte(b))
	return hash.Sum64()
}
