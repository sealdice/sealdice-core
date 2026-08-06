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
			Label:     "国密 SM3",
			Algorithm: "SM3 Hash DRBG",
			Standard:  "GM/T 0105-2021（并结合 SP 800-90B 连续健康检测）",
			ShortDesc: "国密 SM3 Hash DRBG",
			Description: "符合国内商用密码行业标准 GM/T 0105-2021 的随机数生成方案，核心采用国家密码管理局发布的 SM3 杂凑算法构建 Hash DRBG。" +
				"SM3 广泛部署于二代居民身份证、电子护照、金融 IC 卡、电子发票等国密基础设施，是国产密码体系的基础组件。" +
				"结合多源熵输入、已知答案自检（KAT）与 SP 800-90B 连续健康检测机制，在生成质量、规范一致性与安全设计方面表现突出，" +
				"代价是计算与初始化开销相对较高。适合对合规性与国产密码体系有要求的场景。",
		}
	case ModeNIST:
		return ModeSpec{
			Label:     "AES-CTR-DRBG (NIST)",
			Algorithm: "AES-256-CTR-DRBG",
			Standard:  "NIST SP 800-90A Rev.1",
			ShortDesc: "AES-256 计数器模式 DRBG",
			Description: "基于 AES-256-CTR-DRBG 算法，符合美国国家标准与技术研究院（NIST）发布的 SP 800-90A Rev.1 规范。" +
				"以 AES-256 分组密码在计数器（CTR）模式下维护内部状态的确定性随机位生成器；" +
				"AES 是目前全球应用最广的对称加密算法，CTR-DRBG 广泛部署于 TLS/HTTPS、磁盘加密、安全通信等通用密码学场景中。" +
				"本实现额外启用了 prediction resistance（预测抵抗）、实例级 personalization、已知答案自检、连续健康检测、" +
				"定时重播种与密钥轮换等增强策略，在标准一致性、持续保护与审计可解释性方面表现突出。",
		}
	case ModeCRNG:
		return ModeSpec{
			Label:     "操作系统 CRNG",
			Algorithm: "操作系统密码学安全随机数接口",
			Standard:  "Linux getrandom(2) 内核 CRNG（必要时回退 /dev/urandom）；Windows ProcessPrng API",
			ShortDesc: "操作系统内核密码学随机源",
			Description: "不自行实现随机数算法，而是直接向操作系统内核索取。内核拥有用户态程序无法企及的硬件访问权限，" +
				"自系统启动起即持续采集设备 I/O 时序、硬件中断间隔、热噪声等物理熵，经混合搅拌后维护一个全系统共享的密码学随机池。" +
				"内核自身的内存地址随机化（ASLR）、网络协议栈密钥、文件系统加密密钥等同样依赖该随机源，" +
				"其安全性由内核持续维护并经长期的工程验证。性能通常介于高吞吐的 PCG 与较重型的标准 DRBG 实现之间。",
		}
	case ModeHybrid:
		return ModeSpec{
			Label:     "混合模式 (Hybrid)",
			Algorithm: "多源输出按位异或（XOR）混合",
			Standard:  "组合模式：混合 PCG、国密 SM3、AES-CTR-DRBG、操作系统 CRNG 等全部可用来源",
			ShortDesc: "多源异或混合，安全冗余最高、性能最低",
			Description: "将上述多个高质量随机源的输出按位异或（XOR）混合。依据概率论基本结论：若各随机源相互独立，" +
				"且其中至少一个为真随机（均匀分布），则其异或结果必为真随机——无论其余源的分布如何。" +
				"这意味着即使某个源存在偏差、被攻破甚至完全可预测，只要还有一个源保持真随机，最终输出就不可预测。" +
				"这一性质也是一次一密（OTP）密码体制的数学基础，香农于 1949 年正是基于它证明 OTP 达到完美保密。" +
				"各参与源基于不同算法与不同熵路径，具备相互独立性。以最低的性能为代价换取最高的安全冗余，" +
				"适合对随机性安全性有极致要求的场景。",
		}
	case ModePCG:
		fallthrough
	default:
		return ModeSpec{
			Label:     "PCG（默认）",
			Algorithm: "PCG（置换同余发生器）",
			Standard:  "通过 TestU01 BigCrush、PractRand 等主流随机性统计测试套件",
			ShortDesc: "高性能统计随机数生成器（默认）",
			Description: "基于置换同余发生器（Permuted Congruential Generator, PCG）的随机算法，由计算机科学家 Melissa O'Neill 于 2014 年提出。" +
				"通过状态推进与输出置换设计，兼顾生成速度与分布质量，已通过 TestU01 BigCrush、PractRand 等主流随机性统计测试套件，" +
				"在多维分布均匀性上优于传统 rand() 等老式发生器。初始种子由运行时多维信息（时间戳、内存对象地址、进程 ID、调用栈等）" +
				"混合生成，以保证启动时的不可预测性。非密码学安全，但骰点所需的统计均匀性已得到充分保证，且具备极高的吞吐能力，" +
				"是多数场景下的推荐选择。",
		}
	}
}
