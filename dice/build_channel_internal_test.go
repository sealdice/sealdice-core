package dice

import "testing"

// withBuildVerifyState 临时改写缓存的校验结论，绕过 sync.Once 以便覆盖各分支。
func withBuildVerifyState(t *testing.T, state BuildVerifyState) {
	t.Helper()
	buildVerifyOnce.Do(func() {})
	buildVerifyMu.Lock()
	prev := buildVerifyState
	buildVerifyState = state
	buildVerifyMu.Unlock()
	t.Cleanup(func() {
		buildVerifyMu.Lock()
		buildVerifyState = prev
		buildVerifyMu.Unlock()
	})
}

func withDeclaredChannel(t *testing.T, channel string) {
	t.Helper()
	prev := APP_CHANNEL
	APP_CHANNEL = channel
	t.Cleanup(func() { APP_CHANNEL = prev })
}

func TestGetBuildChannel(t *testing.T) {
	cases := []struct {
		name     string
		state    BuildVerifyState
		declared string
		want     BuildChannel
	}{
		// 未注入私钥即自编译，构建自称什么渠道都不改变结论。
		{"未注入私钥即自编译", BuildVerifyUnsigned, "stable", BuildChannelSelfBuilt},
		{"未注入私钥且声明 dev 仍是自编译", BuildVerifyUnsigned, "dev", BuildChannelSelfBuilt},
		// 校验失败说明私钥非官方签发，此时不能沿用构建声明。
		{"校验失败时不沿用声明的 stable", BuildVerifyFailed, "stable", BuildChannelUnknown},
		{"校验通过按声明区分正式版", BuildVerifyPassed, "stable", BuildChannelStable},
		{"校验通过按声明区分开发版", BuildVerifyPassed, "dev", BuildChannelDev},
		{"校验通过但声明无法识别", BuildVerifyPassed, "nightly", BuildChannelUnknown},
		// 接口就绪前沿用声明，避免官方构建被误判成未知。
		{"尚无结论时沿用声明", BuildVerifyPending, "dev", BuildChannelDev},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			withBuildVerifyState(t, c.state)
			withDeclaredChannel(t, c.declared)
			if got := GetBuildChannel(); got != c.want {
				t.Fatalf("期望 %q，实际 %q", c.want, got)
			}
		})
	}
}

func TestVerifyOfficialBuildWithoutKey(t *testing.T) {
	prev := SealTrustedClientPrivateKey
	SealTrustedClientPrivateKey = ""
	t.Cleanup(func() { SealTrustedClientPrivateKey = prev })

	if got := verifyOfficialBuild(); got != BuildVerifyUnsigned {
		t.Fatalf("未注入私钥应为 %q，实际 %q", BuildVerifyUnsigned, got)
	}
}
