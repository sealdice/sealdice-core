package dice

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/slack-go/slack"
	sm "github.com/slack-go/slack/socketmode"
)

func NewSlackConnItem(at string, bt string) *EndPointInfo {
	conn := new(EndPointInfo)
	conn.ID = uuid.New().String()
	conn.Platform = "SLACK"
	conn.ProtocolType = ""
	conn.Enable = false
	conn.RelWorkDir = "extra/slack-" + conn.ID
	conn.Adapter = &PlatformAdapterSlack{
		EndPoint: conn,
		AppToken: at,
		BotToken: bt,
	}
	return conn
}

func ServeSlack(d *Dice, ep *EndPointInfo) {
	defer CrashLog()
	if ep.Platform == "SLACK" {
		conn := ep.Adapter.(*PlatformAdapterSlack)
		conn.Session = d.ImSession
		conn.EndPoint = ep
		if conn.Serve() != 0 {
			ep.State = 3
			//ep.Enable = false
			d.LastUpdatedTime = time.Now().Unix()
			d.Save(false)
			d.Logger.Info("连接失败！")
		}
	}
}

func middlewareSlashCommand(evt *sm.Event, client *sm.Client) {
	cmd, ok := evt.Data.(slack.SlashCommand)
	if !ok {
		fmt.Printf("Ignored %+v\n", evt)
		return
	}

	client.Debugf("Slash command received: %+v", cmd)

	payload := map[string]interface{}{
		"blocks": []slack.Block{
			slack.NewSectionBlock(
				&slack.TextBlockObject{
					Type: slack.MarkdownType,
					Text: "foo",
				},
				nil,
				slack.NewAccessory(
					slack.NewButtonBlockElement(
						"",
						"somevalue",
						&slack.TextBlockObject{
							Type: slack.PlainTextType,
							Text: "bar",
						},
					),
				),
			),
		}}
	client.Ack(*evt.Request, payload)
}
