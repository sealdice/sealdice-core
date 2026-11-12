package dice

import (
	"bytes"
	"context"
	"crypto/md5"  //nolint:gosec // QQ's upload protocol requires MD5 and SHA1.
	"crypto/sha1" //nolint:gosec // QQ's upload protocol requires MD5 and SHA1.
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	qqconstant "github.com/sealdice/botgo/constant"
	_ "golang.org/x/image/webp"

	"sealdice-core/message"
)

type officialQQImageReference struct {
	URL       string `json:"url"`
	FileName  string `json:"file_name"`
	ExpiresAt int64  `json:"expires_at"`
}

func (ref *officialQQImageReference) UnmarshalJSON(data []byte) error {
	var value struct {
		URL       string      `json:"url"`
		FileName  string      `json:"file_name"`
		ExpiresAt json.Number `json:"expires_at"`
	}
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	// QQ 的 int64 字段可能以 JSON 字符串返回。
	expiresAt, err := value.ExpiresAt.Int64()
	if err != nil {
		return fmt.Errorf("读取 QQ 图片素材过期时间失败: %w", err)
	}
	*ref = officialQQImageReference{URL: value.URL, FileName: value.FileName, ExpiresAt: expiresAt}
	return nil
}

func (pa *PlatformAdapterOfficialQQ) requestImageMaterial(ctx context.Context, method, path string, body, result any) error {
	data, err := pa.Api.Transport(ctx, method, qqconstant.APIDomain+path, body)
	if err != nil {
		return err
	}
	var response struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err = json.Unmarshal(data, &response); err != nil {
		return fmt.Errorf("解析 QQ 图片素材响应失败: %w", err)
	}
	if response.Code != 0 {
		return fmt.Errorf("QQ 图片素材错误 %d: %s", response.Code, response.Message)
	}
	return json.Unmarshal(data, result)
}

// markdownImage 将图片写入同一条 Markdown，素材引用可跨群聊/私聊复用。
// https://bot.q.qq.com/wiki/develop/api-v2/server-inter/message/material-management.html
func (pa *PlatformAdapterOfficialQQ) markdownImage(ctx context.Context, file *message.FileElement) (string, error) {
	if file.URL != "" && !isLocalOrNonPublic(file.URL) && !pa.EndPoint.Session.Parent.Config.OfficialQQFileSendBase64 {
		// 公网 URL 由 QQ 转存；括号不能提前结束 Markdown 图片链接。
		imageURL := strings.NewReplacer("(", "%28", ")", "%29", "\n", "%0A", "\r", "%0D").Replace(file.URL)
		return "![图片](" + imageURL + ")", nil
	}
	data, err := getElementBytes(file)
	if err != nil {
		return "", err
	}
	if len(data) > 10_000_000 {
		return "", errors.New("QQ Markdown 图片不能超过 10 MB")
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("读取图片格式和尺寸失败: %w", err)
	}
	switch format {
	case "png", "jpeg", "gif", "webp":
	default:
		return "", fmt.Errorf("QQ Markdown 不支持图片格式 %s", format)
	}
	md5Hex := fmt.Sprintf("%x", md5.Sum(data))   //nolint:gosec // Required by the QQ upload API.
	sha1Hex := fmt.Sprintf("%x", sha1.Sum(data)) //nolint:gosec // Required by the QQ upload API.
	filename := "sealdice-" + sha1Hex + "." + format
	key := pa.AppID + ":" + filename

	pa.imageMaterialMu.Lock()
	defer pa.imageMaterialMu.Unlock()
	if pa.imageMaterials == nil {
		pa.imageMaterials = make(map[string]officialQQImageReference)
	}
	ref, ok := pa.imageMaterials[key]
	if !ok || ref.ExpiresAt <= time.Now().Unix() {
		// 平台最多保存 100 张素材。重启后按内容命名查找，避免重复上传耗尽配额。
		cursor := ""
		for {
			var page struct {
				Images     []officialQQImageReference `json:"images"`
				NextCursor string                     `json:"next_cursor"`
			}
			path := "/v2/media-images"
			if cursor != "" {
				path += "?cursor=" + url.QueryEscape(cursor)
			}
			if err = pa.requestImageMaterial(ctx, http.MethodGet, path, nil, &page); err != nil {
				return "", err
			}
			for _, item := range page.Images {
				if item.FileName == filename && item.ExpiresAt > time.Now().Unix() {
					ref = item
					break
				}
			}
			if ref.ExpiresAt > time.Now().Unix() || page.NextCursor == "" {
				break
			}
			cursor = page.NextCursor
		}
		if ref.ExpiresAt <= time.Now().Unix() {
			ref, err = pa.uploadImageMaterial(ctx, filename, "image/"+format, md5Hex, sha1Hex, data)
			if err != nil {
				return "", err
			}
		}
		if !strings.HasPrefix(ref.URL, "qqbot-media://image/v1/") {
			return "", errors.New("QQ 返回了无效的图片素材引用")
		}
		pa.imageMaterials[key] = ref
	}
	return fmt.Sprintf("![图片 #%dpx #%dpx](%s)", config.Width, config.Height, ref.URL), nil
}

func (pa *PlatformAdapterOfficialQQ) uploadImageMaterial(ctx context.Context, filename, contentType, md5Hex, sha1Hex string, data []byte) (officialQQImageReference, error) {
	var empty officialQQImageReference
	var prepared struct {
		UploadID     string `json:"upload_id"`
		PresignedURL string `json:"presigned_url"`
	}
	body := map[string]any{"file_name": filename, "file_size": len(data), "content_type": contentType, "md5": md5Hex, "sha1": sha1Hex}
	if err := pa.requestImageMaterial(ctx, http.MethodPost, "/v2/media-images/prepare", body, &prepared); err != nil {
		return empty, err
	}
	if prepared.UploadID == "" || prepared.PresignedURL == "" {
		return empty, errors.New("QQ 未返回图片上传地址或任务 ID")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, prepared.PresignedURL, bytes.NewReader(data))
	if err != nil {
		return empty, errors.New("QQ 返回了无效的图片上传地址")
	}
	req.Header.Set("Content-Type", contentType)
	client := &http.Client{Timeout: officialQQAPITimeout}
	resp, err := client.Do(req)
	if err != nil {
		var urlErr *url.Error
		if errors.As(err, &urlErr) {
			err = urlErr.Err // 不把带签名的上传 URL 写入日志。
		}
		return empty, fmt.Errorf("上传 QQ 图片素材失败: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return empty, fmt.Errorf("上传 QQ 图片素材失败: HTTP %d", resp.StatusCode)
	}
	etag := strings.Trim(resp.Header.Get("ETag"), "\"")
	if !strings.EqualFold(etag, md5Hex) {
		return empty, errors.New("QQ 图片上传校验值不一致")
	}
	var completed struct {
		URL string `json:"url"`
		TTL int64  `json:"ttl"`
	}
	if err = pa.requestImageMaterial(ctx, http.MethodPost, "/v2/media-images/complete", map[string]string{"upload_id": prepared.UploadID, "md5": etag}, &completed); err != nil {
		return empty, err
	}
	return officialQQImageReference{URL: completed.URL, FileName: filename, ExpiresAt: time.Now().Unix() + completed.TTL}, nil
}

func officialQQHasImage(elems []message.IMessageElement) bool {
	for _, elem := range elems {
		if _, ok := elem.(*message.ImageElement); ok {
			return true
		}
	}
	return false
}

func officialQQEscapeMarkdownText(text string) string {
	escape := strings.NewReplacer("\\", "\\\\", "`", "\\`", "*", "\\*", "_", "\\_", "~", "\\~", "[", "\\[", "]", "\\]", "#", "\\#", "<", "\\<", ">", "\\>")
	var result strings.Builder
	pos := 0
	for _, match := range officialQQAtRegex.FindAllStringIndex(text, -1) {
		result.WriteString(escape.Replace(text[pos:match[0]]))
		result.WriteString(text[match[0]:match[1]])
		pos = match[1]
	}
	result.WriteString(escape.Replace(text[pos:]))
	return result.String()
}
