package dice

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/vmihailenco/msgpack"

	"sealdice-core/utils/deckstore"
)

// deck_meta 表使用的通用 KV 键（由 L1 定义，规格 §4）。
const (
	metaName        = "name"
	metaAuthor      = "author"
	metaVersion     = "version"
	metaLicense     = "license"
	metaDate        = "date"
	metaUpdateDate  = "update_date"
	metaDesc        = "desc"
	metaFormat      = "format"
	metaFileFormat  = "file_format"
	metaFormatVer   = "format_version"
	metaEnable      = "enable"
	metaErrText     = "err_text"
	metaStoreID     = "store_id"
	metaEtag        = "etag"
	metaUpdateUrls  = "update_urls"
	metaInfo        = "info"
	metaCommand     = "command"
	metaContentHash = "content_hash"
	metaCloud       = "cloud"
	metaCloudItems  = "cloud_items"
)

// cloudDeckItemInfoWire 是 CloudDeckItemInfo 的可序列化中间结构。
// msgpack 仅编码导出字段，因此用带导出字段的结构体承载 Distinct/OptionsUrls。
type cloudDeckItemInfoWire struct {
	Distinct    bool     `msgpack:"distinct"`
	OptionsUrls []string `msgpack:"options_urls"`
}

func cloudInfosToWire(m map[string]*CloudDeckItemInfo) map[string]cloudDeckItemInfoWire {
	if m == nil {
		return nil
	}
	out := make(map[string]cloudDeckItemInfoWire, len(m))
	for k, v := range m {
		if v == nil {
			out[k] = cloudDeckItemInfoWire{}
			continue
		}
		out[k] = cloudDeckItemInfoWire{Distinct: v.Distinct, OptionsUrls: v.OptionsUrls}
	}
	return out
}

func cloudInfosFromWire(w map[string]cloudDeckItemInfoWire) map[string]*CloudDeckItemInfo {
	if w == nil {
		return nil
	}
	out := make(map[string]*CloudDeckItemInfo, len(w))
	for k, v := range w {
		out[k] = &CloudDeckItemInfo{Distinct: v.Distinct, OptionsUrls: v.OptionsUrls}
	}
	return out
}

// deckSource 是一个牌堆来源文件（含扩展包归属）。
type deckSource struct {
	Filename  string
	PackageID string
}

// deckFileKey 生成牌堆库键：sha1(packageID+"\x00"+ToSlash(filename)) 十六进制。
func deckFileKey(filename, packageID string) string {
	sum := sha1.Sum([]byte(packageID + "\x00" + filepath.ToSlash(filename)))
	return hex.EncodeToString(sum[:])
}

// deckImportSource 解析一个来源并写入库。成功返回仅含元数据的 DeckInfo（DeckItems 为 nil）；
// 解析失败返回 DeckInfo{Enable:false, ErrText:err} 与 error，不写库。
func deckImportSource(d *Dice, st deckstore.Store, src deckSource, content []byte) (*DeckInfo, error) {
	contentHash := sha1.Sum(content)
	if isPrefixWithUtf8Bom(content) {
		content = content[3:]
	}
	deckKey := deckFileKey(src.Filename, src.PackageID)
	ext := strings.ToLower(path.Ext(src.Filename))

	switch ext {
	case ".json", ".jsonc":
		di := newDiceEDeckInfo(src, deckKey)
		im, err := st.BeginImport(deckKey)
		if err != nil {
			return di, err
		}
		if err := importDiceEJSON(content, di, im); err != nil {
			im.Abort()
			di.Enable = false
			di.ErrText = err.Error()
			return di, err
		}
		setDeckMeta(im, di, contentHash[:])
		if err := im.Commit(); err != nil {
			return di, err
		}
		return di, nil

	case ".yaml", ".yml", ".toml":
		return importParsedDeck(d, st, src, deckKey, content, contentHash[:], false)

	default:
		// 无扩展名：先试 DiceE，失败回退 SinaNya（与 parseDeck 默认分支一致）。
		di := newDiceEDeckInfo(src, deckKey)
		im, err := st.BeginImport(deckKey)
		if err != nil {
			return di, err
		}
		if err := importDiceEJSON(content, di, im); err == nil {
			setDeckMeta(im, di, contentHash[:])
			if err := im.Commit(); err != nil {
				return di, err
			}
			return di, nil
		}
		im.Abort()
		return importParsedDeck(d, st, src, deckKey, content, contentHash[:], true)
	}
}

func newDiceEDeckInfo(src deckSource, deckKey string) *DeckInfo {
	return &DeckInfo{
		Filename:  src.Filename,
		PackageID: src.PackageID,
		DeckKey:   deckKey,
		Command:   map[string]bool{},
	}
}

// importParsedDeck 走 legacy 解析器（parseDeck / tryParseSinaNya）解析后 drain 进库。
func importParsedDeck(d *Dice, st deckstore.Store, src deckSource, deckKey string, content []byte, contentHash []byte, sinaNyaOnly bool) (*DeckInfo, error) {
	di := &DeckInfo{
		Filename:           src.Filename,
		PackageID:          src.PackageID,
		DeckKey:            deckKey,
		DeckItems:          map[string][]string{},
		Command:            map[string]bool{},
		CloudDeckItemInfos: map[string]*CloudDeckItemInfo{},
	}
	var parseErr error
	if sinaNyaOnly {
		parseErr = tryParseSinaNya(content, di)
	} else if !parseDeck(d, src.Filename, content, di) {
		parseErr = errors.New(di.ErrText)
	}
	if parseErr != nil {
		di.Enable = false
		di.ErrText = parseErr.Error()
		return di, parseErr
	}

	im, err := st.BeginImport(deckKey)
	if err != nil {
		return di, err
	}
	if err := drainLegacyDeck(di, im); err != nil {
		im.Abort()
		return di, err
	}
	setDeckMeta(im, di, contentHash)
	if err := im.Commit(); err != nil {
		return di, err
	}
	return di, nil
}

// importDiceEJSON 流式解析 DiceE 格式（json/jsonc）并写入 import 事务，
// 不构建 map[string]any；jsonc 先经 standardizeJson 归一化。
func importDiceEJSON(data []byte, di *DeckInfo, im deckstore.Import) error {
	standardJson, isRFC, err := standardizeJson(data)
	if err != nil {
		return err
	}
	jsoncDirectly := strings.EqualFold(path.Ext(di.Filename), ".jsonc")

	dec := json.NewDecoder(bytes.NewReader(standardJson))
	tok, err := dec.Token()
	if err != nil {
		return err
	}
	if d, ok := tok.(json.Delim); !ok || d != '{' {
		return errors.New("牌堆文件顶层不是 JSON 对象")
	}

	var (
		title, author, version, license, date, updateDate, brief []string
		storeID, updateUrls, etag                                []string
		keys                                                     []string
		keysPresent                                              bool
		exportList, exportsList                                  []string
		exportPresent, exportsPresent                            bool
		groupNames                                               []string
	)

	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return err
		}
		key, _ := keyTok.(string)

		valTok, err := dec.Token()
		if err != nil {
			return err
		}
		if delim, isDelim := valTok.(json.Delim); isDelim {
			switch delim {
			case '[':
				items, err := readJSONStrings(dec)
				if err != nil {
					return err
				}
				if key == "$schema" {
					continue // 忽略 schema，不写组
				}
				// 与 tryParseDiceE 一致：所有字符串数组键（含 _title 等）都成为组。
				if err := writeDeckGroup(im, key, items); err != nil {
					return err
				}
				groupNames = append(groupNames, key)
				switch key {
				case "_title":
					title = items
				case "_author":
					author = items
				case "_version":
					version = items
				case "_license":
					license = items
				case "_date":
					date = items
				case "_updateDate":
					updateDate = items
				case "_brief":
					brief = items
				case "_storeID":
					storeID = items
				case "_updateUrls":
					updateUrls = items
				case "_etag":
					etag = items
				case "_keys":
					keys = items
					keysPresent = true
				case "_export":
					exportList = items
					exportPresent = true
				case "_exports":
					exportsList = items
					exportsPresent = true
				}
			case '{':
				var discard any
				if err := dec.Decode(&discard); err != nil {
					return err
				}
				if key == "helpdoc" {
					return errors.New("该文件疑似为帮助文档，而不是牌堆文件")
				}
			}
		}
		// 标量值已由 valTok 消费，忽略。
	}
	if _, err := dec.Token(); err != nil { // 消费 '}'
		return err
	}

	// 可见性（复刻 tryParseDiceE 的 _keys/_exports 语义）。
	exportsExists := exportPresent || exportsPresent
	var exports []string
	if exportPresent {
		exports = exportList
	} else {
		exports = exportsList
	}
	if exportsExists {
		for _, i := range exports {
			di.Command[i] = true
		}
	}
	if keysPresent {
		for _, i := range keys {
			di.Command[i] = true
		}
		for _, k := range groupNames {
			if !exportsExists && !strings.HasPrefix(k, "_") {
				if _, exists := di.Command[k]; !exists {
					di.Command[k] = false
				}
			}
		}
	} else {
		for _, k := range groupNames {
			if !exportsExists && !strings.HasPrefix(k, "_") {
				di.Command[k] = true
			}
		}
	}

	di.Name = strings.Join(title, " / ")
	di.Author = strings.Join(author, " / ")
	di.Version = strings.Join(version, " / ")
	di.License = strings.Join(license, " / ")
	di.Date = strings.Join(date, " / ")
	di.UpdateDate = strings.Join(updateDate, " / ")
	di.Desc = strings.Join(brief, "\n")
	di.Format = "Dice!"
	di.FormatVersion = 1
	if !jsoncDirectly && isRFC {
		di.FileFormat = "json"
	} else {
		di.FileFormat = "jsonc"
	}
	di.Enable = true
	if len(storeID) > 0 {
		di.StoreID = storeID[0]
	}
	di.UpdateUrls = updateUrls
	if len(etag) > 0 {
		di.Etag = etag[0]
	}
	return nil
}

// readJSONStrings 从解码器当前位置（数组开始）读取一个字符串数组。
// 数组内非字符串元素静默跳过，与 tryParseDiceE 对数组内元素的处理一致。
func readJSONStrings(dec *json.Decoder) ([]string, error) {
	var out []string
	for dec.More() {
		var v any
		if err := dec.Decode(&v); err != nil {
			return nil, err
		}
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	if _, err := dec.Token(); err != nil { // 消费 ']'
		return nil, err
	}
	return out, nil
}

// writeDeckGroup 把一个组的条目（原文，含 ::N:: 前缀）逐条写入 import 事务。
func writeDeckGroup(im deckstore.Import, name string, items []string) error {
	g := im.BeginGroup(name)
	for _, item := range items {
		w, _ := extractWeight(item)
		g.Add(item, uint32(w))
	}
	_, err := g.Finish()
	return err
}

// drainLegacyDeck 把 legacy DeckInfo.DeckItems 逐组逐条写入 import 事务，随后清空 DeckItems。
func drainLegacyDeck(di *DeckInfo, im deckstore.Import) error {
	for name, items := range di.DeckItems {
		if err := writeDeckGroup(im, name, items); err != nil {
			return err
		}
	}
	di.DeckItems = nil
	return nil
}

// setDeckMeta 把 DeckInfo 元数据写入 import 事务（规格 §4 键列表）。
func setDeckMeta(im deckstore.Import, di *DeckInfo, contentHash []byte) {
	setStr := func(k, v string) { im.SetMeta(k, []byte(v)) }
	setStr(metaName, di.Name)
	setStr(metaAuthor, di.Author)
	setStr(metaVersion, di.Version)
	setStr(metaLicense, di.License)
	setStr(metaDate, di.Date)
	setStr(metaUpdateDate, di.UpdateDate)
	setStr(metaDesc, di.Desc)
	setStr(metaFormat, di.Format)
	setStr(metaFileFormat, di.FileFormat)
	setStr(metaFormatVer, strconv.FormatInt(di.FormatVersion, 10))
	setStr(metaEnable, strconv.FormatBool(di.Enable))
	setStr(metaErrText, di.ErrText)
	setStr(metaStoreID, di.StoreID)
	setStr(metaEtag, di.Etag)

	if b, err := json.Marshal(di.UpdateUrls); err == nil {
		im.SetMeta(metaUpdateUrls, b)
	}
	if b, err := msgpack.Marshal(di.Info); err == nil {
		im.SetMeta(metaInfo, b)
	}
	if b, err := msgpack.Marshal(di.Command); err == nil {
		im.SetMeta(metaCommand, b)
	}
	if di.Cloud {
		im.SetMeta(metaCloud, []byte("1"))
	} else {
		im.SetMeta(metaCloud, []byte("0"))
	}
	if b, err := msgpack.Marshal(cloudInfosToWire(di.CloudDeckItemInfos)); err == nil {
		im.SetMeta(metaCloudItems, b)
	}
	im.SetMeta(metaContentHash, contentHash)
}

// deckLoadMetaRow 从 deck_meta 的 KV 行恢复 DeckInfo 元数据与 content_hash（reload 增量用）。
func deckLoadMetaRow(st deckstore.Store, deckKey string) (di *DeckInfo, hash []byte, ok bool) {
	hash, ok = st.Meta(deckKey, metaContentHash)
	if !ok {
		return nil, nil, false
	}
	di = &DeckInfo{
		DeckKey: deckKey,
		Command: map[string]bool{},
	}
	metaStr := func(k string) string {
		if b, found := st.Meta(deckKey, k); found {
			return string(b)
		}
		return ""
	}
	di.Name = metaStr(metaName)
	di.Author = metaStr(metaAuthor)
	di.Version = metaStr(metaVersion)
	di.License = metaStr(metaLicense)
	di.Date = metaStr(metaDate)
	di.UpdateDate = metaStr(metaUpdateDate)
	di.Desc = metaStr(metaDesc)
	di.Format = metaStr(metaFormat)
	di.FileFormat = metaStr(metaFileFormat)
	di.FormatVersion, _ = strconv.ParseInt(metaStr(metaFormatVer), 10, 64)
	di.Enable, _ = strconv.ParseBool(metaStr(metaEnable))
	di.ErrText = metaStr(metaErrText)
	di.StoreID = metaStr(metaStoreID)
	di.Etag = metaStr(metaEtag)

	if b, found := st.Meta(deckKey, metaUpdateUrls); found {
		_ = json.Unmarshal(b, &di.UpdateUrls)
	}
	if b, found := st.Meta(deckKey, metaInfo); found {
		_ = msgpack.Unmarshal(b, &di.Info)
	}
	if b, found := st.Meta(deckKey, metaCommand); found {
		_ = msgpack.Unmarshal(b, &di.Command)
	}
	di.Cloud, _ = strconv.ParseBool(metaStr(metaCloud))
	if b, found := st.Meta(deckKey, metaCloudItems); found {
		var wire map[string]cloudDeckItemInfoWire
		if err := msgpack.Unmarshal(b, &wire); err == nil {
			di.CloudDeckItemInfos = cloudInfosFromWire(wire)
		}
	}
	return di, hash, true
}
