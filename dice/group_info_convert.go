package dice

import (
	"sealdice-core/model"
)

func GroupInfoFromDB(rec *model.GroupInfoDB, d *Dice) *GroupInfo {
	g := &GroupInfo{}
	g.GroupID = rec.ID
	if rec.GroupId != "" {
		g.GroupID = rec.GroupId
	}
	g.Active = rec.Active
	g.GuildID = rec.GuildId
	g.ChannelID = rec.ChannelId
	g.GroupName = rec.GroupName
	g.DiceSideExpr = rec.DiceSideExpr
	g.System = rec.System
	g.HelpPackages = rec.HelpPackages
	g.CocRuleIndex = rec.CocRuleIndex
	g.LogCurName = rec.LogCurName
	g.LogOn = rec.LogOn
	g.RecentDiceSendTime = rec.RecentDiceSendTime
	g.ShowGroupWelcome = rec.ShowGroupWelcome
	g.GroupWelcomeMessage = rec.GroupWelcomeMessage
	g.EnteredTime = rec.EnteredTime
	g.InviteUserID = rec.InviteUserId
	g.DefaultHelpGroup = rec.DefaultHelpGroup
	g.ExtAppliedVersion = int64(rec.ExtAppliedVersion)
	g.UpdatedAtTime = 0
	if rec.DiceSideNum != 0 {
		g.DiceSideNum = int64(rec.DiceSideNum)
	}
	g.DiceIDActiveMap = new(SyncMap[string, bool])
	for k, v := range rec.DiceIdActiveMap {
		g.DiceIDActiveMap.Store(k, v)
	}
	g.DiceIDExistsMap = new(SyncMap[string, bool])
	for k, v := range rec.DiceIdExistsMap {
		if len(k) >= 9 && k[:9] == "QQ-Group:" {
			continue
		}
		g.DiceIDExistsMap.Store(k, v)
	}
	g.BotList = new(SyncMap[string, bool])
	for k, v := range rec.BotList {
		g.BotList.Store(k, v)
	}
	g.PlayerGroups = new(SyncMap[string, []string])
	for k, v := range rec.PlayerGroups {
		g.PlayerGroups.Store(k, v)
	}
	if g.InactivatedExtSet == nil {
		g.InactivatedExtSet = StringSet{}
	}
	for _, name := range rec.InactivatedExtSet {
		g.InactivatedExtSet[name] = struct{}{}
	}
	var wrappers []*ExtInfo
	for _, name := range rec.ActivatedExtList {
		if name == "" {
			continue
		}
		wrappers = append(wrappers, &ExtInfo{Name: name, IsWrapper: true, TargetName: name})
	}
	g.SetActivatedExtList(wrappers, d)
	return g
}

func GroupInfoToDB(g *GroupInfo) *model.GroupInfoDB {
	rec := &model.GroupInfoDB{
		ID:                  g.GroupID,
		UpdatedAt:           g.UpdatedAtTime,
		Active:              g.Active,
		ActivatedExtList:    []string{},
		InactivatedExtSet:   []string{},
		GroupId:             g.GroupID,
		GuildId:             g.GuildID,
		ChannelId:           g.ChannelID,
		GroupName:           g.GroupName,
		DiceIdActiveMap:     map[string]bool{},
		DiceIdExistsMap:     map[string]bool{},
		BotList:             map[string]bool{},
		DiceSideNum:         int(g.DiceSideNum),
		DiceSideExpr:        g.DiceSideExpr,
		System:              g.System,
		HelpPackages:        g.HelpPackages,
		CocRuleIndex:        g.CocRuleIndex,
		LogCurName:          g.LogCurName,
		LogOn:               g.LogOn,
		RecentDiceSendTime:  g.RecentDiceSendTime,
		ShowGroupWelcome:    g.ShowGroupWelcome,
		GroupWelcomeMessage: g.GroupWelcomeMessage,
		EnteredTime:         g.EnteredTime,
		InviteUserId:        g.InviteUserID,
		DefaultHelpGroup:    g.DefaultHelpGroup,
		PlayerGroups:        map[string][]string{},
		ExtAppliedVersion:   int(g.ExtAppliedVersion),
	}
	g.DiceIDActiveMap.Range(func(key string, value bool) bool {
		rec.DiceIdActiveMap[key] = value
		return true
	})
	g.DiceIDExistsMap.Range(func(key string, value bool) bool {
		rec.DiceIdExistsMap[key] = value
		return true
	})
	g.BotList.Range(func(key string, value bool) bool {
		rec.BotList[key] = value
		return true
	})
	g.PlayerGroups.Range(func(key string, value []string) bool {
		rec.PlayerGroups[key] = value
		return true
	})
	for name := range g.InactivatedExtSet {
		rec.InactivatedExtSet = append(rec.InactivatedExtSet, name)
	}
	for _, ext := range g.GetActivatedExtListRaw() {
		if ext != nil && !ext.IsDeleted && ext.Name != "" {
			rec.ActivatedExtList = append(rec.ActivatedExtList, ext.Name)
		}
	}
	return rec
}
