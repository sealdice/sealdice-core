package dice

import (
	"encoding/json"
	"fmt"
	wr "github.com/mroth/weightedrand"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"sealdice-core/dice/model"
	"strconv"
	"time"
)

//type TextTemplateWithWeight = map[string]map[string]uint
type TextTemplateItem = []interface{} // 实际上是 [](string | int) 类型
type TextTemplateWithWeight = map[string][]TextTemplateItem
type TextTemplateWithWeightDict = map[string]TextTemplateWithWeight

//const CONFIG_TEXT_TEMPLATE_FILE = "./data/configs/text-template.yaml"

func setupTextTemplate(d *Dice) {
	textTemplateFn := filepath.Join(d.BaseConfig.DataDir, "configs/text-template.yaml")
	var texts TextTemplateWithWeightDict

	if _, err := os.Stat(textTemplateFn); err == nil {
		data, err := ioutil.ReadFile(textTemplateFn)
		if err != nil {
			panic(err)
		}
		texts = TextTemplateWithWeightDict{}
		err = yaml.Unmarshal(data, &texts)
		if err != nil {
			panic(err)
		}
	} else {
		texts = TextTemplateWithWeightDict{
			"COC": {
				"设置房规_0": {
					{"已切换房规为0: 出1大成功，不满50出96-100大失败，满50出100大失败(COC7规则书)", 1},
				},
				"设置房规_1": {
					{"已切换房规为1: 不满50出1大成功，不满50出96-100大失败，满50出100大失败", 1},
				},
				"设置房规_2": {
					{"已切换房规为2: 出1-5且判定成功为大成功，出96-100且判定失败为大失败", 1},
				},
				"设置房规_3": {
					{"已切换房规为3: 出1-5大成功，出96-100大失败(即大成功/大失败时无视判定结果)", 1},
				},
				"设置房规_4": {
					{"已切换房规为4: 出1-5且≤(成功率/10)为大成功，不满50出>=96+(成功率/10)为大失败，满50出100大失败", 1},
				},
				"设置房规_5": {
					{"已切换房规为5: 出1-2且≤(成功率/5)为大成功，不满50出96-100大失败，满50出99-100大失败", 1},
				},
				"设置房规_当前": {
					{"当前房规: {$t房规}", 1},
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
					{"成功(困难)", 1},
				},
				"判定_成功_极难": {
					{"成功(极难)", 1},
				},
				"判定_大成功": {
					{"运气不错，大成功！", 1},
					{"大成功!", 1},
				},
				"检定": {
					{`{$t玩家}的{$t原因}检定结果为: {COC:通用_D100判定_带过程}`, 1},
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
				"理智检定": {
					{`{$t玩家}的理智检定:\n{COC:通用_D100判定}\n理智变化: {$t旧值} ➯ {$t新值} (扣除{$t表达式文本}={$t表达式值}点)\n{$t提示_角色疯狂}`, 1},
				},
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
					{"“{$t技能}”增加了{$t表达式文本}={$t增量}点，当前为{$t新值}点", 1},
				},
				"技能成长_结果_失败": {
					{"“{$t技能}”成长失败了！", 1},
				},
				"技能成长_结果_失败变更": {
					{"“{$t技能}”变化{$t表达式文本}={$t增量}点，当前为{$t新值}点", 1},
				},
				"技能成长": {
					{`{COC:技能成长_导入语}\n{COC:通用_D100判定}\n{$t结果文本}`, 1},
				},
			},
			"核心": {
				"骰子名字": {
					{"海豹bot", 1},
				},
				"骰子帮助文本_附加说明": {
					{"SealDice 目前 7*24h 运行于一块陈年OrangePi卡片电脑上，随时可能因为软硬件故障停机（例如过热、被猫打翻）。届时可以来Q群524364253询问。", 1},
				},
				"骰子崩溃": {
					{"已从核心崩溃中恢复，请带指令联系开发者，群号524364253，非常感谢。", 1},
				},
				"骰子开启": {
					{"{常量:APPNAME} 已启用(开发中) {常量:VERSION}", 1},
				},
				"骰子关闭": {
					{"<{核心:骰子名字}> 停止服务", 1},
				},
				"骰子进群": {
					{`<{核心:骰子名字}> 已经就绪。可通过.help查看指令列表\n[CQ:image,file=data/images/sealdice.png]`, 1},
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
				"暗骰_群内": {
					{"黑暗的角落里，传来命运转动的声音", 1},
					{"诸行无常，命数无定，答案只有少数人知晓", 1},
					{"命运正在低语！", 1},
				},
				//"暗骰-私发": {
				//	"来自群<{$群名}>({$t群号})的暗骰，{核心:roll}", 1},
				//},
				"昵称_重置": {
					{"{$tQQ昵称}({$tQQ})的昵称已重置为{$t玩家}", 1},
				},
				"昵称_改名": {
					{"{$tQQ昵称}({$tQQ})的昵称被设定为{$t玩家}", 1},
				},
				"设定默认骰子面数": {
					{"设定默认骰子面数为 {$t个人骰子面数}", 1},
				},
				"设定默认骰子面数_错误": {
					{"设定默认骰子面数: 格式错误", 1},
				},
				"设定默认骰子面数_重置": {
					{"重设默认骰子面数为默认值", 1},
				},
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
			},
			"娱乐": {
				"今日人品": {
					{"{$t玩家}的今日人品为{$t人品}", 1},
				},
			},
		}

		buf, err := yaml.Marshal(texts)
		if err != nil {
			fmt.Println(err)
		} else {
			ioutil.WriteFile(textTemplateFn, buf, 0644)
		}
	}

	d.TextMapRaw = texts
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
		session := d.ImSession
		dNew := Dice{}
		err2 := yaml.Unmarshal(data, &dNew)
		if err2 == nil {
			d.CommandCompatibleMode = dNew.CommandCompatibleMode
			d.ImSession.Conns = dNew.ImSession.Conns

			m := map[string]*ExtInfo{}
			for _, i := range d.ExtList {
				m[i.Name] = i
			}

			session.ServiceAt = dNew.ImSession.ServiceAt
			for _, v := range dNew.ImSession.ServiceAt {
				tmp := []*ExtInfo{}
				for _, i := range v.ActivatedExtList {
					if m[i.Name] != nil {
						tmp = append(tmp, m[i.Name])
					}
				}
				v.ActivatedExtList = tmp
			}

			// 读取新版数据
			for _, g := range d.ImSession.ServiceAt {
				// 群组数据
				data := model.AttrGroupGetAll(d.DB, g.GroupId)
				err := JsonValueMapUnmarshal(data, &g.ValueMap)
				if err != nil {
					d.Logger.Error(err)
				}
				if g.ValueMap == nil {
					g.ValueMap = map[string]VMValue{}
				}

				// 个人群组数据
				for _, p := range g.Players {
					if p.ValueMap == nil {
						p.ValueMap = map[string]VMValue{}
					}
					if p.ValueMapTemp == nil {
						p.ValueMapTemp = map[string]VMValue{}
					}

					data := model.AttrGroupUserGetAll(d.DB, g.GroupId, p.UserId)
					err := JsonValueMapUnmarshal(data, &p.ValueMap)
					if err != nil {
						d.Logger.Error(err)
					}
				}
			}

			d.Logger.Info("serve.yaml loaded")
			//info, _ := yaml.Marshal(Session.ServiceAt)
			//replyGroup(ctx, msg.GroupId, fmt.Sprintf("临时指令：加载配置 似乎成功\n%s", info));
		} else {
			d.Logger.Info("serve.yaml parse failed")
			panic(err2)
		}
	} else {
		d.Logger.Info("serve.yaml not found")
	}

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

			for _, i := range d.ImSession.Conns {
				if i.UserId == 0 {
					i.GetLoginInfo()
				}
			}
		}
	}

	userIds := map[int64]bool{}
	for _, g := range d.ImSession.ServiceAt {
		for _, b := range g.Players {
			userIds[b.UserId] = true
			data, _ := json.Marshal(b.ValueMap)
			model.AttrGroupUserSave(d.DB, g.GroupId, b.UserId, data)
		}

		data, _ := json.Marshal(g.ValueMap)
		model.AttrGroupSave(d.DB, g.GroupId, data)
	}

	// 保存玩家个人全局数据
	for k, v := range d.ImSession.PlayerVarsData {
		if v.Loaded {
			data, _ := json.Marshal(v.ValueMap)
			model.AttrUserSave(d.DB, k, data)
		}
	}
}
