//nolint:testpackage
package dice

import "testing"

// 协议端不透明引用（hash.jpg / cache id）不能被当成缺失的本地文件替换掉。
func TestTextAssetsConvertKeepsOpaqueCQFile(t *testing.T) {
	const in = "[CQ:image,file=3B71E9799730D77111FD3EA5E4076773.jpg]"
	out := textAssetsConvert(in)
	cq := CQParse(out)
	if cq.Type != "image" || cq.Args["file"] != "3B71E9799730D77111FD3EA5E4076773.jpg" {
		t.Fatalf("opaque CQ file was rewritten: %q", out)
	}
}

// 明确的本地路径仍然要保留缺失/越权提示。
func TestTextAssetsConvertRejectsMissingExplicitLocalFile(t *testing.T) {
	out := textAssetsConvert("[CQ:image,file=./definitely-missing.png]")
	if out != "[CQ码找不到文件]" {
		t.Fatalf("missing local file should keep the friendly error, got %q", out)
	}
}
