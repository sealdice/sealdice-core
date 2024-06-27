package dice

type AdvancedConfig struct {
	Show   bool `json:"show" yaml:"show"`     // 显示高级设置页
	Enable bool `json:"enable" yaml:"enable"` // 启用高级设置

	// 跑团日志相关

	StoryLogBackendUrl   string `json:"storyLogBackendUrl" yaml:"storyLogBackendUrl"`     // 自定义后端地址
	StoryLogApiVersion   string `json:"storyLogApiVersion" yaml:"storyLogApiVersion"`     // 后端 api 版本
	StoryLogBackendToken string `json:"storyLogBackendToken" yaml:"storyLogBackendToken"` // 自定义后端 token

	StoreBackendUrl string `json:"storeBackendUrl" yaml:"storeBackendUrl"` // 自定义商店后端地址
}
