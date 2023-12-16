package dice

import (
	"fmt"
	ds "github.com/sealdice/dicescript"
	"hash/fnv"
	"math/rand"
	"regexp"
	"sort"
	"strconv"
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

{$t玩家}一觉醒来，奇怪，太阳在天上怎么还能看见星空？还有天空中这个泡泡形状的巨大黑影是什么|Szzrain结合实际经历创作

打麻将被人连胡了五个国士无双，{$t玩家}哭晕了过去——|蜜瓜包结合实际经历创作
是这样的，{$t玩家}的人格分裂被治好了，跑团的那个人格消失了，所以就完全没办法跑团啦！嗯！|蜜瓜包结合实际经历创作
什么跑团？刚分手，别来烦我！{$t玩家}如是说道|蜜瓜包结合实际经历创作
今天发大水，脑子被水淹了，跑不了团啦！|蜜瓜包结合实际经历创作
`
var emokloreAttrParent = map[string][]string{
	"检索":   {"知力"},
	"洞察":   {"知力"},
	"识路":   {"灵巧", "五感"},
	"直觉":   {"精神", "运势"},
	"鉴定":   {"五感", "知力"},
	"观察":   {"五感"},
	"聆听":   {"五感"},
	"鉴毒":   {"五感"},
	"危机察觉": {"五感", "运势"},
	"灵感":   {"精神", "运势"},
	"社交术":  {"社会"},
	"辩论":   {"知力"},
	"心理":   {"精神", "知力"},
	"魅惑":   {"魅力"},
	"专业知识": {"知力"},
	"万事通":  {"五感", "社会"},
	"业界":   {"社会", "魅力"},
	"速度":   {"身体"},
	"力量":   {"身体"},
	"特技动作": {"身体", "灵巧"},
	"潜泳":   {"身体"},
	"武术":   {"身体"},
	"奥义":   {"身体", "精神", "灵巧"},
	"射击":   {"灵巧", "五感"},
	"耐久":   {"身体"},
	"毅力":   {"精神"},
	"医术":   {"灵巧", "知力"},
	"技巧":   {"灵巧"},
	"艺术":   {"灵巧", "精神", "五感"},
	"操纵":   {"灵巧", "五感", "知力"},
	"暗号":   {"知力"},
	"电脑":   {"知力"},
	"隐匿":   {"灵巧", "社会", "运势"},
	"强运":   {"运势"},
}

var emokloreAttrParent2 = map[string][]string{
	"治疗": {"知力"},
	"复苏": {"知力", "精神"},
}

var emokloreAttrParent3 = map[string][]string{
	"调查": {"灵巧"},
	"知觉": {"五感"},
	"交涉": {"魅力"},
	"知识": {"知力"},
	"信息": {"社会"},
	"运动": {"身体"},
	"格斗": {"身体"},
	"投掷": {"灵巧"},
	"生存": {"身体"},
	"自我": {"精神"},
	"手工": {"灵巧"},
	"幸运": {"运势"},
}

type singleRoulette struct {
	Name string
	Face int64
	Time int
	Pool []int
}

var rouletteMap SyncMap[string, singleRoulette]

func RegisterBuiltinExtFun(self *Dice) {
	cmdGugu := CmdItemInfo{
		Name:      "gugu",
		ShortHelp: ".gugu 来源 // 获取一个随机的咕咕理由，带上来源可以看作者",
		Help:      "人工智能鸽子:\n.gugu 来源 // 获取一个随机的咕咕理由，带上来源可以看作者\n.text // 文本指令",
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			// p := getPlayerInfoBySender(session, msg)
			isShowFrom := cmdArgs.IsArgEqual(1, "from", "showfrom", "来源", "作者")

			reason := DiceFormatTmpl(ctx, "娱乐:鸽子理由")
			reasonInfo := strings.SplitN(reason, "|", 2)

			text := "🕊️: " + reasonInfo[0]
			if isShowFrom && len(reasonInfo) == 2 {
				text += "\n    ——" + reasonInfo[1]
			}
			ReplyToSender(ctx, msg, text)
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}

	cmdJrrp := CmdItemInfo{
		Name:      "jrrp",
		ShortHelp: ".jrrp 获得一个D100随机值，一天内不会变化",
		Help:      "今日人品:\n.jrrp 获得一个D100随机值，一天内不会变化",
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			rpSeed := (time.Now().Unix() + (8 * 60 * 60)) / (24 * 60 * 60)
			rpSeed += int64(fingerprint(ctx.EndPoint.UserID))
			rpSeed += int64(fingerprint(ctx.Player.UserID))
			randItem := rand.NewSource(rpSeed)
			rp := randItem.Int63()%100 + 1

			VarSetValueInt64(ctx, "$t人品", rp)
			ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "娱乐:今日人品"))
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}

	cmdRsr := CmdItemInfo{
		Name:      "rsr",
		ShortHelp: ".rsr <骰数> // 暗影狂奔",
		Help: "暗影狂奔骰点:\n.rsr <骰数>\n" +
			"> 每个被骰出的五或六就称之为一个成功度\n" +
			"> 如果超过半数的骰子投出了一被称之为失误\n" +
			"> 在投出失误的同时没能骰出至少一个成功度被称之为严重失误",
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			val := cmdArgs.GetArgN(1)
			num, err := strconv.ParseInt(val, 10, 64)

			if err == nil && num > 0 {
				successDegrees := int64(0)
				failedCount := int64(0)
				var results []string
				for i := int64(0); i < num; i++ {
					v := DiceRoll64(6)
					if v >= 5 {
						successDegrees++
					} else if v == 1 {
						failedCount++
					}
					// 过大的骰池不显示
					if num < 10 {
						results = append(results, strconv.FormatInt(v, 10))
					}
				}

				var detail string
				if len(results) > 0 {
					detail = "{" + strings.Join(results, "+") + "}\n"
				}

				text := fmt.Sprintf("<%s>骰点%dD6:\n", ctx.Player.Name, num)
				text += detail
				text += fmt.Sprintf("成功度:%d/%d\n", successDegrees, failedCount)

				successRank := int64(0) // 默认
				if failedCount > (num / 2) {
					// 半数失误
					successRank = -1

					if successDegrees == 0 {
						successRank = -2
					}
				}

				switch successRank {
				case -1:
					text += "失误"
				case -2:
					text += "严重失误"
				}
				ReplyToSender(ctx, msg, text)
			} else {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}

			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}

	// Emoklore(共鸣性怪异)规则支持
	helpEk := ".ek <技能名称>(+<奖励骰>) 判定值\n" +
		".ek 检索 // 骰“检索”等级个d10，计算成功数\n" +
		".ek 检索+2 // 在上一条基础上加骰2个d10\n" +
		".ek 检索 6  // 骰“检索”等级个d10，计算小于6的骰个数\n" +
		".ek 检索 知力+检索 // 骰”检索“，判定线为”知力+检索“\n" +
		".ek 5 4 // 骰5个d10，判定值4\n" +
		".ek 检索2 // 未录卡情况下判定2级检索\n" +
		".ek 共鸣 6 // 共鸣判定，成功后手动st共鸣+N\n"
	cmdEk := CmdItemInfo{
		Name:      "ek",
		ShortHelp: helpEk,
		Help:      "共鸣性怪异骰点:\n" + helpEk,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			mctx := ctx

			if cmdArgs.IsArgEqual(1, "help") {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}

			txt := cmdArgs.CleanArgs
			re := regexp.MustCompile(`(?:([^*+\-\s\d]+)(\d+)?|(\d+))\s*(?:([+\-*])\s*(\d+))?`)
			m := re.FindStringSubmatch(txt)
			if len(m) == 0 {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}

			// 读取技能名字和等级
			mustHaveCheckVal := false
			name := m[1]         // .ek 摸鱼
			nameLevelStr := m[2] // .ek 摸鱼3
			if name == "" && nameLevelStr == "" {
				// .ek 3 4
				nameLevelStr = m[3]
				mustHaveCheckVal = true
			}

			var nameLevel int64
			if nameLevelStr != "" {
				nameLevel, _ = strconv.ParseInt(nameLevelStr, 10, 64)
			} else {
				nameLevel, _ = VarGetValueInt64(mctx, name)
			}

			// 附加值 .ek 技能+1
			extraOp := m[4]
			extraValStr := m[5]
			extraVal := int64(0)
			if extraValStr != "" {
				extraVal, _ = strconv.ParseInt(extraValStr, 10, 64)
				if extraOp == "-" {
					extraVal = -extraVal
				}
			}

			restText := txt[len(m[0]):]
			restText = strings.TrimSpace(restText)

			if restText == "" && mustHaveCheckVal {
				ReplyToSender(ctx, msg, "必须填入判定值")
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			// 填充补充部分
			if restText == "" {
				restText = fmt.Sprintf("%s%s", name, nameLevelStr)
				mode := 1
				v := emokloreAttrParent[name]
				if v == nil {
					v = emokloreAttrParent2[name]
					mode = 2
				}
				if v == nil {
					v = emokloreAttrParent3[name]
					mode = 3
				}
				if v != nil {
					maxName := ""
					maxVal := int64(0)
					for _, i := range v {
						val, _ := VarGetValueInt64(mctx, i)
						if val >= maxVal {
							maxVal = val
							maxName = i
						}
					}
					if maxName != "" {
						switch mode {
						case 1:
							// 种类1: 技能+属性
							restText += " + " + maxName
						case 2:
							// 种类2: 属性/2[向上取整]
							restText = fmt.Sprintf("(%s+1)/2", maxName)
						case 3:
							// 种类3: 属性
							restText = maxName
						}
					}
				}
			}

			r, detail, err := DiceExprEvalBase(mctx, restText, RollExtraFlags{
				CocVarNumberMode: true,
				DisableBlock:     true,
			})
			if err != nil {
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			checkVal, _ := r.ReadInt()
			diceNum := nameLevel // 骰子个数为技能等级，至少1个
			if diceNum < 1 {
				diceNum = 1
			}
			if extraOp == "*" {
				diceNum *= extraVal
			} else {
				diceNum += extraVal
			}

			successDegrees := int64(0)
			var results []string
			for i := int64(0); i < diceNum; i++ {
				v := DiceRoll64(10)
				if v <= checkVal {
					successDegrees++
				}
				if v == 1 {
					successDegrees++
				}
				if v == 10 {
					successDegrees--
				}
				// 过大的骰池不显示
				if diceNum < 15 {
					results = append(results, strconv.FormatInt(v, 10))
				}
			}

			var detailPool string
			if len(results) > 0 {
				detailPool = "{" + strings.Join(results, "+") + "}\n"
			}

			// 检定原因
			showName := name
			if showName == "" {
				showName = nameLevelStr
			}
			if nameLevelStr != "" {
				showName += nameLevelStr
			}
			if extraVal > 0 {
				showName += extraOp + extraValStr
			}

			if detail != "" {
				detail = "{" + detail + "}"
			}

			checkText := ""
			switch {
			case successDegrees < 0:
				checkText = "大失败"
			case successDegrees == 0:
				checkText = "失败"
			case successDegrees == 1:
				checkText = "通常成功"
			case successDegrees == 2:
				checkText = "有效成功"
			case successDegrees == 3:
				checkText = "极限成功"
			case successDegrees >= 10:
				checkText = "灾难成功"
			case successDegrees >= 4:
				checkText = "奇迹成功"
			}

			text := fmt.Sprintf("<%s>的“%s”共鸣性怪异规则检定:\n", ctx.Player.Name, showName)
			text += detailPool
			text += fmt.Sprintf("判定值: %d%s\n", checkVal, detail)
			text += fmt.Sprintf("成功数: %d[%s]\n", successDegrees, checkText)

			ReplyToSender(ctx, msg, text)
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}

	helpEkGen := ".ekgen (<数量>) // 制卡指令，生成<数量>组人物属性，最高为10次"
	cmdEkgen := CmdItemInfo{
		Name:      "ekgen",
		ShortHelp: helpEkGen,
		Help:      "共鸣性怪异制卡指令:\n" + helpEkGen,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			n := cmdArgs.GetArgN(1)
			val, err := strconv.ParseInt(n, 10, 64)
			if err != nil {
				// 数量不存在时，视为1次
				val = 1
			}
			if val > 10 {
				val = 10
			}
			var i int64

			var ss []string
			for i = 0; i < val; i++ {
				randMap := map[int64]bool{}
				for j := 0; j < 6; j++ {
					n := DiceRoll64(24)
					if randMap[n] {
						j-- // 如果已经存在，重新roll
					} else {
						randMap[n] = true
					}
				}

				var nums Int64SliceDesc
				for k := range randMap {
					nums = append(nums, k)
				}
				sort.Sort(nums)

				last := int64(25)
				var nums2 []interface{}
				for _, j := range nums {
					val := last - j
					last = j
					nums2 = append(nums2, val)
				}
				nums2 = append(nums2, last)

				// 过滤大于6的
				for {
					// 遍历找出一个大于6的
					isGT6 := false
					var rest int64
					for index, _j := range nums2 {
						j := _j.(int64)
						if j > 6 {
							isGT6 = true
							rest = j - 6
							nums2[index] = int64(6)
							break
						}
					}

					if isGT6 {
						for index, _j := range nums2 {
							j := _j.(int64)
							if j < 6 {
								nums2[index] = j + rest
								break
							}
						}
					} else {
						break
					}
				}
				rand.Shuffle(len(nums2), func(i, j int) {
					nums2[i], nums2[j] = nums2[j], nums2[i]
				})

				text := fmt.Sprintf("身体:%d 灵巧:%d 精神:%d 五感:%d 知力:%d 魅力:%d 社会:%d", nums2...)
				text += fmt.Sprintf(" 运势:%d hp:%d mp:%d", DiceRoll64(6), nums2[0].(int64)+10, nums2[2].(int64)+nums2[4].(int64))

				ss = append(ss, text)
			}
			info := strings.Join(ss, "\n")
			ReplyToSender(ctx, msg, fmt.Sprintf("<%s>的共鸣性怪异人物做成:\n%s", ctx.Player.Name, info))
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}

	readNumber := func(text string, extra string) string {
		if text == "" {
			return ""
		}
		re0 := regexp.MustCompile(`^((\d+)[dDcCaA]|[bBpPfF])(.*)`)
		if re0.MatchString(text) {
			// 这种不需要管，是合法的表达式
			return text
		}

		re := regexp.MustCompile(`^(\d+)(.*)`)
		m := re.FindStringSubmatch(text)
		if len(m) > 0 {
			var rest string
			if len(m) > 2 {
				rest = m[2]
			}
			// 数字 a10 剩下部分
			return fmt.Sprintf("%s%s%s", m[1], extra, rest)
		}

		return text
	}

	cmdDX := CmdItemInfo{
		Name:      "dx",
		ShortHelp: ".dx 3c4",
		Help:      "双重十字规则骰点:\n.dx 3c4 // 也可使用.r 3c4替代",
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			if cmdArgs.GetArgN(1) == "help" {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}

			txt := readNumber(cmdArgs.CleanArgs, "c10")
			if txt == "" {
				txt = "1c10"
				cmdArgs.Args = []string{txt}
			}
			cmdArgs.CleanArgs = txt
			ctx.diceExprOverwrite = "1c10"
			roll := ctx.Dice.CmdMap["roll"]
			return roll.Solve(ctx, msg, cmdArgs)
		},
	}

	helpWW := `.ww 10a5 // 也可使用.r 10a5替代
.ww 10a5k6m7 // a加骰线 k成功线 m面数
.ww 10 // 骰10a10(默认情况下)
.ww set k6 // 修改成功线为6(当前群)
.ww set a8k6m9 // 修改其他默认设定
.ww set clr // 取消修改`
	cmdWW := CmdItemInfo{
		Name:      "ww",
		ShortHelp: helpWW,
		Help:      "骰池(WOD/无限规则骰点):\n" + helpWW,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			switch cmdArgs.GetArgN(1) {
			case "help":
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			case "set":
				arg2 := cmdArgs.GetArgN(2)
				if arg2 == "clr" || arg2 == "clear" {
					ctx.Group.ValueMap.Del("wodThreshold")
					ctx.Group.ValueMap.Del("wodPoints")
					ctx.Group.ValueMap.Del("wodAdd")
					ctx.Group.UpdatedAtTime = time.Now().Unix()
					ReplyToSender(ctx, msg, "骰池设定已恢复默认")
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				var texts []string

				reK := regexp.MustCompile(`[kK](\d+)`)
				if m := reK.FindStringSubmatch(arg2); len(m) > 0 {
					if v, err := strconv.ParseInt(m[1], 10, 64); err == nil {
						if v >= 1 {
							ctx.Group.ValueMap.Set("wodThreshold", &VMValue{TypeID: VMTypeInt64, Value: v})
							ctx.Group.UpdatedAtTime = time.Now().Unix()
							texts = append(texts, fmt.Sprintf("成功线k: 已修改为%d", v))
						} else {
							texts = append(texts, "成功线k: 需要至少为1")
						}
					}
				}
				reM := regexp.MustCompile(`[mM](\d+)`)
				if m := reM.FindStringSubmatch(arg2); len(m) > 0 {
					if v, err := strconv.ParseInt(m[1], 10, 64); err == nil {
						if v >= 1 && v <= 2000 {
							ctx.Group.ValueMap.Set("wodPoints", &VMValue{TypeID: VMTypeInt64, Value: v})
							ctx.Group.UpdatedAtTime = time.Now().Unix()
							texts = append(texts, fmt.Sprintf("骰子面数m: 已修改为%d", v))
						} else {
							texts = append(texts, "骰子面数m: 需要在1-2000之间")
						}
					}
				}
				reA := regexp.MustCompile(`[aA](\d+)`)
				if m := reA.FindStringSubmatch(arg2); len(m) > 0 {
					if v, err := strconv.ParseInt(m[1], 10, 64); err == nil {
						if v >= 2 {
							ctx.Group.ValueMap.Set("wodAdd", &VMValue{TypeID: VMTypeInt64, Value: v})
							ctx.Group.UpdatedAtTime = time.Now().Unix()
							texts = append(texts, fmt.Sprintf("加骰线a: 已修改为%d", v))
						} else {
							texts = append(texts, "加骰线a: 需要至少为2")
						}
					}
				}

				if len(texts) == 0 {
					return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
				}
				ReplyToSender(ctx, msg, strings.Join(texts, "\n"))
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			addNum := int64(10)
			if adding, exists := ctx.Group.ValueMap.Get("wodAdd"); exists {
				if t, ok := adding.(*VMValue); ok {
					addNum, _ = t.ReadInt64()
				}
			}

			txt := readNumber(cmdArgs.CleanArgs, fmt.Sprintf("a%d", addNum))
			if txt == "" {
				txt = fmt.Sprintf("10a%d", addNum)
				cmdArgs.Args = []string{txt}
			}
			cmdArgs.CleanArgs = txt

			roll := ctx.Dice.CmdMap["roll"]
			ctx.diceExprOverwrite = "10a10"
			return roll.Solve(ctx, msg, cmdArgs)
		},
	}

	textHelp := ".text <文本模板> // 文本指令，例: .text 看看手气: {1d16}"
	cmdText := CmdItemInfo{
		Name:      "text",
		ShortHelp: textHelp,
		Help:      "文本模板指令:\n" + textHelp,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			if ctx.Dice.Config.TextCmdTrustOnly {
				// 检查master和信任权限
				refuse := ctx.PrivilegeLevel != 100
				if refuse {
					refuse = ctx.PrivilegeLevel != 70
				}

				// 拒绝无权限访问
				if refuse {
					ReplyToSender(ctx, msg, "你不具备Master权限")
					return CmdExecuteResult{Matched: true, Solved: true}
				}
			}

			val := cmdArgs.GetArgN(1)
			if val != "" {
				ctx.Player.TempValueAlias = nil // 防止dnd的hp被转为“生命值”
				r, _, err := DiceExprTextBase(ctx, cmdArgs.CleanArgs, RollExtraFlags{DisableBlock: false})

				if err == nil && (r.TypeId == ds.VMTypeString || r.TypeId == ds.VMTypeNull) {
					var text string
					if r != nil {
						text = r.Value.(string)
					}

					if kw := cmdArgs.GetKwarg("asm"); r != nil && kw != nil {
						if ctx.PrivilegeLevel >= 40 {

							asm := r.GetAsmText()
							text += "\n" + asm
						}
					}

					seemsCommand := false
					if strings.HasPrefix(text, ".") || strings.HasPrefix(text, "。") || strings.HasPrefix(text, "!") || strings.HasPrefix(text, "/") {
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
			}
			return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
		},
	}

	cmdJsr := CmdItemInfo{
		EnableExecuteTimesParse: true,
		Name:                    "jsr",
		ShortHelp:               ".jsr 3# 10 // 投掷 10 面骰 3 次，结果不重复。结果存入骰池并可用 .drl 抽取。",
		Help: "不重复骰点（Jetter sans répéter）：.jsr 次数# 投骰表达式 (名字)" +
			"\n用例：.jsr 3# 10 // 投掷 10 面骰 3 次，结果不重复，结果存入骰池并可用 .drl 抽取。",
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			t := cmdArgs.SpecialExecuteTimes
			allArgClean := cmdArgs.CleanArgs
			allArgs := strings.Split(allArgClean, " ")
			var m int
			for i, v := range allArgs {
				if strings.HasPrefix(v, "d") {
					v = strings.Replace(v, "d", "", 1)
				}

				if n, err := strconv.Atoi(v); err == nil {
					m = n
					allArgs = append(allArgs[:i], allArgs[i+1:]...)
					break
				}
			}
			if t == 0 {
				t = 1
			}
			if m == 0 {
				m = int(getDefaultDicePoints(ctx))
			}
			if t > int(ctx.Dice.Config.MaxExecuteTime) {
				ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:骰点_轮数过多警告"))
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			if t > m {
				ReplyToSender(ctx, msg, fmt.Sprintf("无法不重复地投掷%d次%d面骰。", t, m))
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			var pool []int
			ma := make(map[int]bool)
			for len(pool) < t {
				n := rand.Intn(m) + 1
				if !ma[n] {
					ma[n] = true
					pool = append(pool, n)
				}
			}
			var results []string
			for _, v := range pool {
				results = append(results, fmt.Sprintf("D%d=%d", m, v))
			}
			allArgClean = strings.Join(allArgs, " ")
			for i := range pool {
				j := rand.Intn(i + 1)
				pool[i], pool[j] = pool[j], pool[i]
			}
			roulette := singleRoulette{
				Face: int64(m),
				Name: allArgClean,
				Pool: pool,
			}

			rouletteMap.Store(ctx.Group.GroupID, roulette)
			VarSetValueStr(ctx, "$t原因", allArgClean)
			if allArgClean != "" {
				forWhatText := DiceFormatTmpl(ctx, "核心:骰点_原因")
				VarSetValueStr(ctx, "$t原因句子", forWhatText)
			} else {
				VarSetValueStr(ctx, "$t原因句子", "")
			}
			VarSetValueInt64(ctx, "$t次数", int64(t))
			VarSetValueStr(ctx, "$t结果文本", strings.Join(results, "\n"))
			reply := DiceFormatTmpl(ctx, "核心:骰点_多轮")
			ReplyToSender(ctx, msg, reply)
			return CmdExecuteResult{
				Matched: true,
				Solved:  true,
			}
		},
	}

	cmdDrl := CmdItemInfo{
		EnableExecuteTimesParse: true,
		Name:                    "drl",
		ShortHelp: ".drl new 10 5# // 在当前群组创建一个面数为 10，能抽取 5 次的骰池\n.drl // 抽取当前群组的骰池\n" +
			".drlh //抽取当前群组的骰池，结果私聊发送",
		Help: "drl（Draw Lot）：.drl new 次数 投骰表达式 (名字) // 在当前群组创建一个骰池\n" +
			"用例：.drl new 10 5# // 在当前群组创建一个面数为 10，能抽取 5 次的骰池\n\n.drl // 抽取当前群组的骰池\n" +
			".drlh //抽取当前群组的骰池，结果私聊发送",
		DisabledInPrivate: true,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			if cmdArgs.IsArgEqual(1, "new") {
				// Make mode
				roulette := singleRoulette{
					Name: "",
					Face: getDefaultDicePoints(ctx),
					Time: 1,
				}
				t := cmdArgs.SpecialExecuteTimes
				if t != 0 {
					roulette.Time = t
				}

				m := cmdArgs.GetArgN(2)
				n := m
				if strings.HasPrefix(m, "d") {
					m = strings.Replace(m, "d", "", 1)
				}
				if i, err := strconv.Atoi(m); err == nil {
					roulette.Face = int64(i)
					text := cmdArgs.GetArgN(3)
					roulette.Name = text
				} else {
					roulette.Name = n
				}

				// NOTE(Xiangze Li): 允许创建更多轮数。使用洗牌算法后并不会很重复计算
				// if roulette.Time > int(ctx.Dice.Config.MaxExecuteTime) {
				// 	ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:骰点_轮数过多警告"))
				// 	return CmdExecuteResult{Matched: true, Solved: true}
				// }

				if int64(roulette.Time) > roulette.Face {
					ReplyToSender(ctx, msg, fmt.Sprintf("创建错误：无法不重复地投掷%d次%d面骰。",
						roulette.Time,
						roulette.Face))
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				// 创建pool后产生随机数，使用F-Y洗牌算法以保证随机性和效率
				var pool = make([]int, roulette.Time)
				var allNum = make([]int, roulette.Face)
				for i := range allNum {
					allNum[i] = i + 1
				}
				for idx := 0; idx < roulette.Time; idx++ {
					i := int(roulette.Face) - 1 - idx
					j := rand.Intn(i + 1)
					allNum[i], allNum[j] = allNum[j], allNum[i]
					pool[idx] = allNum[i]
				}
				roulette.Pool = pool

				rouletteMap.Store(ctx.Group.GroupID, roulette)
				ReplyToSender(ctx, msg, fmt.Sprintf("创建骰池%s成功，骰子面数%d，可抽取%d次。",
					roulette.Name, roulette.Face, roulette.Time))
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			// Draw mode
			var isRouletteEmpty = true
			rouletteMap.Range(func(key string, value singleRoulette) bool {
				isRouletteEmpty = false
				return false
			})
			tryLoad, ok := rouletteMap.Load(ctx.Group.GroupID)
			if isRouletteEmpty || !ok || tryLoad.Face == 0 {
				ReplyToSender(ctx, msg, "当前群组无骰池，请使用.drl new创建一个。")
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			result := fmt.Sprintf("D%d=%d", tryLoad.Face, tryLoad.Pool[0])
			tryLoad.Pool = append(tryLoad.Pool[:0], tryLoad.Pool[1:]...)
			VarSetValueStr(ctx, "$t原因", tryLoad.Name)
			if tryLoad.Name != "" {
				forWhatText := DiceFormatTmpl(ctx, "核心:骰点_原因")
				VarSetValueStr(ctx, "$t原因句子", forWhatText)
			} else {
				VarSetValueStr(ctx, "$t原因句子", "")
			}
			VarSetValueStr(ctx, "$t结果文本", result)
			reply := DiceFormatTmpl(ctx, "核心:骰点")

			if cmdArgs.Command == "drl" {
				if len(tryLoad.Pool) == 0 {
					reply += "\n骰池已经抽空，现在关闭。"
					tryLoad = singleRoulette{}
				}
				ReplyToSender(ctx, msg, reply)
			} else if cmdArgs.Command == "drlh" {
				announce := msg.Sender.Nickname + "进行了抽取。"
				reply += fmt.Sprintf("\n来自群%s(%s)",
					ctx.Group.GroupName, ctx.Group.GroupID)
				if len(tryLoad.Pool) == 0 {
					announce += "\n骰池已经抽空，现在关闭。"
					tryLoad = singleRoulette{}
				}
				ReplyGroup(ctx, msg, announce)
				ReplyPerson(ctx, msg, reply)
			}
			rouletteMap.Store(ctx.Group.GroupID, tryLoad)
			return CmdExecuteResult{
				Matched: true,
				Solved:  true,
			}
		},
	}

	self.RegisterExtension(&ExtInfo{
		Name:            "fun", // 扩展的名称，需要用于指令中，写简短点
		Version:         "1.1.0",
		Brief:           "娱乐扩展，主要提供今日人品、智能鸽子和text指令，以及暂时用于放置小众规则指令",
		AutoActive:      true, // 是否自动开启
		ActiveOnPrivate: true,
		Author:          "木落",
		Official:        true,
		OnCommandReceived: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) {
		},
		OnLoad: func() {
		},
		GetDescText: GetExtensionDesc,
		CmdMap: CmdMapCls{
			"gugu":  &cmdGugu,
			"咕咕":    &cmdGugu,
			"jrrp":  &cmdJrrp,
			"text":  &cmdText,
			"rsr":   &cmdRsr,
			"ek":    &cmdEk,
			"ekgen": &cmdEkgen,
			"dx":    &cmdDX,
			"w":     &cmdWW,
			"ww":    &cmdWW,
			"dxh":   &cmdDX,
			"wh":    &cmdWW,
			"wwh":   &cmdWW,
			"jsr":   &cmdJsr,
			"drl":   &cmdDrl,
			"drlh":  &cmdDrl,
		},
	})
}

func fingerprint(b string) uint64 {
	hash := fnv.New64a()
	_, _ = hash.Write([]byte(b))
	return hash.Sum64()
}
