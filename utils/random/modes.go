package random

import "strings"

type Mode string

const (
	ModePCG    Mode = "pcg"
	ModeGM     Mode = "gm"
	ModeNIST   Mode = "nist"
	ModeCRNG   Mode = "crng"
	ModeHybrid Mode = "hybrid"
)

type ModeSpec struct {
	Label       string
	Algorithm   string
	Standard    string
	ShortDesc   string
	Description string
}

var supportedModes = []Mode{
	ModePCG,
	ModeGM,
	ModeNIST,
	ModeCRNG,
	ModeHybrid,
}

func SupportedModes() []Mode {
	return append([]Mode(nil), supportedModes...)
}

func NormalizeMode(raw string) Mode {
	switch Mode(strings.ToLower(strings.TrimSpace(raw))) {
	case ModeGM:
		return ModeGM
	case ModeNIST:
		return ModeNIST
	case ModeCRNG:
		return ModeCRNG
	case ModeHybrid:
		return ModeHybrid
	default:
		return ModePCG
	}
}

func ParseModeStrict(raw string) (Mode, bool) {
	switch Mode(strings.ToLower(strings.TrimSpace(raw))) {
	case ModePCG:
		return ModePCG, true
	case ModeGM:
		return ModeGM, true
	case ModeNIST:
		return ModeNIST, true
	case ModeCRNG:
		return ModeCRNG, true
	case ModeHybrid:
		return ModeHybrid, true
	default:
		return "", false
	}
}

func HybridBaseModes() []Mode {
	return []Mode{
		ModePCG,
		ModeGM,
		ModeNIST,
		ModeCRNG,
	}
}

func ModeSpecFor(mode Mode) ModeSpec {
	switch mode {
	case ModeGM:
		return ModeSpec{
			Label:     "GM 国密",
			Algorithm: "SM3 Hash DRBG",
			Standard:  "GM/T 0105-2021（并结合 SP 800-90B 健康检测）",
			ShortDesc: "GM 国密，SM3 Hash DRBG",
			Description: "符合国内商密行业标准 GM/T 0105-2021 的国密随机方案，采用 SM3 Hash DRBG，并结合多源熵输入、" +
				"已知答案自检与 SP 800-90B 健康检测机制；兼容国密/商密体系，在生成质量、规范一致性和安全设计方面表现极为突出" +
				"，但计算、初始化开销相对较高。",
		}
	case ModeNIST:
		return ModeSpec{
			Label:     "NIST",
			Algorithm: "AES-CTR-DRBG",
			Standard:  "NIST SP 800-90A Rev.1（启用 prediction resistance、自检、连续健康检测与主动重播种策略）",
			ShortDesc: "NIST，AES-CTR-DRBG",
			Description: "基于 NIST SP 800-90A Rev.1 的 AES-CTR-DRBG 增强方案，本质上以 AES 计数器模式维护内部状态，采用 AES-256、实例级 personalization、" +
				"prediction resistance、已知答案自检、连续健康检测、主动重播种与密钥轮换策略，" +
				"在标准一致性、持续保护与审计可解释性方面表现突出。",
		}
	case ModeCRNG:
		return ModeSpec{
			Label:     "系统级 CRNG",
			Algorithm: "操作系统 CSPRNG/CRNG 接口",
			Standard:  "Linux 优先 getrandom(2) 内核 CRNG，必要时回退 /dev/urandom；Windows 使用 ProcessPrng API（系统 DRBG）",
			ShortDesc: "操作系统 CSPRNG/CRNG",
			Description: "直接使用操作系统提供的密码学安全随机数接口，随机性由内核或系统安全子系统维护的 CRNG/CSPRNG 提供。" +
				"Linux 路径优先走 getrandom(2) 从内核 CRNG 取数，旧环境可能回退到 /dev/urandom；" +
				"Windows 路径调用 ProcessPrng API，从系统级 DRBG 取数。" +
				"安全边界清晰、平台集成度高，性能通常介于高吞吐 PCG 类置换同余生成器与较重型标准 DRBG 实现之间。",
		}
	case ModeHybrid:
		return ModeSpec{
			Label:     "Hybrid 混合",
			Algorithm: "对所有可用随机源输出做按位异或混合",
			Standard:  "组合模式：混合 PCG、GM 国密、NIST、系统级随机源等全部可用来源",
			ShortDesc: "多源异或混合，性能最差",
			Description: "Hybrid 混合模式通过将多个高质量随机源进行按位异或混合，其输出的密码学随机性不会低于任何一个单独的源。" +
				"只要其中一个源是真正不可预测的，最终结果就是不可预测的。即使某个源被攻破或存在弱点，其他源的随机性仍能保证整体安全性。性能最差。",
		}
	case ModePCG:
		fallthrough
	default:
		return ModeSpec{
			Label:     "默认 PCG",
			Algorithm: "PCG",
			Standard:  "通过 TestU01、PractRand 等常见随机性测试",
			ShortDesc: "高性能置换同余发生器",
			Description: "基于置换同余发生器的高性能随机算法，程序结合多维运行时信息生成初始种子以保证其不可预测性，" +
				"以状态推进与输出置换设计，兼顾速度与分布质量。" +
				"通过 TestU01、PractRand 等常见随机性测试，在很多维度上比传统 rand() 或部分老牌发生器更均匀稳定；支持极高吞吐随机调用。",
		}
	}
}
