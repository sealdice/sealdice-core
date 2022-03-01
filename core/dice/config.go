package dice

import (
	"encoding/json"
	"fmt"
	wr "github.com/mroth/weightedrand"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
	"sealdice-core/core"
	"sealdice-core/model"
	"time"
)

type TextTemplateWithWeight = map[string]map[string]uint
type TextTemplateWithWeightDict = map[string]TextTemplateWithWeight

const BASE_CONFIG = "./data/serve.yaml"
const CONFIG_TEXT_TEMPLATE_FILE = "./data/configs/text-template.yaml"

func setupTextTemplate(d *Dice) {
	textTemplateFn := CONFIG_TEXT_TEMPLATE_FILE
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
			"COC7": TextTemplateWithWeight{
				"设置房规-0": {
					"已切换房规为0: 出1大成功，不满50出96-100大失败，满50出100大失败(COC7规则书)": 1,
				},
				"设置房规-1": {
					"已切换房规为1: 不满50出1大成功，不满50出96-100大失败，满50出100大失败": 1,
				},
				"设置房规-2": {
					"已切换房规为2: 出1-5且判定成功为大成功，出96-100且判定失败为大失败": 1,
				},
				"设置房规-3": {
					"已切换房规为3: 出1-5大成功，出96-100大失败(即大成功/大失败时无视判定结果)": 1,
				},
				"设置房规-4": {
					"已切换房规为4: 出1-5且≤(成功率/10)为大成功，不满50出>=96+(成功率/10)为大失败，满50出100大失败": 1,
				},
				"设置房规-5": {
					"已切换房规为5: 出1-2且≤(成功率/5)为大成功，不满50出96-100大失败，满50出99-100大失败": 1,
				},
				"设置房规-当前": {
					"当前房规: {$t房规}": 1,
				},
				"判定-大失败": {
					"大失败！": 1,
				},
				"判定-失败": {
					"失败！": 1,
				},
				"判定-成功-普通": {
					"成功": 1,
				},
				"判定-成功-困难": {
					"成功(困难)": 1,
				},
				"判定-成功-极难": {
					"成功(极难)": 1,
				},
				"判定-大成功": {
					"运气不错，大成功！": 1,
					"大成功!":      1,
				},
			},
			"核心": TextTemplateWithWeight{
				"名字": {
					"SealDice": 1,
				},
				"骰子崩溃": {
					"已从核心崩溃中恢复，请带指令联系开发者。注意不要重复发送本指令以免风控。": 1,
				},
				"骰子开启": {
					"{核心:名字} 已启用(开发中) {常量:VERSION}": 1,
				},
				"骰子关闭": {
					"{核心:名字} 停止服务": 1,
				},
				"骰子退群预告": {
					"收到指令，5s后将退出当前群组": 1,
				},
				"骰子保存设置": {
					"数据已保存": 1,
				},
				//"roll前缀":{
				//	"为了{$t原因}": 1,
				//},
				//"roll": {
				//	"{$t原因}{$t玩家} 掷出了 {$t骰点参数}{$t计算过程}={$t结果}${tASM}": 1,
				//},
				"暗骰-群内": {
					"黑暗的角落里，传来命运转动的声音":    1,
					"诸行无常，命数无定，答案只有少数人知晓": 1,
					"命运正在低语！":             1,
				},
				//"暗骰-私发": {
				//	"来自群<{$群名}>({$t群号})的暗骰，{核心:roll}": 1,
				//},
				"昵称-重置": {
					"{$tQQ昵称}({$tQQ})的昵称已重置为{$t玩家}": 1,
				},
				"昵称-改名": {
					"{$tQQ昵称}({$tQQ})的昵称被设定为{$t玩家}": 1,
				},
				"设定默认骰子面数": {
					"设定默认骰子面数为 {$t个人骰子面数}": 1,
				},
				"设定默认骰子面数-错误": {
					"设定默认骰子面数: 格式错误": 1,
				},
				"设定默认骰子面数-重置": {
					"重设默认骰子面数为默认值": 1,
				},
				"角色管理-加载成功": {
					"角色{$t玩家}加载成功，欢迎回来": 1,
				},
				"角色管理-角色不存在": {
					"无法加载角色：你所指定的角色不存在": 1,
				},
				"角色管理-序列化失败": {
					"无法加载/保存角色：序列化失败": 1,
				},
				"角色管理-储存成功": {
					"角色{$t新角色名}储存成功": 1,
				},
				"角色管理-删除成功": {
					"角色{$t新角色名}删除成功": 1,
				},
				// 角色管理-
			},
			"娱乐": {
				"今日人品": {
					"{$t玩家}的今日人品为 {$t人品}": 1,
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

	// 生成TextMap
	d.TextMap = map[string]*wr.Chooser{}

	for category, item := range texts {
		for k, v := range item {
			choices := []wr.Choice{}
			for t, w := range v {
				choices = append(choices, wr.Choice{Item: t, Weight: w})
			}

			pool, _ := wr.NewChooser(choices...)
			d.TextMap[fmt.Sprintf("%s:%s", category, k)] = pool
		}
	}

}

func (d *Dice) loads() {
	data, err := ioutil.ReadFile(BASE_CONFIG)

	if err == nil {
		session := d.ImSession
		dNew := Dice{}
		err2 := yaml.Unmarshal(data, &dNew)
		if err2 == nil {
			d.CommandCompatibleMode = dNew.CommandCompatibleMode

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
				data := model.AttrGroupGetAll(g.GroupId)
				err := JsonValueMapUnmarshal(data, &g.ValueMap)
				if err != nil {
					core.GetLogger().Error(err)
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

					data := model.AttrGroupUserGetAll(g.GroupId, p.UserId)
					err := JsonValueMapUnmarshal(data, &p.ValueMap)
					if err != nil {
						core.GetLogger().Error(err)
					}
				}
			}

			log.Println("serve.yaml loaded")
			//info, _ := yaml.Marshal(Session.ServiceAt)
			//replyGroup(ctx, msg.GroupId, fmt.Sprintf("临时指令：加载配置 似乎成功\n%s", info));
		} else {
			log.Println("serve.yaml parse failed")
			panic(err2)
		}
	} else {
		log.Println("serve.yaml not found")
	}

	// 读取文本模板
	setupTextTemplate(d)

	picker, _ := wr.NewChooser(wr.Choice{VERSION, 1})
	d.TextMap["常量:VERSION"] = picker
}

func (d *Dice) save() {
	a, err := yaml.Marshal(d)
	if err == nil {
		err := ioutil.WriteFile(BASE_CONFIG, a, 0644)
		if err == nil {
			now := time.Now()
			d.LastSavedTime = &now
			core.GetLogger().Info("保存数据。当前用户ID: ", d.ImSession.UserId)
			if d.ImSession.UserId == 0 {
				d.ImSession.GetLoginInfo()
			}
		}
	}

	userIds := map[int64]bool{}
	for _, g := range d.ImSession.ServiceAt {
		for _, b := range g.Players {
			userIds[b.UserId] = true
			data, _ := json.Marshal(b.ValueMap)
			model.AttrGroupUserSave(g.GroupId, b.UserId, data)
		}

		data, _ := json.Marshal(g.ValueMap)
		model.AttrGroupSave(g.GroupId, data)
	}

	// 保存玩家个人全局数据
	for k, v := range d.ImSession.PlayerVarsData {
		if v.Loaded {
			data, _ := json.Marshal(v.ValueMap)
			model.AttrUserSave(k, data)
		}
	}
}
