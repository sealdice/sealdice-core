package spinner

import (
	"strings"
	"time"

	"github.com/schollz/progressbar/v3"
)

// Lines 供spinner显示的句子
//
// 将在数组中的所有句子中循环使用.
var Lines = []string{
	"青，取之于蓝，而青于蓝。正在更新海豹",
	"剑阁峥嵘而崔嵬，一夫当关，万夫莫开。全新海豹准备中",
	"小荷才露尖尖角，早有蜻蜓立上头。请前往news查看海豹更新日志",
	"海内存知己，天涯若比邻。海豹正在更新，请稍候",
	"有朋自远方来，不亦乐乎。欢迎你更新海豹Dice",
	"休对故人思故国，且将新火试新茶。敬请开始吧",
	"千门万户曈曈日，总把新桃换旧符。海豹更新中，稍等片刻",
}

// SpinnerWithLines 执行耗时任务时在命令行显示一个spinner, 并在最后循环显示Lines中的句子
//   - `chDone` 当耗时任务完成后, 通过这个channel传入一个结束信号
//   - `secondPerLine` 每个句子显示的秒数
//   - `dotPerLine` 每个句子后显示的句点数
//
// Usage:
//
//	go SpinnerWithLines(chDone, 3, 10)
func SpinnerWithLines(chDone <-chan interface{}, secondPerLine int, dotPerLine int) {
	if len(Lines) == 0 {
		panic("SpinnerWithLines cannot be called when Lines is empty")
	}
	if secondPerLine <= 0 {
		panic("SpinnerWithLines cannot be called with secondPerLine <= 0")
	}
	if dotPerLine <= 0 {
		panic("SpinnerWithLines cannot be called with dotPerLine <= 0")
	}

	secondPerLine *= 10
	bar := progressbar.NewOptions(-1, // spinner
		progressbar.OptionSpinnerType(47),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowBytes(false),
		progressbar.OptionSetElapsedTime(true),
		progressbar.OptionShowDescriptionAtLineEnd(),
	)
	it := 0
	for {
		select {
		case <-chDone:
			bar.Exit()
			return
		default:
			bar.Add(1)
			bar.Describe(Lines[(it/secondPerLine)%len(Lines)] + strings.Repeat(".", (it%secondPerLine)/(secondPerLine/dotPerLine)+1))
			it++
			time.Sleep(time.Millisecond * 100) // processbar的spinner并不是随add轮转,而是time/100ms mod size选择要使用的
		}
	}

}
