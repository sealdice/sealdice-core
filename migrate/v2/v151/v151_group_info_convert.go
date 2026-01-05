package v151

import (
	"encoding/json"
	"fmt"

	"gorm.io/gorm"

	"sealdice-core/model"
	"sealdice-core/utils/constant"
	operator "sealdice-core/utils/dboperator/engine"
	upgrade "sealdice-core/utils/upgrader"
)

// 旧版本群信息的轻量 JSON 映射
type oldGroupInfo struct {
	Active           bool `json:"active"`
	ActivatedExtList []struct {
		Name string `json:"name"`
	} `json:"activatedExtList"`
	InactivatedExtSet   any                 `json:"inactivatedExtSet"`
	GroupId             string              `json:"groupId"`
	GuildId             string              `json:"guildId"`
	ChannelId           string              `json:"channelId"`
	GroupName           string              `json:"groupName"`
	DiceIdActiveMap     map[string]bool     `json:"diceIdActiveMap"`
	DiceIdExistsMap     map[string]bool     `json:"diceIdExistsMap"`
	BotList             map[string]bool     `json:"botList"`
	DiceSideNum         int64               `json:"diceSideNum"`
	DiceSideExpr        string              `json:"diceSideExpr"`
	System              string              `json:"system"`
	HelpPackages        []string            `json:"helpPackages"`
	CocRuleIndex        int                 `json:"cocRuleIndex"`
	LogCurName          string              `json:"logCurName"`
	LogOn               bool                `json:"logOn"`
	RecentDiceSendTime  int64               `json:"recentDiceSendTime"`
	ShowGroupWelcome    bool                `json:"showGroupWelcome"`
	GroupWelcomeMessage string              `json:"groupWelcomeMessage"`
	EnteredTime         int64               `json:"enteredTime"`
	InviteUserId        string              `json:"inviteUserId"`
	DefaultHelpGroup    string              `json:"defaultHelpGroup"`
	PlayerGroups        map[string][]string `json:"playerGroups"`
	ExtAppliedVersion   int64               `json:"extAppliedVersion"`
}

// V151GroupInfoConvertMigration 将旧的 group_info BLOB 数据转换为新的列结构
var V151GroupInfoConvertMigration = upgrade.Upgrade{
	ID:          "008_V151GroupInfoConvertMigration",
	Description: "将旧版 group_info 中的 JSON BLOB 数据转换为新结构化列并落库",
	Apply: func(logf func(string), dbop operator.DatabaseOperator) error {
		db := dbop.GetDataDB(constant.WRITE)
		rdb := dbop.GetDataDB(constant.READ)
		// 先确保新结构所需列存在
		if err := db.AutoMigrate(&model.GroupInfoDB{}); err != nil {
			return err
		}

		type row struct {
			ID        string
			UpdatedAt *int64
			Data      []byte
		}

		var total, okCount, failCount int
		logf("[INFO] V151 群信息转换开始")

		if !rdb.Migrator().HasTable(&model.GroupInfo{}) {
			logf("[INFO] 旧版本数据表不存在，可能曾经已经升级过或版本有问题，升级暂时跳过")
			return nil
		}
		rows, err := rdb.Model(&model.GroupInfo{}).Select("id, updated_at, data").Rows()
		if err != nil {
			return err
		}
		defer rows.Close()
		if err = rows.Err(); err != nil {
			return err
		}
		var tempList []model.GroupInfoDB
		for rows.Next() {
			total++
			var r row
			if err = rows.Scan(&r.ID, &r.UpdatedAt, &r.Data); err != nil {
				failCount++
				continue
			}
			var old oldGroupInfo
			if err = json.Unmarshal(r.Data, &old); err != nil {
				failCount++
				continue
			}
			if old.GroupId == "" {
				logf(fmt.Sprintf("[WARN] 群组 %s 缺少 GroupId，请检查数据", r.ID))
				continue // 不插入这条不完了
			}

			rec := &model.GroupInfoDB{
				ID:                  r.ID,
				Active:              old.Active,
				ActivatedExtList:    []string{},
				InactivatedExtSet:   []string{},
				GroupId:             old.GroupId,
				GuildId:             old.GuildId,
				ChannelId:           old.ChannelId,
				GroupName:           old.GroupName,
				DiceIdActiveMap:     map[string]bool{},
				DiceIdExistsMap:     map[string]bool{},
				BotList:             map[string]bool{},
				DiceSideNum:         int(old.DiceSideNum),
				DiceSideExpr:        old.DiceSideExpr,
				System:              old.System,
				HelpPackages:        old.HelpPackages,
				CocRuleIndex:        old.CocRuleIndex,
				LogCurName:          old.LogCurName,
				LogOn:               old.LogOn,
				RecentDiceSendTime:  old.RecentDiceSendTime,
				ShowGroupWelcome:    old.ShowGroupWelcome,
				GroupWelcomeMessage: old.GroupWelcomeMessage,
				EnteredTime:         old.EnteredTime,
				InviteUserId:        old.InviteUserId,
				DefaultHelpGroup:    old.DefaultHelpGroup,
				PlayerGroups:        map[string][]string{},
				ExtAppliedVersion:   int(old.ExtAppliedVersion),
			}
			if r.UpdatedAt != nil {
				rec.UpdatedAt = *r.UpdatedAt
			}

			for _, w := range old.ActivatedExtList {
				if w.Name != "" {
					rec.ActivatedExtList = append(rec.ActivatedExtList, w.Name)
				}
			}

			switch v := old.InactivatedExtSet.(type) {
			case []string:
				rec.InactivatedExtSet = append(rec.InactivatedExtSet, v...)
			case map[string]any:
				for k := range v {
					rec.InactivatedExtSet = append(rec.InactivatedExtSet, k)
				}
			case map[string]struct{}:
				for k := range v {
					rec.InactivatedExtSet = append(rec.InactivatedExtSet, k)
				}
			default:
				// 忽略其他异常格式
			}

			for k, v := range old.DiceIdActiveMap {
				rec.DiceIdActiveMap[k] = v
			}
			for k, v := range old.DiceIdExistsMap {
				// 过滤历史 BUG "QQ-Group:" 前缀
				if len(k) >= 9 && k[:9] == "QQ-Group:" {
					continue
				}
				rec.DiceIdExistsMap[k] = v
			}
			for k, v := range old.BotList {
				rec.BotList[k] = v
			}
			for k, v := range old.PlayerGroups {
				rec.PlayerGroups[k] = v
			}
			// 不能在这里插入，否则会因为连接池只有一个连接，导致死锁
			tempList = append(tempList, *rec)
		}

		// 进行批量创建，开一个事务
		err = db.Transaction(func(tx *gorm.DB) error {
			if err = tx.CreateInBatches(&tempList, 500).Error; err != nil {
				return err
			}
			err = tx.Migrator().DropTable("group_info")
			if err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			logf(fmt.Sprintf("[INFO] V151 群信息转换失败，原因为 %s", err))
			return err
		}
		logf("删除旧版本的历史遗留数据")
		logf(fmt.Sprintf("[INFO] V151 群信息转换完成，共 %d 条，成功 %d 条，失败 %d 条", total, okCount, failCount))
		return nil
	},
}
