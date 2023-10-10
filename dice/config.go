package dice

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sealdice-core/dice/model"
	"sealdice-core/utils"
	"strconv"
	"strings"
	"time"

	"golang.org/x/time/rate"

	"github.com/fy0/lockfree"
	wr "github.com/mroth/weightedrand"
	"gopkg.in/yaml.v3"
)

// type TextTemplateWithWeight = map[string]map[string]uint
type TextTemplateItem = []interface{} // 实际上是 [](string | int) 类型
type TextTemplateItemList []TextTemplateItem

type TextTemplateWithWeight = map[string][]TextTemplateItem
type TextTemplateWithWeightDict = map[string]TextTemplateWithWeight

// TextTemplateHelpItem 辅助信息，用于UI中，大部分自动生成
type TextTemplateHelpItem = struct {
	Filename        []string           `json:"filename"` // 文件名
	Origin          []TextTemplateItem `json:"origin"`   // 初始文本
	Vars            []string           `json:"vars"`     // 可用变量
	Commands        []string           `json:"commands"` // 所属指令
	Modified        bool               `json:"modified"` // 跟初始相比是否有修改
	SubType         string             `json:"subType"`
	ExtraText       string             `json:"extraText"`       // 额外解说
	ExampleCommands []string           `json:"exampleCommands"` // 案例命令
	NotBuiltin      bool               `json:"notBuiltin"`      // 非内置
	TopOrder        int                `json:"topOrder"`        // 置顶序号，越高越靠前
}
type TextTemplateHelpGroup = map[string]*TextTemplateHelpItem
type TextTemplateWithHelpDict = map[string]TextTemplateHelpGroup

//const CONFIG_TEXT_TEMPLATE_FILE = "./data/configs/text-template.yaml"

func (i *TextTemplateItemList) toRandomPool() *wr.Chooser {
	var choices []wr.Choice
	for _, i := range *i {
		//weight, text := extractWeight(i)
		if len(i) == 1 {
			// 一种奇怪的情况，没有第二个值，见过一例，不知道怎么触发的
			i = append(i, 1)
		}

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

func (i *TextTemplateItemList) Clean() {
	for _, i := range *i {
		i[0] = strings.TrimSpace(i[0].(string))
	}
}

func setupBaseTextTemplate(d *Dice) {
	reGugu := regexp.MustCompile(`[\r\n]+`)
	var guguReason []TextTemplateItem

	for _, i := range reGugu.Split(strings.TrimSpace(guguText), -1) {
		guguReason = append(guguReason, []interface{}{i, 1})
	}

	texts := TextTemplateWithWeightDict{
		"COC": {
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
				{"成功了！这要费点力气{$t附加判定结果}", 1},
			},
			"判定_必须_困难_失败": {
				{"失败！还是有点难吧？{$t附加判定结果}", 1},
			},
			"判定_必须_极难_成功": {
				{"居然成功了！运气不错啊！{$t附加判定结果}", 1},
			},
			"判定_必须_极难_失败": {
				{"失败了，不要太勉强自己{$t附加判定结果}", 1},
			},
			"判定_必须_大成功_成功": {
				{"大成功！越过无数失败的命运，你握住了唯一的胜机！", 1},
				{"大成功！这一定是命运石之门的选择！", 1},
			},
			"判定_必须_大成功_失败": {
				{"失败了，不出所料{$t附加判定结果}", 1},
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
				{`{$t原因 ? '由于' + $t原因 + '，'}{$t玩家}的"{$t属性表达式文本}"检定结果为: {$t结果文本}`, 1},
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

			// -------------------- sc --------------------------
			"提示_永久疯狂": {
				{`提示：理智归零，已永久疯狂(可用.ti或.li抽取症状)`, 1},
			},
			"提示_临时疯狂": {
				{`提示：单次损失理智超过5点，若智力检定(.ra 智力)通过，将进入临时性疯狂(可用.ti或.li抽取症状)`, 1},
			},

			"理智检定_单项结果文本": {
				{`{$t检定表达式文本}={$tD100}/{$t判定值}{$t检定计算过程} {$t判定结果}`, 1},
			},
			"理智检定": {
				{"{$t玩家}的理智检定:\n{$t结果文本}\n理智变化: {$t旧值} ➯ {$t新值} (扣除{$t表达式文本}={$t表达式值}点){$t附加语}\n{$t提示_角色疯狂}", 1},
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
			"属性设置_增减_单项": {
				{"{$t属性}: {$t旧值} ➯ {$t新值} ({$t增加或扣除}{$t表达式文本}={$t变化量})", 1},
			},
			"属性设置_增减": {
				{"{$t玩家}的属性变化:\n{$t变更列表}\n{COC:属性设置_保存提醒}", 1},
			},
			//"属性设置_增减": {
			//	{"{$t玩家}的“{$t属性}”变化: {$t旧值} ➯ {$t新值} ({$t增加或扣除}{$t表达式文本}={$t变化量})\n{COC:属性设置_保存提醒}", 1},
			//},
			"属性设置_增减_错误的值": {
				{`"{$t玩家}: 错误的增减值: {$t表达式文本}"`, 1},
			},
			"属性设置_列出": {
				{"{$t玩家}的个人属性为:\n{$t属性信息}", 1},
			},
			"属性设置_列出_未发现记录": {
				{`未发现属性记录`, 1},
			},
			"属性设置_列出_隐藏提示": {
				{`\n注：{$t数量}条属性因<{$t判定值}被隐藏`, 1},
			},
			"属性设置": {
				{`{$t玩家}的{$t规则模板}属性录入完成，本次录入了{$t有效数量}条数据`, 1},
			},
			"属性设置_保存提醒": {
				{`{ $t当前绑定角色 ? '[√] 已绑卡' : '' }`, 1},
			},
			// -------------------- st end --------------------------

			// -------------------- en --------------------------
			"技能成长_导入语": {
				{"{$t玩家}的“{$t技能}”成长检定：", 1},
			},
			"技能成长_错误的属性类型": {
				{"{COC:技能成长_导入语}\n{COC:技能成长_错误的属性类型_无前缀}", 1},
			},
			"技能成长_错误的属性类型_无前缀": {
				{"该属性不能成长", 1},
			},
			"技能成长_错误的失败成长值": {
				{"{COC:技能成长_导入语}\n{COC:技能成长_错误的失败成长值_无前缀}", 1},
			},
			"技能成长_错误的失败成长值_无前缀": {
				{"错误的失败成长值: {$t表达式文本}", 1},
			},
			"技能成长_错误的成功成长值": {
				{"{COC:技能成长_导入语}\n{COC:技能成长_错误的成功成长值_无前缀}", 1},
			},
			"技能成长_错误的成功成长值_无前缀": {
				{"错误的成功成长值: {$t表达式文本}", 1},
			},
			"技能成长_属性未录入": {
				{"{COC:技能成长_导入语}\n{COC:技能成长_属性未录入_无前缀}", 1},
			},
			"技能成长_属性未录入_无前缀": {
				{"你没有使用st录入这个属性，或在en指令中指定属性的值", 1},
			},
			"技能成长_结果_成功": {
				{"{COC:技能成长_结果_成功_无后缀}\n{COC:属性设置_保存提醒}", 1},
			},
			"技能成长_结果_成功_无后缀": {
				{"“{$t技能}”增加了{$t表达式文本}={$t增量}点，当前为{$t新值}点", 1},
			},
			"技能成长_结果_失败": {
				{"“{$t技能}”成长失败了！", 1},
			},
			"技能成长_结果_失败变更": {
				{"{COC:技能成长_结果_失败变更_无后缀}\n{COC:属性设置_保存提醒}", 1},
			},
			"技能成长_结果_失败变更_无后缀": {
				{"“{$t技能}”变化{$t表达式文本}={$t增量}点，当前为{$t新值}点", 1},
			},
			"技能成长": {
				{"{COC:技能成长_导入语}\nD100={$tD100}/{$t判定值} {$t判定结果}\n{$t结果文本}", 1},
			},
			"技能成长_批量_分隔符": {
				{`\n\n`, 1},
			},
			"技能成长_批量_导入语": {
				{"{$t玩家}的{$t数量}项技能的批量成长检定：", 1},
			},
			"技能成长_批量_单条": {
				{"“{$t技能}”：D100={$tD100}/{$t判定值} {$t判定结果}\n{$t结果文本}", 1},
			},
			"技能成长_批量_单条错误前缀": {
				{"“{$t技能}”：", 1},
			},
			"技能成长_批量": {
				{"{COC:技能成长_批量_导入语}\n{$t总结果文本}\n{COC:属性设置_保存提醒}", 1},
			},
			"技能成长_批量_技能过多警告": {
				{`试图成长{$t数量}项技能，但{核心:骰子名字}没有这么多骰子。`, 1},
			},
			// -------------------- en end --------------------------
			"制卡": {
				{"{$t玩家}的七版COC人物作成:\n{$t制卡结果文本}", 1},
			},
			"制卡_分隔符": {
				{"#{SPLIT}", 1},
			},
			"对抗检定": {
				{`对抗检定:
{$t玩家A} {$t玩家A判定式}-> 属性值:{$t玩家A属性} 判定值:{$t玩家A判定值}{$t玩家A判定过程} {$t玩家A判定结果}
{$t玩家B} {$t玩家B判定式}-> 属性值:{$t玩家B属性} 判定值:{$t玩家B判定值}{$t玩家B判定过程} {$t玩家B判定结果}
{% $tWinFlag == -1 ? $t玩家A + '胜出！',
   $tWinFlag == +1 ? $t玩家B + '胜出！',
   $tWinFlag == 0 ? '平手！(请自行根据场景，如属性比较、攻击对反击，攻击对闪避)做出判断'
%}`, 1},
			},
			// -------------------- ti li --------------------------
			"疯狂发作_即时症状": {
				{"{$t玩家}的疯狂发作-即时症状:\n{$t表达式文本}\n{$t疯狂描述}", 1},
			},
			"疯狂发作_总结症状": {
				{"{$t玩家}的疯狂发作-总结症状:\n{$t表达式文本}\n{$t疯狂描述}", 1},
			},
			// -------------------- ti li end --------------------------
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
			"先攻_下一回合": {
				{"【{$t当前回合角色名}】{$t当前回合at}戏份结束了，下面该【{$t下一回合角色名}】{$t下一回合at}出场了！同时请【{$t下下一回合角色名}】{$t下下一回合at}做好准备", 1},
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
			"制卡_分隔符": {
				{`\n`, 1},
			},
		},
		"核心": {
			"骰子名字": {
				{"海豹核心", 1},
			},
			"骰子帮助文本_附加说明": {
				{"========\n.help 骰点/骰主/协议/娱乐/跑团/扩展/查询/其他\n========\n一只海豹罢了", 1},
			},
			"骰子帮助文本_骰主": {
				{"骰主很神秘，什么都没有说——", 1},
			},
			"骰子帮助文本_协议": {
				{"请在遵守以下规则前提下使用:\n1. 遵守国家法律法规\n2. 在跑团相关群进行使用\n3. 不要随意踢出、禁言、刷屏\n4. 务必信任骰主，有事留言\n如不同意使用.bot bye使其退群，谢谢。\n祝玩得愉快。", 1},
			},
			"骰子帮助文本_娱乐": {
				{"帮助:娱乐\n.gugu // 随机召唤一只鸽子\n.jrrp 今日人品", 1},
			},
			"骰子帮助文本_其他": {
				{"帮助:其他\n.find 克苏鲁星之眷族 //查找对应怪物资料\n.find 70尺 法术 // 查找关联资料（仅在全文搜索开启时可用）", 1},
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
				{`<{核心:骰子名字}> 已经就绪。可通过.help查看手册\n[图:data/images/sealdice.png]\nCOC/DND玩家可以使用.set coc/dnd在两种模式中切换\n已搭载自动重连，如遇风控不回可稍作等待`, 1},
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
			"骰子状态附加文本": {
				{"供职于{$t供职群数}个群，其中{$t启用群数}个处于开启状态。{$t群内工作状态}", 1},
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
			"昵称_当前": {
				{"玩家的当前昵称为: {$t玩家}", 1},
			},
			"昵称_重置": {
				{"{$t旧昵称}({$t帐号ID})的昵称已重置为{$t玩家}", 1},
			},
			"昵称_改名": {
				{"{$t旧昵称}({$t帐号ID})的昵称被设定为{$t玩家}", 1},
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
			"角色管理_新建": {
				{"新建角色且自动绑定: {$t角色名}", 1},
			},
			"角色管理_新建_已存在": {
				{"已存在同名角色", 1},
			},
			"角色管理_绑定_成功": {
				{"切换角色\"{$t角色名}\"，绑定成功", 1},
			},
			"角色管理_绑定_失败": {
				{"角色\"{$t角色名}\"绑定失败，角色不存在", 1},
			},
			"角色管理_绑定_解除": {
				{"角色\"{$t角色名}\"绑定已解除，切换至群内角色卡", 1},
			},
			"角色管理_绑定_并未绑定": {
				{"当前群内并未绑定角色", 1},
			},
			"角色管理_加载成功": {
				{"角色{$t玩家}加载成功，欢迎回来", 1},
			},
			"角色管理_加载失败_已绑定": {
				{"当前群内是绑卡状态，请解除绑卡后进行此操作！", 1},
			},
			"角色管理_角色不存在": {
				{"无法加载/删除角色：你所指定的角色不存在", 1},
			},
			"角色管理_序列化失败": {
				{"无法加载/保存角色：序列化失败", 1},
			},
			"角色管理_储存成功": {
				{"角色\"{$t角色名}\"储存成功\n注: 非秘密团不用开团前存卡，跑团后save即可", 1},
			},
			"角色管理_储存失败_已绑定": {
				{"角色卡\"{$t角色名}\"是绑定状态，无法进行save操作", 1},
			},
			"角色管理_删除成功": {
				{"角色\"{$t角色名}\"删除成功", 1},
			},
			"角色管理_删除失败_已绑定": {
				{"角色卡\"{$t角色名}\"是绑定状态，\".pc untagAll {$t角色名}\"解除绑卡后再操作吧", 1},
			},
			"角色管理_删除成功_当前卡": {
				{"由于你删除的角色是当前角色，昵称和属性将被一同清空", 1},
			},
			// -------------------- pc end --------------------------
			"提示_私聊不可用": {
				{"该指令只在群组中可用", 1},
			},
			"提示_无权限": {
				{"你没有权限这样做", 1},
			},
			"提示_无权限_非master/管理/邀请者": {
				{"你不是管理员、邀请者或master", 1},
			},
			"提示_无权限_非master/管理": {
				{"你不是管理员或master", 1},
			},
			"提示_手动退群前缀": {
				{"因长期不使用等原因，骰主后台操作退群", 1},
			},
			"留言_已记录": {
				{"您的留言已被记录，另外注意不要滥用此功能，祝您生活愉快，再会。", 1},
			},
			"拦截_拦截提示_全部模式": {
				{"", 1},
			},
			"拦截_拦截提示_仅命令模式": {
				{"命令包含不当内容，{核心:骰子名字}拒绝响应。", 1},
			},
			"拦截_拦截提示_仅回复模式": {
				{"试图使{核心:骰子名字}回复不当内容，拒绝响应。", 1},
			},
			"拦截_警告内容_提醒级": {
				{"你已多次触发不当内容拦截，{核心:骰子名字}已经无法忍受！", 1},
			},
			"拦截_警告内容_注意级": {
				{"你已多次触发不当内容拦截，{核心:骰子名字}已经无法忍受！", 1},
			},
			"拦截_警告内容_警告级": {
				{"你已多次触发不当内容拦截，{核心:骰子名字}已经无法忍受！", 1},
			},
			"拦截_警告内容_危险级": {
				{"你已多次触发不当内容拦截，{核心:骰子名字}已经无法忍受！", 1},
			},
		},
		"娱乐": {
			"今日人品": {
				{"{$t玩家} 今日人品为{$t人品}，{%\n    $t人品 > 95 ? '人品爆表！',\n    $t人品 > 80 ? '运气还不错！',\n    $t人品 > 50 ? '人品还行吧',\n    $t人品 > 10 ? '今天不太行',\n    1 ? '流年不利啊！'\n%}", 1},
			},
			"鸽子理由": guguReason,
		},
		"其它": {
			"抽牌_列表": {
				{"{$t原始列表}", 1},
			},
			"抽牌_列表_没有牌组": {
				{`呃，没有发现任何牌组`, 1},
			},
			"抽牌_找不到牌组": {
				{"找不到这个牌组", 1},
			},
			"抽牌_找不到牌组_存在类似": {
				{"未找到牌组，但发现一些相似的:", 1},
			},
			"抽牌_分隔符": {
				{`\n\n`, 1},
			},
			"抽牌_结果前缀": {
				{``, 1},
			},
			"随机名字": {
				{"为{$t玩家}生成以下名字：\n{$t随机名字文本}", 1},
			},
			"随机名字_分隔符": {
				{"、", 1},
			},
			"戳一戳": {
				{"{核心:骰子名字}咕踊了一下", 1},
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
				{`找不到记录，请使用.log new新建记录`, 1},
			},
			"记录_开启_失败_未结束的记录": {
				{`当前已有记录中的日志{$t记录名称}，请先将其结束。`, 1},
			},
			"记录_关闭_成功": {
				{`当前记录"{$t记录名称}"已经暂停，已记录文本{$t当前记录条数}条\n结束故事并传送日志请用.log end`, 1},
			},
			"记录_关闭_失败": {
				{`没有找到正在进行的记录，已经是关闭状态。`, 1},
			},
			"记录_取出_未指定记录": {
				{`命令格式错误：当前没有开启状态的记录，或没有通过参数指定要取出的日志。请参考帮助。`, 1},
			},
			"记录_列出_导入语": {
				{`正在列出存在于此群的记录:`, 1},
			},
			"记录_结束": {
				{`故事落下了帷幕。\n记录已经关闭。`, 1},
			},
			"记录_新建_失败_未结束的记录": {
				{`上一段旅程{$t记录名称}还未结束，请先使用.log end结束故事。`, 1},
			},
			"记录_条数提醒": {
				{`提示: 当前故事的文本已经记录了 {$t条数} 条`, 1},
			},
			"记录_导出_邮箱发送前缀": {
				{`已向以下邮箱发送记录邮件：\n`, 1},
			},
			"记录_导出_无格式有效邮箱": {
				{`未提供格式有效的邮箱，将直接发送记录文件`, 1},
			},
			"记录_导出_未指定记录": {
				{`未指定记录名`, 1},
			},
			"记录_导出_文件名前缀": {
				{`【{$t记录名}】{$t日期}{$t时间}`, 1},
			},
			"记录_导出_邮件附言": {
				{"log文件见附件。", 1},
			},
			"OB_开启": {
				{"你将成为观众（自动修改昵称和群名片[如有权限]，并不会给观众发送暗骰结果）。", 1},
			},
			"OB_关闭": {
				{"你不再是观众了（自动修改昵称和群名片[如有权限]）。", 1},
			},
		},
	}

	helpInfo := TextTemplateWithHelpDict{
		"COC": {
			"设置房规_当前": {
				SubType: ".setcoc",
			},
			"判定_大失败": {
				SubType: "判定-常规",
			},
			"判定_失败": {
				SubType: "判定-常规",
			},
			"判定_成功_普通": {
				SubType: "判定-常规",
			},
			"判定_成功_困难": {
				SubType: "判定-常规",
			},
			"判定_成功_极难": {
				SubType: "判定-常规",
			},
			"判定_大成功": {
				SubType: "判定-常规",
			},

			"判定_必须_困难_成功": {
				SubType: "判定-常规",
			},
			"判定_必须_困难_失败": {
				SubType: "判定-常规",
			},
			"判定_必须_极难_成功": {
				SubType: "判定-常规",
			},
			"判定_必须_极难_失败": {
				SubType: "判定-常规",
			},
			"判定_必须_大成功_成功": {
				SubType: "判定-常规",
			},
			"判定_必须_大成功_失败": {
				SubType: "判定-常规",
			},

			"判定_简短_大失败": {
				SubType: "判定-简短",
			},
			"判定_简短_失败": {
				SubType: "判定-简短",
			},
			"判定_简短_成功_普通": {
				SubType: "判定-简短",
			},
			"判定_简短_成功_困难": {
				SubType: "判定-简短",
			},
			"判定_简短_成功_极难": {
				SubType: "判定-简短",
			},
			"判定_简短_大成功": {
				SubType: "判定-简短",
			},

			"检定_单项结果文本": {
				SubType: ".ra/rc",
				Vars:    []string{"$t检定表达式文本", "$tD100", "$t判定值", "$t检定计算过程", "$t判定结果", "$t判定结果_详细", "$t判定结果_简短", "$tSuccessRank"},
			},
			"检定": {
				SubType: ".ra/rc 射击",
				Vars:    []string{"$t原因", "$t玩家", "$t属性表达式文本", "$t结果文本"},
			},
			"检定_多轮": {
				SubType: ".ra/rc 3#射击",
			},
			"检定_轮数过多警告": {
				SubType: ".ra/rc 30#",
			},

			"检定_暗中_私聊_前缀": {
				SubType: ".rah/rch",
			},
			"检定_暗中_群内": {
				SubType: ".rah/rch",
			},
			"检定_格式错误": {
				SubType: ".ra/rc",
			},

			// -------------------- sc --------------------------
			"提示_永久疯狂": {
				SubType: ".sc",
			},
			"提示_临时疯狂": {
				SubType: ".sc",
			},

			"理智检定_单项结果文本": {
				SubType: ".sc",
				Vars:    []string{"$t检定表达式文本", "$tD100", "$t判定值", "$t检定计算过程", "$t判定结果", "$t判定结果_详细", "$t判定结果_简短", "$tSuccessRank"},
			},
			"理智检定": {
				SubType: ".sc",
			},

			"理智检定_附加语_成功": {
				SubType: ".sc",
			},
			"理智检定_附加语_失败": {
				SubType: ".sc",
			},
			"理智检定_附加语_大成功": {
				SubType: ".sc",
			},
			"理智检定_附加语_大失败": {
				SubType: ".sc",
			},
			"理智检定_格式错误": {
				SubType: ".sc",
			},
			// -------------------- sc end --------------------------

			// -------------------- st --------------------------
			"属性设置_删除": {
				SubType: ".st rm A B C",
			},
			"属性设置_清除": {
				SubType: ".st clr",
			},
			"属性设置_增减": {
				SubType: ".st hp+1",
			},
			"属性设置_增减_单项": {
				SubType: ".st hp+1 san-1",
			},
			"属性设置_增减_错误的值": {
				SubType: ".st hp+?",
			},
			"属性设置_列出": {
				SubType: ".st show",
			},
			"属性设置_列出_未发现记录": {
				SubType: ".st show",
			},
			"属性设置_列出_隐藏提示": {
				SubType: ".st show",
			},
			"属性设置": {
				SubType:         ".st 力量70",
				Vars:            []string{"$t玩家", "$t规则模板", "$t有效数量", "$t数量", "$t同义词数量"},
				ExampleCommands: []string{".st 力量70"},
			},
			"属性设置_保存提醒": {
				SubType:         ".st hp+1",
				ExampleCommands: []string{".st hp70", ".st hp+1"},
			},
			// -------------------- st end --------------------------

			// -------------------- en --------------------------
			"技能成长_导入语": {
				SubType: ".en",
			},
			"技能成长_错误的属性类型": {
				SubType: ".en $t玩家",
			},
			"技能成长_错误的属性类型_无前缀": {
				SubType: ".en $t玩家",
			},
			"技能成长_错误的失败成长值": {
				SubType: ".en 斗殴 +?",
			},
			"技能成长_错误的失败成长值_无前缀": {
				SubType: ".en 斗殴 +?",
			},
			"技能成长_错误的成功成长值": {
				SubType: ".en 斗殴 +?",
			},
			"技能成长_错误的成功成长值_无前缀": {
				SubType: ".en 斗殴 +?",
			},
			"技能成长_属性未录入": {
				SubType: ".en 斗殴",
			},
			"技能成长_属性未录入_无前缀": {
				SubType: ".en 斗殴",
			},
			"技能成长_结果_成功": {
				SubType: ".en 斗殴",
			},
			"技能成长_结果_成功_无后缀": {
				SubType: ".en 斗殴",
			},
			"技能成长_结果_失败": {
				SubType: ".en 斗殴",
			},
			"技能成长_结果_失败变更": {
				SubType: ".en 斗殴",
			},
			"技能成长_结果_失败变更_无后缀": {
				SubType: ".en 斗殴",
			},
			"技能成长": {
				SubType: ".en",
			},
			"技能成长_批量_分隔符": {
				SubType: ".en 批量",
			},
			"技能成长_批量_导入语": {
				SubType: ".en 批量",
			},
			"技能成长_批量_单条": {
				SubType: ".en 批量",
			},
			"技能成长_批量_单条错误前缀": {
				SubType: ".en 批量",
			},
			"技能成长_批量": {
				SubType: ".en 批量",
			},
			"技能成长_批量_技能过多警告": {
				SubType: ".en 批量",
			},
			// -------------------- en end --------------------------
			"制卡": {
				SubType: ".coc 2",
			},
			"制卡_分隔符": {
				SubType: ".coc 2",
			},
			"对抗检定": {
				SubType: ".rav/.rcv",
			},
			// -------------------- ti li --------------------------
			"疯狂发作_即时症状": {
				Vars:    []string{"$t玩家", "$t表达式文本", "$t疯狂描述", "$t选项值", "$t附加值1", "$t附加值2"},
				SubType: ".ti",
			},
			"疯狂发作_总结症状": {
				Vars:    []string{"$t玩家", "$t表达式文本", "$t疯狂描述", "$t选项值", "$t附加值1", "$t附加值2"},
				SubType: ".li",
			},
			// -------------------- ti li end --------------------------
		},
		"娱乐": {
			"今日人品": {
				Vars:     []string{"$t玩家", "$t人品"},
				Commands: []string{"jrrp"},
				SubType:  ".jrrp",
			},
			"鸽子理由": {
				SubType: ".gugu",
			},
		},
		"DND": {
			"属性设置_删除": {
				SubType: ".st rm",
			},
			"属性设置_清除": {
				SubType: ".st clr",
			},
			"属性设置_列出": {
				SubType: ".st show",
			},
			"属性设置_列出_未发现记录": {
				SubType: ".st show",
			},
			"属性设置_列出_隐藏提示": {
				SubType: ".st show",
			},
			"BUFF设置_删除": {
				SubType: ".buff rm",
			},
			"BUFF设置_清除": {
				SubType: ".buff clr",
			},
			"先攻_查看_前缀": {
				SubType: ".init",
			},
			"先攻_下一回合": {
				SubType: ".init ed",
			},
			"先攻_移除_前缀": {
				SubType: ".init rm",
			},
			"先攻_清除列表": {
				SubType: ".init clr",
			},
			"先攻_设置_指定单位": {
				SubType: ".init set",
			},
			"先攻_设置_前缀": {
				SubType: ".ri",
			},
			"先攻_设置_格式错误": {
				SubType: ".ri",
			},
			"死亡豁免_D20_附加语": {
				SubType: ".ds/死亡豁免",
			},
			"死亡豁免_D1_附加语": {
				SubType: ".ds/死亡豁免",
			},
			"死亡豁免_成功_附加语": {
				SubType: ".ds/死亡豁免",
			},
			"死亡豁免_失败_附加语": {
				SubType: ".ds/死亡豁免",
			},
			"死亡豁免_结局_伤势稳定": {
				SubType: ".ds/死亡豁免",
			},
			"死亡豁免_结局_角色死亡": {
				SubType: ".ds/死亡豁免",
			},
			"制卡_分隔符": {
				SubType: ".dnd 2/.dndx 3",
			},
		},
		"核心": {
			"骰子名字": {
				SubType:  "通用",
				TopOrder: 1,
			},
			"骰子帮助文本_附加说明": {
				SubType:  ".help",
				TopOrder: 1,
			},
			"骰子帮助文本_骰主": {
				SubType: ".help",
			},
			"骰子帮助文本_协议": {
				SubType: ".help",
			},
			"骰子帮助文本_娱乐": {
				SubType: ".help",
			},
			"骰子帮助文本_其他": {
				SubType: ".help",
			},
			"骰子执行异常": {
				SubType:  "通用",
				TopOrder: 1,
			},
			"骰子开启": {
				SubType:  ".bot on",
				TopOrder: 1,
			},
			"骰子关闭": {
				SubType:  ".bot off",
				TopOrder: 1,
			},
			"骰子进群": {
				SubType:  "通用",
				TopOrder: 1,
			},
			//"骰子群内迎新": {
			//	{`欢迎，{$新人昵称}，祝你在这里过得愉快`, 1},
			//},
			"骰子成为好友": {
				SubType:  "通用",
				TopOrder: 1,
			},
			"骰子退群预告": {
				SubType:  ".bot bye",
				TopOrder: 1,
			},
			"骰子保存设置": {
				SubType: ".bot save",
			},
			"骰子状态附加文本": {
				SubType: ".bot about",
				Vars:    []string{"$t供职群数", "$t启用群数", "$t群内工作状态", "$t群内工作状态_仅状态"},
			},
			//"roll前缀":{
			//	"为了{$t原因}", 1},
			//},
			//"roll": {
			//	"{$t原因}{$t玩家} 掷出了 {$t骰点参数}{$t计算过程}={$t结果}${tASM}", 1},
			//},
			// -------------------- roll --------------------------
			"骰点_原因": {
				SubType: ".r",
			},
			"骰点_单项结果文本": {
				SubType: ".r",
			},
			"骰点": {
				SubType: ".r",
			},
			"骰点_多轮": {
				SubType: ".r 3#",
			},
			"骰点_轮数过多警告": {
				SubType: ".r 30#",
			},
			// -------------------- roll end --------------------------
			"暗骰_群内": {
				SubType: ".rh",
			},
			"暗骰_私聊_前缀": {
				SubType: ".rh",
			},
			"昵称_当前": {
				SubType: ".nn",
			},
			"昵称_重置": {
				SubType: ".nn clr",
				Vars:    []string{"$t旧昵称", "$t帐号昵称", "$t帐号ID", "$t玩家"},
			},
			"昵称_改名": {
				SubType: ".nn",
				Vars:    []string{"$t旧昵称", "$t帐号ID", "$t玩家"},
			},
			"设定默认骰子面数": {
				SubType: ".set 30 --my",
			},
			"设定默认群组骰子面数": {
				SubType: ".set 100",
			},
			"设定默认骰子面数_错误": {
				SubType: ".set ?",
			},
			"设定默认骰子面数_重置": {
				SubType: ".set clr",
			},
			// -------------------- ch --------------------------
			"角色管理_新建": {
				SubType: ".pc new",
				Vars:    []string{"$t角色名"},
			},
			"角色管理_新建_已存在": {
				SubType: ".pc new",
				Vars:    []string{"$t角色名"},
			},
			"角色管理_绑定_成功": {
				SubType: ".pc tag",
				Vars:    []string{"$t角色名"},
			},
			"角色管理_绑定_失败": {
				SubType: ".pc tag",
				Vars:    []string{"$t角色名"},
			},
			"角色管理_绑定_解除": {
				SubType: ".pc tag",
				Vars:    []string{"$t角色名"},
			},
			"角色管理_绑定_并未绑定": {
				SubType: ".pc tag",
				Vars:    []string{"$t角色名"},
			},
			"角色管理_加载成功": {
				SubType: ".pc load",
				Vars:    []string{"$t角色名", "$t玩家"},
			},
			"角色管理_角色不存在": {
				SubType: ".pc load",
			},
			"角色管理_加载失败_已绑定": {
				SubType: ".pc load",
			},
			"角色管理_序列化失败": {
				SubType: ".pc load/save",
			},
			"角色管理_储存成功": {
				SubType: ".pc save",
			},
			"角色管理_储存失败_已绑定": {
				SubType: ".pc save",
			},
			"角色管理_删除成功": {
				SubType: ".pc rm",
			},
			"角色管理_删除失败_已绑定": {
				SubType: ".pc rm",
			},
			"角色管理_删除成功_当前卡": {
				SubType: ".pc rm",
			},
			// -------------------- pc end --------------------------
			"提示_私聊不可用": {
				SubType: "通用",
			},
			"提示_无权限": {
				SubType: "通用",
			},
			"提示_无权限_非master/管理/邀请者": {
				SubType: "通用",
			},
			"提示_无权限_非master/管理": {
				SubType: "通用",
			},
			"提示_手动退群前缀": {
				SubType: "通用",
			},
			"留言_已记录": {
				SubType: ".send",
			},
			"拦截_拦截提示_全部模式": {
				SubType: "拦截",
			},
			"拦截_拦截提示_仅命令模式": {
				SubType: "拦截",
			},
			"拦截_拦截提示_仅回复模式": {
				SubType: "拦截",
			},
			"拦截_警告内容_提醒级": {
				SubType: "拦截",
			},
			"拦截_警告内容_注意级": {
				SubType: "拦截",
			},
			"拦截_警告内容_警告级": {
				SubType: "拦截",
			},
			"拦截_警告内容_危险级": {
				SubType: "拦截",
			},
		},
		"其它": {
			"抽牌_列表": {
				SubType: ".draw keys",
			},
			"抽牌_列表_没有牌组": {
				SubType: ".draw keys",
			},
			"抽牌_找不到牌组": {
				SubType: ".draw",
				Vars:    []string{"$t牌组"},
			},
			"抽牌_找不到牌组_存在类似": {
				SubType: ".draw",
				Vars:    []string{"$t牌组"},
			},
			"抽牌_结果前缀": {
				SubType:   ".draw",
				ExtraText: "举例: 你从牌堆抽出 xxxx",
				Vars:      []string{"$t牌组"},
			},
			"抽牌_分隔符": {
				SubType:   ".draw",
				ExtraText: "多个抽取结果之间的分隔符",
			},
			"随机名字": {
				SubType: ".name/.namednd",
			},
			"随机名字_分隔符": {
				SubType: ".name/.namednd",
			},
			"戳一戳": {
				SubType: "手机QQ功能",
			},
		},
		"日志": {
			"记录_新建": {
				SubType: ".log new",
				Vars:    []string{"$t记录名称"},
			},
			"记录_开启_成功": {
				SubType: ".log on",
			},
			"记录_开启_失败_无此记录": {
				SubType: ".log on",
			},
			"记录_开启_失败_尚未新建": {
				SubType:   ".log on",
				ExtraText: "当 log new 之后，会有一个默认的记录名。此时可以直接log on和log off而不加参数。\n一旦log end之后，默认记录名没有了，就会出这个提示。",
			},
			"记录_开启_失败_未结束的记录": {
				SubType: ".log on",
				Vars:    []string{"$t记录名称"},
			},
			"记录_关闭_成功": {
				SubType: ".log off",
			},
			"记录_关闭_失败": {
				SubType: ".log off",
			},
			"记录_取出_未指定记录": {
				SubType: ".log get",
			},
			"记录_列出_导入语": {
				SubType: ".log list",
			},
			"记录_结束": {
				SubType: ".log end",
			},
			"记录_新建_失败_未结束的记录": {
				SubType: ".log new",
				Vars:    []string{"$t记录名称"},
			},
			"记录_条数提醒": {
				SubType: ".log",
			},
			"记录_导出_邮箱发送前缀": {
				SubType: ".log export",
			},
			"记录_导出_无格式有效邮箱": {
				SubType: ".log export",
			},
			"记录_导出_未指定记录": {
				SubType: ".log export",
			},
			"记录_导出_文件名前缀": {
				SubType: ".log export",
			},
			"记录_导出_邮件附言": {
				SubType:   ".log export",
				ExtraText: "发送的跑团log提取邮件附带的文案。",
			},
			"OB_开启": {
				SubType: ".ob",
			},
			"OB_关闭": {
				SubType: ".ob exit",
			},
		},
	}
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
				helpInfoItem.NotBuiltin = true
				v1[keyName] = helpInfoItem

				//vars := []string{}
				//existsMap := map[string]bool{}
				//for _, i := range v2 {
				//	re := regexp.MustCompile(`{(\S+?)}`)
				//	m := re.FindAllStringSubmatch(i[0].(string), -1)
				//	for _, j := range m {
				//		if !existsMap[j[1]] {
				//			existsMap[j[1]] = true
				//			vars = append(vars, j[1])
				//		}
				//	}
				//}
				//helpInfoItem.Vars = vars
			} else {
				//d.Logger.Debugf("词条覆盖: %s, %s", keyName, fn)
				// 如果和最初有变化，标记为修改
				var modified bool

				if len(helpInfoItem.Origin) == 0 {
					// 判断为初次
					helpInfoItem.Origin = append([]TextTemplateItem{}, v2...)
				}

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

			if len(helpInfoItem.Vars) == 0 {
				var vars []string
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
			}
		}
	}
}

func loadTextTemplate(d *Dice, fn string) {
	textPath := filepath.Join(d.BaseConfig.DataDir, fn)

	if _, err := os.Stat(textPath); err == nil {
		data, err := os.ReadFile(textPath)
		if err != nil {
			panic(err)
		}
		texts := TextTemplateWithWeightDict{}
		err = yaml.Unmarshal(data, &texts)
		if err != nil {
			panic(err)
		}

		if texts["牌堆"] != nil {
			texts["其它"] = texts["牌堆"]
			delete(texts, "牌堆")
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
			var choices []wr.Choice
			for _, textItem := range v {
				choices = append(choices, wr.Choice{Item: textItem[0].(string), Weight: getNumVal(textItem[1])})
			}

			pool, _ := wr.NewChooser(choices...)
			d.TextMap[fmt.Sprintf("%s:%s", category, k)] = pool
		}
	}

	picker, _ := wr.NewChooser(wr.Choice{Item: APPNAME, Weight: 1})
	d.TextMap["常量:APPNAME"] = picker

	picker, _ = wr.NewChooser(wr.Choice{Item: VERSION, Weight: 1})
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
	data, err := os.ReadFile(filepath.Join(d.BaseConfig.DataDir, "serve.yaml"))

	// 配置这块弄得比较屎，有机会换个方案。。。
	if err == nil {
		dNew := Dice{}
		err2 := yaml.Unmarshal(data, &dNew)
		if err2 == nil {
			//d.CommandCompatibleMode = dNew.CommandCompatibleMode
			d.CommandCompatibleMode = true // 一直为true即可
			d.ImSession.EndPoints = dNew.ImSession.EndPoints
			d.CommandPrefix = dNew.CommandPrefix
			d.DiceMasters = dNew.DiceMasters
			d.VersionCode = dNew.VersionCode
			d.MessageDelayRangeStart = dNew.MessageDelayRangeStart
			d.MessageDelayRangeEnd = dNew.MessageDelayRangeEnd
			d.WorkInQQChannel = dNew.WorkInQQChannel
			d.QQChannelLogMessage = dNew.QQChannelLogMessage
			d.QQChannelAutoOn = dNew.QQChannelAutoOn
			d.QQEnablePoke = dNew.QQEnablePoke
			d.TextCmdTrustOnly = dNew.TextCmdTrustOnly
			d.IgnoreUnaddressedBotCmd = dNew.IgnoreUnaddressedBotCmd
			d.UILogLimit = dNew.UILogLimit
			d.FriendAddComment = dNew.FriendAddComment
			d.AutoReloginEnable = dNew.AutoReloginEnable
			d.NoticeIds = dNew.NoticeIds
			d.ExtDefaultSettings = dNew.ExtDefaultSettings
			d.CustomReplyConfigEnable = dNew.CustomReplyConfigEnable
			d.RefuseGroupInvite = dNew.RefuseGroupInvite
			d.DefaultCocRuleIndex = dNew.DefaultCocRuleIndex
			d.UpgradeWindowId = dNew.UpgradeWindowId
			d.UpgradeEndpointId = dNew.UpgradeEndpointId
			d.BotExtFreeSwitch = dNew.BotExtFreeSwitch
			d.RateLimitEnabled = dNew.RateLimitEnabled
			d.TrustOnlyMode = dNew.TrustOnlyMode
			d.AliveNoticeEnable = dNew.AliveNoticeEnable
			d.AliveNoticeValue = dNew.AliveNoticeValue
			d.ReplyDebugMode = dNew.ReplyDebugMode
			d.LogSizeNoticeCount = dNew.LogSizeNoticeCount
			d.LogSizeNoticeEnable = dNew.LogSizeNoticeEnable
			d.PlayerNameWrapEnable = dNew.PlayerNameWrapEnable
			d.MailEnable = dNew.MailEnable
			d.MailFrom = dNew.MailFrom
			d.MailPassword = dNew.MailPassword
			d.MailSmtp = dNew.MailSmtp
			d.JsEnable = dNew.JsEnable
			d.DisabledJsScripts = dNew.DisabledJsScripts
			d.NewsMark = dNew.NewsMark

			d.EnableCensor = dNew.EnableCensor
			d.CensorMode = dNew.CensorMode
			d.CensorThresholds = dNew.CensorThresholds
			d.CensorHandlers = dNew.CensorHandlers
			d.CensorScores = dNew.CensorScores
			d.CensorCaseSensitive = dNew.CensorCaseSensitive
			d.CensorMatchPinyin = dNew.CensorMatchPinyin
			d.CensorFilterRegexStr = dNew.CensorFilterRegexStr

			if dNew.BanList != nil {
				d.BanList.BanBehaviorRefuseReply = dNew.BanList.BanBehaviorRefuseReply
				d.BanList.BanBehaviorRefuseInvite = dNew.BanList.BanBehaviorRefuseInvite
				d.BanList.BanBehaviorQuitLastPlace = dNew.BanList.BanBehaviorQuitLastPlace
				d.BanList.ScoreReducePerMinute = dNew.BanList.ScoreReducePerMinute

				d.BanList.ThresholdWarn = dNew.BanList.ThresholdWarn
				d.BanList.ThresholdBan = dNew.BanList.ThresholdBan
				d.BanList.ScoreGroupMuted = dNew.BanList.ScoreGroupMuted
				d.BanList.ScoreGroupKicked = dNew.BanList.ScoreGroupKicked
				d.BanList.ScoreTooManyCommand = dNew.BanList.ScoreTooManyCommand

				d.BanList.JointScorePercentOfGroup = dNew.BanList.JointScorePercentOfGroup
				d.BanList.JointScorePercentOfInviter = dNew.BanList.JointScorePercentOfInviter
			}

			d.MaxExecuteTime = dNew.MaxExecuteTime
			if d.MaxExecuteTime == 0 {
				d.MaxExecuteTime = 12
			}

			d.MaxCocCardGen = dNew.MaxCocCardGen
			if d.MaxCocCardGen == 0 {
				d.MaxCocCardGen = 5
			}

			d.CustomReplenishRate = dNew.CustomReplenishRate
			if d.CustomReplenishRate == "" {
				d.CustomReplenishRate = "@every 3s"
				d.ParsedReplenishRate = rate.Every(time.Second * 3)
			} else {
				if parsed, errParse := utils.ParseRate(d.CustomReplenishRate); errParse == nil {
					d.ParsedReplenishRate = parsed
				} else {
					d.Logger.Errorf("解析CustomReplenishRate失败: %v", errParse)
					d.CustomReplenishRate = "@every 3s"
					d.ParsedReplenishRate = rate.Every(time.Second * 3)
				}

			}

			d.CustomBurst = dNew.CustomBurst
			if d.CustomBurst == 0 {
				d.CustomBurst = 3
			}

			if d.DiceMasters == nil || len(d.DiceMasters) == 0 {
				d.DiceMasters = []string{"UI:1001"}
			}
			var newDiceMasters []string
			for _, i := range d.DiceMasters {
				if i != "<平台,如QQ>:<帐号,如QQ号>" {
					newDiceMasters = append(newDiceMasters, i)
				}
			}
			d.DiceMasters = newDiceMasters
			// 装载ServiceAt
			d.ImSession.ServiceAtNew = map[string]*GroupInfo{}
			//d.ImSession.ServiceAtNew = model.GroupInfoListGet(d.DBData)
			_ = model.GroupInfoListGet(d.DBData, func(id string, updatedAt int64, data []byte) {
				var groupInfo GroupInfo
				err := json.Unmarshal(data, &groupInfo)
				if err == nil {
					groupInfo.GroupId = id
					groupInfo.UpdatedAtTime = updatedAt

					// 找出其中以群号开头的，这是1.2版本的bug
					var toDelete []string
					if groupInfo.DiceIdExistsMap != nil {
						groupInfo.DiceIdExistsMap.Range(func(key string, value bool) bool {
							if strings.HasPrefix(key, "QQ-Group:") {
								toDelete = append(toDelete, key)
							}
							return true
						})
						for _, i := range toDelete {
							groupInfo.DiceIdExistsMap.Delete(i)
						}
					}
					//data = bytes.ReplaceAll(data, []byte(`"diceIds":{`), []byte(`"diceIdActiveMap":{`))

					//fmt.Println("????", id, groupInfo.GroupId)
					d.ImSession.ServiceAtNew[id] = &groupInfo
				} else {
					d.Logger.Errorf("加载群信息失败: %s", id)
				}
			})

			m := map[string]*ExtInfo{}
			for _, i := range d.ExtList {
				m[i.Name] = i
			}

			// 设置群扩展
			for _, v := range d.ImSession.ServiceAtNew {
				var tmp []*ExtInfo
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

				data := model.AttrGroupGetAll(d.DBData, g.GroupId)
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
				if g.DiceIdActiveMap == nil {
					g.DiceIdActiveMap = new(SyncMap[string, bool])
				}
				if g.DiceIdExistsMap == nil {
					g.DiceIdExistsMap = new(SyncMap[string, bool])
				}
				if g.BotList == nil {
					g.BotList = new(SyncMap[string, bool])
				}
			}

			if d.VersionCode != 0 && d.VersionCode < 10000 {
				d.CustomReplyConfigEnable = false
			}

			if d.VersionCode != 0 && d.VersionCode < 10001 {
				d.AliveNoticeValue = "@every 3h"
			}

			if d.VersionCode != 0 && d.VersionCode < 10003 {
				d.Logger.Infof("进行配置文件版本升级: %d -> %d", d.VersionCode, 10003)
				d.LogSizeNoticeCount = 500
				d.LogSizeNoticeEnable = true
				d.CustomReplyConfigEnable = true
			}

			if d.VersionCode != 0 && d.VersionCode < 10004 {
				d.AutoReloginEnable = false
			}

			if d.VersionCode != 0 && d.VersionCode < 10005 {
				d.RunAfterLoaded = append(d.RunAfterLoaded, func() {
					d.Logger.Info("正在自动升级自定义文案文件")
					for index, text := range d.TextMapRaw["核心"]["昵称_重置"] {
						srcText := text[0].(string)
						srcText = strings.ReplaceAll(srcText, "{$tQQ昵称}", "{$t旧昵称}")
						srcText = strings.ReplaceAll(srcText, "{$t帐号昵称}", "{$t旧昵称}")
						d.TextMapRaw["核心"]["昵称_重置"][index][0] = srcText
					}

					for index, text := range d.TextMapRaw["核心"]["角色管理_删除成功"] {
						srcText := text[0].(string)
						srcText = strings.ReplaceAll(srcText, "{$t新角色名}", "{$t角色名}")
						d.TextMapRaw["核心"]["角色管理_删除成功"][index][0] = srcText
					}

					SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
					d.GenerateTextMap()
					d.SaveText()
				})
			}

			// 1.2 版本
			if d.VersionCode != 0 && d.VersionCode < 10200 {
				d.TextCmdTrustOnly = true
				d.QQEnablePoke = true
				d.PlayerNameWrapEnable = true

				isUI1001Master := false
				for _, i := range d.DiceMasters {
					if i == "UI:1001" {
						isUI1001Master = true
						break
					}
				}
				if !isUI1001Master {
					d.DiceMasters = append(d.DiceMasters, "UI:1001")
				}

				d.RunAfterLoaded = append(d.RunAfterLoaded, func() {
					// 更正写反的部分
					d.Logger.Info("正在自动升级自定义文案文件")
					for index := range d.TextMapRaw["COC"]["属性设置_保存提醒"] {
						//srcText := text[0].(string)
						//srcText = strings.ReplaceAll(
						//	srcText,
						//	`{ $t当前绑定角色 ? '角色信息已经变更，别忘了使用.pc save来进行保存！' : '' }`,
						//	`{ $t当前绑定角色 ? '[√] 已绑卡' : '' }`,
						//)
						srcText := `{ $t当前绑定角色 ? '[√] 已绑卡' : '' }`
						d.TextMapRaw["COC"]["属性设置_保存提醒"][index][0] = srcText
					}

					SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
					d.GenerateTextMap()
					d.SaveText()
				})
			}

			// 1.2 版本
			if d.VersionCode != 0 && d.VersionCode < 10203 {
				d.RunAfterLoaded = append(d.RunAfterLoaded, func() {
					// 更正写反的部分
					d.Logger.Info("正在自动升级自定义文案文件")
					for index := range d.TextMapRaw["COC"]["属性设置_增减_单项"] {
						srcText := "{$t属性}: {$t旧值} ➯ {$t新值} ({$t增加或扣除}{$t表达式文本}={$t变化量})"
						d.TextMapRaw["COC"]["属性设置_增减_单项"][index][0] = srcText
					}

					SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
					d.GenerateTextMap()
					d.SaveText()
				})
			}

			// 1.3 版本
			if d.VersionCode != 0 && d.VersionCode < 10300 {
				d.JsEnable = true

				d.RunAfterLoaded = append(d.RunAfterLoaded, func() {
					// 更正写反的部分
					d.Logger.Info("正在自动升级自定义文案文件")
					for index, text := range d.TextMapRaw["娱乐"]["鸽子理由"] {
						srcText := text[0].(string)
						srcText = strings.ReplaceAll(srcText, "在互联网上约到可爱美少女不惜搁置跑团前去约会的{$t玩家}，还不知道这个叫奈亚的妹子隐藏着什么", "空山不见人，但闻咕咕声。 —— {$t玩家}")
						d.TextMapRaw["娱乐"]["鸽子理由"][index][0] = srcText
					}

					SetupTextHelpInfo(d, d.TextMapHelpInfo, d.TextMapRaw, "configs/text-template.yaml")
					d.GenerateTextMap()
					d.SaveText()
				})
			}

			// 设置全局群名缓存和用户名缓存
			dm := d.Parent
			now := time.Now().Unix()
			for k, v := range d.ImSession.ServiceAtNew {
				dm.GroupNameCache.Set(k, &GroupNameCacheItem{Name: v.GroupName, time: now})
				// 这块暂时不存在了
				//for k2, v2 := range v.Players {
				//	dm.UserNameCache.Set(k2, &GroupNameCacheItem{Name: v2.Name, time: now})
				//}
			}

			d.Logger.Info("serve.yaml loaded")
			//info, _ := yaml.Marshal(Session.ServiceAt)
			//replyGroup(ctx, msg.GroupId, fmt.Sprintf("临时指令：加载配置 似乎成功\n%s", info));
		} else {
			d.Logger.Error("serve.yaml parse failed")
			panic(err2)
		}
	} else {
		// 这里是没有加载到配置文件，所以写默认设置项
		d.AutoReloginEnable = false
		d.WorkInQQChannel = true
		d.CustomReplyConfigEnable = false
		d.AliveNoticeValue = "@every 3h"
		d.Logger.Info("serve.yaml not found")

		d.LogSizeNoticeCount = 500
		d.LogSizeNoticeEnable = true

		// 1.2
		d.QQEnablePoke = true
		d.TextCmdTrustOnly = true
		d.PlayerNameWrapEnable = true
		d.DiceMasters = []string{"UI:1001"}

		// 1.3
		d.JsEnable = true
	}

	_ = model.BanItemList(d.DBData, func(id string, banUpdatedAt int64, data []byte) {
		var v BanListInfoItem
		err := json.Unmarshal(data, &v)
		if err == nil {
			v.BanUpdatedAt = banUpdatedAt
			d.BanList.Map.Store(id, &v)
		}
	})

	for _, i := range d.ImSession.EndPoints {
		i.Session = d.ImSession
		i.AdapterSetup()
	}

	if d.NoticeIds == nil {
		d.NoticeIds = []string{}
	}

	if len(d.CommandPrefix) == 0 {
		d.CommandPrefix = []string{
			"!",
			".",
			"。",
			"/",
		}
	}

	d.VersionCode = 10300 // TODO: 记得修改！！！
	d.LogWriter.LogLimit = d.UILogLimit

	// 设置扩展选项
	d.ApplyExtDefaultSettings()

	// 读取文本模板
	setupTextTemplate(d)
	d.MarkModified()
}

func (d *Dice) SaveText() {
	buf, err := yaml.Marshal(d.TextMapRaw)
	if err != nil {
		fmt.Println(err)
	} else {
		newFn := filepath.Join(d.BaseConfig.DataDir, "configs/text-template.yaml")
		bakFn := filepath.Join(d.BaseConfig.DataDir, "configs/text-template.yaml.bak")
		//ioutil.WriteFile(filepath.Join(d.BaseConfig.DataDir, "configs/text-template.yaml"), buf, 0644)
		current, err := os.ReadFile(newFn)
		if err != nil {
			_ = os.WriteFile(bakFn, current, 0644)
		}

		_ = os.WriteFile(newFn, buf, 0644)
	}
}

// ApplyExtDefaultSettings 应用扩展默认配置
func (d *Dice) ApplyExtDefaultSettings() {
	// 遍历两个列表
	exts1 := map[string]*ExtDefaultSettingItem{}
	for _, i := range d.ExtDefaultSettings {
		exts1[i.Name] = i
	}

	exts2 := map[string]*ExtInfo{}
	for _, i := range d.ExtList {
		exts2[i.Name] = i
	}

	// 如果存在于扩展列表，但不存在于默认列表中的，那么放入末尾
	for k, v := range exts2 {
		if _, exists := exts1[k]; !exists {
			item := &ExtDefaultSettingItem{Name: k, AutoActive: v.AutoActive, DisabledCommand: map[string]bool{}}
			d.ExtDefaultSettings = append(d.ExtDefaultSettings, item)
			exts1[k] = item
		}
	}

	// 遍历设置表，将其插入扩展信息
	for k, v := range exts1 {
		extInfo, exists := exts2[k]
		if exists {
			v.ExtItem = extInfo
			v.Loaded = true
			extInfo.DefaultSetting = v

			// 为了避免锁问题，这里做一个新的map
			m := map[string]bool{}

			// 将改扩展拥有的指令，塞入DisabledCommand
			names := map[string]bool{}
			for _, v := range extInfo.CmdMap {
				names[v.Name] = true
			}
			// 去掉无效指令
			for k, v := range v.DisabledCommand {
				if names[k] {
					m[k] = v
				}
			}
			// 塞入之前没有的指令
			for k := range names {
				if _, exists := m[k]; !exists {
					m[k] = false // false 因为默认不禁用
				}
			}
			v.DisabledCommand = m
		} else {
			// 需要吗?
			// 也许需要, 模糊地感觉可能造成内存泄漏
			// v.ExtItem = nil
			v.Loaded = false
		}
	}

	// 不好分辨，直接标记
	d.MarkModified()
}

func (d *Dice) Save(isAuto bool) {
	if d.LastUpdatedTime != 0 {
		a, err := yaml.Marshal(d)

		if err == nil {
			err := os.WriteFile(filepath.Join(d.BaseConfig.DataDir, "serve.yaml"), a, 0644)
			if err == nil {
				now := time.Now()
				d.LastSavedTime = &now
				if isAuto {
					d.Logger.Info("自动保存")
				} else {
					d.Logger.Info("保存数据")
				}
				d.LastUpdatedTime = 0
			} else {
				d.Logger.Errorln("保存serve.yaml出错", err)
			}
		}
	}

	for _, g := range d.ImSession.ServiceAtNew {
		// 保存群内玩家信息
		if g.Players != nil {
			g.Players.Range(func(key string, value *GroupPlayerInfo) bool {
				if value.UpdatedAtTime != 0 {
					_ = model.GroupPlayerInfoSave(d.DBData, g.GroupId, key, (*model.GroupPlayerInfoBase)(value))
					value.UpdatedAtTime = 0
				}

				// 保存群组卡
				if value.Vars != nil && value.Vars.Loaded {
					if value.Vars.LastWriteTime != 0 {
						data, _ := json.Marshal(LockFreeMapToMap(value.Vars.ValueMap))
						model.AttrGroupUserSave(d.DBData, g.GroupId, key, data)
						value.Vars.LastWriteTime = 0
					}
				}

				return true
			})
		}

		if g.UpdatedAtTime != 0 {
			data, err := json.Marshal(g)
			if err == nil {
				err := model.GroupInfoSave(d.DBData, g.GroupId, g.UpdatedAtTime, data)
				if err != nil {
					d.Logger.Warnf("保存群组数据失败 %v : %v", g.GroupId, err.Error())
				}
				g.UpdatedAtTime = 0
			}
		}

		// TODO: 这里其实还能优化
		data, _ := json.Marshal(LockFreeMapToMap(g.ValueMap))
		model.AttrGroupSave(d.DBData, g.GroupId, data)
	}

	// 同步绑定的角色卡数据
	chPrefix := "$:ch-bind-mtime:"
	chPrefixData := "$:ch-bind-data:"
	for _, v := range d.ImSession.PlayerVarsData {
		if v.Loaded {
			if v.LastWriteTime != 0 {
				var toDelete []string
				syncMap := map[string]bool{}
				allCh := map[string]lockfree.HashMap{}

				_ = v.ValueMap.Iterate(func(_k interface{}, _v interface{}) error {
					if k, ok := _k.(string); ok {
						if strings.HasPrefix(k, chPrefixData) {
							v := _v.(lockfree.HashMap)
							allCh[k[len(chPrefixData):]] = v
						}
						if strings.HasPrefix(k, chPrefix) {
							// 只要存在，就是修改过，数值多少不重要
							syncMap[k[len(chPrefix):]] = true
							toDelete = append(toDelete, k)
						}
					}
					return nil
				})

				for _, i := range toDelete {
					v.ValueMap.Del(i)
				}

				//fmt.Println("!!!!!!!!", toDelete, syncMap, allCh)
				// 这里面的角色是需要同步的
				for name := range syncMap {
					chData := allCh[name]
					if chData != nil {
						val, err := json.Marshal(LockFreeMapToMap(chData))
						if err == nil {
							varName := "$ch:" + name
							v.ValueMap.Set(varName, &VMValue{
								TypeId: VMTypeString,
								Value:  string(val),
							})
							//fmt.Println("XXXXXXX", varName, string(val))
						}
					} else {
						// 过期了，可能该角色已经被删除
						v.ValueMap.Del("$:ch-bind-data:" + name)
					}
				}
			}
		}
	}

	// 保存玩家个人全局数据
	for k, v := range d.ImSession.PlayerVarsData {
		if v.Loaded {
			if v.LastWriteTime != 0 {
				data, _ := json.Marshal(LockFreeMapToMap(v.ValueMap))
				model.AttrUserSave(d.DBData, k, data)
				v.LastWriteTime = 0
			}
		}
	}

	// 保存黑名单数据
	// TODO: 增加更新时间检测
	//model.BanMapSet(d.DBData, d.BanList.MapToJSON())

	// endpoint数据额外更新到数据库
	for _, ep := range d.ImSession.EndPoints {
		// 为了避免Restore时没有UserId, Dump时有UserId, 导致空白数据被错误落库的情况, 这里提前做判断
		if len(ep.UserId) > 0 {
			/* NOTE(Xiangze Li): 按理说Restore只需要在每个ep新增时做一次. 但是许多ep都是异步
			   连接, 并且在连接真正完成之后才有UserId. 所以干脆每次保存数据都尝试一次Restore. */
			ep.StatsRestore(d)
			ep.StatsDump(d)
		}
	}
}
