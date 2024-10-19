package dice

import "github.com/Masterminds/semver/v3"

var (
	APPNAME = "SealDice"

	// VERSION 版本号，按固定格式，action 在构建时可能会自动注入部分信息
	// 正式：主版本号+yyyyMMdd，如 1.4.5+20240308
	// dev：主版本号-dev+yyyyMMdd.7位hash，如 1.4.5-dev+20240308.1a2b3c4
	// rc：主版本号-rc.序号+yyyyMMdd.7位hash如 1.4.5-rc.0+20240308.1a2b3c4，1.4.5-rc.1+20240309.2a3b4c4，……
	VERSION = semver.MustParse(VERSION_MAIN + VERSION_PRERELEASE + VERSION_BUILD_METADATA)

	// VERSION_MAIN 主版本号
	VERSION_MAIN = "1.5.0"
	// VERSION_PRERELEASE 先行版本号
	VERSION_PRERELEASE = "-dev"
	// VERSION_BUILD_METADATA 版本编译信息
	VERSION_BUILD_METADATA = ""

	// APP_CHANNEL 更新频道，stable/dev，在 action 构建时自动注入
	APP_CHANNEL = "dev" //nolint:revive

	VERSION_CODE = int64(1004006) //nolint:revive

	VERSION_JSAPI_COMPATIBLE = []*semver.Version{
		VERSION,
		semver.MustParse("1.4.7"),
		semver.MustParse("1.4.6"),
		semver.MustParse("1.4.5"),
		semver.MustParse("1.4.4"),
		semver.MustParse("1.4.3"),
	}
)
