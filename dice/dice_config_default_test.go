package dice_test

import (
	"reflect"
	"testing"

	"sealdice-core/dice"
)

func TestDefaultNoticeIDsIncludeDefaultDiceMasters(t *testing.T) {
	if !reflect.DeepEqual(dice.DefaultConfig.NoticeIDs, dice.DefaultConfig.DiceMasters) {
		t.Fatalf(
			"新安装的通知列表应与默认骰主列表一致，diceMasters=%v noticeIds=%v",
			dice.DefaultConfig.DiceMasters,
			dice.DefaultConfig.NoticeIDs,
		)
	}
}
