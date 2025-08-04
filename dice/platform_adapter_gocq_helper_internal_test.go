package dice

import "testing"

var testSignServerConfig = &SignServerConfig{
	SignServers: []*SignServer{
		{
			URL:           "http://127.0.0.1:8080",
			Key:           "114514",
			Authorization: "-",
		},
		{
			URL:           "https://signserver.example.com",
			Key:           "114514",
			Authorization: "Bearer xxxx",
		},
	},
	RuleChangeSignServer: 1,
	MaxCheckCount:        0,
	SignServerTimeout:    60,
	AutoRegister:         false,
	AutoRefreshToken:     false,
	RefreshInterval:      40,
}

func Test_generateOldSignServerConfigStr(t *testing.T) {
	type args struct {
		config *SignServerConfig
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"default",
			args{config: testSignServerConfig},
			`
  # 旧版签名服务相关配置信息
  sign-server: 'http://127.0.0.1:8080'
  # 如果签名服务器的版本在1.1.0及以下, 请将下面的参数改成true
  # 该字段在新签名配置信息中也存在，防止重复此处不配置
  # is-below-110: false
  # 签名服务器所需要的apikey, 如果签名服务器的版本在1.1.0及以下则此项无效
  key: '114514'
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := generateOldSignServerConfigStr(tt.args.config); got != tt.want {
				t.Errorf("generateOldSignServerConfigStr() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_generateNewSignServerConfigStr(t *testing.T) {
	type args struct {
		config *SignServerConfig
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"default",
			args{config: testSignServerConfig},
			`  # 新版签名服务相关配置信息
  # 数据包的签名服务器列表，第一个作为主签名服务器，后续作为备用
  # 兼容 https://github.com/fuqiuluo/unidbg-fetch-qsign
  # 如果遇到 登录 45 错误, 或者发送信息风控的话需要填入一个或多个服务器
  # 不建议设置过多，设置主备各一个即可，超过 5 个只会取前五个
  # 示例:
  # sign-servers:
  #   - url: 'http://127.0.0.1:8080' # 本地签名服务器
  #     key: "114514"  # 相应 key
  #     authorization: "-"   # authorization 内容, 依服务端设置
  #   - url: 'https://signserver.example.com' # 线上签名服务器
  #     key: "114514"
  #     authorization: "-"
  #   ...
  #
  # 服务器可使用docker在本地搭建或者使用他人开放的服务
  sign-servers:
    - url: 'http://127.0.0.1:8080'
      key: "114514"
      authorization: "-"
    - url: 'https://signserver.example.com'
      key: "114514"
      authorization: "Bearer xxxx"

  # 判断签名服务不可用（需要切换）的额外规则
  # 0: 不设置 （此时仅在请求无法返回结果时判定为不可用）
  # 1: 在获取到的 sign 为空 （若选此建议关闭 auto-register，一般为实例未注册但是请求签名的情况）
  # 2: 在获取到的 sign 或 token 为空（若选此建议关闭 auto-refresh-token ）
  rule-change-sign-server: 1

  # 连续寻找可用签名服务器最大尝试次数
  # 为 0 时会在连续 3 次没有找到可用签名服务器后保持使用主签名服务器，不再尝试进行切换备用
  # 否则会在达到指定次数后 **退出** 主程序
  max-check-count: 0
  # 签名服务请求超时时间(s)
  sign-server-timeout: 60
  # 如果签名服务器的版本在1.1.0及以下, 请将下面的参数改成true
  # 建议使用 1.1.6 以上版本，低版本普遍半个月冻结一次
  is-below-110: false
  # 在实例可能丢失（获取到的签名为空）时是否尝试重新注册
  # 为 true 时，在签名服务不可用时可能每次发消息都会尝试重新注册并签名。
  # 为 false 时，将不会自动注册实例，在签名服务器重启或实例被销毁后需要重启 go-cqhttp 以获取实例
  # 否则后续消息将不会正常签名。关闭此项后可以考虑开启签名服务器端 auto_register 避免需要重启
  # 由于实现问题，当前建议关闭此项，推荐开启签名服务器的自动注册实例
  auto-register: false
  # 是否在 token 过期后立即自动刷新签名 token（在需要签名时才会检测到，主要防止 token 意外丢失）
  # 独立于定时刷新
  auto-refresh-token: false
  # 定时刷新 token 间隔时间，单位为分钟, 建议 30~40 分钟, 不可超过 60 分钟
  # 目前丢失token也不会有太大影响，可设置为 0 以关闭，推荐开启
  refresh-interval: 40
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := generateNewSignServerConfigStr(tt.args.config); got != tt.want {
				t.Errorf("generateNewSignServerConfigStr() = %v, want %v", got, tt.want)
			}
		})
	}
}
