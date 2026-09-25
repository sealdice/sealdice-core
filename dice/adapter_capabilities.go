package dice

import (
	"sort"
	"strings"
	"sync"
)

// AdapterEventSpec 描述适配器可发射的一个事件。
type AdapterEventSpec struct {
	Name        string `jsbind:"name"        json:"name"`
	Description string `jsbind:"description" json:"description"`
	RequestOnly bool   `jsbind:"requestOnly" json:"request_only"` // 请求类（仅通知）
}

// AdapterRawActionSpec 描述适配器支持的一个 sendRaw 动作。
type AdapterRawActionSpec struct {
	Name        string            `jsbind:"name"        json:"name"`
	Description string            `jsbind:"description" json:"description"`
	Params      map[string]string `jsbind:"params"      json:"params,omitempty"` // 参数名 -> 类型说明
}

// AdapterCapabilitySet 一个适配器（按 ProtocolType 区分，如 milky/onebot/gocq）的能力清单。
type AdapterCapabilitySet struct {
	ProtocolType string                          `json:"protocol_type"` // 如 "milky"、"onebot"、"gocq"
	Platform     string                          `json:"platform"`      // 如 "QQ"、"DISCORD"
	EmitEvents   map[string]AdapterEventSpec     `json:"emit_events"`
	RawActions   map[string]AdapterRawActionSpec `json:"raw_actions"`
}

var (
	adapterCapabilityMu   sync.RWMutex
	adapterCapabilities   = map[string]AdapterCapabilitySet{} // key: ProtocolType
	adapterCapabilityOnce sync.Once
)

// ensureAdapterCapabilitiesRegistered 惰性注册各适配器能力（替代 init()，规避 gochecknoinits）。
func ensureAdapterCapabilitiesRegistered() {
	adapterCapabilityOnce.Do(registerAllAdapterCapabilities)
}

func registerAllAdapterCapabilities() {
	registerMilkyAdapterCapabilities()
	registerOnebotAdapterCapabilities()
	registerGocqAdapterCapabilities()
	registerMiscAdapterCapabilities()
}

// RegisterAdapterCapabilities 注册适配器能力清单。适配器在各自文件的 init() 中调用。
// 重复注册同一 ProtocolType 时以后注册者为准（测试中允许重复注册）。
func RegisterAdapterCapabilities(set AdapterCapabilitySet) {
	adapterCapabilityMu.Lock()
	defer adapterCapabilityMu.Unlock()
	adapterCapabilities[set.ProtocolType] = set
}

// GetAdapterCapabilities 按协议类型查询能力清单。
func GetAdapterCapabilities(protocolType string) (AdapterCapabilitySet, bool) {
	ensureAdapterCapabilitiesRegistered()
	adapterCapabilityMu.RLock()
	defer adapterCapabilityMu.RUnlock()
	set, ok := adapterCapabilities[protocolType]
	return set, ok
}

// adapterCapabilitiesFor 按端点查询能力清单：优先按 ProtocolType，为空时回退到平台小写名。
func adapterCapabilitiesFor(ep *EndPointInfo) (AdapterCapabilitySet, bool) {
	if ep.ProtocolType != "" {
		if set, ok := GetAdapterCapabilities(ep.ProtocolType); ok {
			return set, true
		}
	}
	return GetAdapterCapabilities(strings.ToLower(ep.Platform))
}

// GetAdapterCapabilitiesByPlatform 按平台聚合该平台下所有协议的能力清单（能力查询 UI/JS 用）。
func GetAdapterCapabilitiesByPlatform(platform string) []AdapterCapabilitySet {
	ensureAdapterCapabilitiesRegistered()
	adapterCapabilityMu.RLock()
	defer adapterCapabilityMu.RUnlock()
	var out []AdapterCapabilitySet
	for _, set := range adapterCapabilities {
		if set.Platform == platform {
			out = append(out, set)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ProtocolType < out[j].ProtocolType })
	return out
}
