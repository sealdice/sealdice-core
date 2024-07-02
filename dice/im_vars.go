package dice

// 用户变量相关

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/samber/lo"
	ds "github.com/sealdice/dicescript"
)

func VarSetValueStr(ctx *MsgContext, s string, v string) {
	VarSetValue(ctx, s, ds.NewStrVal(v))
}

func VarSetValueInt64(ctx *MsgContext, s string, v int64) {
	VarSetValue(ctx, s, ds.NewIntVal(ds.IntType(v)))
}

func _VarSetValueV1(ctx *MsgContext, s string, v *VMValue) {
	VarSetValue(ctx, s, v.ConvertToV2())
}

func VarSetValue(ctx *MsgContext, s string, v *ds.VMValue) {
	ctx.CreateVmIfNotExists()
	am := ctx.Dice.AttrsManager
	name := ctx.Player.GetValueNameByAlias(s, nil)
	vClone := v.Clone()

	// 临时变量
	if strings.HasPrefix(s, "$t") {
		// 如果是内部设置的临时变量，不需要长期存活
		//  if ctx.Player.ValueMapTemp == nil {
		//  	ctx.Player.ValueMapTemp = &ds.ValueMap{}
		//  }
		//  ctx.Player.ValueMapTemp.Store(s, vClone)
		// 用这个替代 temp 吗？我不太确定
		ctx.vm.StoreNameLocal(s, vClone)
		return
	}

	// 个人变量
	if strings.HasPrefix(s, "$m") {
		if ctx.Session != nil && ctx.Player != nil {
			playerAttrs := lo.Must(am.LoadById(ctx.Player.UserID))
			playerAttrs.Store(name, vClone)
		}
		return
	}

	// 群变量
	if ctx.Group != nil && strings.HasPrefix(s, "$g") {
		groupAttrs := lo.Must(am.LoadById(ctx.Group.GroupID))
		groupAttrs.Store(s, vClone)
		return
	}

	// 个人属性
	if ctx.Session != nil && ctx.Player != nil {
		curAttrs := lo.Must(am.LoadByCtx(ctx))
		curAttrs.Store(name, vClone)
	}
}

func VarDelValue(ctx *MsgContext, s string) {
	am := ctx.Dice.AttrsManager
	name := ctx.Player.GetValueNameByAlias(s, nil)

	// 临时变量
	if strings.HasPrefix(s, "$t") {
		ctx.Player.ValueMapTemp.Delete(s)
		curAttrs := lo.Must(am.LoadByCtx(ctx))
		curAttrs.Delete(name)
		return
	}

	// 个人变量
	if strings.HasPrefix(s, "$m") {
		if ctx.Session != nil && ctx.Player != nil {
			playerAttrs := lo.Must(am.LoadById(ctx.Player.UserID))
			playerAttrs.Delete(name)
			return
		}
	}

	// 群变量
	if ctx.Group != nil && strings.HasPrefix(s, "$g") {
		groupAttrs := lo.Must(am.LoadById(ctx.Group.GroupID))
		groupAttrs.Delete(s)
		return
	}

	curAttrs := lo.Must(am.LoadByCtx(ctx))
	curAttrs.Delete(name)
}

func VarGetValueInt64(ctx *MsgContext, s string) (int64, bool) {
	v, exists := VarGetValue(ctx, s)
	if exists && v.TypeId == ds.VMTypeInt {
		return int64(v.MustReadInt()), true
	}
	return 0, false
}

func VarGetValueStr(ctx *MsgContext, s string) (string, bool) {
	v, exists := _VarGetValueV1(ctx, s)
	if exists && v.TypeID == VMTypeString {
		return v.Value.(string), true
	}
	return "", false
}

func VarGetValue(ctx *MsgContext, s string) (*ds.VMValue, bool) {
	name := ctx.Player.GetValueNameByAlias(s, nil)
	am := ctx.Dice.AttrsManager

	// 临时变量
	if strings.HasPrefix(s, "$t") {
		if ctx.vm != nil {
			v, ok := ctx.vm.Attrs.Load(s)
			if ok {
				return v, ok
			}
		}
		// 跟入群致辞闪退的一个bug有关，当时是报 _v, exists := ctx.Player.ValueMapTemp.Get(s) 这一行 nil pointer
		if ctx.Player.ValueMapTemp == nil {
			ctx.Player.ValueMapTemp = &ds.ValueMap{}
			return nil, false
		}
		if v, ok := ctx.Player.ValueMapTemp.Load(s); ok {
			return v, ok
		}
	}

	// 个人全局变量
	if strings.HasPrefix(s, "$m") {
		if ctx.Session != nil && ctx.Player != nil {
			playerAttrs := lo.Must(am.LoadById(ctx.Player.UserID))
			return playerAttrs.LoadX(name)
		}
	}

	// 群变量
	if ctx.Group != nil && strings.HasPrefix(s, "$g") {
		groupAttrs := lo.Must(am.LoadById(ctx.Group.GroupID))
		return groupAttrs.LoadX(s)
	}

	// 个人群变量
	if ctx.Player != nil {
		curAttrs := lo.Must(am.LoadByCtx(ctx))
		return curAttrs.LoadX(name)
	}

	return nil, false
}

func _VarGetValueV1(ctx *MsgContext, s string) (*VMValue, bool) {
	if v, ok := VarGetValue(ctx, s); ok {
		return dsValueToRollVMv1(v), ok
	}
	return nil, false
}

func (i *GroupPlayerInfo) GetValueNameByAlias(s string, alias map[string][]string) string {
	name := s

	if alias == nil {
		// 当私聊的时候，i就会是nil
		if i != nil && i.TempValueAlias != nil {
			alias = *i.TempValueAlias
		}
	}

	for k, v := range alias {
		if strings.EqualFold(s, k) {
			name = k // 防止一手大小写不一致
			break    // 名字本身就是确定值，不用修改
		}
		for _, i := range v {
			if strings.EqualFold(s, i) {
				name = k
				break
			}
		}
	}

	return name
}

func GetValueNameByAlias(s string, alias map[string][]string) string {
	name := s

	for k, v := range alias {
		if strings.EqualFold(s, k) {
			name = k // 防止一手大小写不一致
			break    // 名字本身就是确定值，不用修改
		}
		for _, i := range v {
			if strings.EqualFold(s, i) {
				name = k
				break
			}
		}
	}

	return name
}

func SetTempVars(ctx *MsgContext, qqNickname string) {
	// 设置临时变量
	if ctx.Player != nil {
		pcName := ctx.Player.Name
		pcName = strings.ReplaceAll(pcName, "\n", "")
		pcName = strings.ReplaceAll(pcName, "\r", "")
		pcName = strings.ReplaceAll(pcName, `\n`, "")
		pcName = strings.ReplaceAll(pcName, `\r`, "")
		pcName = strings.ReplaceAll(pcName, `\f`, "")

		VarSetValueStr(ctx, "$t玩家", fmt.Sprintf("<%s>", pcName))
		if ctx.Dice != nil && !ctx.Dice.PlayerNameWrapEnable {
			VarSetValueStr(ctx, "$t玩家", pcName)
		}
		VarSetValueStr(ctx, "$t玩家_RAW", pcName)
		VarSetValueStr(ctx, "$tQQ昵称", fmt.Sprintf("<%s>", qqNickname))
		VarSetValueStr(ctx, "$t帐号昵称", fmt.Sprintf("<%s>", qqNickname))
		VarSetValueStr(ctx, "$t账号昵称", fmt.Sprintf("<%s>", qqNickname))
		VarSetValueStr(ctx, "$t帐号ID", ctx.Player.UserID)
		VarSetValueStr(ctx, "$t账号ID", ctx.Player.UserID)
		VarSetValueInt64(ctx, "$t个人骰子面数", int64(ctx.Player.DiceSideNum))
		VarSetValueStr(ctx, "$tQQ", ctx.Player.UserID)
		VarSetValueStr(ctx, "$t骰子帐号", ctx.EndPoint.UserID)
		VarSetValueStr(ctx, "$t骰子账号", ctx.EndPoint.UserID)
		VarSetValueStr(ctx, "$t骰子昵称", ctx.EndPoint.Nickname)
		VarSetValueStr(ctx, "$t帐号ID_RAW", UserIDExtract(ctx.Player.UserID))
		VarSetValueStr(ctx, "$t账号ID_RAW", UserIDExtract(ctx.Player.UserID))
		VarSetValueStr(ctx, "$t平台", ctx.EndPoint.Platform)

		rpSeed := (time.Now().Unix() + (8 * 60 * 60)) / (24 * 60 * 60)
		rpSeed += int64(fingerprint(ctx.EndPoint.UserID))
		rpSeed += int64(fingerprint(ctx.Player.UserID))
		randItem := rand.NewSource(rpSeed)
		rp := randItem.Int63()%100 + 1
		VarSetValueInt64(ctx, "$t人品", rp)

		now := time.Now()
		t, _ := strconv.ParseInt(now.Format("20060102"), 10, 64)
		VarSetValueInt64(ctx, "$tDate", t)
		wd := int64(now.Weekday())
		if wd == 0 {
			wd = 7
		}
		VarSetValueInt64(ctx, "$tWeekday", wd)

		t, _ = strconv.ParseInt(now.Format("2006"), 10, 64)
		VarSetValueInt64(ctx, "$tYear", t)
		t, _ = strconv.ParseInt(now.Format("01"), 10, 64)
		VarSetValueInt64(ctx, "$tMonth", t)
		t, _ = strconv.ParseInt(now.Format("02"), 10, 64)
		VarSetValueInt64(ctx, "$tDay", t)
		t, _ = strconv.ParseInt(now.Format("15"), 10, 64)
		VarSetValueInt64(ctx, "$tHour", t)
		t, _ = strconv.ParseInt(now.Format("04"), 10, 64)
		VarSetValueInt64(ctx, "$tMinute", t)
		t, _ = strconv.ParseInt(now.Format("05"), 10, 64)
		VarSetValueInt64(ctx, "$tSecond", t)
		VarSetValueInt64(ctx, "$tTimestamp", now.Unix())
	}

	if ctx.Group != nil {
		if ctx.MessageType == "group" {
			VarSetValueStr(ctx, "$t群号", ctx.Group.GroupID)
			VarSetValueStr(ctx, "$t群名", ctx.Group.GroupName)
			VarSetValueStr(ctx, "$t群号_RAW", UserIDExtract(ctx.Group.GroupID))
		}
		VarSetValueInt64(ctx, "$t群组骰子面数", ctx.Group.DiceSideNum)
		VarSetValueInt64(ctx, "$t当前骰子面数", getDefaultDicePoints(ctx))
		if ctx.Group.System == "" {
			ctx.Group.System = "coc7"
			ctx.Group.UpdatedAtTime = time.Now().Unix()
		}
		VarSetValueStr(ctx, "$t游戏模式", ctx.Group.System)
		VarSetValueStr(ctx, "$t规则模板", ctx.Group.System)
		VarSetValueStr(ctx, "$tSystem", ctx.Group.System)
		VarSetValueStr(ctx, "$t当前记录", ctx.Group.LogCurName)
	}

	if ctx.MessageType == "group" {
		VarSetValueStr(ctx, "$t消息类型", "group")
	} else {
		VarSetValueStr(ctx, "$t消息类型", "private")
	}
}
