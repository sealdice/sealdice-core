package dice

import (
	"encoding/json"
	"fmt"
	"github.com/fy0/lockfree"
	wr "github.com/mroth/weightedrand"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sealdice-core/dice/model"
	"strconv"
	"strings"
	"time"
)

//type TextTemplateWithWeight = map[string]map[string]uint
type TextTemplateItem = []interface{} // 实际上是 [](string | int) 类型
type TextTemplateItemList []TextTemplateItem

type TextTemplateWithWeight = map[string][]TextTemplateItem
type TextTemplateWithWeightDict = map[string]TextTemplateWithWeight

// TextTemplateHelpItem 辅助信息，用于UI中，大部分自动生成
type TextTemplateHelpItem = struct {
	Filename []string           `json:"filename"` // 文件名
	Origin   []TextTemplateItem `json:"origin"`   // 初始文本
	Vars     []string           `json:"vars"`     // 可用变量
	Modified bool               `json:"modified"` // 跟初始相比是否有修改
}
type TextTemplateHelpGroup = map[string]*TextTemplateHelpItem
type TextTemplateWithHelpDict = map[string]TextTemplateHelpGroup

//const CONFIG_TEXT_TEMPLATE_FILE = "./data/configs/text-template.yaml"

func (i *TextTemplateItemList) toRandomPool() *wr.Chooser {
	var choices []wr.Choice
	for _, i := range *i {
		//weight, text := extractWeight(i)
		if w, ok := i[1].(int); ok {
			choices = append(choices, wr.Choice{Item: i[0].(string), Weight: uint(w)})
		}

		if w, ok := i[1].(float64); ok {
			choices = append(choices, wr.Choice{Item: i[0].(string), Weight: uint(w)})
		}
	}
	randomPool, _ := wr.NewChooser(choices...)
	return randomPool
}

func setupBaseTextTemplate(d *Dice) {
	reGugu := regexp.MustCompile(`[\r\n]+`)
	var guguReason []TextTemplateItem

	for _, i := range reGugu.Split(strings.TrimSpace(guguText), -1) {
		guguReason = append(guguReason, []interface{}{i, 1})
	}

	texts := TextTemplateWithWeightDict{
		"COC": {
			"设置房规_0": {
				{`已切换房规为0:\n出1大成功\n不满50出96-100大失败，满50出100大失败(COC7规则书)`, 1},
			},
			"设置房规_1": {
				{`已切换房规为1:\n不满50出1大成功，满50出1-5大成功\n不满50出96-100大失败，满50出100大失败`, 1},
			},
			"设置房规_2": {
				{`已切换房规为2:\n出1-5且判定成功为大成功\n出96-100且判定失败为大失败`, 1},
			},
			"设置房规_3": {
				{`已切换房规为3:\n出1-5大成功\n出96-100大失败(即大成功/大失败时无视判定结果)`, 1},
			},
			"设置房规_4": {
				{`已切换房规为4:\n出1-5且≤(成功率/10)为大成功\n不满50出>=96+(成功率/10)为大失败，满50出100大失败`, 1},
			},
			"设置房规_5": {
				{`已切换房规为5:\n出1-2且≤(成功率/5)为大成功\n不满50出96-100大失败，满50出99-100大失败`, 1},
			},
			"设置房规_DeltaGreen": {
				{`已切换房规为DeltaGreen:\n出1或检定成功基础上个位十位相同为大成功\n出100或检定失败基础上个位十位相同为大失败\n此规则无困难成功或极难成功`, 1},
			},
			"设置房规_当前": {
				{`当前房规: {$t房规}\n{$t房规文本}`, 1},
			},
			"判定_大失败": {
				{"大失败！", 1},
			},
			"判定_失败": {
				{"失败！", 1},
			},
			"判定_成功_普通": {
				{"成功", 1},
			},
			"判定_成功_困难": {
				{"困难成功", 1},
			},
			"判定_成功_极难": {
				{"极难成功", 1},
			},
			"判定_大成功": {
				{"运气不错，大成功！", 1},
				{"大成功!", 1},
			},

			"判定_必须_困难_成功": {
				{"成功了！这要费点力气", 1},
			},
			"判定_必须_困难_失败": {
				{"失败！还是有点难吧？", 1},
			},
			"判定_必须_极难_成功": {
				{"居然成功了！运气不错啊！", 1},
			},
			"判定_必须_极难_失败": {
				{"失败了，不要太勉强自己", 1},
			},
			"判定_必须_大成功_成功": {
				{"大成功！越过无数失败的命运，你握住了唯一的胜机！", 1},
				{"大成功！这一定是命运石之门的选择！", 1},
			},
			"判定_必须_大成功_失败": {
				{"失败了，不出所料。", 1},
			},

			"判定_简短_大失败": {
				{"大失败", 1},
			},
			"判定_简短_失败": {
				{"失败", 1},
			},
			"判定_简短_成功_普通": {
				{"成功", 1},
			},
			"判定_简短_成功_困难": {
				{"困难成功", 1},
			},
			"判定_简短_成功_极难": {
				{"极难成功", 1},
			},
			"判定_简短_大成功": {
				{"大成功", 1},
			},

			"检定_单项结果文本": {
				{`{$t检定表达式文本}={$tD100}/{$t判定值}{$t检定计算过程} {$t判定结果}`, 1},
			},
			"检定": {
				{`{$t玩家}的"{$t属性表达式文本}"检定结果为: {$t结果文本}`, 1},
			},
			"检定_多轮": {
				{`对{$t玩家}的"{$t属性表达式文本}"进行了{$t次数}次检定，结果为:\n{$t结果文本}`, 1},
			},
			"检定_轮数过多警告": {
				{`你真的需要这么多轮检定？{核心:骰子名字}将对你提高警惕！`, 1},
				{`不支持连续检定{$t次数}次，{核心:骰子名字}觉得这太多了。`, 1},
			},

			"检定_暗中_私聊_前缀": {
				{`来自群<{$t群名}>({$t群号})的暗中检定:\n`, 1},
			},
			"检定_暗中_群内": {
				{"{$t玩家}悄悄进行了一项{$t属性表达式文本}检定", 1},
			},
			"检定_格式错误": {
				{`检定命令格式错误`, 1},
			},
			"通用_D100判定": {
				{"D100={$tD100}/{$t判定值} {$t判定结果}", 1},
			},
			"通用_D100判定_带过程": {
				{"D100={$tD100}/{$t判定值}{$t计算过程} {$t判定结果}", 1},
			},
			"提示_永久疯狂": {
				{`提示：理智归零，已永久疯狂(可用.ti或.li抽取症状)`, 1},
			},
			"提示_临时疯狂": {
				{`提示：单次损失理智超过5点，若智力检定(.ra 智力)通过，将进入临时性疯狂(可用.ti或.li抽取症状)`, 1},
			},

			// -------------------- sc --------------------------
			"理智检定_单项结果文本": {
				{`{$t检定表达式文本}={$tD100}/{$t判定值}{$t检定计算过程} {$t判定结果}`, 1},
			},
			"理智检定": {
				{`{$t玩家}的理智检定:\n{$t结果文本}\n理智变化: {$t旧值} ➯ {$t新值} (扣除{$t表达式文本}={$t表达式值}点){$t附加语}\n{$t提示_角色疯狂}`, 1},
			},

			"理智检定_附加语_成功": {
				{"", 1},
			},
			"理智检定_附加语_失败": {
				{"", 1},
			},
			"理智检定_附加语_大成功": {
				{`\n不错的结果，可惜毫无意义`, 1},
			},
			"理智检定_附加语_大失败": {
				{`\n你很快就能洞悉一切`, 1},
				{`\n精彩的演出，希望主演不要太快谢幕`, 1},
			},
			"理智检定_格式错误": {
				{`理智检定命令格式错误`, 1},
			},
			// -------------------- sc end --------------------------

			// -------------------- st --------------------------
			"属性设置_删除": {
				{`{$t玩家}的如下属性被成功删除:{$t属性列表}，失败{$t失败数量}项`, 1},
			},
			"属性设置_清除": {
				{`{$t玩家}的属性数据已经清除，共计{$t数量}条`, 1},
			},
			"属性设置_增减": {
				{`{$t玩家}的“{$t属性}”变化: {$t旧值} ➯ {$t新值} ({$t增加或扣除}{$t表达式文本}={$t变化量})\n{COC:属性设置_保存提醒}`, 1},
			},
			"属性设置_增减_错误的值": {
				{`"{$t玩家}: 错误的增减值: {$t表达式文本}"`, 1},
			},
			"属性设置_列出": {
				{`{$t玩家}的个人属性为:\n{$t属性信息}`, 1},
			},
			"属性设置_列出_未发现记录": {
				{`未发现属性记录`, 1},
			},
			"属性设置_列出_隐藏提示": {
				{`\n注：{$t数量}条属性因<{$t判定值}被隐藏`, 1},
			},
			"属性设置": {
				{`{$t玩家}的属性录入完成，本次录入了{$t有效数量}条数据(共读入{$t数量}条，同义词{$t同义词数量}条)`, 1},
			},
			"属性设置_保存提醒": {
				{`角色信息已经变更，别忘了使用.ch save来进行保存！`, 1},
			},
			// -------------------- st end --------------------------

			// -------------------- en --------------------------
			"技能成长_导入语": {
				{"{$t玩家}的“{$t技能}”成长检定：", 1},
			},
			"技能成长_错误的属性类型": {
				{`{COC:技能成长_导入语}\n该属性不能成长`, 1},
			},
			"技能成长_错误的失败成长值": {
				{`{COC:技能成长_导入语}\n错误的失败成长值: {$t表达式文本}`, 1},
			},
			"技能成长_错误的成功成长值": {
				{`{COC:技能成长_导入语}\n错误的成功成长值: {$t表达式文本}`, 1},
			},
			"技能成长_属性未录入": {
				{`{COC:技能成长_导入语}\n你没有使用st录入这个属性，或在en指令中指定属性的值`, 1},
			},
			"技能成长_结果_成功": {
				{`“{$t技能}”增加了{$t表达式文本}={$t增量}点，当前为{$t新值}点\n{COC:属性设置_保存提醒}`, 1},
			},
			"技能成长_结果_失败": {
				{"“{$t技能}”成长失败了！", 1},
			},
			"技能成长_结果_失败变更": {
				{`“{$t技能}”变化{$t表达式文本}={$t增量}点，当前为{$t新值}点\n{COC:属性设置_保存提醒}`, 1},
			},
			"技能成长": {
				{`{COC:技能成长_导入语}\n{COC:通用_D100判定}\n{$t结果文本}`, 1},
			},
			// -------------------- en end --------------------------
		},
		"DND": {
			"属性设置_删除": {
				{`{$t玩家}的如下属性被成功删除:{$t属性列表}，失败{$t失败数量}项`, 1},
			},
			"属性设置_清除": {
				{`{$t玩家}的属性数据已经清除，共计{$t数量}条`, 1},
			},
			"属性设置_列出": {
				{`{$t玩家}的个人属性为:\n{$t属性信息}`, 1},
			},
			"属性设置_列出_未发现记录": {
				{`未发现属性记录`, 1},
			},
			"属性设置_列出_隐藏提示": {
				{`\n注：{$t数量}条属性因<{$t判定值}被隐藏`, 1},
			},
			"BUFF设置_删除": {
				{`{$t玩家}的如下BUFF被成功删除:{$t属性列表}，失败{$t失败数量}项`, 1},
			},
			"BUFF设置_清除": {
				{`{$t玩家}的BUFF数据已经清除，共计{$t数量}条`, 1},
			},
			"先攻_查看_前缀": {
				{`当前先攻列表为:\n`, 1},
			},
			"先攻_移除_前缀": {
				{`{$t玩家}将以下单位从先攻列表中移除:\n`, 1},
			},
			"先攻_清除列表": {
				{`先攻列表已清除`, 1},
			},
			"先攻_设置_指定单位": {
				{"{$t玩家}已设置 {$t目标} 的先攻点为{$t表达式}{% $t计算过程 ? `{$t计算过程} =` %} {$t点数}\\n", 1},
			},
			"先攻_设置_前缀": {
				{`{$t玩家}对先攻点数设置如下:\n`, 1},
			},
			"先攻_设置_格式错误": {
				{`{$t玩家}的ri格式不正确!`, 1},
			},
			"死亡豁免_D20_附加语": {
				{`你觉得你还可以抢救一下！HP回复1点！`, 1},
			},
			"死亡豁免_D1_附加语": {
				{`伤势莫名加重了！死亡豁免失败+2！`, 1},
			},
			"死亡豁免_成功_附加语": {
				{`伤势暂时得到控制！死亡豁免成功+1`, 1},
			},
			"死亡豁免_失败_附加语": {
				{`有些不妙！死亡豁免失败+1`, 1},
			},
			"死亡豁免_结局_伤势稳定": {
				{`累计获得了3次死亡豁免检定成功，伤势稳定了！`, 1},
			},
			"死亡豁免_结局_角色死亡": {
				{`累计获得了3次死亡豁免检定失败，不幸去世了！`, 1},
			},
			"受到伤害_超过HP上限_附加语": {
				{`\n{$t玩家}遭受了{$t伤害点数}点过量伤害，超过了他的承受能力，一命呜呼了！`, 1},
			},
			"受到伤害_昏迷中_附加语": {
				{`\n{$t玩家}在昏迷状态下遭受了{$t伤害点数}点过量伤害，死亡豁免失败+1！`, 1},
			},
			"受到伤害_进入昏迷_附加语": {
				{`\n{$t玩家}遭受了{$t伤害点数}点过量伤害，生命值降至0，陷入了昏迷！`, 1},
			},
		},
		"核心": {
			"骰子名字": {
				{"海豹核心", 1},
			},
			"骰子帮助文本_附加说明": {
				{"一只海豹罢了", 1},
			},
			"骰子执行异常": {
				{"指令执行异常，请联系开发者，群号524364253，非常感谢。", 1},
			},
			"骰子开启": {
				{"{常量:APPNAME} 已启用 {常量:VERSION}", 1},
			},
			"骰子关闭": {
				{"<{核心:骰子名字}> 停止服务", 1},
			},
			"骰子进群": {
				{`<{核心:骰子名字}> 已经就绪。可通过.help查看手册\n[图:data/images/sealdice.png]\nDND玩家可以使用.ext dnd5e on开启特化指令支持\n已搭载自动重连，如遇风控不回可稍作等待`, 1},
			},
			//"骰子群内迎新": {
			//	{`欢迎，{$新人昵称}，祝你在这里过得愉快`, 1},
			//},
			"骰子成为好友": {
				{`<{核心:骰子名字}> 已经就绪。可通过.help查看手册，请拉群测试，私聊容易被企鹅吃掉。\n[图:data/images/sealdice.png]`, 1},
			},
			"骰子退群预告": {
				{"收到指令，5s后将退出当前群组", 1},
			},
			"骰子保存设置": {
				{"数据已保存", 1},
			},
			//"roll前缀":{
			//	"为了{$t原因}", 1},
			//},
			//"roll": {
			//	"{$t原因}{$t玩家} 掷出了 {$t骰点参数}{$t计算过程}={$t结果}${tASM}", 1},
			//},
			// -------------------- roll --------------------------
			"骰点_原因": {
				{"由于{$t原因}，", 1},
			},
			"骰点_单项结果文本": {
				{`{$t表达式文本}{$t计算过程}={$t计算结果}`, 1},
			},
			"骰点": {
				{"{$t原因句子}{$t玩家}掷出了 {$t结果文本}", 1},
			},
			"骰点_多轮": {
				{`{$t原因句子}{$t玩家}掷骰{$t次数}次:\n{$t结果文本}`, 1},
			},
			"骰点_轮数过多警告": {
				{`骰点的轮数多到异常，{核心:骰子名字}将对你提高警惕！`, 1},
				{`骰点{$t次数}轮？{核心:骰子名字}没有这么多骰子。`, 1},
			},
			// -------------------- roll end --------------------------
			"暗骰_群内": {
				{"黑暗的角落里，传来命运转动的声音", 1},
				{"诸行无常，命数无定，答案只有少数人知晓", 1},
				{"命运正在低语！", 1},
			},
			"暗骰_私聊_前缀": {
				{`来自群<{$t群名}>({$t群号})的暗骰:\n`, 1},
			},
			"昵称_重置": {
				{"{$t帐号昵称}({$t帐号ID})的昵称已重置为{$t玩家}", 1},
			},
			"昵称_改名": {
				{"{$t帐号昵称}({$t帐号ID})的昵称被设定为{$t玩家}", 1},
			},
			"设定默认骰子面数": {
				{"设定个人默认骰子面数为 {$t个人骰子面数}", 1},
			},
			"设定默认群组骰子面数": {
				{"设定群组默认骰子面数为 {$t群组骰子面数}", 1},
			},
			"设定默认骰子面数_错误": {
				{"设定默认骰子面数: 格式错误", 1},
			},
			"设定默认骰子面数_重置": {
				{"重设默认骰子面数为默认值", 1},
			},
			// -------------------- ch --------------------------
			"角色管理_加载成功": {
				{"角色{$t玩家}加载成功，欢迎回来", 1},
			},
			"角色管理_角色不存在": {
				{"无法加载/删除角色：你所指定的角色不存在", 1},
			},
			"角色管理_序列化失败": {
				{"无法加载/保存角色：序列化失败", 1},
			},
			"角色管理_储存成功": {
				{"角色{$t新角色名}储存成功", 1},
			},
			"角色管理_删除成功": {
				{"角色{$t新角色名}删除成功", 1},
			},
			"角色管理_删除成功_当前卡": {
				{"由于你删除的角色是当前角色，昵称和属性将被一同清空", 1},
			},
			// -------------------- pc end --------------------------
			"提示_私聊不可用": {
				{"该指令只在群组中可用", 1},
			},
		},
		"娱乐": {
			"今日人品": {
				{"{$t玩家} 今日人品为{$t人品}，{%\n    $t人品 > 95 ? '人品爆表！',\n    $t人品 > 80 ? '运气还不错！',\n    $t人品 > 50 ? '人品还行吧',\n    $t人品 > 10 ? '今天不太行',\n    1 ? '流年不利啊！'\n%}", 1},
			},
			"鸽子理由": guguReason,
		},
		"牌堆": {
			"抽牌_列表_没有牌组": {
				{`呃，没有发现任何牌组`, 1},
			},
			"抽牌_找不到牌组": {
				{"找不到这个牌组", 1},
			},
		},
		"日志": {
			"记录_新建": {
				{`新的故事开始了，祝旅途愉快！\n记录已经开启`, 1},
			},
			"记录_开启_成功": {
				{`故事"{$t记录名称}"的记录已经继续开启，当前已记录文本{$t当前记录条数}`, 1},
			},
			"记录_开启_失败_无此记录": {
				{`无法继续，没能找到记录: {$t记录名称}`, 1},
			},
			"记录_开启_失败_尚未新建": {
				{`旅程尚未开始，请使用.log new新建记录`, 1},
			},
			"记录_关闭_成功": {
				{`当前故事"{$t记录名称}"的记录已经暂停，当前已记录文本{$t当前记录条数}条\n结束故事并上传日志请用.log end`, 1},
			},
			"记录_关闭_失败": {
				{`旅途甚至尚未开始，又谈何停下来呢？`, 1},
			},
			"记录_列出_导入语": {
				{`正在列出存在于此群的记录:`, 1},
			},
			"记录_结束": {
				{`故事落下了帷幕。\n记录已经关闭。`, 1},
			},
			"记录_新建_失败_未结束的记录": {
				{`上一段旅程还未结束，请先使用.log end结束故事。或使用.log new <名称>，强行建立新日志`, 1},
			},
		},
	}

	helpInfo := TextTemplateWithHelpDict{}
	d.TextMapRaw = texts
	d.TextMapHelpInfo = helpInfo

	SetupTextHelpInfo(d, helpInfo, texts, "configs/text-template.yaml")
}

func SetupTextHelpInfo(d *Dice, helpInfo TextTemplateWithHelpDict, texts TextTemplateWithWeightDict, fn string) {
	diceTexts := d.TextMapRaw

	for groupName, v := range texts {
		v1, exists := helpInfo[groupName]
		if !exists {
			v1 = TextTemplateHelpGroup{}
			helpInfo[groupName] = v1
		}

		if diceTexts[groupName] == nil {
			// 如果TextMapRaw没有此分类，将其创建
			diceTexts[groupName] = TextTemplateWithWeight{}
		}

		for keyName, v2 := range v {
			helpInfoItem, exists := v1[keyName]

			if !exists {
				helpInfoItem = &TextTemplateHelpItem{}

				// 检测到新词条，将其写入
				diceTexts[groupName][keyName] = v2

				// 创建信息
				helpInfoItem.Origin = v2
				helpInfoItem.Modified = false
				helpInfoItem.Filename = []string{fn}
				v1[keyName] = helpInfoItem

				vars := []string{}
				existsMap := map[string]bool{}
				for _, i := range v2 {
					re := regexp.MustCompile(`{(\S+?)}`)
					m := re.FindAllStringSubmatch(i[0].(string), -1)
					for _, j := range m {
						if !existsMap[j[1]] {
							existsMap[j[1]] = true
							vars = append(vars, j[1])
						}
					}
				}
				helpInfoItem.Vars = vars
			} else {
				//d.Logger.Debugf("词条覆盖: %s, %s", keyName, fn)
				// 如果和最初有变化，标记为修改
				var modified bool

				if len(v2) != len(helpInfoItem.Origin) {
					modified = true
				} else {
					for index := range v2 {
						a := v2[index]
						b := helpInfoItem.Origin[index]
						if a[0] != b[0] || a[1] != b[1] {
							modified = true
							break
						}
					}
				}

				diceTexts[groupName][keyName] = v2 // 不管怎样都要复制，以免出现a改b改a，而留下b的情况
				helpInfoItem.Modified = modified

				// 添加到文件属性中
				filenameExists := false
				for _, i := range helpInfoItem.Filename {
					if i == fn {
						filenameExists = true
						break
					}
				}

				if !filenameExists {
					helpInfoItem.Filename = append(helpInfoItem.Filename, fn)
				}
			}
		}
	}
}

func loadTextTemplate(d *Dice, fn string) {
	textPath := filepath.Join(d.BaseConfig.DataDir, fn)

	if _, err := os.Stat(textPath); err == nil {
		data, err := ioutil.ReadFile(textPath)
		if err != nil {
			panic(err)
		}
		texts := TextTemplateWithWeightDict{}
		err = yaml.Unmarshal(data, &texts)
		if err != nil {
			panic(err)
		}

		SetupTextHelpInfo(d, d.TextMapHelpInfo, texts, fn)
	} else {
		d.Logger.Info("未检测到自定义文本文件，即将进行创建。")
	}
}

func setupTextTemplate(d *Dice) {
	// 加载预设
	setupBaseTextTemplate(d)

	// 加载硬盘设置
	loadTextTemplate(d, "configs/text-template.yaml")

	d.SaveText()
	d.GenerateTextMap()
}

func (d *Dice) GenerateTextMap() {
	// 生成TextMap
	d.TextMap = map[string]*wr.Chooser{}

	for category, item := range d.TextMapRaw {
		for k, v := range item {
			choices := []wr.Choice{}
			for _, textItem := range v {
				choices = append(choices, wr.Choice{Item: textItem[0].(string), Weight: getNumVal(textItem[1])})
			}

			pool, _ := wr.NewChooser(choices...)
			d.TextMap[fmt.Sprintf("%s:%s", category, k)] = pool
		}
	}

	picker, _ := wr.NewChooser(wr.Choice{APPNAME, 1})
	d.TextMap["常量:APPNAME"] = picker

	picker, _ = wr.NewChooser(wr.Choice{VERSION, 1})
	d.TextMap["常量:VERSION"] = picker
}

func getNumVal(i interface{}) uint {
	switch reflect.TypeOf(i).Kind() {
	case reflect.Int:
		return uint(i.(int))
	case reflect.Int8:
		return uint(i.(int8))
	case reflect.Int16:
		return uint(i.(int16))
	case reflect.Int32:
		return uint(i.(int32))
	case reflect.Int64:
		return uint(i.(int64))
	case reflect.Uint:
		return uint(i.(uint))
	case reflect.Uint8:
		return uint(i.(uint8))
	case reflect.Uint16:
		return uint(i.(uint16))
	case reflect.Uint32:
		return uint(i.(uint32))
	case reflect.Uint64:
		return uint(i.(uint64))
	case reflect.Uintptr:
		return uint(i.(uintptr))
	case reflect.Float32:
		return uint(i.(float32))
	case reflect.Float64:
		return uint(i.(float64))
	case reflect.String:
		v, _ := strconv.ParseInt(i.(string), 10, 64)
		return uint(v)
	}
	return 0
}

func (d *Dice) loads() {
	data, err := ioutil.ReadFile(filepath.Join(d.BaseConfig.DataDir, "serve.yaml"))

	if err == nil {
		dNew := Dice{}
		err2 := yaml.Unmarshal(data, &dNew)
		if err2 == nil {
			d.CommandCompatibleMode = dNew.CommandCompatibleMode
			d.ImSession.EndPoints = dNew.ImSession.EndPoints
			d.CommandPrefix = dNew.CommandPrefix
			d.DiceMasters = dNew.DiceMasters
			d.VersionCode = dNew.VersionCode
			d.MessageDelayRangeStart = dNew.MessageDelayRangeStart
			d.MessageDelayRangeEnd = dNew.MessageDelayRangeEnd
			d.WorkInQQChannel = dNew.WorkInQQChannel
			d.QQChannelLogMessage = dNew.QQChannelLogMessage
			d.QQChannelAutoOn = dNew.QQChannelAutoOn
			d.UILogLimit = dNew.UILogLimit
			d.FriendAddComment = dNew.FriendAddComment
			d.AutoReloginEnable = dNew.AutoReloginEnable
			d.NoticeIds = dNew.NoticeIds

			if d.DiceMasters == nil || len(d.DiceMasters) == 0 {
				d.DiceMasters = []string{}
			}
			newDiceMasters := []string{}
			for _, i := range d.DiceMasters {
				if i != "<平台,如QQ>:<帐号,如QQ号>" {
					newDiceMasters = append(newDiceMasters, i)
				}
			}
			d.DiceMasters = newDiceMasters
			d.ImSession.ServiceAtNew = dNew.ImSession.ServiceAtNew

			m := map[string]*ExtInfo{}
			for _, i := range d.ExtList {
				m[i.Name] = i
			}

			if d.VersionCode != 0 && d.VersionCode < 9913 {
				// 进行配置文件的升级
				d.Logger.Infof("进行配置文件版本升级: %d -> %d", d.VersionCode, 9913)
				d.MessageDelayRangeStart = 0.4
				d.MessageDelayRangeEnd = 0.9

				// connections
				for _, i := range dNew.ImSession.LegacyConns {
					fmt.Println("连接信息转换: ", i.Nickname, i.UserId)
					platform := i.Platform
					if platform == "" {
						platform = "QQ"
					}
					ep := &EndPointInfo{
						EndPointInfoBase{
							Id:                  i.Id,
							Nickname:            i.Nickname,
							State:               i.State,
							UserId:              FormatDiceIdQQ(i.UserId),
							GroupNum:            i.GroupNum,
							CmdExecutedNum:      i.CmdExecutedNum,
							CmdExecutedLastTime: i.CmdExecutedLastTime,
							OnlineTotalTime:     i.OnlineTotalTime,

							Platform:     platform,
							RelWorkDir:   i.RelWorkDir,
							Enable:       i.Enable,
							ProtocolType: i.Type,
						},
						&PlatformAdapterQQOnebot{
							ConnectUrl:                       i.ConnectUrl,
							UseInPackGoCqhttp:                i.UseInPackGoCqhttp,
							InPackGoCqHttpLoginSucceeded:     i.InPackGoCqHttpLoginSucceeded,
							InPackGoCqHttpLastRestrictedTime: i.InPackGoCqHttpLastRestrictedTime,
							InPackGoCqHttpProtocol:           i.InPackGoCqHttpProtocol,
							InPackGoCqHttpPassword:           i.InPackGoCqHttpPassword,
						},
					}
					d.ImSession.EndPoints = append(d.ImSession.EndPoints, ep)
				}

				// 这个似乎不用转换
				//for _, i := range d.ImSession.LegacyPlayerVarsData {
				//}

				// 群数据迁移
				d.ImSession.ServiceAtNew = make(map[string]*GroupInfo)
				for oldId, i := range dNew.ImSession.LegacyServiceAt {
					fmt.Println("群数据迁移: ", i.GroupName, i.GroupId)
					group := GroupInfo{
						Active:           i.Active,
						ActivatedExtList: i.ActivatedExtList,
						NotInGroup:       i.NotInGroup,

						GroupId:     FormatDiceIdQQGroup(i.GroupId),
						GroupName:   i.GroupName,
						DiceIds:     i.DiceIds,
						BotList:     i.BotList,
						DiceSideNum: i.DiceSideNum,

						CocRuleIndex: i.CocRuleIndex,
						LogCurName:   i.LogCurName,
						LogOn:        i.LogOn,
					}

					players := map[string]*GroupPlayerInfo{}
					for _, j := range i.Players {
						uid := FormatDiceIdQQ(j.UserId)
						players[uid] = &GroupPlayerInfo{
							GroupPlayerInfoBase{
								Name:            j.Name,
								UserId:          uid,
								InGroup:         j.InGroup,
								LastCommandTime: j.LastUpdateTime,
								DiceSideNum:     j.DiceSideNum,
							},
						}
					}
					group.Players = players

					d.ImSession.ServiceAtNew[FormatDiceIdQQGroup(oldId)] = &group
				}
				model.AttrTryUpdate(d.DB)
			}

			// 设置群扩展
			for _, v := range d.ImSession.ServiceAtNew {
				tmp := []*ExtInfo{}
				for _, i := range v.ActivatedExtList {
					if m[i.Name] != nil {
						tmp = append(tmp, m[i.Name])
					}
				}
				v.ActivatedExtList = tmp
			}

			// 读取群变量
			for _, g := range d.ImSession.ServiceAtNew {
				// 群组数据
				if g.ValueMap == nil {
					g.ValueMap = lockfree.NewHashMap()
				}

				data := model.AttrGroupGetAll(d.DB, g.GroupId)
				if len(data) != 0 {
					mapData := make(map[string]*VMValue)
					err := JsonValueMapUnmarshal(data, &mapData)
					if err != nil {
						d.Logger.Error("读取群变量失败: ", err)
					}
					for k, v := range mapData {
						g.ValueMap.Set(k, v)
					}
				}
				if g.DiceIds == nil {
					g.DiceIds = map[string]bool{}
				}
				if g.BotList == nil {
					g.BotList = map[string]bool{}
				}

				if d.VersionCode != 0 && d.VersionCode < 9909 {
					ei := d.ExtFind("story")
					g.ExtActive(ei)
					ei = d.ExtFind("dnd5e")
					g.ExtActive(ei)
					ei = d.ExtFind("coc7")
					g.ExtActive(ei)
				}

				// 个人群组数据
				//for _, p := range g.Players {
				//	if p.ValueMap == nil {
				//		p.ValueMap = map[string]*VMValue{}
				//	}
				//	if p.ValueMapTemp == nil {
				//		p.ValueMapTemp = map[string]*VMValue{}
				//	}
				//
				//	data := model.AttrGroupUserGetAll(d.DB, g.GroupId, p.UserId)
				//	err := JsonValueMapUnmarshal(data, &p.ValueMap)
				//	if err != nil {
				//		d.Logger.Error(err)
				//	}
				//}
			}

			if d.VersionCode != 0 && d.VersionCode < 9914 {
				d.AutoReloginEnable = true
				d.WorkInQQChannel = true
			}

			d.Logger.Info("serve.yaml loaded")
			//info, _ := yaml.Marshal(Session.ServiceAt)
			//replyGroup(ctx, msg.GroupId, fmt.Sprintf("临时指令：加载配置 似乎成功\n%s", info));
		} else {
			d.Logger.Info("serve.yaml parse failed")
			panic(err2)
		}
	} else {
		d.AutoReloginEnable = true
		d.WorkInQQChannel = true
		d.Logger.Info("serve.yaml not found")
	}

	for _, i := range d.ImSession.EndPoints {
		i.Session = d.ImSession
		i.AdapterSetup()
	}

	if d.NoticeIds == nil {
		d.NoticeIds = []string{}
	}

	if len(d.CommandPrefix) == 0 {
		d.CommandPrefix = []string{
			".",
			"。",
			"/",
		}
	}

	d.VersionCode = 9914
	d.LogWriter.LogLimit = d.UILogLimit

	// 读取文本模板
	setupTextTemplate(d)
}

func (d *Dice) SaveText() {
	buf, err := yaml.Marshal(d.TextMapRaw)
	if err != nil {
		fmt.Println(err)
	} else {
		ioutil.WriteFile(filepath.Join(d.BaseConfig.DataDir, "configs/text-template.yaml"), buf, 0644)
	}
}

func (d *Dice) Save(isAuto bool) {
	a, err := yaml.Marshal(d)
	if err == nil {
		err := ioutil.WriteFile(filepath.Join(d.BaseConfig.DataDir, "serve.yaml"), a, 0644)
		if err == nil {
			now := time.Now()
			d.LastSavedTime = &now
			if isAuto {
				d.Logger.Debug("自动保存")
			} else {
				d.Logger.Info("保存数据")
			}

			//for _, i := range d.ImSession.EndPoints {
			//	if i.UserId == 0 {
			//		i.GetLoginInfo()
			//	}
			//}
		}
	}

	userIds := map[string]bool{}
	for _, g := range d.ImSession.ServiceAtNew {
		for _, b := range g.Players {
			userIds[b.UserId] = true
			if b.Vars != nil && b.Vars.Loaded {
				if b.Vars.LastWriteTime != 0 {
					data, _ := json.Marshal(LockFreeMapToMap(b.Vars.ValueMap))
					model.AttrGroupUserSave(d.DB, g.GroupId, b.UserId, data)
					b.Vars.LastWriteTime = 0
				}
			}
		}

		data, _ := json.Marshal(LockFreeMapToMap(g.ValueMap))
		model.AttrGroupSave(d.DB, g.GroupId, data)
	}

	// 保存玩家个人全局数据
	for k, v := range d.ImSession.PlayerVarsData {
		if v.Loaded {
			if v.LastWriteTime != 0 {
				data, _ := json.Marshal(LockFreeMapToMap(v.ValueMap))
				model.AttrUserSave(d.DB, k, data)
				v.LastWriteTime = 0
			}
		}
	}
}
