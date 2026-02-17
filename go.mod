module sealdice-core

go 1.24.6

require (
	github.com/Masterminds/semver/v3 v3.4.0
	github.com/Milly/go-base2048 v0.1.0
	github.com/PaienNate/pineutil v0.0.0-20251018153346-e4acff10b752
	github.com/ShiraazMoollatjie/goluhn v0.0.0-20211017190329-0d86158c056a
	github.com/Szzrain/DingTalk-go v0.0.8-alpha
	github.com/Szzrain/Milky-go-sdk v1.0.1
	github.com/Szzrain/dodo-open-go v0.2.8
	github.com/acarl005/stripansi v0.0.0-20180116102854-5a71ef0e047d
	github.com/alexmullins/zip v0.0.0-20180717182244-4affb64b04d0
	github.com/antlabs/strsim v0.0.3
	github.com/armon/go-socks5 v0.0.0-20160902184237-e75332964ef5
	github.com/avast/retry-go v3.0.0+incompatible
	github.com/blevesearch/bleve/v2 v2.5.2
	github.com/blevesearch/bleve_index_api v1.2.10
	github.com/bwmarrin/discordgo v0.29.0
	github.com/bytedance/sonic v1.14.1
	github.com/danielgtaylor/huma/v2 v2.34.1
	github.com/dop251/goja v0.0.0-20260216154549-8b74ce4618c5
	github.com/dop251/goja_nodejs v0.0.0-20250409162600-f7acab6894b0
	github.com/evanw/esbuild v0.25.11
	github.com/fy0/go-autostart v0.0.0-20220515100644-a25d81ed766b
	github.com/fy0/gojax v0.0.0-20221225152702-4140cf8509bd
	github.com/fy0/systray v1.2.2
	github.com/fyrchik/go-shlex v0.0.0-20210215145004-cd7f49bfd959
	github.com/gen2brain/beeep v0.11.1
	github.com/go-creed/sat v1.0.3
	github.com/go-gorm/caches/v4 v4.0.5
	github.com/go-telegram-bot-api/telegram-bot-api/v5 v5.5.1
	github.com/gofrs/flock v0.13.0
	github.com/golang-module/carbon v1.7.3
	github.com/google/uuid v1.6.0
	github.com/gorilla/websocket v1.5.3
	github.com/grokify/html-strip-tags-go v0.1.0
	github.com/jessevdk/go-flags v1.6.1
	github.com/jmoiron/sqlx v1.4.0
	github.com/joho/godotenv v1.5.1
	github.com/juliangruber/go-intersect v1.1.0
	github.com/kardianos/service v1.2.4
	github.com/labstack/echo/v4 v4.13.4
	github.com/lonelyevil/kook v0.0.31
	github.com/lonelyevil/kook/log_adapter/plog v0.0.31
	github.com/looplab/fsm v1.0.0
	github.com/lxn/win v0.0.0-20210218163916-a377121e959e
	github.com/matoous/go-nanoid/v2 v2.1.0
	github.com/mattn/go-sqlite3 v1.14.32
	github.com/maypok86/otter v1.2.4
	github.com/mitchellh/mapstructure v1.5.0
	github.com/monaco-io/request v1.0.16
	github.com/mozillazg/go-pinyin v0.21.0
	github.com/mroth/weightedrand v1.0.0
	github.com/ncruces/go-sqlite3 v0.29.0
	github.com/ncruces/go-sqlite3/gormlite v0.24.0
	github.com/open-dingtalk/dingtalk-stream-sdk-go v0.9.1
	github.com/otiai10/copy v1.14.1
	github.com/panjf2000/ants/v2 v2.11.3
	github.com/parquet-go/parquet-go v0.25.1
	github.com/pelletier/go-toml/v2 v2.2.4
	github.com/phuslu/log v1.0.119
	github.com/pilagod/gorm-cursor-paginator/v2 v2.7.0
	github.com/pkg/errors v0.9.1
	github.com/robfig/cron/v3 v3.0.1
	github.com/sacOO7/gowebsocket v0.0.0-20221109081133-70ac927be105
	github.com/sahilm/fuzzy v0.1.1
	github.com/samber/lo v1.52.0
	github.com/schollz/progressbar/v3 v3.18.0
	github.com/sealdice/botgo v0.0.0-20240102160217-e61d5bdfe083
	github.com/sealdice/dicescript v0.0.0-20260103130502-bd5621347a4e
	github.com/slack-go/slack v0.17.3
	github.com/sunshineplan/imgconv v1.1.14
	github.com/tailscale/hujson v0.0.0-20250605163823-992244df8c5a
	github.com/tdewolff/minify/v2 v2.24.3
	github.com/tidwall/buntdb v1.3.2
	github.com/tidwall/gjson v1.18.0
	github.com/tidwall/sjson v1.2.5
	github.com/vmihailenco/msgpack v4.0.4+incompatible
	github.com/xuri/excelize/v2 v2.10.0
	github.com/yuin/goldmark v1.7.13
	go.etcd.io/bbolt v1.4.3
	go.uber.org/zap v1.27.0
	golang.org/x/crypto v0.45.0
	golang.org/x/exp v0.0.0-20250911091902-df9299821621
	golang.org/x/sys v0.38.0
	golang.org/x/text v0.34.0
	golang.org/x/time v0.14.0
	gopkg.in/elazarl/goproxy.v1 v1.0.0-20180725130230-947c36da3153
	gopkg.in/gomail.v2 v2.0.0-20160411212932-81ebce5c23df
	gopkg.in/natefinch/lumberjack.v2 v2.2.1
	gopkg.in/yaml.v3 v3.0.1
	gorm.io/driver/mysql v1.6.0
	gorm.io/driver/postgres v1.6.0
	gorm.io/driver/sqlite v1.6.0
	gorm.io/gorm v1.31.0
	moul.io/zapgorm2 v1.3.0
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	git.sr.ht/~jackmordaunt/go-toast v1.1.2 // indirect
	github.com/HugoSmits86/nativewebp v1.2.0 // indirect
	github.com/RoaringBitmap/roaring/v2 v2.8.0 // indirect
	github.com/andybalholm/brotli v1.2.0 // indirect
	github.com/bits-and-blooms/bitset v1.22.0 // indirect
	github.com/bits-and-blooms/bloom/v3 v3.7.0 // indirect
	github.com/blevesearch/geo v0.2.3 // indirect
	github.com/blevesearch/go-faiss v1.0.25 // indirect
	github.com/blevesearch/go-porterstemmer v1.0.3 // indirect
	github.com/blevesearch/gtreap v0.1.1 // indirect
	github.com/blevesearch/mmap-go v1.0.4 // indirect
	github.com/blevesearch/scorch_segment_api/v2 v2.3.10 // indirect
	github.com/blevesearch/segment v0.9.1 // indirect
	github.com/blevesearch/snowballstem v0.9.0 // indirect
	github.com/blevesearch/upsidedown_store_api v1.0.2 // indirect
	github.com/blevesearch/vellum v1.1.0 // indirect
	github.com/blevesearch/zapx/v11 v11.4.2 // indirect
	github.com/blevesearch/zapx/v12 v12.4.2 // indirect
	github.com/blevesearch/zapx/v13 v13.4.2 // indirect
	github.com/blevesearch/zapx/v14 v14.4.2 // indirect
	github.com/blevesearch/zapx/v15 v15.4.2 // indirect
	github.com/blevesearch/zapx/v16 v16.2.4 // indirect
	github.com/bytedance/gopkg v0.1.3 // indirect
	github.com/bytedance/sonic/loader v0.3.0 // indirect
	github.com/cloudwego/base64x v0.1.6 // indirect
	github.com/dlclark/regexp2 v1.11.5 // indirect
	github.com/dolthub/maphash v0.1.0 // indirect
	github.com/elazarl/goproxy v0.0.0-20230808193330-2592e75ae04a // indirect
	github.com/elazarl/goproxy/ext v0.0.0-20230808193330-2592e75ae04a // indirect
	github.com/esiqveland/notify v0.13.3 // indirect
	github.com/gammazero/deque v1.1.0 // indirect
	github.com/getlantern/context v0.0.0-20220418194847-3d5e7a086201 // indirect
	github.com/getlantern/errors v1.0.4 // indirect
	github.com/getlantern/golog v0.0.0-20230503153817-8e72de7e0a65 // indirect
	github.com/getlantern/hex v0.0.0-20220104173244-ad7e4b9194dc // indirect
	github.com/getlantern/hidden v0.0.0-20220104173330-f221c5a24770 // indirect
	github.com/getlantern/ops v0.0.0-20231025133620-f368ab734534 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/go-resty/resty/v2 v2.16.5 // indirect
	github.com/go-sourcemap/sourcemap v2.1.4+incompatible // indirect
	github.com/go-sql-driver/mysql v1.9.3 // indirect
	github.com/go-stack/stack v1.8.1 // indirect
	github.com/gobuffalo/envy v1.10.2 // indirect
	github.com/gobuffalo/packd v1.0.2 // indirect
	github.com/gobuffalo/packr v1.30.1 // indirect
	github.com/godbus/dbus/v5 v5.1.0 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/golang/snappy v1.0.0 // indirect
	github.com/google/pprof v0.0.0-20260202012954-cb029daf43ef // indirect
	github.com/hhrutter/lzw v1.0.0 // indirect
	github.com/hhrutter/pkcs7 v0.2.0 // indirect
	github.com/hhrutter/tiff v1.0.2 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.7.5 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/jackmordaunt/icns/v3 v3.0.1 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/klauspost/cpuid/v2 v2.2.10 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/labstack/gommon v0.4.2 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/mitchellh/colorstring v0.0.0-20190213212951-d06e56a500db // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/mschoch/smat v0.2.0 // indirect
	github.com/ncruces/julianday v1.0.0 // indirect
	github.com/nfnt/resize v0.0.0-20180221191011-83c6a9932646 // indirect
	github.com/otiai10/mint v1.6.3 // indirect
	github.com/oxtoacart/bpool v0.0.0-20190530202638-03653db5a59c // indirect
	github.com/pdfcpu/pdfcpu v0.11.0 // indirect
	github.com/pierrec/lz4/v4 v4.1.22 // indirect
	github.com/richardlehane/mscfb v1.0.4 // indirect
	github.com/richardlehane/msoleps v1.0.4 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	github.com/sacOO7/go-logger v0.0.0-20180719173527-9ac9add5a50d // indirect
	github.com/sergeymakinen/go-bmp v1.0.0 // indirect
	github.com/sergeymakinen/go-ico v1.0.0-beta.0 // indirect
	github.com/sunshineplan/pdf v1.0.8 // indirect
	github.com/tadvi/systray v0.0.0-20190226123456-11a2b8fa57af // indirect
	github.com/tdewolff/parse/v2 v2.8.3 // indirect
	github.com/tetratelabs/wazero v1.9.0 // indirect
	github.com/tidwall/btree v1.8.0 // indirect
	github.com/tidwall/grect v0.1.4 // indirect
	github.com/tidwall/match v1.2.0 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/tidwall/rtred v0.1.2 // indirect
	github.com/tidwall/tinyqueue v0.1.1 // indirect
	github.com/tiendc/go-deepcopy v1.7.1 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasttemplate v1.2.2 // indirect
	github.com/xuri/efp v0.0.1 // indirect
	github.com/xuri/nfp v0.0.2-0.20250530014748-2ddeb826f9a9 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/otel v1.37.0 // indirect
	go.opentelemetry.io/otel/metric v1.37.0 // indirect
	go.opentelemetry.io/otel/trace v1.37.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/arch v0.21.0 // indirect
	golang.org/x/image v0.32.0 // indirect
	golang.org/x/mod v0.32.0 // indirect
	golang.org/x/net v0.47.0 // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/term v0.37.0 // indirect
	google.golang.org/appengine v1.6.8 // indirect
	google.golang.org/protobuf v1.36.6 // indirect
	gopkg.in/alexcesaro/quotedprintable.v3 v3.0.0-20150716171945-2caba252f4dc // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)

replace (
	github.com/Szzrain/dodo-open-go v0.2.7 => github.com/sealdice/dodo-open-go v0.2.8
	// Try to fix arm64 bug with better snappy.
	github.com/blevesearch/zapx/v16 v16.1.8 => github.com/PaienNate/zapx/v16 v16.1.9
	// Try to fix sqlite in cgofree
	// github.com/glebarez/sqlite v1.11.0 => github.com/PaienNate/sqlite v0.0.0-20241102151933-067d82f14685
	github.com/dop251/goja_nodejs v0.0.0-20250409162600-f7acab6894b0 => github.com/PaienNate/goja_nodejs v0.0.0-20250924024212-bac2e5ba5231
	github.com/lonelyevil/kook v0.0.31 => github.com/sealdice/kook v0.0.3
	github.com/sacOO7/gowebsocket v0.0.0-20221109081133-70ac927be105 => github.com/fy0/GoWebsocket v0.0.0-20231128163937-aa5c110b25c6
// github.com/PaienNate/pineutil v0.0.0-20251014070632-4c5ebfdc92e5 => C:\Users\11018\Documents\goutil
)
