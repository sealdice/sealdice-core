package dice

import (
	"reflect"
	"testing"
)

func TestDefaultNoticeIDsIncludeDefaultDiceMasters(t *testing.T) {
	if !reflect.DeepEqual(DefaultConfig.NoticeIDs, DefaultConfig.DiceMasters) {
		t.Fatalf(
			"新安装的通知列表应与默认骰主列表一致，diceMasters=%v noticeIds=%v",
			DefaultConfig.DiceMasters,
			DefaultConfig.NoticeIDs,
		)
	}
}
