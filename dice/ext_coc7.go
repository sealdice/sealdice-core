package dice

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

var fearListText = `
1) 洗澡恐惧症（Ablutophobia）：对于洗涤或洗澡的恐惧。
2) 恐高症（Acrophobia）：对于身处高处的恐惧。
3) 飞行恐惧症（Aerophobia）：对飞行的恐惧。
4) 广场恐惧症（Agoraphobia）：对于开放的（拥挤）公共场所的恐惧。
5) 恐鸡症（Alektorophobia）：对鸡的恐惧。
6) 大蒜恐惧症（Alliumphobia）：对大蒜的恐惧。
7) 乘车恐惧症（Amaxophobia）：对于乘坐地面载具的恐惧。
8) 恐风症（Ancraophobia）：对风的恐惧。
9) 男性恐惧症（Androphobia）：对于成年男性的恐惧。
10) 恐英症（Anglophobia）：对英格兰或英格兰文化的恐惧。
11) 恐花症（Anthophobia）：对花的恐惧。
12) 截肢者恐惧症（Apotemnophobia）：对截肢者的恐惧。
13) 蜘蛛恐惧症（Arachnophobia）：对蜘蛛的恐惧。
14) 闪电恐惧症（Astraphobia）：对闪电的恐惧。
15) 废墟恐惧症（Atephobia）：对遗迹或残址的恐惧。
16) 长笛恐惧症（Aulophobia）：对长笛的恐惧。
17) 细菌恐惧症（Bacteriophobia）：对细菌的恐惧。
18) 导弹/子弹恐惧症（Ballistophobia）：对导弹或子弹的恐惧。
19) 跌落恐惧症（Basophobia）：对于跌倒或摔落的恐惧。
20) 书籍恐惧症（Bibliophobia）：对书籍的恐惧。
21) 植物恐惧症（Botanophobia）：对植物的恐惧。
22) 美女恐惧症（Caligynephobia）：对美貌女性的恐惧。
23) 寒冷恐惧症（Cheimaphobia）：对寒冷的恐惧。
24) 恐钟表症（Chronomentrophobia）：对于钟表的恐惧。
25) 幽闭恐惧症（Claustrophobia）：对于处在封闭的空间中的恐惧。
26) 小丑恐惧症（Coulrophobia）：对小丑的恐惧。
27) 恐犬症（Cynophobia）：对狗的恐惧。
28) 恶魔恐惧症（Demonophobia）：对邪灵或恶魔的恐惧。
29) 人群恐惧症（Demophobia）：对人群的恐惧。
30) 牙科恐惧症①（Dentophobia）：对牙医的恐惧。
31) 丢弃恐惧症（Disposophobia）：对于丢弃物件的恐惧（贮藏癖）。
32) 皮毛恐惧症（Doraphobia）：对动物皮毛的恐惧。
33) 过马路恐惧症（Dromophobia）：对于过马路的恐惧。
34) 教堂恐惧症（Ecclesiophobia）：对教堂的恐惧。
35) 镜子恐惧症（Eisoptrophobia）：对镜子的恐惧。
36) 针尖恐惧症（Enetophobia）：对针或大头针的恐惧。
37) 昆虫恐惧症（Entomophobia）：对昆虫的恐惧。
38) 恐猫症（Felinophobia）：对猫的恐惧。
39) 过桥恐惧症（Gephyrophobia）：对于过桥的恐惧。
40) 恐老症（Gerontophobia）：对于老年人或变老的恐惧。
41) 恐女症（Gynophobia）：对女性的恐惧。
42) 恐血症（Haemaphobia）：对血的恐惧。
43) 宗教罪行恐惧症（Hamartophobia）：对宗教罪行的恐惧。
44) 触摸恐惧症（Haphophobia）：对于被触摸的恐惧。
45) 爬虫恐惧症（Herpetophobia）：对爬行动物的恐惧。
46) 迷雾恐惧症（Homichlophobia）：对雾的恐惧。
47) 火器恐惧症（Hoplophobia）：对火器的恐惧。
48) 恐水症（Hydrophobia）：对水的恐惧。
49) 催眠恐惧症①（Hypnophobia）：对于睡眠或被催眠的恐惧。
50) 白袍恐惧症（Iatrophobia）：对医生的恐惧。
51) 鱼类恐惧症（Ichthyophobia）：对鱼的恐惧。
52) 蟑螂恐惧症（Katsaridaphobia）：对蟑螂的恐惧。
53) 雷鸣恐惧症（Keraunophobia）：对雷声的恐惧。
54) 蔬菜恐惧症（Lachanophobia）：对蔬菜的恐惧。
55) 噪音恐惧症（Ligyrophobia）：对刺耳噪音的恐惧。
56) 恐湖症（Limnophobia）：对湖泊的恐惧。
57) 机械恐惧症（Mechanophobia）：对机器或机械的恐惧。
58) 巨物恐惧症（Megalophobia）：对于庞大物件的恐惧。
59) 捆绑恐惧症（Merinthophobia）：对于被捆绑或紧缚的恐惧。
60) 流星恐惧症（Meteorophobia）：对流星或陨石的恐惧。
61) 孤独恐惧症（Monophobia）：对于一人独处的恐惧。
62) 不洁恐惧症（Mysophobia）：对污垢或污染的恐惧。
63) 黏液恐惧症（Myxophobia）：对黏液（史莱姆）的恐惧。
64) 尸体恐惧症（Necrophobia）：对尸体的恐惧。
65) 数字 8 恐惧症（Octophobia）：对数字 8 的恐惧。
66) 恐牙症（Odontophobia）：对牙齿的恐惧。
67) 恐梦症（Oneirophobia）：对梦境的恐惧。
68) 称呼恐惧症（Onomatophobia）：对于特定词语的恐惧。
69) 恐蛇症（Ophidiophobia）：对蛇的恐惧。
70) 恐鸟症（Ornithophobia）：对鸟的恐惧。
71) 寄生虫恐惧症（Parasitophobia）：对寄生虫的恐惧。
72) 人偶恐惧症（Pediophobia）：对人偶的恐惧。
73) 吞咽恐惧症（Phagophobia）：对于吞咽或被吞咽的恐惧。
74) 药物恐惧症（Pharmacophobia）：对药物的恐惧。
75) 幽灵恐惧症（Phasmophobia）：对鬼魂的恐惧。
76) 日光恐惧症（Phenogophobia）：对日光的恐惧。
77) 胡须恐惧症（Pogonophobia）：对胡须的恐惧。
78) 河流恐惧症（Potamophobia）：对河流的恐惧。
79) 酒精恐惧症（Potophobia）：对酒或酒精的恐惧。
80) 恐火症（Pyrophobia）：对火的恐惧。
81) 魔法恐惧症（Rhabdophobia）：对魔法的恐惧。
82) 黑暗恐惧症（Scotophobia）：对黑暗或夜晚的恐惧。
83) 恐月症（Selenophobia）：对月亮的恐惧。
84) 火车恐惧症（Siderodromophobia）：对于乘坐火车出行的恐惧。
85) 恐星症（Siderophobia）：对星星的恐惧。
86) 狭室恐惧症（Stenophobia）：对狭小物件或地点的恐惧。
87) 对称恐惧症（Symmetrophobia）：对对称的恐惧。
88) 活埋恐惧症（Taphephobia）：对于被活埋或墓地的恐惧。
89) 公牛恐惧症（Taurophobia）：对公牛的恐惧。
90) 电话恐惧症（Telephonophobia）：对电话的恐惧。
91) 怪物恐惧症①（Teratophobia）：对怪物的恐惧。
92) 深海恐惧症（Thalassophobia）：对海洋的恐惧。
93) 手术恐惧症（Tomophobia）：对外科手术的恐惧。
94) 十三恐惧症（Triskadekaphobia）：对数字 13 的恐惧症。
95) 衣物恐惧症（Vestiphobia）：对衣物的恐惧。
96) 女巫恐惧症（Wiccaphobia）：对女巫与巫术的恐惧。
97) 黄色恐惧症（Xanthophobia）：对黄色或“黄”字的恐惧。
98) 外语恐惧症（Xenoglossophobia）：对外语的恐惧。
99) 异域恐惧症（Xenophobia）：对陌生人或外国人的恐惧。
100) 动物恐惧症（Zoophobia）：对动物的恐惧。
`

var ManiaListText = `
1) 沐浴癖（Ablutomania）：执着于清洗自己。
2) 犹豫癖（Aboulomania）：病态地犹豫不定。
3) 喜暗狂（Achluomania）：对黑暗的过度热爱。
4) 喜高狂（Acromaniaheights）：狂热迷恋高处。
5) 亲切癖（Agathomania）：病态地对他人友好。
6) 喜旷症（Agromania）：强烈地倾向于待在开阔空间中。
7) 喜尖狂（Aichmomania）：痴迷于尖锐或锋利的物体。
8) 恋猫狂（Ailuromania）：近乎病态地对猫友善。
9) 疼痛癖（Algomania）：痴迷于疼痛。
10) 喜蒜狂（Alliomania）：痴迷于大蒜。
11) 乘车癖（Amaxomania）：痴迷于乘坐车辆。
12) 欣快癖（Amenomania）：不正常地感到喜悦。
13) 喜花狂（Anthomania）：痴迷于花朵。
14) 计算癖（Arithmomania）：狂热地痴迷于数字。
15) 消费癖（Asoticamania）：鲁莽冲动地消费。
16) 隐居癖*（Automania）：过度地热爱独自隐居。（原文如此，存疑，Automania 实际上是恋车癖）
17) 芭蕾癖（Balletmania）：痴迷于芭蕾舞。
18) 窃书癖（Biliokleptomania）：无法克制偷窃书籍的冲动。
19) 恋书狂（Bibliomania）：痴迷于书籍和/或阅读
20) 磨牙癖（Bruxomania）：无法克制磨牙的冲动。
21) 灵臆症（Cacodemomania）：病态地坚信自己已被一个邪恶的灵体占据。
22) 美貌狂（Callomania）：痴迷于自身的美貌。
23) 地图狂（Cartacoethes）：在何时何处都无法控制查阅地图的冲动。
24) 跳跃狂（Catapedamania）：痴迷于从高处跳下。
25) 喜冷症（Cheimatomania）：对寒冷或寒冷的物体的反常喜爱。
26) 舞蹈狂（Choreomania）：无法控制地起舞或发颤。
27) 恋床癖（Clinomania）：过度地热爱待在床上。
28) 恋墓狂（Coimetormania）：痴迷于墓地。
29) 色彩狂（Coloromania）：痴迷于某种颜色。
30) 小丑狂（Coulromania）：痴迷于小丑。
31) 恐惧狂（Countermania）：执着于经历恐怖的场面。
32) 杀戮癖（Dacnomania）：痴迷于杀戮。
33) 魔臆症（Demonomania）：病态地坚信自己已被恶魔附身。
34) 抓挠癖（Dermatillomania）：执着于抓挠自己的皮肤。
35) 正义狂（Dikemania）：痴迷于目睹正义被伸张。
36) 嗜酒狂（Dipsomania）：反常地渴求酒精。
37) 毛皮狂（Doramania）：痴迷于拥有毛皮。（存疑）
38) 赠物癖（Doromania）：痴迷于赠送礼物。
39) 漂泊症（Drapetomania）：执着于逃离。
40) 漫游癖（Ecdemiomania）：执着于四处漫游。
41) 自恋狂（Egomania）：近乎病态地以自我为中心或自我崇拜。
42) 职业狂（Empleomania）：对于工作的无尽病态渴求。
43) 臆罪症（Enosimania）：病态地坚信自己带有罪孽。
44) 学识狂（Epistemomania）：痴迷于获取学识。
45) 静止癖（Eremiomania）：执着于保持安静。
46) 乙醚上瘾（Etheromania）：渴求乙醚。
47) 求婚狂（Gamomania）：痴迷于进行奇特的求婚。
48) 狂笑癖（Geliomania）：无法自制地，强迫性的大笑。
49) 巫术狂（Goetomania）：痴迷于女巫与巫术。
50) 写作癖（Graphomania）：痴迷于将每一件事写下来。
51) 裸体狂（Gymnomania）：执着于裸露身体。
52) 妄想狂（Habromania）：近乎病态地充满愉快的妄想（而不顾现实状况如何）。
53) 蠕虫狂（Helminthomania）：过度地喜爱蠕虫。
54) 枪械狂（Hoplomania）：痴迷于火器。
55) 饮水狂（Hydromania）：反常地渴求水分。
56) 喜鱼癖（Ichthyomania）：痴迷于鱼类。
57) 图标狂（Iconomania）：痴迷于图标与肖像
58) 偶像狂（Idolomania）：痴迷于甚至愿献身于某个偶像。
59) 信息狂（Infomania）：痴迷于积累各种信息与资讯。
60) 射击狂（Klazomania）：反常地执着于射击。
61) 偷窃癖（Kleptomania）：反常地执着于偷窃。
62) 噪音癖（Ligyromania）：无法自制地执着于制造响亮或刺耳的噪音。
63) 喜线癖（Linonomania）：痴迷于线绳。
64) 彩票狂（Lotterymania）：极端地执着于购买彩票。
65) 抑郁症（Lypemania）：近乎病态的重度抑郁倾向。
66) 巨石狂（Megalithomania）：当站在石环中或立起的巨石旁时，就会近乎病态地写出各种奇怪的创意。
67) 旋律狂（Melomania）：痴迷于音乐或一段特定的旋律。
68) 作诗癖（Metromania）：无法抑制地想要不停作诗。
69) 憎恨癖（Misomania）：憎恨一切事物，痴迷于憎恨某个事物或团体。
70) 偏执狂（Monomania）：近乎病态地痴迷与专注某个特定的想法或创意。
71) 夸大癖（Mythomania）：以一种近乎病态的程度说谎或夸大事物。
72) 臆想症（Nosomania）：妄想自己正在被某种臆想出的疾病折磨。
73) 记录癖（Notomania）：执着于记录一切事物（例如摄影）
74) 恋名狂（Onomamania）：痴迷于名字（人物的、地点的、事物的）
75) 称名癖（Onomatomania）：无法抑制地不断重复某个词语的冲动。
76) 剔指癖（Onychotillomania）：执着于剔指甲。
77) 恋食癖（Opsomania）：对某种食物的病态热爱。
78) 抱怨癖（Paramania）：一种在抱怨时产生的近乎病态的愉悦感。
79) 面具狂（Personamania）：执着于佩戴面具。
80) 幽灵狂（Phasmomania）：痴迷于幽灵。
81) 谋杀癖（Phonomania）：病态的谋杀倾向。
82) 渴光癖（Photomania）：对光的病态渴求。
83) 背德癖（Planomania）：病态地渴求违背社会道德（原文如此，存疑，Planomania 实际上是漂泊症）
84) 求财癖（Plutomania）：对财富的强迫性的渴望。
85) 欺骗狂（Pseudomania）：无法抑制的执着于撒谎。
86) 纵火狂（Pyromania）：执着于纵火。
87) 提问狂（Questiong-Asking Mania）：执着于提问。
88) 挖鼻癖（Rhinotillexomania）：执着于挖鼻子。
89) 涂鸦癖（Scribbleomania）：沉迷于涂鸦。
90) 列车狂（Siderodromomania）：认为火车或类似的依靠轨道交通的旅行方式充满魅力。
91) 臆智症（Sophomania）：臆想自己拥有难以置信的智慧。
92) 科技狂（Technomania）：痴迷于新的科技。
93) 臆咒狂（Thanatomania）：坚信自己已被某种死亡魔法所诅咒。
94) 臆神狂（Theomania）：坚信自己是一位神灵。
95) 抓挠癖（Titillomaniac）：抓挠自己的强迫倾向。
96) 手术狂（Tomomania）：对进行手术的不正常爱好。
97) 拔毛癖（Trichotillomania）：执着于拔下自己的头发。
98) 臆盲症（Typhlomania）：病理性的失明。
99) 嗜外狂（Xenomania）：痴迷于异国的事物。
100) 喜兽癖（Zoomania）：对待动物的态度近乎疯狂地友好
`

var difficultyPrefixMap = map[string]int{
	"":    1,
	"常规":  1,
	"困难":  2,
	"极难":  3,
	"大成功": 4,
	"困難":  2,
	"極難":  3,
	"常規":  1,
}

func cardRuleCheck(mctx *MsgContext, msg *Message) *GameSystemTemplate {
	cardType := ReadCardType(mctx)
	if cardType != "" && cardType != mctx.Group.System {
		ReplyToSender(mctx, msg, fmt.Sprintf("阻止操作：当前卡规则为 %s，群规则为 %s。\n为避免损坏此人物卡，请先更换角色卡，或使用.st fmt强制转卡", cardType, mctx.Group.System))
		return nil
	}
	tmpl := mctx.Group.GetCharTemplate(mctx.Dice)
	if tmpl == nil {
		ReplyToSender(mctx, msg, fmt.Sprintf("阻止操作：未发现人物卡使用的规则: %s，可能相关扩展已经卸载，请联系骰主", cardType))
		return nil
	}
	cmdStCharFormat(mctx, tmpl) // 转一下卡
	return tmpl
}

func RegisterBuiltinExtCoc7(self *Dice) {
	// 初始化疯狂列表
	reFear := regexp.MustCompile(`(\d+)\)\s+([^\n]+)`)
	m := reFear.FindAllStringSubmatch(fearListText, -1)
	fearMap := map[int]string{}
	for _, i := range m {
		n, _ := strconv.Atoi(i[1])
		fearMap[n] = i[2]
	}

	m = reFear.FindAllStringSubmatch(ManiaListText, -1)
	maniaMap := map[int]string{}
	for _, i := range m {
		n, _ := strconv.Atoi(i[1])
		maniaMap[n] = i[2]
	}

	// 初始化规则模板
	self.GameSystemTemplateAdd(getCoc7CharTemplate())

	helpRc := "" +
		".ra/rc <属性表达式> // 属性检定指令，当前者小于等于后者，检定通过\n" +
		".ra <难度><属性> // 如 .ra 困难侦查\n" +
		".ra b <属性表达式> // 奖励骰或惩罚骰\n" +
		".ra p2 <属性表达式> // 多个奖励骰或惩罚骰\n" +
		".ra 3#p <属性表达式> // 多重检定\n" +
		".ra <属性表达式> (@某人) // 对某人做检定(使用他的属性)\n" +
		".rch/rah // 暗中检定，和检定指令用法相同"

	cmdRc := &CmdItemInfo{
		EnableExecuteTimesParse: true,
		Name:                    "rc/ra",
		ShortHelp:               helpRc,
		Help:                    "检定指令:\n" + helpRc,
		AllowDelegate:           true,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			if len(cmdArgs.Args) == 0 {
				ctx.DelegateText = ""
				ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "COC:检定_格式错误"))
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			if cmdArgs.IsArgEqual(1, "help") {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}

			mctx := GetCtxProxyFirst(ctx, cmdArgs)
			mctx.DelegateText = ctx.DelegateText
			mctx.SystemTemplate = mctx.Group.GetCharTemplate(ctx.Dice)
			restText := cmdArgs.CleanArgs

			tmpl := cardRuleCheck(mctx, msg)
			if tmpl == nil {
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			mctx.Player.TempValueAlias = &tmpl.Alias // 兼容性支持

			reBP := regexp.MustCompile(`^[bBpP(]`)
			re2 := regexp.MustCompile(`([^\d]+)\s+([\d]+)`)

			if !reBP.MatchString(restText) {
				restText = re2.ReplaceAllString(restText, "$1$2")
				restText = "D100 " + restText
			} else {
				replaced := true
				if len(restText) > 1 {
					// 为了避免一种分支情况: .ra  b 50 测试，b和50中间的空格被消除
					ch2 := restText[1]
					if unicode.IsSpace(rune(ch2)) { // 暂不考虑太过奇葩的空格
						replaced = true
						restText = restText[:1] + " " + re2.ReplaceAllString(restText[2:], "$1$2")
					}
				}

				if !replaced {
					restText = re2.ReplaceAllString(restText, "$1$2")
				}
			}

			cocRule := mctx.Group.CocRuleIndex
			if cmdArgs.Command == "rc" {
				// 强制规则书
				cocRule = 0
			}

			var reason string
			var commandInfoItems []interface{}
			rollOne := func(manyTimes bool) *CmdExecuteResult {
				difficultyRequire := 0
				// 试图读取检定表达式
				swap := false
				r1, detail1, err := mctx.Dice.ExprEvalBase(restText, mctx, RollExtraFlags{
					CocVarNumberMode: true,
					CocDefaultAttrOn: true,
					DisableBlock:     true,
				})

				if err != nil {
					ReplyToSender(mctx, msg, "解析出错: "+restText)
					return &CmdExecuteResult{Matched: true, Solved: true}
				}

				difficultyRequire2 := difficultyPrefixMap[r1.Parser.CocFlagVarPrefix]
				if difficultyRequire2 > difficultyRequire {
					difficultyRequire = difficultyRequire2
				}
				expr1Text := r1.Matched
				expr2Text := r1.restInput

				// 如果读取完了，那么说明刚才读取的实际上是属性表达式
				if expr2Text == "" {
					expr2Text = "D100"
					swap = true
				}

				r2, detail2, err := mctx.Dice.ExprEvalBase(expr2Text, mctx, RollExtraFlags{
					CocVarNumberMode: true,
					CocDefaultAttrOn: true,
					DisableBlock:     true,
				})

				if err != nil {
					ReplyToSender(mctx, msg, "解析出错: "+expr2Text)
					return &CmdExecuteResult{Matched: true, Solved: true}
				}

				expr2Text = r2.Matched
				reason = r2.restInput

				difficultyRequire2 = difficultyPrefixMap[r2.Parser.CocFlagVarPrefix]
				if difficultyRequire2 > difficultyRequire {
					difficultyRequire = difficultyRequire2
				}

				if swap {
					r1, r2 = r2, r1
					detail1, detail2 = detail2, detail1 //nolint
					expr1Text, expr2Text = expr2Text, expr1Text
				}

				if r1.TypeID != VMTypeInt64 || r2.TypeID != VMTypeInt64 {
					ReplyToSender(mctx, msg, "你输入的表达式并非文本类型")
					return &CmdExecuteResult{Matched: true, Solved: true}
				}

				if r1.Matched == "d100" || r1.Matched == "D100" {
					// 此时没有必要
					detail1 = ""
				}

				var outcome = r1.Value.(int64)
				var attrVal = r2.Value.(int64)

				successRank, criticalSuccessValue := ResultCheck(mctx, cocRule, outcome, attrVal, difficultyRequire)
				// 根据难度需求，修改判定值
				checkVal := attrVal
				switch difficultyRequire {
				case 2:
					checkVal /= 2
				case 3:
					checkVal /= 5
				case 4:
					checkVal = criticalSuccessValue
				}
				VarSetValueInt64(mctx, "$tD100", outcome)
				VarSetValueInt64(mctx, "$t判定值", checkVal)
				VarSetValueInt64(mctx, "$tSuccessRank", int64(successRank))

				var suffix string
				var suffixFull string
				var suffixShort string
				if difficultyRequire > 1 {
					// 此时两者内容相同这样做是为了避免失败文本被计算两次
					suffixFull = GetResultTextWithRequire(mctx, successRank, difficultyRequire, false)
					suffixShort = suffixFull
				} else {
					suffixFull = GetResultTextWithRequire(mctx, successRank, difficultyRequire, false)
					suffixShort = GetResultTextWithRequire(mctx, successRank, difficultyRequire, true)
				}

				if manyTimes {
					suffix = suffixShort
				} else {
					suffix = suffixFull
				}

				VarSetValueStr(mctx, "$t判定结果", suffix)
				VarSetValueStr(mctx, "$t判定结果_详细", suffixFull)
				VarSetValueStr(mctx, "$t判定结果_简短", suffixShort)

				detailWrap := ""
				if detail1 != "" {
					detailWrap = ", (" + detail1 + ")"
				}

				// 指令信息标记
				infoItem := map[string]interface{}{
					"expr1":    expr1Text,
					"expr2":    expr2Text,
					"outcome":  outcome,
					"attrVal":  attrVal,
					"checkVal": checkVal,
					"rank":     successRank,
				}
				commandInfoItems = append(commandInfoItems, infoItem)

				VarSetValueStr(mctx, "$t检定表达式文本", expr1Text)
				VarSetValueStr(mctx, "$t属性表达式文本", expr2Text)
				VarSetValueStr(mctx, "$t检定计算过程", detailWrap)
				VarSetValueStr(mctx, "$t计算过程", detailWrap)

				SetTempVars(mctx, mctx.Player.Name) // 信息里没有QQ昵称，用这个顶一下
				VarSetValueStr(mctx, "$t结果文本", DiceFormatTmpl(mctx, "COC:检定_单项结果文本"))
				return nil
			}

			var text string
			if cmdArgs.SpecialExecuteTimes > 1 {
				VarSetValueInt64(mctx, "$t次数", int64(cmdArgs.SpecialExecuteTimes))
				if cmdArgs.SpecialExecuteTimes > int(ctx.Dice.MaxExecuteTime) {
					ReplyToSender(mctx, msg, DiceFormatTmpl(mctx, "COC:检定_轮数过多警告"))
					return CmdExecuteResult{Matched: true, Solved: true}
				}
				texts := []string{}
				for i := 0; i < cmdArgs.SpecialExecuteTimes; i++ {
					ret := rollOne(true)
					if ret != nil {
						return *ret
					}
					texts = append(texts, DiceFormatTmpl(mctx, "COC:检定_单项结果文本"))
				}

				VarSetValueStr(mctx, "$t原因", reason)
				VarSetValueStr(mctx, "$t结果文本", strings.Join(texts, `\n`))
				text = DiceFormatTmpl(mctx, "COC:检定_多轮")
			} else {
				ret := rollOne(false)
				if ret != nil {
					return *ret
				}
				VarSetValueStr(mctx, "$t原因", reason)
				VarSetValueStr(mctx, "$t结果文本", DiceFormatTmpl(mctx, "COC:检定_单项结果文本"))
				text = DiceFormatTmpl(mctx, "COC:检定")
			}

			isHide := cmdArgs.Command == "rah" || cmdArgs.Command == "rch"

			// 指令信息
			commandInfo := map[string]interface{}{
				"cmd":     "ra",
				"rule":    "coc7",
				"pcName":  mctx.Player.Name,
				"cocRule": cocRule,
				"items":   commandInfoItems,
			}
			if isHide {
				commandInfo["hide"] = isHide
			}
			mctx.CommandInfo = commandInfo

			if kw := cmdArgs.GetKwarg("ci"); kw != nil {
				info, err := json.Marshal(mctx.CommandInfo)
				if err == nil {
					text += "\n" + string(info)
				} else {
					text += "\n" + "指令信息无法序列化"
				}
			}

			if isHide {
				if msg.Platform == "QQ-CH" {
					ReplyToSender(mctx, msg, "QQ频道内尚不支持暗骰")
					return CmdExecuteResult{Matched: true, Solved: true}
				}
				if mctx.IsPrivate {
					ReplyToSender(mctx, msg, DiceFormatTmpl(mctx, "核心:提示_私聊不可用"))
				} else {
					mctx.CommandHideFlag = mctx.Group.GroupID
					ReplyGroup(mctx, msg, DiceFormatTmpl(mctx, "COC:检定_暗中_群内"))
					ReplyPerson(mctx, msg, DiceFormatTmpl(mctx, "COC:检定_暗中_私聊_前缀")+text)
				}
			} else {
				ReplyToSender(mctx, msg, text)
			}
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}

	helpSetCOC := ".setcoc 0-5 // 设置常见的0-5房规\n" +
		".setcoc dg // delta green 扩展规则"
	cmdSetCOC := &CmdItemInfo{
		Name:      "setcoc",
		ShortHelp: helpSetCOC,
		Help:      "设置房规:\n" + helpSetCOC,
		HelpFunc: func(isShort bool) string {
			help := ".setcoc 0-5 // 设置常见的0-5房规，0为规则书，2为国内常用规则\n" +
				".setcoc dg // delta green 扩展规则\n" +
				".setcoc details // 列出所有规则及其解释文本 \n"

			// 自定义
			for _, i := range self.CocExtraRules {
				n := strings.ReplaceAll(i.Desc, "\n", " ")
				help += fmt.Sprintf(".setcoc %d/%s // %s\n", i.Index, i.Key, n)
			}

			if isShort {
				return help
			}
			return "设置房规:\n" + help
		},
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			n := cmdArgs.GetArgN(1)
			suffix := "\nCOC7规则扩展已自动开启"
			setRuleByName(ctx, "coc7")

			switch n {
			case "0":
				ctx.Group.CocRuleIndex = 0
				text := fmt.Sprintf("已切换房规为%s:\n%s%s", SetCocRulePrefixText[ctx.Group.CocRuleIndex], SetCocRuleText[ctx.Group.CocRuleIndex], suffix)
				ReplyToSender(ctx, msg, text)
			case "1":
				ctx.Group.CocRuleIndex = 1
				text := fmt.Sprintf("已切换房规为%s:\n%s%s", SetCocRulePrefixText[ctx.Group.CocRuleIndex], SetCocRuleText[ctx.Group.CocRuleIndex], suffix)
				ReplyToSender(ctx, msg, text)
			case "2":
				ctx.Group.CocRuleIndex = 2
				text := fmt.Sprintf("已切换房规为%s:\n%s%s", SetCocRulePrefixText[ctx.Group.CocRuleIndex], SetCocRuleText[ctx.Group.CocRuleIndex], suffix)
				ReplyToSender(ctx, msg, text)
			case "3":
				ctx.Group.CocRuleIndex = 3
				text := fmt.Sprintf("已切换房规为%s:\n%s%s", SetCocRulePrefixText[ctx.Group.CocRuleIndex], SetCocRuleText[ctx.Group.CocRuleIndex], suffix)
				ReplyToSender(ctx, msg, text)
			case "4":
				ctx.Group.CocRuleIndex = 4
				text := fmt.Sprintf("已切换房规为%s:\n%s%s", SetCocRulePrefixText[ctx.Group.CocRuleIndex], SetCocRuleText[ctx.Group.CocRuleIndex], suffix)
				ReplyToSender(ctx, msg, text)
			case "5":
				ctx.Group.CocRuleIndex = 5
				text := fmt.Sprintf("已切换房规为%s:\n%s%s", SetCocRulePrefixText[ctx.Group.CocRuleIndex], SetCocRuleText[ctx.Group.CocRuleIndex], suffix)
				ReplyToSender(ctx, msg, text)
			case "dg":
				ctx.Group.CocRuleIndex = 11
				text := fmt.Sprintf("已切换房规为%s:\n%s%s", SetCocRulePrefixText[ctx.Group.CocRuleIndex], SetCocRuleText[ctx.Group.CocRuleIndex], suffix)
				ReplyToSender(ctx, msg, text)
			case "details":
				help := "当前有coc7规则如下:\n"
				for i := 0; i < 6; i++ {
					basicStr := strings.ReplaceAll(SetCocRuleText[i], "\n", " ")
					help += fmt.Sprintf(".setcoc %d // %s\n", i, basicStr)
				}
				// dg
				dgStr := strings.ReplaceAll(SetCocRuleText[11], "\n", " ")
				help += fmt.Sprintf(".setcoc dg // %s\n", dgStr)

				// 自定义
				for _, i := range self.CocExtraRules {
					ruleStr := strings.ReplaceAll(i.Desc, "\n", " ")
					help += fmt.Sprintf(".setcoc %d/%s // %s\n", i.Index, i.Key, ruleStr)
				}
				ReplyToSender(ctx, msg, help)
			case "help":
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			default:
				nInt, _ := strconv.ParseInt(n, 10, 64)
				for _, i := range ctx.Dice.CocExtraRules {
					if i.Key == n || nInt == int64(i.Index) {
						ctx.Group.CocRuleIndex = i.Index
						text := fmt.Sprintf("已切换房规为%s:\n%s%s", i.Name, i.Desc, suffix)
						ReplyToSender(ctx, msg, text)
						return CmdExecuteResult{Matched: true, Solved: true}
					}
				}

				if text, ok := SetCocRuleText[ctx.Group.CocRuleIndex]; ok {
					VarSetValueStr(ctx, "$t房规文本", text)
					VarSetValueStr(ctx, "$t房规", SetCocRulePrefixText[ctx.Group.CocRuleIndex])
					VarSetValueInt64(ctx, "$t房规序号", int64(ctx.Group.CocRuleIndex))
					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "COC:设置房规_当前"))
				} else {
					// TODO: 修改这种找规则的方式
					var rule *CocRuleInfo
					nInt64 := ctx.Group.CocRuleIndex
					for _, i := range ctx.Dice.CocExtraRules {
						if nInt64 == i.Index {
							rule = i
							break
						}
					}

					VarSetValueStr(ctx, "$t房规文本", rule.Desc)
					VarSetValueStr(ctx, "$t房规", rule.Name)
					VarSetValueInt64(ctx, "$t房规序号", int64(rule.Index))
					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "COC:设置房规_当前"))
				}
			}

			ctx.Group.ExtActive(ctx.Dice.ExtFind("coc7"))
			ctx.Group.System = "coc7"
			ctx.Group.UpdatedAtTime = time.Now().Unix()
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}

	helpRcv := ".rav/.rcv <技能> @某人 // 自己和某人进行对抗检定\n" +
		".rav <技能1> <技能2> @某A @某B // 对A和B两人做对抗检定，分别使用输入的两个技能数值\n" +
		"// 注: <技能>写法举例: 侦查、侦查40、困难侦查、40、侦查+10"
	cmdRcv := &CmdItemInfo{
		Name:          "rcv/rav",
		ShortHelp:     helpRcv,
		Help:          "对抗检定:\n" + helpRcv,
		AllowDelegate: true, // 特殊的代骰
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			val := cmdArgs.GetArgN(1)
			ctx.DelegateText = "" // 消除代骰文本，避免单人代骰出问题

			switch val {
			case "help", "":
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			default:
				// 至少@一人，检定才成立
				if len(cmdArgs.At) == 0 {
					return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
				}
				ctx1 := ctx
				ctx2 := ctx

				if cmdArgs.AmIBeMentionedFirst {
					// 第一个at的是骰子，不计为 at的人
					if len(cmdArgs.At) == 2 {
						// 单人
						ctx2, _ = cmdArgs.At[1].CopyCtx(ctx)
					} else if len(cmdArgs.At) == 3 {
						ctx1, _ = cmdArgs.At[1].CopyCtx(ctx)
						ctx2, _ = cmdArgs.At[2].CopyCtx(ctx)
					}
				} else {
					if len(cmdArgs.At) == 1 {
						// 单人
						ctx2, _ = cmdArgs.At[0].CopyCtx(ctx)
					} else if len(cmdArgs.At) == 2 {
						ctx1, _ = cmdArgs.At[0].CopyCtx(ctx)
						ctx2, _ = cmdArgs.At[1].CopyCtx(ctx)
					}
				}

				restText := cmdArgs.CleanArgs
				var lastMatched string
				readOneVal := func(mctx *MsgContext) (*CmdExecuteResult, int64, string, string) {
					r, _, err := mctx.Dice.ExprEvalBase(restText, mctx, RollExtraFlags{
						CocVarNumberMode: true,
						CocDefaultAttrOn: true,
						DisableBlock:     true,
					})

					if err != nil {
						ReplyToSender(ctx, msg, "解析出错: "+restText)
						return &CmdExecuteResult{Matched: true, Solved: true}, 0, "", ""
					}
					val, ok := r.ReadInt64()
					if !ok {
						ReplyToSender(ctx, msg, "类型不是数字: "+r.Matched)
						return &CmdExecuteResult{Matched: true, Solved: true}, 0, "", ""
					}
					lastMatched = r.Matched
					restText = r.restInput
					return nil, val, r.Parser.CocFlagVarPrefix, r.Matched
				}

				readOneOutcomeVal := func(mctx *MsgContext) (*CmdExecuteResult, int64, string) {
					restText = strings.TrimSpace(restText)
					if strings.HasPrefix(restText, ",") || strings.HasPrefix(restText, "，") {
						re := regexp.MustCompile(`[,，](.*)`)
						m := re.FindStringSubmatch(restText)
						restText = m[1]
						r, detail, err := mctx.Dice.ExprEvalBase(restText, mctx, RollExtraFlags{DisableBlock: true})
						if err != nil {
							ReplyToSender(ctx, msg, "解析出错: "+restText)
							return &CmdExecuteResult{Matched: true, Solved: true}, 0, ""
						}
						val, ok := r.ReadInt64()
						if !ok {
							ReplyToSender(ctx, msg, "类型不是数字: "+r.Matched)
							return &CmdExecuteResult{Matched: true, Solved: true}, 0, ""
						}
						restText = r.restInput
						return nil, val, "[" + detail + "]"
					}
					return nil, DiceRoll64(100), ""
				}

				ret, val1, difficult1, expr1 := readOneVal(ctx1)
				if ret != nil {
					return *ret
				}
				ret, outcome1, rollDetail1 := readOneOutcomeVal(ctx1)
				if ret != nil {
					return *ret
				}

				if restText == "" {
					restText = lastMatched
				}

				// lastMatched
				ret, val2, difficult2, expr2 := readOneVal(ctx2)
				if ret != nil {
					return *ret
				}
				ret, outcome2, rollDetail2 := readOneOutcomeVal(ctx2)
				if ret != nil {
					return *ret
				}

				cocRule := ctx.Group.CocRuleIndex
				if cmdArgs.Command == "rcv" {
					// 强制规则书
					cocRule = 0
				}

				successRank1, _ := ResultCheck(ctx, cocRule, outcome1, val1, 0)
				difficultyRequire1 := difficultyPrefixMap[difficult1]
				checkPass1 := successRank1 >= difficultyRequire1 // A是否通过检定

				successRank2, _ := ResultCheck(ctx, cocRule, outcome2, val2, 0)
				difficultyRequire2 := difficultyPrefixMap[difficult2]
				checkPass2 := successRank2 >= difficultyRequire2 // B是否通过检定

				winNum := 0
				switch {
				case checkPass1 && checkPass2:
					if successRank1 > successRank2 {
						// A 胜出
						winNum = -1
					} else if successRank1 < successRank2 {
						// B 胜出
						winNum = 1
					} else { //nolint:gocritic
						// 这里状况复杂，属性检定时，属性高的人胜出
						// 攻击时，成功等级相同，视为被攻击者胜出(目标选择闪避)
						// 攻击时，成功等级相同，视为攻击者胜出(目标选择反击)
						// 技能高的人胜出

						if cocRule == 11 {
							// dg规则下，似乎并不区分情况，比骰点大小即可
							if outcome1 < outcome2 {
								winNum = -1
							}
							if outcome1 > outcome2 {
								winNum = 1
							}
						} /* else {
							这段代码不能使用，因为如果是反击，那么技能是相同的，然而攻击方必胜
							reX := regexp.MustCompile("\\d+$")
							expr1X := reX.ReplaceAllString(expr1, "")
							expr2X := reX.ReplaceAllString(expr2, "")
							if expr1X != "" && expr1X == expr2X {
								if val1 > val2 {
									winNum = -1
								}
								if val1 < val2 {
									winNum = 1
								}
							}
						} */
					}
				case checkPass1 && !checkPass2:
					winNum = -1 // A胜
				case !checkPass1 && checkPass2:
					winNum = 1 // B胜
				default: /*no-op*/
				}

				suffix1 := GetResultTextWithRequire(ctx1, successRank1, difficultyRequire1, true)
				suffix2 := GetResultTextWithRequire(ctx2, successRank2, difficultyRequire2, true)

				p1Name := ctx1.Player.Name
				p2Name := ctx2.Player.Name
				if p1Name == "" {
					p1Name = "玩家A"
				}
				if p2Name == "" {
					p2Name = "玩家B"
				}

				VarSetValueStr(ctx, "$t玩家A", p1Name)
				VarSetValueStr(ctx, "$t玩家B", p2Name)

				VarSetValueStr(ctx, "$t玩家A判定式", expr1)
				VarSetValueStr(ctx, "$t玩家B判定式", expr2)

				VarSetValueInt64(ctx, "$t玩家A属性", val1) // 这个才是真正的判定值（判定线值，属性+难度影响）
				VarSetValueInt64(ctx, "$t玩家B属性", val2)

				VarSetValueInt64(ctx, "$t玩家A判定值", outcome1) // 这两个由下面的替换
				VarSetValueInt64(ctx, "$t玩家B判定值", outcome2) // 这两个由下面的替换
				VarSetValueInt64(ctx, "$t玩家A出目", outcome1)
				VarSetValueInt64(ctx, "$t玩家B出目", outcome2)

				VarSetValueStr(ctx, "$t玩家A判定过程", rollDetail1)
				VarSetValueStr(ctx, "$t玩家B判定过程", rollDetail2)

				VarSetValueStr(ctx, "$t玩家A判定结果", suffix1)
				VarSetValueStr(ctx, "$t玩家B判定结果", suffix2)

				VarSetValueInt64(ctx, "$tWinFlag", int64(winNum))

				ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "COC:对抗检定"))
			}
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}

	cmdSt := getCmdStBase()

	helpEn := `.en <技能名称>(技能点数) (+(<失败成长值>/)<成功成长值>) // 整体格式，可以直接看下面几个分解格式
.en <技能名称> // 骰D100，若点数大于当前值，属性成长1d10
.en <技能名称>(技能点数) // 骰D100，若点数大于技能点数，属性=技能点数+1d10
.en <技能名称>(技能点数) +<成功成长值> // 骰D100，若点数大于当前值，属性成长成功成长值点
.en <技能名称>(技能点数) +<失败成长值>/<成功成长值> // 骰D100，若点数大于当前值，属性成长成功成长值点，否则增加失败
.en <技能名称1> <技能名称2> // 批量技能成长，支持上述多种格式，复杂情况建议用|隔开每个技能`

	cmdEn := &CmdItemInfo{
		Name:          "en",
		ShortHelp:     helpEn,
		Help:          "成长指令:\n" + helpEn,
		AllowDelegate: false,
		Solve: func(mctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			if cmdArgs.IsArgEqual(1, "help") {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}

			// .en [技能名称]([技能值])+(([失败成长值]/)[成功成长值])
			// FIXME: 实在是被正则绕晕了，把多组和每组的正则分开了
			re := regexp.MustCompile(`([a-zA-Z_\p{Han}]+)\s*(\d+)?\s*(\+\s*([-+\ddD]+\s*/)?\s*([-+\ddD]+))?[^|]*?`)
			// 支持多组技能成长
			skills := re.FindAllString(cmdArgs.CleanArgs, -1)

			type enCheckResult struct {
				valid         bool
				invalidReason error

				varName     string
				varValueStr string
				successExpr string
				failExpr    string

				varValue    int64
				rollValue   int64
				successRank int
				success     bool
				resultText  string
				increment   int64
				newVarValue int64
			}
			RuleNotMatch := fmt.Errorf("rule not match")
			FormatMismatch := fmt.Errorf("format mismatch")
			SkillNotEntered := fmt.Errorf("skill not entered")
			SkillTypeError := fmt.Errorf("skill value type error")
			SuccessExprFormatError := fmt.Errorf("success expr format error")
			FailExprFormatError := fmt.Errorf("fail expr format error")
			singleRe := regexp.MustCompile(`([a-zA-Z_\p{Han}]+)\s*(\d+)?\s*(\+(([^/]+)/)?\s*(.+))?`)
			check := func(skill string) (checkResult enCheckResult) {
				checkResult.valid = true
				m := singleRe.FindStringSubmatch(skill)
				tmpl := cardRuleCheck(mctx, msg)
				if tmpl == nil {
					checkResult.valid = false
					checkResult.invalidReason = RuleNotMatch
					return
				}

				if m == nil {
					checkResult.valid = false
					checkResult.invalidReason = FormatMismatch
					return
				}

				varName := m[1]     // 技能名称
				varValueStr := m[2] // 技能值 - 字符串
				successExpr := m[6] // 成功的加值表达式
				failExpr := m[5]    // 失败的加值表达式

				var varValue int64
				checkResult.varName = varName
				checkResult.varValueStr = varValueStr

				// 首先，试图读取技能的值
				if varValueStr != "" {
					varValue, _ = strconv.ParseInt(varValueStr, 10, 64)
				} else {
					val, exists := VarGetValue(mctx, varName)
					if !exists {
						// 没找到，尝试取得默认值
						val, _, _, exists = tmpl.GetDefaultValueEx0(mctx, varName)
					}
					if !exists {
						checkResult.valid = false
						checkResult.invalidReason = SkillNotEntered
						return
					}
					if val.TypeID != VMTypeInt64 {
						checkResult.valid = false
						checkResult.invalidReason = SkillTypeError
						return
					}
					varValue = val.Value.(int64)
				}

				d100 := DiceRoll64(100)
				// 注意一下，这里其实是，小于失败 大于成功
				successRank, _ := ResultCheck(mctx, mctx.Group.CocRuleIndex, d100, varValue, 0)
				var resultText string
				// 若玩家投出了高于当前技能值的结果，或者结果大于95，则调查员该技能获得改善：骰1D10并且立即将结果加到当前技能值上。技能可通过此方式超过100%。
				if d100 > 95 {
					successRank = -1
				}
				var success bool
				if successRank > 0 {
					resultText = "失败"
					success = false
				} else {
					resultText = "成功"
					success = true
				}

				checkResult.rollValue = d100
				checkResult.varValue = varValue
				checkResult.resultText = resultText
				checkResult.successRank = successRank
				checkResult.success = success

				if success {
					if successExpr == "" {
						successExpr = "1d10"
					}

					r, _, err := mctx.Dice.ExprEval(successExpr, mctx)
					checkResult.successExpr = successExpr
					if err != nil {
						checkResult.valid = false
						checkResult.invalidReason = SuccessExprFormatError
						return
					}

					increment := r.VMValue.Value.(int64)
					checkResult.increment = increment
					checkResult.newVarValue = varValue + increment
				} else {
					if failExpr == "" {
						checkResult.increment = 0
						checkResult.newVarValue = varValue
					} else {
						r, _, err := mctx.Dice.ExprEval(failExpr, mctx)
						checkResult.failExpr = failExpr
						if err != nil {
							checkResult.valid = false
							checkResult.invalidReason = FailExprFormatError
							return
						}

						increment := r.VMValue.Value.(int64)
						checkResult.increment = increment
						checkResult.newVarValue = varValue + increment
					}
				}
				return
			}

			VarSetValueInt64(mctx, "$t数量", int64(len(skills)))
			if len(skills) < 1 { //nolint:nestif
				ReplyToSender(mctx, msg, "指令格式不匹配")
				return CmdExecuteResult{Matched: true, Solved: true}
			} else if len(skills) > 10 {
				ReplyToSender(mctx, msg, DiceFormatTmpl(mctx, "COC:技能成长_批量_技能过多警告"))
				return CmdExecuteResult{Matched: true, Solved: true}
			} else if len(skills) == 1 {
				checkResult := check(skills[0])
				VarSetValueStr(mctx, "$t技能", checkResult.varName)
				VarSetValueInt64(mctx, "$tD100", checkResult.rollValue)
				VarSetValueInt64(mctx, "$t判定值", checkResult.varValue)
				VarSetValueStr(mctx, "$t判定结果", checkResult.resultText)
				VarSetValueInt64(mctx, "$tSuccessRank", int64(checkResult.successRank))
				VarSetValueInt64(mctx, "$t旧值", checkResult.varValue)
				VarSetValueInt64(mctx, "$t增量", checkResult.increment)
				VarSetValueInt64(mctx, "$t新值", checkResult.newVarValue)
				if checkResult.valid {
					VarSetValueInt64(mctx, checkResult.varName, checkResult.newVarValue)
					if checkResult.success {
						VarSetValueStr(mctx, "$t表达式文本", checkResult.successExpr)
						VarSetValueStr(mctx, "$t结果文本", DiceFormatTmpl(mctx, "COC:技能成长_结果_成功"))
					} else {
						VarSetValueStr(mctx, "$t表达式文本", checkResult.failExpr)
						if checkResult.failExpr == "" {
							VarSetValueStr(mctx, "$t结果文本", DiceFormatTmpl(mctx, "COC:技能成长_结果_失败"))
						} else {
							VarSetValueStr(mctx, "$t结果文本", DiceFormatTmpl(mctx, "COC:技能成长_结果_失败变更"))
						}
					}
					VarSetValueInt64(mctx, "$t数量", int64(1))

					VarSetValueStr(mctx, "$t当前绑定角色", mctx.ChBindCurGet())
					if mctx.Player.AutoSetNameTemplate != "" {
						_, _ = SetPlayerGroupCardByTemplate(mctx, mctx.Player.AutoSetNameTemplate)
					}
					ReplyToSender(mctx, msg, DiceFormatTmpl(mctx, "COC:技能成长"))
				} else {
					switch {
					case errors.Is(checkResult.invalidReason, RuleNotMatch):
						// skip
						return CmdExecuteResult{Matched: true, Solved: true}
					case errors.Is(checkResult.invalidReason, FormatMismatch):
						ReplyToSender(mctx, msg, "指令格式不匹配")
						return CmdExecuteResult{Matched: true, Solved: true}
					case errors.Is(checkResult.invalidReason, SkillNotEntered):
						ReplyToSender(mctx, msg, DiceFormatTmpl(mctx, "COC:技能成长_属性未录入"))
					case errors.Is(checkResult.invalidReason, SkillTypeError):
						ReplyToSender(mctx, msg, DiceFormatTmpl(mctx, "COC:技能成长_错误的属性类型"))
					case errors.Is(checkResult.invalidReason, SuccessExprFormatError):
						ReplyToSender(mctx, msg, DiceFormatTmpl(mctx, "COC:技能成长_错误的成功成长值"))
					case errors.Is(checkResult.invalidReason, FailExprFormatError):
						ReplyToSender(mctx, msg, DiceFormatTmpl(mctx, "COC:技能成长_错误的失败成长值"))
					}
				}
			} else {
				var checkResultStrs []string
				var checkResults []enCheckResult
				for _, skill := range skills {
					checkResult := check(skill)
					checkResults = append(checkResults, checkResult)
				}
				for _, checkResult := range checkResults {
					VarSetValueStr(mctx, "$t技能", checkResult.varName)
					VarSetValueInt64(mctx, "$tD100", checkResult.rollValue)
					VarSetValueInt64(mctx, "$t判定值", checkResult.varValue)
					VarSetValueStr(mctx, "$t判定结果", checkResult.resultText)
					VarSetValueInt64(mctx, "$tSuccessRank", int64(checkResult.successRank))
					VarSetValueInt64(mctx, "$t旧值", checkResult.varValue)
					VarSetValueInt64(mctx, "$t增量", checkResult.increment)
					VarSetValueInt64(mctx, "$t新值", checkResult.newVarValue)
					if checkResult.valid {
						VarSetValueInt64(mctx, checkResult.varName, checkResult.newVarValue)
						if checkResult.success {
							VarSetValueStr(mctx, "$t表达式文本", checkResult.successExpr)
							VarSetValueStr(mctx, "$t结果文本", DiceFormatTmpl(mctx, "COC:技能成长_结果_成功_无后缀"))
						} else {
							VarSetValueStr(mctx, "$t表达式文本", checkResult.failExpr)
							if checkResult.failExpr == "" {
								VarSetValueStr(mctx, "$t结果文本", DiceFormatTmpl(mctx, "COC:技能成长_结果_失败"))
							} else {
								VarSetValueStr(mctx, "$t结果文本", DiceFormatTmpl(mctx, "COC:技能成长_结果_失败变更_无后缀"))
							}
						}
						resStr := DiceFormatTmpl(mctx, "COC:技能成长_批量_单条")
						checkResultStrs = append(checkResultStrs, resStr)
					} else {
						temp := DiceFormatTmpl(mctx, "COC:技能成长_批量_单条错误前缀")
						switch {
						case errors.Is(checkResult.invalidReason, RuleNotMatch):
							// skip
							return CmdExecuteResult{Matched: true, Solved: true}
						case errors.Is(checkResult.invalidReason, FormatMismatch):
							ReplyToSender(mctx, msg, "指令格式不匹配")
							return CmdExecuteResult{Matched: true, Solved: true}
						case errors.Is(checkResult.invalidReason, SkillNotEntered):
							temp += DiceFormatTmpl(mctx, "COC:技能成长_属性未录入_无前缀")
						case errors.Is(checkResult.invalidReason, SkillTypeError):
							temp += DiceFormatTmpl(mctx, "COC:技能成长_错误的属性类型_无前缀")
						case errors.Is(checkResult.invalidReason, SuccessExprFormatError):
							temp += DiceFormatTmpl(mctx, "COC:技能成长_错误的成功成长值_无前缀")
						case errors.Is(checkResult.invalidReason, FailExprFormatError):
							temp += DiceFormatTmpl(mctx, "COC:技能成长_错误的失败成长值_无前缀")
						}
						checkResultStrs = append(checkResultStrs, temp)
					}
				}
				sep := DiceFormatTmpl(mctx, "COC:技能成长_批量_分隔符")
				resultStr := strings.Join(checkResultStrs, sep)
				VarSetValueStr(mctx, "$t总结果文本", resultStr)
				VarSetValueStr(mctx, "$t当前绑定角色", mctx.ChBindCurGet())
				if mctx.Player.AutoSetNameTemplate != "" {
					_, _ = SetPlayerGroupCardByTemplate(mctx, mctx.Player.AutoSetNameTemplate)
				}
				ReplyToSender(mctx, msg, DiceFormatTmpl(mctx, "COC:技能成长_批量"))
			}
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}

	cmdTi := &CmdItemInfo{
		Name:      "ti",
		ShortHelp: ".ti // 抽取一个临时性疯狂症状",
		Help:      "抽取临时性疯狂症状:\n.li // 抽取一个临时性疯狂症状",
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			if cmdArgs.IsArgEqual(1, "help") {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}
			num := DiceRoll(10)
			VarSetValueStr(ctx, "$t表达式文本", fmt.Sprintf("1D10=%d", num))
			VarSetValueInt64(ctx, "$t选项值", int64(num))

			var desc string
			extraNum1 := DiceRoll(10)
			VarSetValueInt64(ctx, "$t附加值1", int64(extraNum1))
			switch num {
			case 1:
				desc += fmt.Sprintf("失忆：调查员会发现自己只记得最后身处的安全地点，却没有任何来到这里的记忆。例如，调查员前一刻还在家中吃着早饭，下一刻就已经直面着不知名的怪物。这将会持续 1D10=%d 轮。", extraNum1)
			case 2:
				desc += fmt.Sprintf("假性残疾：调查员陷入了心理性的失明，失聪以及躯体缺失感中，持续 1D10=%d 轮。", extraNum1)
			case 3:
				desc += fmt.Sprintf("暴力倾向：调查员陷入了六亲不认的暴力行为中，对周围的敌人与友方进行着无差别的攻击，持续 1D10=%d 轮。", extraNum1)
			case 4:
				desc += fmt.Sprintf("偏执：调查员陷入了严重的偏执妄想之中。有人在暗中窥视着他们，同伴中有人背叛了他们，没有人可以信任，万事皆虚。持续 1D10=%d 轮", extraNum1)
			case 5:
				desc += fmt.Sprintf("人际依赖：守秘人适当参考调查员的背景中重要之人的条目，调查员因为一些原因而将他人误认为了他重要的人并且努力的会与那个人保持那种关系，持续 1D10=%d 轮", extraNum1)
			case 6:
				desc += fmt.Sprintf("昏厥：调查员当场昏倒，并需要 1D10=%d 轮才能苏醒。", extraNum1)
			case 7:
				desc += fmt.Sprintf("逃避行为：调查员会用任何的手段试图逃离现在所处的位置，即使这意味着开走唯一一辆交通工具并将其它人抛诸脑后，调查员会试图逃离 1D10=%d 轮。", extraNum1)
			case 8:
				desc += fmt.Sprintf("竭嘶底里：调查员表现出大笑，哭泣，嘶吼，害怕等的极端情绪表现，持续 1D10=%d 轮。", extraNum1)
			case 9:
				desc += fmt.Sprintf("恐惧：调查员通过一次 D100 或者由守秘人选择，来从恐惧症状表中选择一个恐惧源，就算这一恐惧的事物是并不存在的，调查员的症状会持续 1D10=%d 轮。", extraNum1)
				extraNum2 := DiceRoll(100)
				desc += fmt.Sprintf("\n1D100=%d\n", extraNum2)
				desc += fearMap[extraNum2]
				VarSetValueInt64(ctx, "$t附加值2", int64(extraNum2))
			case 10:
				desc += fmt.Sprintf("躁狂：调查员通过一次 D100 或者由守秘人选择，来从躁狂症状表中选择一个躁狂的诱因，这个症状将会持续 1D10=%d 轮。", extraNum1)
				extraNum2 := DiceRoll(100)
				desc += fmt.Sprintf("\n1D100=%d\n", extraNum2)
				desc += maniaMap[extraNum2]
				VarSetValueInt64(ctx, "$t附加值2", int64(extraNum2))
			}
			VarSetValueStr(ctx, "$t疯狂描述", desc)

			ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "COC:疯狂发作_即时症状"))
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}

	cmdLi := &CmdItemInfo{
		Name:      "li",
		ShortHelp: ".li // 抽取一个总结性疯狂症状",
		Help:      "抽取总结性疯狂症状:\n.li // 抽取一个总结性疯狂症状",
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			if cmdArgs.IsArgEqual(1, "help") {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}
			num := DiceRoll(10)
			VarSetValueStr(ctx, "$t表达式文本", fmt.Sprintf("1D10=%d", num))
			VarSetValueInt64(ctx, "$t选项值", int64(num))

			var desc string
			extraNum1 := DiceRoll(10)
			VarSetValueInt64(ctx, "$t附加值1", int64(extraNum1))
			switch num {
			case 1:
				desc += "失忆：回过神来，调查员们发现自己身处一个陌生的地方，并忘记了自己是谁。记忆会随时间恢复。"
			case 2:
				desc += fmt.Sprintf("被窃：调查员在 1D10=%d 小时后恢复清醒，发觉自己被盗，身体毫发无损。如果调查员携带着宝贵之物（见调查员背景），做幸运检定来决定其是否被盗。所有有价值的东西无需检定自动消失。", extraNum1)
			case 3:
				desc += fmt.Sprintf("遍体鳞伤：调查员在 1D10=%d 小时后恢复清醒，发现自己身上满是拳痕和瘀伤。生命值减少到疯狂前的一半，但这不会造成重伤。调查员没有被窃。这种伤害如何持续到现在由守秘人决定。", extraNum1)
			case 4:
				desc += "暴力倾向：调查员陷入强烈的暴力与破坏欲之中。调查员回过神来可能会理解自己做了什么也可能毫无印象。调查员对谁或何物施以暴力，他们是杀人还是仅仅造成了伤害，由守秘人决定。"
			case 5:
				desc += "极端信念：查看调查员背景中的思想信念，调查员会采取极端和疯狂的表现手段展示他们的思想信念之一。比如一个信教者会在地铁上高声布道。"
			case 6:
				desc += fmt.Sprintf("重要之人：考虑调查员背景中的重要之人，及其重要的原因。在 1D10=%d 小时或更久的时间中，调查员将不顾一切地接近那个人，并为他们之间的关系做出行动。", extraNum1)
			case 7:
				desc += "被收容：调查员在精神病院病房或警察局牢房中回过神来，他们可能会慢慢回想起导致自己被关在这里的事情。"
			case 8:
				desc += "逃避行为：调查员恢复清醒时发现自己在很远的地方，也许迷失在荒郊野岭，或是在驶向远方的列车或长途汽车上。"
			case 9:
				desc += fmt.Sprintf("恐惧：调查员患上一个新的恐惧症状。在恐惧症状表上骰 1 个 D100 来决定症状，或由守秘人选择一个。调查员在 1D10=%d 小时后回过神来，并开始为避开恐惧源而采取任何措施。", extraNum1)
				extraNum2 := DiceRoll(100)
				desc += fmt.Sprintf("\n1D100=%d\n", extraNum2)
				desc += fearMap[extraNum2]
				VarSetValueInt64(ctx, "$t附加值2", int64(extraNum2))
			case 10:
				desc += fmt.Sprintf("狂躁：调查员患上一个新的狂躁症状。在狂躁症状表上骰 1 个 d100 来决定症状，或由守秘人选择一个。调查员会在 1D10=%d 小时后恢复理智。在这次疯狂发作中，调查员将完全沉浸于其新的狂躁症状。这症状是否会表现给旁人则取决于守秘人和此调查员。", extraNum1)
				extraNum2 := DiceRoll(100)
				desc += fmt.Sprintf("\n1D100=%d\n", extraNum2)
				desc += maniaMap[extraNum2]
				VarSetValueInt64(ctx, "$t附加值2", int64(extraNum2))
			}
			VarSetValueStr(ctx, "$t疯狂描述", desc)

			ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "COC:疯狂发作_总结症状"))
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}

	helpSc := ".sc <成功时掉san>/<失败时掉san> // 对理智进行一次D100检定，根据结果扣除理智\n" +
		".sc <失败时掉san> //同上，简易写法 \n" +
		".sc (b/p) (<成功时掉san>/)<失败时掉san> // 加上奖惩骰"
	cmdSc := &CmdItemInfo{
		Name:          "sc",
		ShortHelp:     helpSc,
		Help:          "理智检定:\n" + helpSc,
		AllowDelegate: true,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			// http://www.antagonistes.com/files/CoC%20CheatSheet.pdf
			// v2: (worst) FAIL — REGULAR SUCCESS — HARD SUCCESS — EXTREME SUCCESS (best)

			if len(cmdArgs.Args) == 0 {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}
			if cmdArgs.IsArgEqual(1, "help") {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}

			mctx := GetCtxProxyFirst(ctx, cmdArgs)

			tmpl := cardRuleCheck(mctx, msg)
			if tmpl == nil {
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			mctx.Player.TempValueAlias = &tmpl.Alias

			// 首先读取一个值
			// 试图读取 /: 读到了，当前是成功值，转入读取单项流程，试图读取失败值
			// 试图读取 ,: 读到了，当前是失败值，试图转入下一项
			// 试图读取表达式: 读到了，当前是判定值

			defaultSuccessExpr := "0"
			argText := cmdArgs.CleanArgs

			splitDiv := func(text string) (int, string, string) {
				ret := strings.SplitN(text, "/", 2)
				if len(ret) == 1 {
					return 1, ret[0], ""
				}
				return 2, ret[0], ret[1]
			}

			getOnePiece := func() (string, string, string, int) {
				expr1 := "d100" // 先假设为常见情况，也就是D100
				expr2 := ""
				expr3 := ""

				innerGetOnePiece := func() int {
					var err error
					r, _, err := mctx.Dice.ExprEvalBase(argText, mctx, RollExtraFlags{IgnoreDiv0: true, DisableBlock: true})
					if err != nil {
						// 情况1，完全不能解析
						return 1
					}

					num, t1, t2 := splitDiv(r.Matched)
					if num == 2 {
						expr2 = t1
						expr3 = t2
						argText = r.restInput
						return 0
					}

					// 现在可以肯定并非是 .sc 1/1 形式，那么判断一下
					// .sc 1 或 .sc 1 1/1 或 .sc 1 1
					if strings.HasPrefix(r.restInput, ",") || r.restInput == "" {
						// 结束了，所以这是 .sc 1
						expr2 = defaultSuccessExpr
						expr3 = r.Matched
						argText = r.restInput
						return 0
					}

					// 可能是 .sc 1 1 或 .sc 1 1/1
					expr1 = r.Matched
					r2, _, err := mctx.Dice.ExprEvalBase(r.restInput, mctx, RollExtraFlags{DisableBlock: true})
					if err != nil {
						return 2
					}
					num, t1, t2 = splitDiv(r2.Matched)
					if num == 2 {
						// sc 1 1
						expr2 = t1
						expr3 = t2
						argText = r2.restInput
						return 0
					}

					// sc 1/1
					expr2 = defaultSuccessExpr
					expr3 = t1
					argText = r2.restInput
					return 0
				}

				return expr1, expr2, expr3, innerGetOnePiece()
			}

			expr1, expr2, expr3, code := getOnePiece()

			switch code {
			case 1:
				// 这输入的是啥啊，完全不能解析
				ReplyToSender(mctx, msg, DiceFormatTmpl(mctx, "COC:理智检定_格式错误"))
			case 2:
				// 已经匹配了/，失败扣除血量不正确
				ReplyToSender(mctx, msg, DiceFormatTmpl(mctx, "COC:理智检定_格式错误"))
			case 3:
				// 第一个式子对了，第二个是啥东西？
				ReplyToSender(mctx, msg, DiceFormatTmpl(mctx, "COC:理智检定_格式错误"))

			case 0:
				// 完全正确
				var d100 int64
				var san int64

				// 获取判定值
				rCond, detailCond, err := mctx.Dice.ExprEval(expr1, mctx)
				if err == nil && rCond.TypeID == VMTypeInt64 {
					d100 = rCond.Value.(int64)
				}
				detailWrap := ""
				if detailCond != "" {
					if expr1 != "d100" {
						detailWrap = ", (" + detailCond + ")"
					}
				}

				// 读取san值
				r, _, err := mctx.Dice.ExprEval("san", mctx)
				if err == nil && r.TypeID == VMTypeInt64 {
					san = r.Value.(int64)
				}
				_san, err := strconv.ParseInt(argText, 10, 64)
				if err == nil {
					san = _san
				}

				// 进行检定
				successRank, _ := ResultCheck(mctx, mctx.Group.CocRuleIndex, d100, san, 0)
				suffix := GetResultText(mctx, successRank, false)
				suffixShort := GetResultText(mctx, successRank, true)

				VarSetValueStr(mctx, "$t检定表达式文本", expr1)
				VarSetValueStr(mctx, "$t检定计算过程", detailWrap)

				VarSetValueInt64(mctx, "$tD100", d100)
				VarSetValueInt64(mctx, "$t判定值", san)
				VarSetValueStr(mctx, "$t判定结果", suffix)
				VarSetValueStr(mctx, "$t判定结果_详细", suffix)
				VarSetValueStr(mctx, "$t判定结果_简短", suffixShort)
				VarSetValueInt64(ctx, "$tSuccessRank", int64(successRank))
				VarSetValueInt64(mctx, "$t旧值", san)

				SetTempVars(mctx, mctx.Player.Name) // 信息里没有QQ昵称，用这个顶一下
				VarSetValueStr(mctx, "$t结果文本", DiceFormatTmpl(mctx, "COC:理智检定_单项结果文本"))

				// 计算扣血
				var reduceSuccess int64
				var reduceFail int64
				var text1 string
				var sanNew int64

				r, _, err = mctx.Dice.ExprEvalBase(expr2, mctx, RollExtraFlags{DisableBlock: true})
				if err == nil {
					reduceSuccess = r.Value.(int64)
				}

				r, _, err = mctx.Dice.ExprEvalBase(expr3, mctx, RollExtraFlags{BigFailDiceOn: successRank == -2, DisableBlock: true})
				if err == nil {
					reduceFail = r.Value.(int64)
				}

				if successRank > 0 {
					sanNew = san - reduceSuccess
					text1 = expr2
				} else {
					sanNew = san - reduceFail
					text1 = expr3
				}

				if sanNew < 0 {
					sanNew = 0
				}

				name := mctx.Player.GetValueNameByAlias("理智", tmpl.Alias)
				VarSetValueInt64(mctx, name, sanNew)

				// 输出结果
				offset := san - sanNew
				VarSetValueInt64(mctx, "$t新值", sanNew)
				VarSetValueStr(mctx, "$t表达式文本", text1)
				VarSetValueInt64(mctx, "$t表达式值", offset)

				var crazyTip string
				if sanNew == 0 {
					crazyTip += DiceFormatTmpl(mctx, "COC:提示_永久疯狂") + "\n"
				} else if offset >= 5 {
					crazyTip += DiceFormatTmpl(mctx, "COC:提示_临时疯狂") + "\n"
				}
				VarSetValueStr(mctx, "$t提示_角色疯狂", crazyTip)

				switch successRank {
				case -2:
					VarSetValueStr(mctx, "$t附加语", DiceFormatTmpl(ctx, "COC:理智检定_附加语_大失败"))
				case -1:
					VarSetValueStr(mctx, "$t附加语", DiceFormatTmpl(ctx, "COC:理智检定_附加语_失败"))
				case 1, 2, 3:
					VarSetValueStr(mctx, "$t附加语", DiceFormatTmpl(ctx, "COC:理智检定_附加语_成功"))
				case 4:
					VarSetValueStr(mctx, "$t附加语", DiceFormatTmpl(ctx, "COC:理智检定_附加语_大成功"))
				default:
					VarSetValueStr(mctx, "$t附加语", "")
				}

				// 指令信息
				commandInfo := map[string]interface{}{
					"cmd":     "sc",
					"rule":    "coc7",
					"pcName":  mctx.Player.Name,
					"cocRule": mctx.Group.CocRuleIndex,
					"items": []interface{}{
						map[string]interface{}{
							"outcome": d100,
							"exprs":   []string{expr1, expr2, expr3},
							"rank":    successRank,
							"sanOld":  san,
							"sanNew":  sanNew,
						},
					},
				}
				ctx.CommandInfo = commandInfo

				text := DiceFormatTmpl(mctx, "COC:理智检定")
				if kw := cmdArgs.GetKwarg("ci"); kw != nil {
					info, err := json.Marshal(ctx.CommandInfo)
					if err == nil {
						text += "\n" + string(info)
					} else {
						text += "\n" + "指令信息无法序列化"
					}
				}

				ReplyToSender(mctx, msg, text)
			}

			if mctx.Player.AutoSetNameTemplate != "" {
				_, _ = SetPlayerGroupCardByTemplate(mctx, mctx.Player.AutoSetNameTemplate)
			}
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}

	cmdCoc := &CmdItemInfo{
		Name:      "coc",
		ShortHelp: ".coc (<数量>) // 制卡指令，返回<数量>组人物属性",
		Help:      "COC制卡指令:\n.coc (<数量>) // 制卡指令，返回<数量>组人物属性",
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			n := cmdArgs.GetArgN(1)
			val, err := strconv.ParseInt(n, 10, 64)
			if err != nil {
				if n == "" {
					val = 1 // 数量不存在时，视为1次
				} else {
					return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
				}
			}
			if val > ctx.Dice.MaxCocCardGen {
				val = ctx.Dice.MaxCocCardGen
			}
			var i int64

			var ss []string
			for i = 0; i < val; i++ {
				result, _, err := self.ExprText(`力量:{$t1=3d6*5} 敏捷:{$t2=3d6*5} 意志:{$t3=3d6*5}\n体质:{$t4=3d6*5} 外貌:{$t5=3d6*5} 教育:{$t6=(2d6+6)*5}\n体型:{$t7=(2d6+6)*5} 智力:{$t8=(2d6+6)*5}\nHP:{($t4+$t7)/10} 幸运:{$t9=3d6*5} [{$t1+$t2+$t3+$t4+$t5+$t6+$t7+$t8}/{$t1+$t2+$t3+$t4+$t5+$t6+$t7+$t8+$t9}]`, ctx)
				if err != nil {
					break
				}
				result = strings.ReplaceAll(result, `\n`, "\n")
				ss = append(ss, result)
			}
			sep := DiceFormatTmpl(ctx, "COC:制卡_分隔符")
			info := strings.Join(ss, sep)
			VarSetValueStr(ctx, "$t制卡结果文本", info)
			text := DiceFormatTmpl(ctx, "COC:制卡")
			// fmt.Sprintf("<%s>的七版COC人物作成:\n%s", ctx.Player.Name, info)
			ReplyToSender(ctx, msg, text)
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}

	theExt := &ExtInfo{
		Name:       "coc7",
		Version:    "1.0.0",
		Brief:      "第七版克苏鲁的呼唤TRPG跑团指令集",
		AutoActive: true,
		Author:     "木落",
		Official:   true,
		ConflictWith: []string{
			"dnd5e",
		},
		OnLoad: func() {

		},
		OnCommandReceived: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) {
			tmpl := getCoc7CharTemplate()
			ctx.Player.TempValueAlias = &tmpl.Alias
		},
		GetDescText: GetExtensionDesc,
		CmdMap: CmdMapCls{
			"en":     cmdEn,
			"setcoc": cmdSetCOC,
			"ti":     cmdTi,
			"li":     cmdLi,
			"ra":     cmdRc,
			"rc":     cmdRc,
			"rch":    cmdRc,
			"rah":    cmdRc,
			"cra":    cmdRc,
			"crc":    cmdRc,
			"crch":   cmdRc,
			"crah":   cmdRc,
			"rav":    cmdRcv,
			"rcv":    cmdRcv,
			"sc":     cmdSc,
			"coc":    cmdCoc,
			"st":     cmdSt,
			"cst":    cmdSt,
		},
	}
	self.RegisterExtension(theExt)
}

func GetResultTextWithRequire(ctx *MsgContext, successRank int, difficultyRequire int, userShortVersion bool) string {
	if difficultyRequire > 1 {
		isSuccess := successRank >= difficultyRequire

		if successRank > difficultyRequire && successRank == 4 {
			// 大成功
			VarSetValueStr(ctx, "$t附加判定结果", fmt.Sprintf("(%s)", GetResultText(ctx, successRank, true)))
		} else if !isSuccess && successRank == -2 {
			// 大失败
			VarSetValueStr(ctx, "$t附加判定结果", fmt.Sprintf("(%s)", GetResultText(ctx, successRank, true)))
		} else {
			VarSetValueStr(ctx, "$t附加判定结果", "")
		}

		var suffix string
		switch difficultyRequire {
		case +2:
			if isSuccess {
				suffix = DiceFormatTmpl(ctx, "COC:判定_必须_困难_成功")
			} else {
				suffix = DiceFormatTmpl(ctx, "COC:判定_必须_困难_失败")
			}
		case +3:
			if isSuccess {
				suffix = DiceFormatTmpl(ctx, "COC:判定_必须_极难_成功")
			} else {
				suffix = DiceFormatTmpl(ctx, "COC:判定_必须_极难_失败")
			}
		case +4:
			if isSuccess {
				suffix = DiceFormatTmpl(ctx, "COC:判定_必须_大成功_成功")
			} else {
				suffix = DiceFormatTmpl(ctx, "COC:判定_必须_大成功_失败")
			}
		}
		return suffix
	}
	return GetResultText(ctx, successRank, userShortVersion)
}

func GetResultText(ctx *MsgContext, successRank int, userShortVersion bool) string {
	var suffix string
	if userShortVersion {
		switch successRank {
		case -2:
			suffix = DiceFormatTmpl(ctx, "COC:判定_简短_大失败")
		case -1:
			suffix = DiceFormatTmpl(ctx, "COC:判定_简短_失败")
		case +1:
			suffix = DiceFormatTmpl(ctx, "COC:判定_简短_成功_普通")
		case +2:
			suffix = DiceFormatTmpl(ctx, "COC:判定_简短_成功_困难")
		case +3:
			suffix = DiceFormatTmpl(ctx, "COC:判定_简短_成功_极难")
		case +4:
			suffix = DiceFormatTmpl(ctx, "COC:判定_简短_大成功")
		}
	} else {
		switch successRank {
		case -2:
			suffix = DiceFormatTmpl(ctx, "COC:判定_大失败")
		case -1:
			suffix = DiceFormatTmpl(ctx, "COC:判定_失败")
		case +1:
			suffix = DiceFormatTmpl(ctx, "COC:判定_成功_普通")
		case +2:
			suffix = DiceFormatTmpl(ctx, "COC:判定_成功_困难")
		case +3:
			suffix = DiceFormatTmpl(ctx, "COC:判定_成功_极难")
		case +4:
			suffix = DiceFormatTmpl(ctx, "COC:判定_大成功")
		}
	}
	return suffix
}

type CocRuleCheckRet struct {
	SuccessRank          int   `jsbind:"successRank"`          // 成功级别
	CriticalSuccessValue int64 `jsbind:"criticalSuccessValue"` // 大成功数值
}

type CocRuleInfo struct {
	Index int    `jsbind:"index"` // 序号
	Key   string `jsbind:"key"`   // .setcoc key
	Name  string `jsbind:"name"`  // 已切换至规则 Name: Desc
	Desc  string `jsbind:"desc"`  // 规则描述

	Check func(ctx *MsgContext, d100 int64, checkValue int64, difficultyRequired int) CocRuleCheckRet `jsbind:"check"`
}

func ResultCheck(ctx *MsgContext, cocRule int, d100 int64, attrValue int64, difficultyRequired int) (successRank int, criticalSuccessValue int64) {
	if cocRule >= 20 {
		d := ctx.Dice
		val, exists := d.CocExtraRules[cocRule]
		if !exists {
			cocRule = 0
		} else {
			ret := val.Check(ctx, d100, attrValue, difficultyRequired)
			return ret.SuccessRank, ret.CriticalSuccessValue
		}
	}
	return ResultCheckBase(cocRule, d100, attrValue, difficultyRequired)
}

/*
大失败：骰出 100。若成功需要的值低于 50，大于等于 96 的结果都是大失败 -> -2
失败：骰出大于角色技能或属性值（但不是大失败） -> -1
常规成功：骰出小于等于角色技能或属性值 -> 1
困难成功：骰出小于等于角色技能或属性值的一半 -> 2
极难成功：骰出小于等于角色技能或属性值的五分之一 -> 3
大成功：骰出1 -> 4
*/
func ResultCheckBase(cocRule int, d100 int64, attrValue int64, difficultyRequired int) (successRank int, criticalSuccessValue int64) {
	criticalSuccessValue = int64(1) // 大成功阈值
	fumbleValue := int64(100)       // 大失败阈值

	checkVal := attrValue
	switch difficultyRequired {
	case 2:
		checkVal /= 2
	case 3:
		checkVal /= 5
	case 4:
		checkVal = criticalSuccessValue
	}

	if d100 <= checkVal {
		successRank = 1
	} else {
		successRank = -1
	}

	// 分支规则设定
	switch cocRule {
	case 0:
		// 规则书规则
		// 不满50出96-100大失败，满50出100大失败
		if checkVal < 50 {
			fumbleValue = 96
		}
	case 1:
		// 不满50出1大成功，满50出1-5大成功
		// 不满50出96-100大失败，满50出100大失败
		if attrValue >= 50 {
			criticalSuccessValue = 5
		}
		if attrValue < 50 {
			fumbleValue = 96
		}
	case 2:
		// 出1-5且<=成功率大成功
		// 出100或出96-99且>成功率大失败
		criticalSuccessValue = 5
		if attrValue < criticalSuccessValue {
			criticalSuccessValue = attrValue
		}
		fumbleValue = 96
		if attrValue >= fumbleValue {
			fumbleValue = attrValue + 1
			if fumbleValue > 100 {
				fumbleValue = 100
			}
		}
	case 3:
		// 出1-5大成功
		// 出100或出96-99大失败
		criticalSuccessValue = 5
		fumbleValue = 96
	case 4:
		// 出1-5且<=成功率/10大成功
		// 不满50出>=96+成功率/10大失败，满50出100大失败
		// 规则4 -> 大成功判定值 = min(5, 判定值/10)，大失败判定值 = min(96+判定值/10, 100)
		criticalSuccessValue = attrValue / 10
		if criticalSuccessValue > 5 {
			criticalSuccessValue = 5
		}
		fumbleValue = 96 + attrValue/10
		if 100 < fumbleValue {
			fumbleValue = 100
		}
	case 5:
		// 出1-2且<成功率/5大成功
		// 不满50出96-100大失败，满50出99-100大失败
		criticalSuccessValue = attrValue / 5
		if criticalSuccessValue > 2 {
			criticalSuccessValue = 2
		}
		if attrValue < 50 {
			fumbleValue = 96
		} else {
			fumbleValue = 99
		}
	case 11: // dg
		criticalSuccessValue = 1
		fumbleValue = 100
	}

	// 成功判定
	if successRank == 1 || d100 <= criticalSuccessValue {
		// 区分大成功、困难成功、极难成功等
		if d100 <= attrValue/2 {
			// suffix = "成功(困难)"
			successRank = 2
		}
		if d100 <= attrValue/5 {
			// suffix = "成功(极难)"
			successRank = 3
		}
		if d100 <= criticalSuccessValue {
			// suffix = "大成功！"
			successRank = 4
		}
	} else if d100 >= fumbleValue {
		// suffix = "大失败！"
		successRank = -2
	}

	if cocRule == 0 || cocRule == 1 || cocRule == 2 {
		if d100 == 1 {
			// 为 1 必是大成功，即使判定线是0
			// 根据群友说法，相关描述见40周年版407页 / 89-90页
			// 保守起见，只在规则0、1、2下生效 [规则1与官方规则相似]
			successRank = 4
		}
	}

	// 默认规则改判，为100必然是大失败
	if d100 == 100 && cocRule == 0 {
		successRank = -2
	}

	// 规则3的改判，强行大成功或大失败
	if cocRule == 3 {
		if d100 <= criticalSuccessValue {
			// suffix = "大成功！"
			successRank = 4
		}
		if d100 >= fumbleValue {
			// suffix = "大失败！"
			successRank = -2
		}
	}

	// 规则DG改判，检定成功基础上，个位十位相同大成功
	// 检定失败基础上，个位十位相同大失败
	if cocRule == 11 {
		numUnits := d100 % 10
		numTens := d100 % 100 / 10
		dgCheck := numUnits == numTens

		if successRank > 0 {
			if dgCheck {
				successRank = 4
			} else {
				successRank = 1 // 抹除困难极难成功
			}
		} else {
			if dgCheck {
				successRank = -2
			} else {
				successRank = -1
			}
		}

		// 23.3 根据dg规则书修正: 为1大成功
		if d100 == 1 {
			successRank = 4
		}
	}

	return successRank, criticalSuccessValue
}
