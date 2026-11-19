//nolint:testpackage // Verify QQ's material upload protocol and outgoing messages together.
package dice

import (
	"bytes"
	"context"
	"crypto/md5"  //nolint:gosec // Required by the upload protocol.
	"crypto/sha1" //nolint:gosec // Required by the upload protocol.
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/sealdice/botgo/dto"
	"github.com/sealdice/botgo/dto/keyboard"
	"github.com/sealdice/botgo/openapi"
	"github.com/sealdice/botgo/openapi/options"

	"sealdice-core/message"
)

const testQQImageURL = "qqbot-media://image/v1/test-image"
const testQQInlineImage = "![图片 #2px #2px](" + testQQImageURL + ")"

type officialQQMediaTestAPI struct {
	openapi.OpenAPI
	messages      []*dto.MessageToCreate
	uploads       [][]byte
	prepared      map[string]any
	prepareCalls  int
	completeCalls int
	putURL        string
	existing      []officialQQImageReference
	failStage     string
}

func (api *officialQQMediaTestAPI) Transport(_ context.Context, method, target string, body any) ([]byte, error) {
	switch {
	case method == http.MethodGet && strings.HasSuffix(target, "/v2/media-images"):
		return json.Marshal(map[string]any{"images": api.existing})
	case method == http.MethodPost && strings.HasSuffix(target, "/v2/media-images/prepare"):
		api.prepareCalls++
		if api.failStage == "prepare" {
			return []byte(`{"code":40035007,"message":"image quota exceeded"}`), nil
		}
		api.prepared = body.(map[string]any)
		return json.Marshal(map[string]any{"upload_id": "upload", "presigned_url": api.putURL})
	case method == http.MethodPost && strings.HasSuffix(target, "/v2/media-images/complete"):
		api.completeCalls++
		if api.failStage == "complete" {
			return []byte(`{"code":40035003,"message":"upload mismatch"}`), nil
		}
		request := body.(map[string]string)
		if request["upload_id"] != "upload" || request["md5"] != api.prepared["md5"] {
			return nil, fmt.Errorf("invalid completion: %#v", request)
		}
		return json.Marshal(map[string]any{"url": testQQImageURL, "ttl": 7776000})
	default:
		return nil, fmt.Errorf("unexpected request: %s %s", method, target)
	}
}

func (api *officialQQMediaTestAPI) PostGroupMessage(_ context.Context, _ string, msg dto.APIMessage, _ ...options.Option) (*dto.Message, error) {
	copyMsg := *msg.(*dto.MessageToCreate)
	api.messages = append(api.messages, &copyMsg)
	return &dto.Message{ID: fmt.Sprintf("sent-%d", len(api.messages))}, nil
}

func (api *officialQQMediaTestAPI) PostC2CMessage(ctx context.Context, target string, msg dto.APIMessage, opt ...options.Option) (*dto.Message, error) {
	return api.PostGroupMessage(ctx, target, msg, opt...)
}

func officialQQTestLocalImage(t *testing.T) []byte {
	t.Helper()
	t.Chdir(t.TempDir())
	if err := os.MkdirAll("data/images", 0o755); err != nil {
		t.Fatal(err)
	}
	var data bytes.Buffer
	if err := jpeg.Encode(&data, image.NewRGBA(image.Rect(0, 0, 2, 2)), nil); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("data/images/zcnydhcg.jpg", data.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	return data.Bytes()
}

func newOfficialQQMediaTestAdapter(t *testing.T, markdown, nameCard bool) (*PlatformAdapterOfficialQQ, *MsgContext, *officialQQMediaTestAPI) {
	t.Helper()
	ctx, _ := newNameCardReplyTestContext()
	ctx.Dice.Config.OfficialQQUseMarkdown = markdown
	ctx.Dice.Config.OfficialQQUseNameCard = nameCard
	ctx.Player.AutoSetNameTemplate = "茶叶 HP10/10"
	ctx.Session = &IMSession{Parent: ctx.Dice}
	ctx.EndPoint.Session = ctx.Session
	api := &officialQQMediaTestAPI{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.Header.Get("Content-Type") != "image/jpeg" || r.Header.Get("Authorization") != "" {
			t.Errorf("invalid presigned upload: method=%s headers=%v", r.Method, r.Header)
		}
		data, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		api.uploads = append(api.uploads, data)
		if api.failStage == "put" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		etag := fmt.Sprintf("%x", md5.Sum(data)) //nolint:gosec // Required by the upload protocol.
		if api.failStage == "checksum" {
			etag = "wrong"
		}
		w.Header().Set("ETag", `"`+etag+`"`)
	}))
	t.Cleanup(server.Close)
	api.putURL = server.URL
	pa := &PlatformAdapterOfficialQQ{EndPoint: ctx.EndPoint, Api: api, UIN: "12345", AppID: "app"}
	ctx.EndPoint.Adapter = pa
	ctx.Group.GroupID = formatDiceIDOfficialQQGroupOpenID(pa.UIN, "group")
	ctx.Player.UserID = formatDiceIDOfficialQQUserOpenID(pa.UIN, "user")
	VarSetValueStr(ctx, "$tMsgID", "incoming")
	VarSetValueStr(ctx, "$tEventID", "")
	return pa, ctx, api
}

func TestOfficialQQImageReplySingleBubble(t *testing.T) {
	imageData := officialQQTestLocalImage(t)
	const img = "[图:data/images/zcnydhcg.jpg]"
	for _, mode := range []struct {
		name               string
		markdown, nameCard bool
	}{{"plain", false, false}, {"markdown", true, false}, {"name_card", true, true}} {
		for _, target := range []string{"group", "private"} {
			for _, raw := range []bool{false, true} {
				for _, input := range []string{"大成功！\n" + img, img + "大成功！", "前" + img + "后", "前" + img + "中" + img + "后", img + "\n" + img} {
					t.Run(fmt.Sprintf("%s/%s/raw=%t/%s", mode.name, target, raw, input), func(t *testing.T) {
						pa, ctx, api := newOfficialQQMediaTestAdapter(t, mode.markdown, mode.nameCard)
						ctx.MessageType = target
						text := withOfficialQQNameCard(ctx, input)
						want := strings.ReplaceAll(text, img, testQQInlineImage)
						var err error
						if raw {
							if target == "group" {
								_, err = pa.sendQQGroupMsgRaw(ctx, "incoming", "group", text, nil)
							} else {
								_, err = pa.sendC2CMsgRaw(ctx, "incoming", "user", text, nil)
							}
						} else if target == "group" {
							pa.SendToGroup(ctx, ctx.Group.GroupID, text, "")
						} else {
							pa.SendToPerson(ctx, ctx.Player.UserID, text, "")
						}
						if err != nil {
							t.Fatal(err)
						}
						if len(api.messages) != 1 {
							t.Fatalf("sent %d messages, want one bubble", len(api.messages))
						}
						msg := api.messages[0]
						if msg.MsgType != 2 || msg.Media != nil || msg.Markdown == nil || msg.Markdown.Content != want || !msg.Markdown.ForceVerifyImageResource {
							t.Fatalf("message must contain text and image in one verified Markdown: %#v, want %q", msg, want)
						}
						if msg.MsgID != "incoming" || msg.MsgSeq == 0 {
							t.Error("passive reply metadata missing")
						}
						if len(api.uploads) != 1 || !bytes.Equal(api.uploads[0], imageData) || api.prepareCalls != 1 || api.completeCalls != 1 {
							t.Fatal("image must upload once, even when repeated")
						}
						if api.prepared["md5"] != fmt.Sprintf("%x", md5.Sum(imageData)) || api.prepared["sha1"] != fmt.Sprintf("%x", sha1.Sum(imageData)) || api.prepared["file_size"] != len(imageData) || api.prepared["content_type"] != "image/jpeg" { //nolint:gosec // Protocol checksums.
							t.Fatalf("incorrect upload metadata: %#v", api.prepared)
						}
					})
				}
			}
		}
	}
}

func TestOfficialQQImageMaterialReuseAfterRestart(t *testing.T) {
	data := officialQQTestLocalImage(t)
	pa, ctx, api := newOfficialQQMediaTestAdapter(t, true, false)
	api.existing = []officialQQImageReference{{URL: testQQImageURL, FileName: fmt.Sprintf("sealdice-%x.jpeg", sha1.Sum(data)), ExpiresAt: time.Now().Add(time.Hour).Unix()}} //nolint:gosec // Content addressing for QQ.
	for range 2 {
		_, err := pa.sendQQGroupMsgRaw(ctx, "incoming", "group", "大成功！\n[图:data/images/zcnydhcg.jpg]", nil)
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(api.uploads) != 0 || api.prepareCalls != 0 || len(api.messages) != 2 {
		t.Fatal("existing platform material must be reused")
	}
}

func TestOfficialQQImageMaterialFailureIsNotSilent(t *testing.T) {
	officialQQTestLocalImage(t)
	for _, stage := range []string{"prepare", "put", "checksum", "complete"} {
		t.Run(stage, func(t *testing.T) {
			pa, ctx, api := newOfficialQQMediaTestAdapter(t, true, true)
			api.failStage = stage
			_, err := pa.sendQQGroupMsgRaw(ctx, "incoming", "group", "大成功！\n[图:data/images/zcnydhcg.jpg]", nil)
			if err == nil || len(api.messages) != 0 {
				t.Fatalf("err=%v sent=%d: failed image must not silently become a text-only reply", err, len(api.messages))
			}
		})
	}
}

func TestOfficialQQImageReplyReferenceKeyboardAndEscaping(t *testing.T) {
	officialQQTestLocalImage(t)
	pa, ctx, api := newOfficialQQMediaTestAdapter(t, false, false)
	kb := &keyboard.MessageKeyboard{}
	_, err := pa.sendQQGroupMsgRaw(ctx, "incoming", "group", "[CQ:reply,id=original]<角色> *成功*\n[图:data/images/zcnydhcg.jpg]\n", kb)
	if err != nil || len(api.messages) != 1 {
		t.Fatalf("err=%v messages=%d", err, len(api.messages))
	}
	msg := api.messages[0]
	if msg.MessageReference == nil || msg.MessageReference.MessageID != "original" || msg.Keyboard != kb {
		t.Fatal("reply reference or keyboard missing")
	}
	want := "\\<角色\\> \\*成功\\*\n" + testQQInlineImage + "\n"
	if msg.Markdown.Content != want {
		t.Fatalf("content=%q, want %q", msg.Markdown.Content, want)
	}
}

func TestOfficialQQPublicMarkdownImage(t *testing.T) {
	pa, _, api := newOfficialQQMediaTestAdapter(t, true, false)
	got, err := pa.markdownImage(t.Context(), &message.FileElement{URL: "https://example.com/image(1).png"})
	if err != nil || got != "![图片](https://example.com/image%281%29.png)" || api.prepareCalls != 0 {
		t.Fatalf("public image=%q, err=%v", got, err)
	}
}

func TestOfficialQQImagePlainTextEscapingKeepsMentions(t *testing.T) {
	input := "<角色> *成功* <@user> <qqbot-at-user id=\"another\" />"
	want := "\\<角色\\> \\*成功\\* <@user> <qqbot-at-user id=\"another\" />"
	if got := officialQQEscapeMarkdownText(input); got != want {
		t.Fatalf("escaped=%q, want %q", got, want)
	}
}

func TestOfficialQQImageReferenceExpiryJSON(t *testing.T) {
	for _, expiry := range []string{`1802247519`, `"1802247519"`} {
		var ref officialQQImageReference
		data := `{"url":"` + testQQImageURL + `","file_name":"image.jpeg","expires_at":` + expiry + `}`
		if err := json.Unmarshal([]byte(data), &ref); err != nil {
			t.Fatal(err)
		}
		if ref.ExpiresAt != 1802247519 || ref.URL != testQQImageURL || ref.FileName != "image.jpeg" {
			t.Fatalf("unexpected image reference: %#v", ref)
		}
	}
}
