package dice

import (
	"sync"
)

// BuildChannel 发布渠道，是「构建声明的渠道」与「官方构建校验结果」共同得出的结论，
// 不等于 APP_CHANNEL。APP_CHANNEL 只是 ldflags 注入的字符串，自编译时可以随意填写，
// 因此不能单独作为可信度的依据。
type BuildChannel string

const (
	// BuildChannelStable 正式版：通过官方构建校验，且构建声明为 stable
	BuildChannelStable BuildChannel = "stable"
	// BuildChannelDev 开发版：通过官方构建校验，且构建声明为 dev
	BuildChannelDev BuildChannel = "dev"
	// BuildChannelSelfBuilt 自编译：没有注入可信客户端私钥，构建方无法声明官方身份
	BuildChannelSelfBuilt BuildChannel = "self-built"
	// BuildChannelUnknown 未知：注入了私钥，但未能通过官方校验，无法判断来源
	BuildChannelUnknown BuildChannel = "unknown"
)

// BuildVerifyState 官方构建校验状态
type BuildVerifyState string

const (
	// BuildVerifyUnsigned 未注入可信客户端私钥，无需校验，必然不是官方构建
	BuildVerifyUnsigned BuildVerifyState = "unsigned"
	// BuildVerifyPending 已注入私钥，但尚未取得校验结论
	BuildVerifyPending BuildVerifyState = "pending"
	// BuildVerifyPassed 已注入私钥且通过官方校验
	BuildVerifyPassed BuildVerifyState = "passed"
	// BuildVerifyFailed 已注入私钥但未通过官方校验，私钥并非官方签发
	BuildVerifyFailed BuildVerifyState = "failed"
)

var (
	buildVerifyOnce  sync.Once
	buildVerifyMu    sync.RWMutex
	buildVerifyState = BuildVerifyPending
)

// verifyOfficialBuild 向官方服务器确认本机注入的可信客户端私钥确实由官方签发。
//
// 只检查私钥是否为空是不够的：自编译时同样可以用 ldflags 注入一把自制私钥，
// 因此「是否官方构建」必须由持有对应公钥的一方判定，客户端不能自证。
//
// TODO: 校验接口细节尚未确定，待官方后端提供后接入。预期形态为用
// SealTrustedClientPrivateKey 对服务端下发的 nonce 做 EcdsaSignRow 签名并提交，
// 由服务端用对应公钥验签后返回结论。在此之前一律返回 BuildVerifyPending，
// 表示「尚无结论」而不是「校验失败」——把官方构建误判成未知比暂不判定更糟。
func verifyOfficialBuild() BuildVerifyState {
	if len(SealTrustedClientPrivateKey) == 0 {
		return BuildVerifyUnsigned
	}
	return BuildVerifyPending
}

// BuildVerify 返回官方构建校验状态，结论只求取一次并缓存。
func BuildVerify() BuildVerifyState {
	buildVerifyOnce.Do(func() {
		state := verifyOfficialBuild()
		buildVerifyMu.Lock()
		buildVerifyState = state
		buildVerifyMu.Unlock()
	})
	buildVerifyMu.RLock()
	defer buildVerifyMu.RUnlock()
	return buildVerifyState
}

// GetBuildChannel 返回对外展示的发布渠道。
//
// 未注入私钥即自编译，无需联网即可确定。注入了私钥但校验失败的是未知：
// 私钥存在却不被官方认可，来源无法判断，此时不应沿用构建自称的渠道。
// 尚无校验结论时沿用构建声明的渠道，避免官方构建在接口就绪前一律显示为未知。
func GetBuildChannel() BuildChannel {
	switch BuildVerify() {
	case BuildVerifyUnsigned:
		return BuildChannelSelfBuilt
	case BuildVerifyFailed:
		return BuildChannelUnknown
	case BuildVerifyPassed, BuildVerifyPending:
		return declaredBuildChannel()
	}
	return BuildChannelUnknown
}

// declaredBuildChannel 构建时通过 APP_CHANNEL 声明的渠道，只在校验未否定它时使用。
func declaredBuildChannel() BuildChannel {
	switch APP_CHANNEL {
	case "stable":
		return BuildChannelStable
	case "dev":
		return BuildChannelDev
	default:
		return BuildChannelUnknown
	}
}
