package main

import (
	"fmt"
	wr "github.com/mroth/weightedrand"
	"math/rand"
	"time"
)

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
	"我家猫生病了，带他去看病",
	"医生说今天疫苗到了，带猫打疫苗",
	"我鸽某人今天就是要咕口牙！",
	"当你们都觉得我要咕的时候，我咕了，这其实是一种不咕",

	"说咕就咕怎么能算是咕咕了呢！",
}


func (self *Dice) registerBuiltinExtFun() {
	choices := []wr.Choice{}
	for _, i := range gugu {
		choices = append(choices, wr.Choice{Item: i, Weight: 1})
	}
	guguRandomPool, _ := wr.NewChooser(choices...)

	cmdGugu := CmdItemInfo{
		name: "gugu",
		Brief: "获取一个随机的咕咕理由",
		solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) struct{ success bool } {
			if ctx.isCurGroupBotOn || msg.MessageType == "private" {
				//p := getPlayerInfoBySender(session, msg)
				rand.Seed(time.Now().UTC().UnixNano()) // always seed random!
				replyToSender(ctx, msg, "🕊️: " + guguRandomPool.Pick().(string))
			}
			return struct{ success bool }{
				success: true,
			}
		},
	}

	jrrpTexts := map[string]string{
		"rp": "<%s> 的今日人品为 %d",
	}
	cmdJrrp := CmdItemInfo{
		name: "jrrp",
		Brief: "获得一个D100随机值，一天内不会变化",
		texts: jrrpTexts,
		solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) struct{ success bool } {
			if msg.MessageType == "group" {
				if ctx.isCurGroupBotOn {
					p := ctx.player
					todayTime := time.Now().Format("2006-01-02")

					rp := 0
					if p.RpTime == todayTime {
						rp = p.RpToday
					} else {
						rp = DiceRoll(100)
						p.RpTime = todayTime
						p.RpToday = rp
					}

					replyGroup(ctx, msg.GroupId, fmt.Sprintf(jrrpTexts["rp"], p.Name, rp))
				}
			}

			return struct{ success bool }{
				success: true,
			}
		},
	}

	self.extList = append(self.extList, &ExtInfo{
		Name:       "fun", // 扩展的名称，需要用于指令中，写简短点
		version:    "0.0.1",
		Brief: "娱乐扩展，主打抽牌功能、智能鸽子",
		autoActive: true, // 是否自动开启
		ActiveOnPrivate: true,
		Author: "木落",
		OnCommandReceived: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) {
			//p := getPlayerInfoBySender(session, msg)
			//p.TempValueAlias = &ac.Alias;
		},
		OnLoad: func() {
		},
		GetDescText: func (i *ExtInfo) string {
			return "> " + i.Brief + "\n" + "提供命令:\n.gugu / .咕咕  // 获取一个随机的咕咕理由\n.deck <牌堆> // 从牌堆抽牌\n.jrrp 今日人品"
		},
		cmdMap: CmdMapCls{
			"gugu": &cmdGugu,
			"咕咕": &cmdGugu,
			"jrrp": &cmdJrrp,
			"deck": &CmdItemInfo{
				name: "deck",
				Brief: "从牌堆抽牌",
				solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) struct{ success bool } {
					replyToSender(ctx, msg, "尚未开发完成，敬请期待")
					return struct{ success bool }{
						success: true,
					}
				},
			},
		},
	})
}
