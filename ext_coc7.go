package main

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"regexp"
	"sort"
	"strconv"
	"time"
)

var fearListText string = `
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

var ManiaListText string = `
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

func (self *Dice) registerBuiltinExtCoc7() {
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

	ac := AttributeConfigs{}
	af, err := ioutil.ReadFile(CONFIG_ATTRIBUTE_FILE)
	if err == nil {
		err = yaml.Unmarshal(af, &ac)
		if err != nil {
			panic(err)
		}
	}

	self.extList = append(self.extList, &ExtInfo{
		Name: "coc7",
		version: "0.0.1",
		Brief: "第七版克苏鲁的呼唤TRPG跑团扩展指令集",
		autoActive: true,
		Author: "木落",
		OnPrepare: func (session *IMSession, msg *Message, cmdArgs *CmdArgs) {
			p := getPlayerInfoBySender(session, msg)
			p.TempValueAlias = &ac.Alias;
		},
		GetDescText: func (ei *ExtInfo) string {
			text := "> " + ei.Brief + "\n" + "提供命令:\n"
			keys := make([]string, 0, len(ei.cmdMap))
			for k := range ei.cmdMap {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			for _, i := range keys {
				i := ei.cmdMap[i]
				brief := i.Brief
				if brief != "" {
					brief = " // " + brief
				}
				text += "." + i.name + brief + "\n"
			}

			return text
		},
		cmdMap: CmdMapCls{
			"ti": &CmdItemInfo{
				name: "ti",
				Brief: "随机抽取一个临时性疯狂症状",
				solve: func(session *IMSession, msg *Message, cmdArgs *CmdArgs) struct{ success bool } {
					// 临时性疯狂
					if isCurGroupBotOn(session, msg) {
						p := getPlayerInfoBySender(session, msg)
						foo := func(tmpl string) string {
							val, _, _ := self.exprText(tmpl, p)
							return val
						}

						num := DiceRoll(10)
						text := fmt.Sprintf("<%s>的疯狂发作-即时症状:\n1D10=%d\n", p.Name, num)

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
							text += foo("逃避行为：调查员会用任何的手段试图逃离现在所处的位置，即使这意味着开走唯一一辆交通工具并将其它人抛诸脑后，调查员会试图逃离 1D10轮。")
						case 8:
							text += foo("竭嘶底里：调查员表现出大笑，哭泣，嘶吼，害怕等的极端情绪表现，持续 1D10={1d10} 轮。")
						case 9:
							text += foo("恐惧：调查员通过一次 D100 或者由守秘人选择，来从恐惧症状表中选择一个恐惧源，就算这一恐惧的事物是并不存在的，调查员的症状会持续1D10 轮。")
							num2 := DiceRoll(100)
							text += fmt.Sprintf("\n1D100=%d\n", num2)
							text += fearMap[num2]
						case 10:
							text += foo("躁狂：调查员通过一次 D100 或者由守秘人选择，来从躁狂症状表中选择一个躁狂的诱因，这个症状将会持续 1D10={1d10} 轮。")
							num2 := DiceRoll(100)
							text += fmt.Sprintf("\n1D100=%d\n", num2)
							text += maniaMap[num2]
						}

						replyGroup(session.Socket, msg.GroupId, text);
					}
					return struct{ success bool }{
						success: true,
					}
				},
			},
			"li": &CmdItemInfo{
				name: "li",
				Brief: "随机抽取一个总结性疯狂症状",
				solve: func(session *IMSession, msg *Message, cmdArgs *CmdArgs) struct{ success bool } {
					// 总结性疯狂
					if isCurGroupBotOn(session, msg) {
						p := getPlayerInfoBySender(session, msg)
						foo := func(tmpl string) string {
							val, _, _ := self.exprText(tmpl, p)
							return val
						}

						num := DiceRoll(10)
						text := fmt.Sprintf("<%s>的疯狂发作-总结症状:\n1D10=%d\n", p.Name, num)

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

						replyGroup(session.Socket, msg.GroupId, text);
					}
					return struct{ success bool }{
						success: true,
					}
				},
			},

			"ra": &CmdItemInfo{
				name: "ra <属性>",
				Brief: "属性检定指令，骰一个D100，当有“D100 ≤ 属性”时，检定通过",
				solve: func(session *IMSession, msg *Message, cmdArgs *CmdArgs) struct{ success bool } {
					if isCurGroupBotOn(session, msg) && len(cmdArgs.Args) >= 1 {
						var cond int64
						p := getPlayerInfoBySender(session, msg)

						if len(cmdArgs.Args) >= 1 {
							var err error
							var suffix, detail string
							var r *vmStack
							d100 := DiceRoll64(100)
							r, detail, err = session.parent.exprEval(cmdArgs.RawArgs, p)
							if r != nil && r.typeId == 0 {
								cond = r.value.(int64)
							}

							if d100 <= cond {
								suffix = "成功"
								if d100 <= cond / 2 {
									suffix = "成功(困难)"
								}
								if d100 <= cond / 4 {
									suffix = "成功(极难)"
								}
								if d100 <= 1 {
									suffix = "大成功！"
								}
							} else {
								if d100 > 95 {
									suffix = "大失败！"
								} else {
									suffix = "失败！"
								}
							}

							if err == nil {
								detailWrap := ""
								if detail != "" {
									detailWrap = "=(" + detail + ")"
								}

								text := fmt.Sprintf("<%s>的“%s”检定结果为: D100=%d/%d%s %s", p.Name, cmdArgs.RawArgs, d100, cond, detailWrap, suffix)
								replyGroup(session.Socket, msg.GroupId, text);
							} else {
								replyGroup(session.Socket, msg.GroupId, "表达式不正确，可能是找不到属性");
							}
						}
					}
					return struct{ success bool }{
						success: true,
					}
				},
			},
			"sc": &CmdItemInfo{
				name: "sc <成功时掉san>/<失败时掉san>",
				Brief: "对理智进行一次D100检定，根据结果扣除理智。如“.sc 0/1d3”为成功不扣除理智，失败扣除1d3。大失败时按掷骰最大值扣除。支持复杂表达式。如.sc 1d2+3/1d(知识+1)",
				solve: func(session *IMSession, msg *Message, cmdArgs *CmdArgs) struct{ success bool } {
					if isCurGroupBotOn(session, msg) && len(cmdArgs.Args) >= 1 {
						//var cond int64
						p := getPlayerInfoBySender(session, msg)
						if len(cmdArgs.Args) >= 1 {
							var san int64
							var reduceSuccess int64
							var reduceFail int64
							var text1 string

							// 读取san值
							r, _, err := session.parent.exprEval("san", p)
							if err == nil {
								san = r.value.(int64)
							}

							// roll
							d100 := DiceRoll64(100)
							bigFail := d100 > 95

							// 计算成功或失败
							re := regexp.MustCompile(`(.+?)/(.+?)(\s+(\d+))?$`)
							m := re.FindStringSubmatch(cmdArgs.RawArgs)
							if len(m) > 0 {
								successExpr := m[1]
								failedExpr := m[2]
								text1 = successExpr + "/" + failedExpr
								_san, err := strconv.ParseInt(m[4], 10, 64)
								if err == nil {
									san = _san
								}

								r, _, err = session.parent.exprEvalBase(successExpr, p, false)
								if err == nil {
									reduceSuccess = r.value.(int64)
								}
								r, _, err = session.parent.exprEvalBase(failedExpr, p, bigFail)
								if err == nil {
									reduceFail = r.value.(int64)
								}

								var sanNew int64
								var suffix string
								if d100 <= san {
									suffix = "成功"
									sanNew = san - reduceSuccess
									text1 = successExpr
								} else {
									if d100 > 95 {
										suffix = "大失败！"
									} else {
										suffix = "失败！"
									}
									sanNew = san - reduceFail
									text1 = failedExpr
								}

								if sanNew < 0 {
									sanNew = 0
								}

								p.SetValueInt64("理智", sanNew, ac.Alias)

								//输出结果
								offset := san - sanNew
								text := fmt.Sprintf("<%s>的理智检定:\nD100=%d/%d %s\n理智变化: %d ➯ %d (扣除%s=%d点)\n", p.Name, d100, san, suffix, san, sanNew, text1, offset)

								if sanNew == 0 {
									text += "提示：理智归零，已永久疯狂(可用.ti或.li抽取症状)\n"
								} else {
									if offset >= 5 {
										text += "提示：单次损失理智超过5点，若智力检定(.ra 智力)通过，将进入临时性疯狂(可用.ti或.li抽取症状)\n"
									}
								}

								// 临时疯狂
								replyGroup(session.Socket, msg.GroupId, text);
							} else {
								replyGroup(session.Socket, msg.GroupId, "命令格式错误");
							}
						}
					}

					return struct{ success bool }{
						success: true,
					}
				},
			},
			"st": &CmdItemInfo{
				name: "st show <最小数值> / <属性><数值> / <属性>±<表达式>",
				Brief: "复杂指令，详见文档。举例: “.st 力量50“ ”.st 力量+1d10““.st show 40“ ",
				solve: func(session *IMSession, msg *Message, cmdArgs *CmdArgs) struct{ success bool } {
					// .st show
					// .st help
					// .st (<Name>[0-9]+)+
					// .st (<Name>)
					// .st (<Name>)+-<表达式>
					if isCurGroupBotOn(session, msg) && len(cmdArgs.Args) >= 0 {
						var param1 string
						if len(cmdArgs.Args) == 0 {
							param1 = ""
						} else {
							param1 = cmdArgs.Args[0]
						}
						switch param1 {
						case "help", "":
							text := "属性设置指令，支持分支指令如下：\n"
							text += ".st show/list <数值> // 展示个人属性，若加<数值>则不显示小于该数值的属性\n"
							text += ".st clr/clear // 清除属性\n"
							text += ".st del <属性名1> <属性名2> ... // 删除属性，可多项，以空格间隔\n"
							text += ".st help // 帮助\n"
							text += ".st <属性名><值> // 例：.st 敏捷50"
							replyGroup(session.Socket, msg.GroupId, text);

						case "del":
							p := getPlayerInfoBySender(session, msg)
							vm := p.ValueNumMap
							nums := []string{}
							failed := []string{}

							for _, varname := range cmdArgs.Args[1:] {
								_, ok := vm[varname]
								if ok {
									nums = append(nums, varname)
									delete(p.ValueNumMap, varname)
								} else {
									failed = append(failed, varname)
								}
							}

							text := fmt.Sprintf("<%s>的如下属性被成功删除:%s，失败%d项\n", p.Name, nums, len(failed))
							replyGroup(session.Socket, msg.GroupId, text);

						case "clr", "clear":
							p := getPlayerInfoBySender(session, msg)
							num := len(p.ValueNumMap)
							p.ValueNumMap = map[string]int64{};
							text := fmt.Sprintf("<%s>的属性数据已经清除，共计%d条", p.Name, num)
							replyGroup(session.Socket, msg.GroupId, text);

						case "show", "list":
							info := ""
							name := msg.Sender.Nickname

							p := getPlayerInfoBySender(session, msg)
							name = p.Name

							useLimit := false
							usePickItem := false
							var limit int64

							if len(cmdArgs.Args) >= 2 {
								arg2, _ := cmdArgs.GetArgN(2)
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
									pickItems[i] = 1
								}
							}

							tick := 0
							if len(p.ValueNumMap) == 0 {
								info = "未发现属性记录"
							} else {
								// 按照配置文件排序
								attrKeys := []string{}
								used := map[string]bool{}
								for _, i := range ac.Order.Top {
									key := p.GetValueNameByAlias(i, ac.Alias)
									if used[key] {
										continue
									}
									attrKeys = append(attrKeys, key)
									used[key] = true
								}

								// 其余按字典序
								topNum := len(attrKeys)
								attrKeys2 := []string{}
								for k, _ := range p.ValueNumMap {
									attrKeys2 = append(attrKeys2, k)
								}
								sort.Strings(attrKeys2)
								for _, key := range attrKeys2 {
									if used[key] {
										continue
									}
									attrKeys = append(attrKeys, key)
								}

								// 遍历输出
								for index, k := range attrKeys {
									v := p.ValueNumMap[k]

									if index >= topNum {
										if useLimit && v < limit {
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
									info += fmt.Sprintf("%s: %d\t", k, v)
									if tick % 4 == 0 {
										info += fmt.Sprintf("\n")
									}
								}
							}

							text := fmt.Sprintf("<%s>的个人属性为：\n%s", name, info)
							replyGroup(session.Socket, msg.GroupId, text);

						default:
							re1, _ := regexp.Compile(`([^\d]+?)([+-])=?(.+)$`)
							m := re1.FindStringSubmatch(cmdArgs.cleanArgs)
							if len(m) > 0 {
								p := getPlayerInfoBySender(session, msg)
								val, exists := p.GetValueInt64(m[1], ac.Alias)
								if !exists {
									text := fmt.Sprintf("<%s>: 无法找到名下属性 %s，不能作出修改", p.Name, m[1])
									replyGroup(session.Socket, msg.GroupId, text);
								} else {
									v, _, err := self.exprEval(m[3], p)
									if err == nil && v.typeId == 0 {
										var newVal int64
										rightVal := v.value.(int64)
										signText := ""

										if m[2] == "+" {
											signText = "增加"
											newVal = val + rightVal
										} else {
											signText = "扣除"
											newVal = val - rightVal
										}
										p.SetValueInt64(m[1], newVal, ac.Alias)

										text := fmt.Sprintf("<%s>的“%s”变化: %d ➯ %d (%s%s=%d)\n", p.Name, m[1], val, newVal, signText, m[3], rightVal)
										replyGroup(session.Socket, msg.GroupId, text);
									} else {
										text := fmt.Sprintf("<%s>: 错误的增减值: %s", p.Name, m[3])
										replyGroup(session.Socket, msg.GroupId, text);
									}
								}
							} else {
								valueMap := map[string]int64{};
								re, _ := regexp.Compile(`([^\d]+?)[:=]?(\d+)`)

								// 读取所有参数中的值
								stText := cmdArgs.cleanArgs

								m := re.FindAllStringSubmatch(RemoveSpace(stText), -1)

								for _, i := range m {
									num, err := strconv.ParseInt(i[2], 10, 64);
									if err == nil {
										valueMap[i[1]] = num;
									}
								}

								for _, v := range cmdArgs.Kwargs {
									vint, err := strconv.ParseInt(v.Value, 10, 64)
									if err == nil {
										valueMap[v.Name] = vint
									}
								}

								count := 0
								synonymsCount := 0
								p := getPlayerInfoBySender(session, msg)

								for k, v := range valueMap {
									name := p.GetValueNameByAlias(k, ac.Alias)
									if k != name {
										synonymsCount += 1
									} else {
										count += 1
									}
									p.SetValueInt64(name, v, ac.Alias)
								}

								p.lastUpdateTime = time.Now().Unix();
								//s, _ := json.Marshal(valueMap)
								text := fmt.Sprintf("<%s>的属性录入完成，本次共记录了%d条数据 (其中%d条为同义词)", p.Name, len(valueMap), synonymsCount)
								replyGroup(session.Socket, msg.GroupId, text);
							}
						}
					}
					return struct{ success bool }{
						success: true,
					}
				},
			},
		},
	})
}
