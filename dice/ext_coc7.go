package dice

import (
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
	"regexp"
	"sort"
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

var difficultPrefixMap = map[string]int{
	"":    1,
	"常规":  1,
	"困难":  2,
	"极难":  3,
	"大成功": 4,
	"困難":  2,
	"極難":  3,
	"常規":  1,
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
	tmpl := getCoc7CharTemplate()
	self.GameSystemTemplateAdd(tmpl)

	helpRc := "" +
		".ra/rc <属性表达式> // 属性检定指令，当前者小于等于后者，检定通过\n" +
		".ra <难度><属性> // 如 .ra 困难侦查\n" +
		".ra b <属性表达式> // 奖励骰或惩罚骰\n" +
		".ra p2 <属性表达式> // 多个奖励骰或惩罚骰\n" +
		".ra 3#p <属性表达式> // 多重检定\n" +
		".ra <属性表达式> (@某人) // 对某人做检定(使用他的属性)\n" +
		".rch/rah // 暗中检定，和检定指令用法相同"

	helpSt := ""
	helpSt += ".st show // 展示个人属性\n"
	helpSt += ".st show <属性1> <属性2> ... // 展示特定的属性数值\n"
	helpSt += ".st show <数字> // 展示高于<数字>的属性，如.st show 30\n"
	helpSt += ".st clr/clear // 清除属性\n"
	helpSt += ".st del <属性1> <属性2> ... // 删除属性，可多项，以空格间隔\n"
	helpSt += ".st help // 帮助\n"
	helpSt += ".st <属性><值> // 例：.st 敏捷50\n"
	helpSt += ".st <属性>±<表达式> // 例：.st 敏捷+1d50，请注意目前+或-要跟在属性后面，不得空格"

	cmdRc := &CmdItemInfo{
		Name:          "rc/ra",
		ShortHelp:     helpRc,
		Help:          "检定指令:\n" + helpRc,
		AllowDelegate: true,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			if len(cmdArgs.Args) >= 1 {
				mctx := GetCtxProxyFirst(ctx, cmdArgs)
				mctx.DelegateText = ctx.DelegateText
				mctx.SystemTemplate = mctx.Group.GetCharTemplate(ctx.Dice)
				restText := cmdArgs.CleanArgs

				reBP := regexp.MustCompile(`^[bBpP]`)
				re2 := regexp.MustCompile(`([^\d]+)\s+([\d]+)`)
				//re2 := regexp.MustCompile(`([^\d])\s+([\d])|([\d])\s+([^\d])`)

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
					difficultRequire := 0
					// 试图读取检定表达式
					swap := false
					r1, detail1, err := mctx.Dice.ExprEvalBase(restText, mctx, RollExtraFlags{
						CocVarNumberMode: true,
						CocDefaultAttrOn: true,
					})

					if err != nil {
						ReplyToSender(mctx, msg, "解析出错: "+restText)
						return &CmdExecuteResult{Matched: true, Solved: true}
					}

					difficultRequire2 := difficultPrefixMap[r1.Parser.CocFlagVarPrefix]
					if difficultRequire2 > difficultRequire {
						difficultRequire = difficultRequire2
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
					})

					if err != nil {
						ReplyToSender(mctx, msg, "解析出错: "+expr2Text)
						return &CmdExecuteResult{Matched: true, Solved: true}
					}

					expr2Text = r2.Matched
					reason = r2.restInput

					difficultRequire2 = difficultPrefixMap[r2.Parser.CocFlagVarPrefix]
					if difficultRequire2 > difficultRequire {
						difficultRequire = difficultRequire2
					}

					if swap {
						r1, detail1, r2, detail2 = r2, detail2, r1, detail1
						expr1Text, expr2Text = expr2Text, expr1Text
					}

					if r1.TypeId != VMTypeInt64 || r2.TypeId != VMTypeInt64 {
						ReplyToSender(mctx, msg, "你输入的表达式并非文本类型")
						return &CmdExecuteResult{Matched: true, Solved: true}
					}

					if r1.Matched == "d100" || r1.Matched == "D100" {
						// 此时没有必要
						detail1 = ""
					}

					var checkVal = r1.Value.(int64)
					var attrVal = r2.Value.(int64)

					successRank, criticalSuccessValue := ResultCheck(mctx, cocRule, checkVal, attrVal)
					var suffix string
					var suffixFull string
					var suffixShort string

					if difficultRequire > 1 {
						// 此时两者内容相同这样做是为了避免失败文本被计算两次
						suffixFull = GetResultTextWithRequire(mctx, successRank, difficultRequire, false)
						suffixShort = suffixFull
					} else {
						suffixFull = GetResultTextWithRequire(mctx, successRank, difficultRequire, false)
						suffixShort = GetResultTextWithRequire(mctx, successRank, difficultRequire, true)
					}

					if manyTimes {
						suffix = suffixShort
					} else {
						suffix = suffixFull
					}

					// 根据难度需求，修改判定值
					switch difficultRequire {
					case 2:
						attrVal /= 2
					case 3:
						attrVal /= 5
					case 4:
						attrVal = criticalSuccessValue
					}
					VarSetValueInt64(mctx, "$tD100", checkVal)
					VarSetValueInt64(mctx, "$t判定值", attrVal)
					VarSetValueStr(mctx, "$t判定结果", suffix)
					VarSetValueInt64(mctx, "$tSuccessRank", int64(successRank))
					VarSetValueStr(mctx, "$t判定结果_详细", suffixFull)
					VarSetValueStr(mctx, "$t判定结果_简短", suffixShort)

					if err == nil {
						detailWrap := ""
						if detail1 != "" {
							detailWrap = ", (" + detail1 + ")"
						}

						// 指令信息标记
						infoItem := map[string]interface{}{
							"expr1":    expr1Text,
							"expr2":    expr2Text,
							"checkVal": checkVal,
							"attrVal":  attrVal,
							"rank":     successRank,
						}
						commandInfoItems = append(commandInfoItems, infoItem)

						VarSetValueStr(mctx, "$t检定表达式文本", expr1Text)
						VarSetValueStr(mctx, "$t属性表达式文本", expr2Text)
						VarSetValueStr(mctx, "$t检定计算过程", detailWrap)
						VarSetValueStr(mctx, "$t计算过程", detailWrap)

						//text := fmt.Sprintf("<%s>的“%s”检定结果为: D100=%d/%d%s %s", ctx.Player.Name, cmdArgs.CleanArgs, d100, cond, detailWrap, suffix)
						SetTempVars(mctx, mctx.Player.Name) // 信息里没有QQ昵称，用这个顶一下
						VarSetValueStr(mctx, "$t结果文本", DiceFormatTmpl(mctx, "COC:检定_单项结果文本"))
					}
					return nil
				}

				var text string
				if cmdArgs.SpecialExecuteTimes > 1 {
					VarSetValueInt64(mctx, "$t次数", int64(cmdArgs.SpecialExecuteTimes))
					if cmdArgs.SpecialExecuteTimes > 12 {
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
						mctx.CommandHideFlag = mctx.Group.GroupId
						ReplyGroup(mctx, msg, DiceFormatTmpl(mctx, "COC:检定_暗中_群内"))
						ReplyPerson(mctx, msg, DiceFormatTmpl(mctx, "COC:检定_暗中_私聊_前缀")+text)
					}
				} else {
					ReplyToSender(mctx, msg, text)
				}
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			ctx.DelegateText = ""
			ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "COC:检定_格式错误"))
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
			help := ""
			for i := 0; i < 6; i++ {
				n := strings.ReplaceAll(SetCocRuleText[i], "\n", " ")
				help += fmt.Sprintf(".setcoc %d // %s\n", i, n)
			}
			// dg
			n := strings.ReplaceAll(SetCocRuleText[11], "\n", " ")
			help += fmt.Sprintf(".setcoc dg // %s\n", n)

			// 自定义
			for _, i := range self.CocExtraRules {
				n := strings.ReplaceAll(i.Desc, "\n", " ")
				help += fmt.Sprintf(".setcoc %d/%s // %s\n", i.Index, i.Key, n)
			}

			if isShort {
				return help
			} else {
				return "设置房规:\n" + help
			}
		},
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			n := cmdArgs.GetArgN(1)
			suffix := ""
			ctx.Group.ExtActive(ctx.Group.ExtGetActive("coc7"))
			suffix = "\nCOC7规则扩展已自动开启"
			ctx.Group.System = "coc7"
			ctx.Group.UpdatedAtTime = time.Now().Unix()

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
							//text := fmt.Sprintf("已切换房规为%s:\n%s%s", i.Name, i.Desc, suffix)
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
				if len(cmdArgs.At) == 1 {
					// 单人
					ctx2, _ = cmdArgs.At[0].CopyCtx(ctx)
				}
				if len(cmdArgs.At) == 2 {
					ctx1, _ = cmdArgs.At[0].CopyCtx(ctx)
					ctx2, _ = cmdArgs.At[1].CopyCtx(ctx)
				}

				restText := cmdArgs.CleanArgs
				var lastMatched string
				readOneVal := func(mctx *MsgContext) (*CmdExecuteResult, int64, string, string) {
					r, _, err := mctx.Dice.ExprEvalBase(restText, mctx, RollExtraFlags{
						CocVarNumberMode: true,
						CocDefaultAttrOn: true,
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

				readOneCheckVal := func(mctx *MsgContext) (*CmdExecuteResult, int64, string) {
					restText = strings.TrimSpace(restText)
					if strings.HasPrefix(restText, ",") || strings.HasPrefix(restText, "，") {
						re := regexp.MustCompile(`[,，](.*)`)
						m := re.FindStringSubmatch(restText)
						restText = m[1]
						r, detail, err := mctx.Dice.ExprEvalBase(restText, mctx, RollExtraFlags{})
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
				ret, checkVal1, rollDetail1 := readOneCheckVal(ctx1)
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
				ret, checkVal2, rollDetail2 := readOneCheckVal(ctx2)
				if ret != nil {
					return *ret
				}

				cocRule := ctx.Group.CocRuleIndex
				if cmdArgs.Command == "rcv" {
					// 强制规则书
					cocRule = 0
				}

				successRank1, _ := ResultCheck(ctx, cocRule, checkVal1, val1)
				difficultRequire1 := difficultPrefixMap[difficult1]
				checkPass1 := successRank1 >= difficultRequire1 // A是否通过检定

				successRank2, _ := ResultCheck(ctx, cocRule, checkVal2, val2)
				difficultRequire2 := difficultPrefixMap[difficult2]
				checkPass2 := successRank2 >= difficultRequire2 // B是否通过检定

				winNum := 0
				if checkPass1 && checkPass2 {
					if successRank1 > successRank2 {
						// A 胜出
						winNum = -1
					} else if successRank1 < successRank2 {
						// B 胜出
						winNum = 1
					} else {
						// 这里状况复杂，属性检定时，属性高的人胜出
						// 攻击时，成功等级相同，视为被攻击者胜出(目标选择闪避)
						// 攻击时，成功等级相同，视为攻击者胜出(目标选择反击)
						// 技能高的人胜出
					}
				} else {
					if !checkPass1 && !checkPass2 {
						// 双方都失败，无事发生
					} else if checkPass1 && !checkPass2 {
						winNum = -1 // A胜
					} else if !checkPass1 && checkPass2 {
						winNum = 1 // B胜
					}
				}

				suffix1 := GetResultTextWithRequire(ctx1, successRank1, difficultRequire1, true)
				suffix2 := GetResultTextWithRequire(ctx2, successRank2, difficultRequire2, true)

				//switch winNum {
				//case -1:
				//	resultText = fmt.Sprintf("<%s>胜出！", ctx1.Player.Name)
				//case +1:
				//	resultText = fmt.Sprintf("<%s>胜出！", ctx2.Player.Name)
				//case 0:
				//	resultText = "平手！(请自行根据场景，如属性比较、攻击对反击，攻击对闪避)做出判断"
				//}

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

				VarSetValueInt64(ctx, "$t玩家A属性", val1)
				VarSetValueInt64(ctx, "$t玩家B属性", val2)

				VarSetValueInt64(ctx, "$t玩家A判定值", checkVal1)
				VarSetValueInt64(ctx, "$t玩家B判定值", checkVal2)

				VarSetValueStr(ctx, "$t玩家A判定过程", rollDetail1)
				VarSetValueStr(ctx, "$t玩家B判定过程", rollDetail2)

				VarSetValueStr(ctx, "$t玩家A判定结果", suffix1)
				VarSetValueStr(ctx, "$t玩家B判定结果", suffix2)

				VarSetValueInt64(ctx, "$tWinFlag", int64(winNum))

				ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "COC:对抗检定"))
				//ReplyToSender(ctx, msg, fmt.Sprintf("对抗检定:\n"+
				//	"<%s> %s-> 属性值:%d 判定值:%d%s %s\n"+
				//	"<%s> %s-> 属性值:%d 判定值:%d%s %s\n%s",
				//	ctx1.Player.Name, expr1, val1, checkVal1, rollDetail1, suffix1,
				//	ctx2.Player.Name, expr2, val2, checkVal2, rollDetail2, suffix2,
				//	resultText))
			}
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}

	cmdSt := &CmdItemInfo{
		Name:          "st",
		ShortHelp:     helpSt,
		Help:          "COC属性设置指令，支持分支指令如下:\n" + helpSt,
		AllowDelegate: true,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			// .st show
			// .st help
			// .st (<Name>[0-9]+)+
			// .st (<Name>)
			// .st (<Name>)+-<表达式>
			if true { // 这里就不改结构了，慌
				mctx := GetCtxProxyFirst(ctx, cmdArgs)
				cmdArgs.ChopPrefixToArgsWith("help", "del", "rm", "show", "list")

				if len(cmdArgs.Args) == 0 || cmdArgs.IsArgEqual(1, "help") {
					return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
				} else if cmdArgs.IsArgEqual(1, "del", "rm") {
					var nums []string
					var failed []string

					doDelete := func(varname string) {
						vars, _ := mctx.ChVarsGet()
						_, ok := vars.Get(varname)
						if ok {
							nums = append(nums, varname)
							vars.Del(varname)
						} else {
							failed = append(failed, varname)
						}
					}

					for _, varname := range cmdArgs.Args[1:] {
						doDelete(varname)
					}
					if len(nums) > 0 {
						mctx.ChVarsUpdateTime()
					}

					//text := fmt.Sprintf("<%s>的如下属性被成功删除:%s，失败%d项\n", p.Name, nums, len(failed))
					VarSetValueStr(mctx, "$t属性列表", strings.Join(nums, " "))
					VarSetValueInt64(mctx, "$t失败数量", int64(len(failed)))
					ReplyToSender(mctx, msg, DiceFormatTmpl(mctx, "COC:属性设置_删除"))
				} else if cmdArgs.IsArgEqual(1, "clr", "clear") {
					num := mctx.ChVarsClear()
					VarSetValueInt64(mctx, "$t数量", int64(num))
					//text := fmt.Sprintf("<%s>的属性数据已经清除，共计%d条", p.Name, num)
					ReplyToSender(mctx, msg, DiceFormatTmpl(mctx, "COC:属性设置_清除"))
				} else if cmdArgs.IsArgEqual(1, "show", "list") {
					info := ""
					p := mctx.Player

					useLimit := false
					usePickItem := false
					limktSkipCount := 0
					var limit int64

					if len(cmdArgs.Args) >= 2 {
						arg2 := cmdArgs.GetArgN(2)
						_limit, err := strconv.ParseInt(arg2, 10, 64)
						if err == nil {
							limit = _limit
							useLimit = true
						} else {
							usePickItem = true
						}
					}

					pickItems := map[string]int{}

					if usePickItem {
						for _, i := range cmdArgs.Args[1:] {
							key := p.GetValueNameByAlias(i, tmpl.Alias)
							pickItems[key] = 1
						}
					}

					tick := 0
					if mctx.ChVarsNumGet() == 0 {
						info = DiceFormatTmpl(mctx, "COC:属性设置_列出_未发现记录")
					} else {
						// 按照配置文件排序
						attrKeys := []string{}
						used := map[string]bool{}
						for _, i := range tmpl.AttrSettings.Top {
							key := p.GetValueNameByAlias(i, tmpl.Alias)
							if used[key] {
								continue
							}
							attrKeys = append(attrKeys, key)
							used[key] = true
						}

						// 其余按字典序
						topNum := len(attrKeys)
						attrKeys2 := []string{}

						vars, _ := mctx.ChVarsGet()
						_ = vars.Iterate(func(_k interface{}, _v interface{}) error {
							attrKeys2 = append(attrKeys2, _k.(string))
							return nil
						})

						sort.Strings(attrKeys2)
						for _, key := range attrKeys2 {
							if used[key] {
								continue
							}
							attrKeys = append(attrKeys, key)
						}

						// 遍历输出
						for index, k := range attrKeys {
							if strings.HasPrefix(k, "$") {
								continue
							}
							var v *VMValue
							_v, exists := vars.Get(k)
							if !exists {
								// 不存在的值，强行补0
								v = &VMValue{TypeId: VMTypeInt64, Value: int64(0)}
							} else {
								v = _v.(*VMValue)
							}

							if index >= topNum {
								if useLimit && v.TypeId == VMTypeInt64 && v.Value.(int64) < limit {
									limktSkipCount += 1
									continue
								}
							}

							if usePickItem {
								_, ok := pickItems[k]
								if !ok {
									continue
								}
							}

							tick += 1
							info += fmt.Sprintf("%s:%s\t", k, v.ToString())
							if tick%4 == 0 {
								info = strings.TrimSpace(info) // 去除末尾空格
								info += fmt.Sprintf("\n")
							}
						}

						if info == "" {
							info = DiceFormatTmpl(mctx, "COC:属性设置_列出_未发现记录")
						}
					}

					if useLimit {
						VarSetValueInt64(mctx, "$t数量", int64(limktSkipCount))
						VarSetValueInt64(mctx, "$t判定值", int64(limit))
						info += DiceFormatTmpl(mctx, "COC:属性设置_列出_隐藏提示")
						//info += fmt.Sprintf("\n注：%d条属性因≤%d被隐藏", limktSkipCount, limit)
					}

					VarSetValueStr(mctx, "$t属性信息", info)
					extra := ReadCardTypeEx(mctx, "coc7")
					ReplyToSender(mctx, msg, DiceFormatTmpl(mctx, "COC:属性设置_列出")+extra)
				} else {
					re1, _ := regexp.Compile(`([^\d\s]+)\s*([+\-＋－])=?(.+)$`)
					m := re1.FindStringSubmatch(cmdArgs.CleanArgs)
					if len(m) > 0 {
						p := mctx.Player

						name := p.GetValueNameByAlias(m[1], tmpl.Alias)
						val, exists := VarGetValueInt64(mctx, name)
						if !exists {
							text := fmt.Sprintf("<%s>: 无法找到名下属性 %s，不能作出修改", p.Name, m[1])
							ReplyToSender(mctx, msg, text)
						} else {
							v, _, err := self.ExprEval(m[2]+m[3], mctx)
							if err == nil && v.TypeId == 0 {
								var newVal int64
								rightVal := v.Value.(int64)
								signText := ""

								if m[2] == "+" || m[2] == "＋" {
									signText = "增加"
									newVal = val + rightVal
								} else {
									signText = "扣除"
									newVal = val + rightVal
								}
								name := p.GetValueNameByAlias(m[1], tmpl.Alias)
								VarSetValueInt64(mctx, name, newVal)

								//text := fmt.Sprintf("<%s>的“%s”变化: %d ➯ %d (%s%s=%d)\n", p.Name, m[1], val, newVal, signText, m[3], rightVal)
								VarSetValueStr(mctx, "$t属性", m[1])
								VarSetValueInt64(mctx, "$t旧值", val)
								VarSetValueInt64(mctx, "$t新值", newVal)
								VarSetValueStr(mctx, "$t增加或扣除", signText)
								VarSetValueStr(mctx, "$t表达式文本", v.Matched)
								VarSetValueInt64(mctx, "$t变化量", rightVal)

								VarSetValueStr(mctx, "$t当前绑定角色", mctx.ChBindCurGet())
								text := DiceFormatTmpl(mctx, "COC:属性设置_增减")

								// 指令信息
								commandInfo := map[string]interface{}{
									"cmd":    "st",
									"rule":   "coc7",
									"pcName": mctx.Player.Name,
									"items": []interface{}{
										map[string]interface{}{
											"type":    "mod",
											"attr":    m[1],
											"modExpr": v.Matched,
											"valOld":  val,
											"valNew":  newVal,
											"isInc":   signText == "增加", // 增加还是扣除
										},
									},
								}
								ctx.CommandInfo = commandInfo

								if kw := cmdArgs.GetKwarg("ci"); kw != nil {
									info, err := json.Marshal(ctx.CommandInfo)
									if err == nil {
										text += "\n" + string(info)
									} else {
										text += "\n" + "指令信息无法序列化"
									}
								}

								ReplyToSender(mctx, msg, text)
							} else {
								VarSetValueStr(mctx, "$t表达式文本", m[3])
								//text := fmt.Sprintf("<%s>: 错误的增减值: %s", p.Name, m[3])
								text := DiceFormatTmpl(mctx, "COC:属性设置_增减_错误的值")
								ReplyToSender(mctx, msg, text)
							}
						}
					} else {
						valueMap := map[string]int64{}
						re, _ := regexp.Compile(`([^\d]+?)[:=：＝]?(\d+)`)

						// 读取所有参数中的值
						stText := cmdArgs.CleanArgs

						m := re.FindAllStringSubmatch(RemoveSpace(stText), -1)

						for _, i := range m {
							num, err := strconv.ParseInt(i[2], 10, 64)
							if err == nil {
								valueMap[i[1]] = num
							}
						}

						for _, v := range cmdArgs.Kwargs {
							vint, err := strconv.ParseInt(v.Value, 10, 64)
							if err == nil {
								valueMap[v.Name] = vint
							}
						}

						nameMap := map[string]bool{}
						synonymsCount := 0
						p := mctx.Player

						for k, v := range valueMap {
							name := p.GetValueNameByAlias(k, tmpl.Alias)
							nameMap[name] = true
							if k != name {
								synonymsCount += 1
							}

							VarSetValueInt64(mctx, name, v)
						}

						if len(m) == 0 {
							ReplyToSender(mctx, msg, "无法识别的属性: "+stText)
							return CmdExecuteResult{Matched: true, Solved: true}
						}

						//s, _ := json.Marshal(valueMap)
						VarSetValueInt64(mctx, "$t数量", int64(len(valueMap)))
						VarSetValueInt64(mctx, "$t有效数量", int64(len(nameMap)))
						VarSetValueInt64(mctx, "$t同义词数量", int64(synonymsCount))
						text := DiceFormatTmpl(mctx, "COC:属性设置")
						SetCardType(mctx, "coc7")
						//text := fmt.Sprintf("<%s>的属性录入完成，本次共记录了%d条数据 (其中%d条为同义词)", p.Name, len(valueMap), synonymsCount)
						ReplyToSender(mctx, msg, text)
					}
				}

				if ctx.Player.AutoSetNameTemplate != "" {
					_, _ = SetPlayerGroupCardByTemplate(ctx, ctx.Player.AutoSetNameTemplate)
				}
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			return CmdExecuteResult{Matched: true, Solved: false}
		},
	}

	helpEn := `.en <技能名称>(技能点数) (+(<失败成长值>/)<成功成长值>) // 整体格式，可以直接看下面几个分解格式
.en <技能名称> // 骰D100，若点数大于当前值，属性成长1d10
.en <技能名称>(技能点数) // 骰D100，若点数大于技能点数，属性=技能点数+1d10
.en <技能名称>(技能点数) +<成功成长值> // 骰D100，若点数大于当前值，属性成长成功成长值点
.en <技能名称>(技能点数) +<失败成长值>/<成功成长值> // 骰D100，若点数大于当前值，属性成长成功成长值点，否则增加失败`

	cmdEn := &CmdItemInfo{
		Name:          "en",
		ShortHelp:     helpEn,
		Help:          "成长指令:\n" + helpEn,
		AllowDelegate: false,
		Solve: func(mctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			// 首先处理单参数形式
			// .en [技能名称]([技能值])+(([失败成长值]/)[成功成长值])
			re := regexp.MustCompile(`([a-zA-Z_\p{Han}]+)\s*(\d+)?\s*(\+(([^/]+)/)?\s*(.+))?`)
			m := re.FindStringSubmatch(cmdArgs.CleanArgs)

			if m != nil {
				varName := m[1]     // 技能名称
				varValueStr := m[2] // 技能值 - 字符串
				successExpr := m[6] // 成功的加值表达式
				failExpr := m[5]    // 失败的加值表达式

				var varValue int64
				VarSetValueStr(mctx, "$t技能", varName)

				// 首先，试图读取技能的值
				if varValueStr != "" {
					varValue, _ = strconv.ParseInt(varValueStr, 10, 64)
				} else {
					val, exists := VarGetValue(mctx, varName)
					if !exists {
						ReplyToSender(mctx, msg, DiceFormatTmpl(mctx, "COC:技能成长_属性未录入"))
						return CmdExecuteResult{Matched: true, Solved: false}
					}
					if val.TypeId != VMTypeInt64 {
						ReplyToSender(mctx, msg, DiceFormatTmpl(mctx, "COC:技能成长_错误的属性类型"))
						return CmdExecuteResult{Matched: true, Solved: false}
					}
					varValue = val.Value.(int64)
				}

				d100 := DiceRoll64(100)
				// 注意一下，这里其实是，小于失败 大于成功
				successRank, _ := ResultCheck(mctx, mctx.Group.CocRuleIndex, d100, varValue)
				var resultText string
				if successRank > 0 {
					resultText = "失败"
				} else {
					resultText = "成功"
				}

				VarSetValueInt64(mctx, "$tD100", d100)
				VarSetValueInt64(mctx, "$t判定值", varValue)
				VarSetValueStr(mctx, "$t判定结果", resultText)
				VarSetValueStr(mctx, "$t判定结果", resultText)
				VarSetValueInt64(mctx, "$tSuccessRank", int64(successRank))

				if successRank < 0 {
					// 如果成功
					if successExpr == "" {
						successExpr = "1d10"
					}

					r, _, err := mctx.Dice.ExprEval(successExpr, mctx)
					VarSetValueStr(mctx, "$t表达式文本", successExpr)
					if err != nil {
						ReplyToSender(mctx, msg, DiceFormatTmpl(mctx, "COC:技能成长_错误的成功成长值"))
						return CmdExecuteResult{Matched: true, Solved: false}
					}

					VarSetValueInt64(mctx, "$t旧值", varValue)
					varValue += r.VMValue.Value.(int64)
					nv := &VMValue{TypeId: VMTypeInt64, Value: varValue}

					VarSetValue(mctx, "$t增量", &r.VMValue)
					VarSetValue(mctx, "$t新值", nv)
					VarSetValue(mctx, varName, nv)

					VarSetValueStr(mctx, "$t当前绑定角色", mctx.ChBindCurGet())
					VarSetValueStr(mctx, "$t结果文本", DiceFormatTmpl(mctx, "COC:技能成长_结果_成功"))
				} else {
					// 如果失败
					if failExpr == "" {
						VarSetValueStr(mctx, "$t结果文本", DiceFormatTmpl(mctx, "COC:技能成长_结果_失败"))
					} else {
						r, _, err := mctx.Dice.ExprEval(failExpr, mctx)
						VarSetValueStr(mctx, "$t表达式文本", failExpr)
						if err != nil {
							ReplyToSender(mctx, msg, DiceFormatTmpl(mctx, "COC:技能成长_错误的失败成长值"))
							return CmdExecuteResult{Matched: true, Solved: false}
						}

						VarSetValueInt64(mctx, "$t旧值", varValue)
						varValue += r.VMValue.Value.(int64)
						nv := &VMValue{TypeId: VMTypeInt64, Value: varValue}

						VarSetValue(mctx, "$t增量", &r.VMValue)
						VarSetValue(mctx, "$t新值", nv)
						VarSetValue(mctx, varName, nv)

						VarSetValueStr(mctx, "$t当前绑定角色", mctx.ChBindCurGet())
						VarSetValueStr(mctx, "$t结果文本", DiceFormatTmpl(mctx, "COC:技能成长_结果_失败变更"))
					}

				}

				if mctx.Player.AutoSetNameTemplate != "" {
					_, _ = SetPlayerGroupCardByTemplate(mctx, mctx.Player.AutoSetNameTemplate)
				}
				ReplyToSender(mctx, msg, DiceFormatTmpl(mctx, "COC:技能成长"))
			} else {
				ReplyToSender(mctx, msg, "指令格式不匹配")
			}
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}

	cmdTi := &CmdItemInfo{
		Name:      "ti",
		ShortHelp: ".ti // 抽取一个临时性疯狂症状",
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			// 临时性疯狂
			foo := func(tmpl string) string {
				val, _, _ := self.ExprText(tmpl, ctx)
				return val
			}

			num := DiceRoll(10)
			text := fmt.Sprintf("<%s>的疯狂发作-即时症状:\n1D10=%d\n", ctx.Player.Name, num)

			switch num {
			case 1:
				text += foo("失忆：调查员会发现自己只记得最后身处的安全地点，却没有任何来到这里的记忆。例如，调查员前一刻还在家中吃着早饭，下一刻就已经直面着不知名的怪物。这将会持续 1D10={1d10} 轮。")
			case 2:
				text += foo("假性残疾：调查员陷入了心理性的失明，失聪以及躯体缺失感中，持续 1D10={1d10} 轮。")
			case 3:
				text += foo("暴力倾向：调查员陷入了六亲不认的暴力行为中，对周围的敌人与友方进行着无差别的攻击，持续 1D10={1d10} 轮。")
			case 4:
				text += foo("偏执：调查员陷入了严重的偏执妄想之中。有人在暗中窥视着他们，同伴中有人背叛了他们，没有人可以信任，万事皆虚。持续 1D10={1d10} 轮")
			case 5:
				text += foo("人际依赖：守秘人适当参考调查员的背景中重要之人的条目，调查员因为一些原因而降他人误认为了他重要的人并且努力的会与那个人保持那种关系，持续 1D10={1d10} 轮")
			case 6:
				text += foo("昏厥：调查员当场昏倒，并需要 1D10={1d10} 轮才能苏醒。")
			case 7:
				text += foo("逃避行为：调查员会用任何的手段试图逃离现在所处的位置，即使这意味着开走唯一一辆交通工具并将其它人抛诸脑后，调查员会试图逃离 1D10={1d10} 轮。")
			case 8:
				text += foo("竭嘶底里：调查员表现出大笑，哭泣，嘶吼，害怕等的极端情绪表现，持续 1D10={1d10} 轮。")
			case 9:
				text += foo("恐惧：调查员通过一次 D100 或者由守秘人选择，来从恐惧症状表中选择一个恐惧源，就算这一恐惧的事物是并不存在的，调查员的症状会持续 1D10={1d10} 轮。")
				num2 := DiceRoll(100)
				text += fmt.Sprintf("\n1D100=%d\n", num2)
				text += fearMap[num2]
			case 10:
				text += foo("躁狂：调查员通过一次 D100 或者由守秘人选择，来从躁狂症状表中选择一个躁狂的诱因，这个症状将会持续 1D10={1d10} 轮。")
				num2 := DiceRoll(100)
				text += fmt.Sprintf("\n1D100=%d\n", num2)
				text += maniaMap[num2]
			}

			ReplyToSender(ctx, msg, text)
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}

	cmdLi := &CmdItemInfo{
		Name:      "li",
		ShortHelp: ".li // 抽取一个总结性疯狂症状",
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			// 总结性疯狂
			foo := func(tmpl string) string {
				val, _, _ := self.ExprText(tmpl, ctx)
				return val
			}

			num := DiceRoll(10)
			text := fmt.Sprintf("<%s>的疯狂发作-总结症状:\n1D10=%d\n", ctx.Player.Name, num)

			switch num {
			case 1:
				text += foo("失忆：回过神来，调查员们发现自己身处一个陌生的地方，并忘记了自己是谁。记忆会随时间恢复。")
			case 2:
				text += foo("被窃：调查员在 1D10={1d10} 小时后恢复清醒，发觉自己被盗，身体毫发无损。如果调查员携带着宝贵之物（见调查员背景），做幸运检定来决定其是否被盗。所有有价值的东西无需检定自动消失。")
			case 3:
				text += foo("遍体鳞伤：调查员在 1D10={1d10} 小时后恢复清醒，发现自己身上满是拳痕和瘀伤。生命值减少到疯狂前的一半，但这不会造成重伤。调查员没有被窃。这种伤害如何持续到现在由守秘人决定。")
			case 4:
				text += foo("暴力倾向：调查员陷入强烈的暴力与破坏欲之中。调查员回过神来可能会理解自己做了什么也可能毫无印象。调查员对谁或何物施以暴力，他们是杀人还是仅仅造成了伤害，由守秘人决定。")
			case 5:
				text += foo("极端信念：查看调查员背景中的思想信念，调查员会采取极端和疯狂的表现手段展示他们的思想信念之一。比如一个信教者会在地铁上高声布道。")
			case 6:
				text += foo("重要之人：考虑调查员背景中的重要之人，及其重要的原因。在 1D10={1d10} 小时或更久的时间中，调查员将不顾一切地接近那个人，并为他们之间的关系做出行动。")
			case 7:
				text += foo("被收容：调查员在精神病院病房或警察局牢房中回过神来，他们可能会慢慢回想起导致自己被关在这里的事情。")
			case 8:
				text += foo("逃避行为：调查员恢复清醒时发现自己在很远的地方，也许迷失在荒郊野岭，或是在驶向远方的列车或长途汽车上。")
			case 9:
				text += foo("恐惧：调查员患上一个新的恐惧症状。在恐惧症状表上骰 1 个 D100 来决定症状，或由守秘人选择一个。调查员在 1D10={1d10} 小时后回过神来，并开始为避开恐惧源而采取任何措施。")
				num2 := DiceRoll(100)
				text += fmt.Sprintf("\n1D100=%d\n", num2)
				text += fearMap[num2]
			case 10:
				text += foo("狂躁：调查员患上一个新的狂躁症状。在狂躁症状表上骰 1 个 d100 来决定症状，或由守秘人选择一个。调查员会在 1D10={1d10} 小时后恢复理智。在这次疯狂发作中，调查员将完全沉浸于其新的狂躁症状。这症状是否会表现给旁人则取决于守秘人和此调查员。")
				num2 := DiceRoll(100)
				text += fmt.Sprintf("\n1D100=%d\n", num2)
				text += maniaMap[num2]
			}

			ReplyToSender(ctx, msg, text)
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}

	cmdSc := &CmdItemInfo{
		Name: "sc",
		ShortHelp: ".sc <成功时掉san>/<失败时掉san> // 对理智进行一次D100检定，根据结果扣除理智\n" +
			".sc <失败时掉san> //同上，简易写法 \n" +
			".sc (b/p) (<成功时掉san>/)<失败时掉san> // 加上奖惩骰",
		//".sc <成功掉san>/<失败掉san> (,<成功掉san>/<失败掉san>)+",
		AllowDelegate: true,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			// http://www.antagonistes.com/files/CoC%20CheatSheet.pdf
			// v2: (worst) FAIL — REGULAR SUCCESS — HARD SUCCESS — EXTREME SUCCESS (best)

			if len(cmdArgs.Args) == 0 {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}
			mctx := GetCtxProxyFirst(ctx, cmdArgs)
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
					r, _, err := mctx.Dice.ExprEvalBase(argText, mctx, RollExtraFlags{IgnoreDiv0: true})
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
					r2, _, err := mctx.Dice.ExprEvalBase(r.restInput, mctx, RollExtraFlags{})
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
			//fmt.Println("???", expr1, "|", expr2, "|", expr3, "x", code)

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
				if err == nil && rCond.TypeId == VMTypeInt64 {
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
				if err == nil && r.TypeId == VMTypeInt64 {
					san = r.Value.(int64)
				}
				_san, err := strconv.ParseInt(argText, 10, 64)
				if err == nil {
					san = _san
				}

				// 进行检定
				successRank, _ := ResultCheck(ctx, mctx.Group.CocRuleIndex, d100, san)
				suffix := GetResultText(ctx, successRank, false)
				suffixShort := GetResultText(ctx, successRank, true)

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

				text1 = expr2 + "/" + expr3

				r, _, err = mctx.Dice.ExprEvalBase(expr2, mctx, RollExtraFlags{})
				if err == nil {
					reduceSuccess = r.Value.(int64)
				}

				r, _, err = mctx.Dice.ExprEvalBase(expr3, mctx, RollExtraFlags{BigFailDiceOn: successRank == -2})
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

				name := ctx.Player.GetValueNameByAlias("理智", tmpl.Alias)
				VarSetValueInt64(ctx, name, sanNew)

				//输出结果
				offset := san - sanNew
				VarSetValueInt64(mctx, "$t新值", sanNew)
				VarSetValueStr(mctx, "$t表达式文本", text1)
				VarSetValueInt64(mctx, "$t表达式值", offset)
				//text := fmt.Sprintf("<%s>的理智检定:\nD100=%d/%d %s\n理智变化: %d ➯ %d (扣除%s=%d点)\n", ctx.Player.Name, d100, san, suffix, san, sanNew, text1, offset)

				var crazyTip string
				if sanNew == 0 {
					crazyTip += DiceFormatTmpl(mctx, "COC:提示_永久疯狂") + "\n"
				} else {
					if offset >= 5 {
						crazyTip += DiceFormatTmpl(mctx, "COC:提示_临时疯狂") + "\n"
					}
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
							"checkVal": d100,
							"exprs":    []string{expr1, expr2, expr3},
							"rank":     successRank,
							"sanOld":   san,
							"sanNew":   sanNew,
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

			if ctx.Player.AutoSetNameTemplate != "" {
				_, _ = SetPlayerGroupCardByTemplate(ctx, ctx.Player.AutoSetNameTemplate)
			}
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}

	cmdCoc := &CmdItemInfo{
		Name:      "coc",
		ShortHelp: ".coc (<数量>) // 制卡指令，返回<数量>组人物属性",
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
		ConflictWith: []string{
			"dnd5e",
		},
		OnLoad: func() {

		},
		OnCommandReceived: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) {
			ctx.Player.TempValueAlias = &tmpl.Alias
		},
		GetDescText: func(ei *ExtInfo) string {
			return GetExtensionDesc(ei)
		},
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

func GetResultTextWithRequire(ctx *MsgContext, successRank int, difficultRequire int, userShortVersion bool) string {
	if difficultRequire > 1 {
		isSuccess := successRank >= difficultRequire
		var suffix string
		switch difficultRequire {
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
	} else {
		return GetResultText(ctx, successRank, userShortVersion)
	}
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

	Check func(ctx *MsgContext, d100 int64, checkValue int64) CocRuleCheckRet `jsbind:"check"`
}

func ResultCheck(ctx *MsgContext, cocRule int, d100 int64, checkValue int64) (successRank int, criticalSuccessValue int64) {
	if cocRule >= 20 {
		d := ctx.Dice
		val, exists := d.CocExtraRules[cocRule]
		if !exists {
			cocRule = 0
		} else {
			ret := val.Check(ctx, d100, checkValue)
			return ret.SuccessRank, ret.CriticalSuccessValue
		}
	}
	return ResultCheckBase(cocRule, d100, checkValue)
}

/*
大失败：骰出 100。若成功需要的值低于 50，大于等于 96 的结果都是大失败 -> -2
失败：骰出大于角色技能或属性值（但不是大失败） -> -1
常规成功：骰出小于等于角色技能或属性值 -> 1
困难成功：骰出小于等于角色技能或属性值的一半 -> 2
极难成功：骰出小于等于角色技能或属性值的五分之一 -> 3
大成功：骰出1 -> 4
*/
func ResultCheckBase(cocRule int, d100 int64, checkValue int64) (successRank int, criticalSuccessValue int64) {
	if d100 <= checkValue {
		successRank = 1
	} else {
		successRank = -1
	}

	criticalSuccessValue = int64(1) // 大成功阈值
	fumbleValue := int64(100)       // 大失败阈值

	// 分支规则设定
	switch cocRule {
	case 0:
		// 规则书规则
		//不满50出96-100大失败，满50出100大失败
		if checkValue < 50 {
			fumbleValue = 96
		}
	case 1:
		//不满50出1大成功，满50出1-5大成功
		//不满50出96-100大失败，满50出100大失败
		if checkValue >= 50 {
			criticalSuccessValue = 5
		}
		if checkValue < 50 {
			fumbleValue = 96
		}
	case 2:
		//出1-5且<=成功率大成功
		//出100或出96-99且>成功率大失败
		criticalSuccessValue = 5
		fumbleValue = 96
	case 3:
		//出1-5大成功
		//出100或出96-99大失败
		criticalSuccessValue = 5
		fumbleValue = 96
	case 4:
		//出1-5且<=成功率/10大成功
		//不满50出>=96+成功率/10大失败，满50出100大失败
		//规则4 -> 大成功判定值 = min(5, 判定值/10)，大失败判定值 = min(96+判定值/10, 100)
		criticalSuccessValue = checkValue / 10
		if criticalSuccessValue > 5 {
			criticalSuccessValue = 5
		}
		fumbleValue = 96 + checkValue/10
		if 100 < fumbleValue {
			fumbleValue = 100
		}
	case 5:
		//出1-2且<成功率/5大成功
		//不满50出96-100大失败，满50出99-100大失败
		criticalSuccessValue = checkValue / 5
		if criticalSuccessValue > 2 {
			criticalSuccessValue = 2
		}
		if checkValue < 50 {
			fumbleValue = 96
		} else {
			fumbleValue = 99
		}
	case 11: //dg
		criticalSuccessValue = 1
		fumbleValue = 100
	}

	// 成功判定
	if successRank == 1 {
		// 区分大成功、困难成功、极难成功等
		if d100 <= checkValue/2 {
			//suffix = "成功(困难)"
			successRank = 2
		}
		if d100 <= checkValue/5 {
			//suffix = "成功(极难)"
			successRank = 3
		}
		if d100 <= criticalSuccessValue {
			//suffix = "大成功！"
			successRank = 4
		}
	} else {
		if d100 >= fumbleValue {
			//suffix = "大失败！"
			successRank = -2
		}
	}

	// 默认规则改判，为100必然是大失败
	if d100 == 100 && cocRule == 0 {
		successRank = -2
	}

	// 规则3的改判，强行大成功或大失败
	if cocRule == 3 {
		if d100 <= criticalSuccessValue {
			//suffix = "大成功！"
			successRank = 4
		}
		if d100 >= fumbleValue {
			//suffix = "大失败！"
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
	}

	return successRank, criticalSuccessValue
}

func setupConfig(d *Dice) AttributeConfigs {
	attrConfigFn := d.GetExtConfigFilePath("coc7", "attribute.yaml")

	if _, err := os.Stat(attrConfigFn); err == nil && false {
		// 如果文件存在，那么读取
		ac := AttributeConfigs{}
		af, err := ioutil.ReadFile(attrConfigFn)
		if err == nil {
			err = yaml.Unmarshal(af, &ac)
			if err != nil {
				panic(err)
			}
		}
		return ac
	} else {
		// 如果不存在，新建

		defaultVals := AttributeConfigs{
			Alias: map[string][]string{
				"理智": {"san", "san值", "理智值", "理智点数", "心智", "心智点数", "心智點數", "理智點數"},
				"力量": {"str"},
				"体质": {"con", "體質"},
				"体型": {"siz", "體型", "体形", "體形"},
				"敏捷": {"dex"},
				"外貌": {"app", "外表"},
				"意志": {"pow"},
				"教育": {"edu", "知识", "知識"}, // 教育和知识等值而不是一回事，注意
				"智力": {"int", "灵感", "靈感"}, // 智力和灵感等值而不是一回事，注意

				"幸运":     {"luck", "幸运值", "运气", "幸運", "運氣", "幸運值"},
				"生命值":    {"hp", "生命", "体力", "體力", "血量", "耐久值"},
				"生命值上限":  {"hpmax", "生命上限", "体力上限", "體力上限", "血量上限", "耐久值上限"},
				"魔法值":    {"mp", "魔法", "魔力", "魔力值"},
				"护甲":     {"装甲", "護甲", "裝甲"},
				"枪械":     {"火器", "射击", "槍械", "射擊"},
				"会计":     {"會計"},
				"人类学":    {"人類學"},
				"估价":     {"估價"},
				"考古学":    {"考古學"},
				"魅惑":     {"取悦", "取悅"},
				"攀爬":     {"攀岩", "攀登"},
				"计算机使用":  {"电脑使用", "計算機使用", "電腦使用", "计算机", "电脑", "計算機", "電腦"},
				"信用评级":   {"信誉", "信用", "信誉度", "cr", "信用評級", "信譽", "信譽度"},
				"克苏鲁神话":  {"cm", "克苏鲁", "克苏鲁神话知识", "克蘇魯", "克蘇魯神話", "克蘇魯神話知識"},
				"乔装":     {"喬裝"},
				"闪避":     {"閃避"},
				"汽车驾驶":   {"汽車駕駛", "汽车", "驾驶", "汽車", "駕駛"},
				"电气维修":   {"电器维修", "电工", "電氣維修", "電器維修", "電工"},
				"电子学":    {"電子學"},
				"话术":     {"快速交谈", "話術", "快速交談"},
				"历史":     {"歷史"},
				"恐吓":     {"恐嚇"},
				"跳跃":     {"跳躍"},
				"母语":     {"母語"},
				"图书馆使用":  {"圖書館使用", "图书馆", "图书馆利用", "圖書館", "圖書館利用"},
				"聆听":     {"聆聽"},
				"锁匠":     {"开锁", "撬锁", "钳工", "鎖匠", "鉗工", "開鎖", "撬鎖"},
				"机械维修":   {"机器维修", "机修", "機器維修", "機修", "機械維修"},
				"医学":     {"醫學"},
				"博物学":    {"自然", "自然学", "自然史", "自然學", "博物學"},
				"领航":     {"导航", "領航", "導航"},
				"神秘学":    {"神秘學"},
				"操作重型机械": {"重型操作", "重型机械", "重型", "重机", "操作重型機械", "重型機械", "重機"},
				"说服":     {"辩论", "议价", "演讲", "說服", "辯論", "議價", "演講"},
				"精神分析":   {"心理分析"},
				"心理学":    {"心理學"},
				"骑术":     {"騎術"},
				"妙手":     {"藏匿", "盗窃", "盜竊"},
				"侦查":     {"侦察", "偵查", "偵察"},
				"潜行":     {"躲藏", "潛行"},
				"投掷":     {"投擲"},
				"追踪":     {"跟踪", "追蹤", "跟蹤"},
				"驯兽":     {"驯养", "动物驯养", "馴獸", "動物馴養", "馴養"},
				"读唇":     {"唇语", "讀唇", "唇語"},
				"炮术":     {"炮術"},
				"学识":     {"学问", "學識", "學問"},
				"艺术与手艺":  {"艺术和手艺", "艺术", "手艺", "工艺", "技艺", "藝術與手藝", "藝術和手藝", "藝術", "手藝", "工藝", "技藝"},
				"美术":     {"美術"},
				"伪造":     {"偽造"},
				"摄影":     {"攝影"},
				"理发":     {"理髮"},
				"书法":     {"書法"},
				"木匠":     {"木工"},
				"厨艺":     {"烹饪", "廚藝", "烹飪"},
				"写作":     {"文学", "寫作", "文學"},
				"歌剧歌唱":   {"歌劇歌唱"},
				"技术制图":   {"技術製圖"},
				"裁缝":     {"裁縫"},
				"声乐":     {"聲樂"},
				"喜剧":     {"喜劇"},
				"器乐":     {"器樂"},
				"速记":     {"速記"},
				"园艺":     {"園藝"},
				"斗殴":     {"鬥毆"},
				"剑":      {"剑术", "劍", "劍術"},
				"斧":      {"斧头", "斧子", "斧頭"},
				"链锯":     {"电锯", "油锯", "鏈鋸", "電鋸", "油鋸"},
				"链枷":     {"连枷", "連枷", "鏈枷"},
				"绞索":     {"绞具", "絞索", "絞具"},
				"手枪":     {"手槍"},
				"步枪":     {"霰弹枪", "步霰", "步枪/霰弹枪", "散弹枪", "步槍", "霰彈槍", "步霰", "步槍/霰彈槍", "散彈槍"},
				"弓":      {"弓术", "弓箭", "弓術"},
				"火焰喷射器":  {"火焰噴射器"},
				"机枪":     {"機槍"},
				"矛":      {"投矛"},
				"冲锋枪":    {"衝鋒槍"},
				"天文学":    {"天文學"},
				"生物学":    {"生物學"},
				"植物学":    {"植物學"},
				"化学":     {"化學"},
				"密码学":    {"密碼學"},
				"工程学":    {"工程學"},
				"司法科学":   {"司法科學"},
				"地质学":    {"地理学", "地質學", "地理學"},
				"数学":     {"數學"},
				"气象学":    {"氣象學"},
				"药学":     {"藥學"},
				"物理学":    {"物理", "物理學"},
				"动物学":    {"動物學"},
				"船":      {"开船", "驾驶船", "開船", "駕駛船"},
				"飞行器":    {"开飞行器", "驾驶飞行器", "飛行器", "開飛行器", "駕駛飛行器"},
				"科学":     {"科學"},
				"海洋":     {"海上"},
				"极地":     {"極地"},
				"语言":     {"外语", "語言", "外語"},
				"潜水":     {"潛水"},
			},
			Order: AttributeOrder{
				Top:    []string{"力量", "敏捷", "体质", "体型", "外貌", "智力", "意志", "教育", "理智", "克苏鲁神话", "生命值", "魔法值"},
				Others: AttributeOrderOthers{SortBy: "Name"},
			},
		}

		buf, err := yaml.Marshal(defaultVals)
		if err != nil {
			fmt.Println(err)
		} else {
			_ = os.WriteFile(attrConfigFn, buf, 0644)
		}
		return defaultVals
	}
}

// 一个sc的废案，当时没考虑到除号问题
//	read2n3 := func(r0 *VmResult) int {
//		if strings.HasPrefix(r0.restInput, "/") {
//			// 当前值为成功值
//			expr2 = r0.Matched
//
//			// 匹配失败值，必须匹配
//			r, _, err := ctx.Dice.ExprEvalBase(r0.restInput[1:], mctx, RollExtraFlags{})
//			if err != nil {
//				return 2
//			}
//
//			expr3 = r.Matched
//			argText = r.restInput
//			return 0
//		}
//
//		fmt.Println("333 rest", r0.restInput, len(r0.restInput), r0.restInput == "")
//		if strings.HasPrefix(r0.restInput, ",") || r0.restInput == "" {
//			expr2 = defaultSuccessExpr
//			expr3 = r.Matched
//			argText = r.restInput
//			return 0
//		}
//
//		return -1
//	}
//
//	code := read2n3(r)
//	if code == -1 {
//		// 读取到表达式，所以r是判定值
//		r2, _, err := ctx.Dice.ExprEvalBase(r.restInput, mctx, RollExtraFlags{})
//		if err != nil {
//			// 情况3，格式错误
//			return 3
//		}
//		expr1 = r.Matched
//
//		// 读取剩下两个值
//		return read2n3(r2)
//	}
//
//	return code
//}
