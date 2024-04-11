package dice

import (
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

type AddDiscordEcho struct {
	Token              string
	ProxyURL           string
	ReverseProxyUrl    string
	ReverseProxyCDNUrl string
}

// NewDiscordConnItem 本来没必要写这个的，但是不知道为啥依赖出问题
func NewDiscordConnItem(v AddDiscordEcho) *EndPointInfo {
	conn := new(EndPointInfo)
	conn.ID = uuid.New().String()
	conn.Platform = "DISCORD"
	conn.ProtocolType = ""
	conn.Enable = false
	conn.RelWorkDir = "extra/discord-" + conn.ID
	conn.Adapter = &PlatformAdapterDiscord{
		EndPoint:           conn,
		Token:              v.Token,
		ProxyURL:           v.ProxyURL,
		ReverseProxyUrl:    v.ReverseProxyUrl,
		ReverseProxyCDNUrl: v.ReverseProxyCDNUrl,
	}
	return conn
}

// ServeDiscord gocqhttp_helper 中有一个相同的待重构方法，为了避免阻碍重构，先不写在一起了
func ServeDiscord(d *Dice, ep *EndPointInfo) {
	defer CrashLog()
	if ep.Platform == "DISCORD" {
		conn := ep.Adapter.(*PlatformAdapterDiscord)
		d.Logger.Infof("DiscordGo 尝试连接")
		if conn.Serve() != 0 {
			d.Logger.Errorf("连接Discord服务失败")
			ep.State = 3
			ep.Enable = false
			d.LastUpdatedTime = time.Now().Unix()
			d.Save(false)
		}
	}
}

func regenerateDiscordEndPoint(endPointDiscord string) {
	discordgo.EndpointDiscord = endPointDiscord
	discordgo.EndpointAPI = discordgo.EndpointDiscord + "api/v" + discordgo.APIVersion + "/"
	discordgo.EndpointGuilds = discordgo.EndpointAPI + "guilds/"
	discordgo.EndpointChannels = discordgo.EndpointAPI + "channels/"
	discordgo.EndpointUsers = discordgo.EndpointAPI + "users/"
	discordgo.EndpointGateway = discordgo.EndpointAPI + "gateway"
	discordgo.EndpointGatewayBot = discordgo.EndpointGateway + "/bot"
	discordgo.EndpointWebhooks = discordgo.EndpointAPI + "webhooks/"
	discordgo.EndpointStickers = discordgo.EndpointAPI + "stickers/"
	discordgo.EndpointStageInstances = discordgo.EndpointAPI + "stage-instances"
	discordgo.EndpointVoice = discordgo.EndpointAPI + "/voice/"
	discordgo.EndpointVoiceRegions = discordgo.EndpointVoice + "regions"

	discordgo.EndpointGuildCreate = discordgo.EndpointAPI + "guilds"

	discordgo.EndpointApplications = discordgo.EndpointAPI + "applications"

	discordgo.EndpointOAuth2 = discordgo.EndpointAPI + "oauth2/"
	discordgo.EndpointOAuth2Applications = discordgo.EndpointOAuth2 + "applications"
}

func regenerateDiscordEndPointCDN(endPointDiscordCDN string) {
	discordgo.EndpointCDN = endPointDiscordCDN
	discordgo.EndpointCDNAttachments = discordgo.EndpointCDN + "attachments/"
	discordgo.EndpointCDNAvatars = discordgo.EndpointCDN + "avatars/"
	discordgo.EndpointCDNIcons = discordgo.EndpointCDN + "icons/"
	discordgo.EndpointCDNSplashes = discordgo.EndpointCDN + "splashes/"
	discordgo.EndpointCDNChannelIcons = discordgo.EndpointCDN + "channel-icons/"
	discordgo.EndpointCDNBanners = discordgo.EndpointCDN + "banners/"
	discordgo.EndpointCDNGuilds = discordgo.EndpointCDN + "guilds/"
}
