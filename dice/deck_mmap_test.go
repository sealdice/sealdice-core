//nolint:testpackage // Exercise actual deck loading, drawing and recursive formatting.
package dice

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	ds "github.com/sealdice/dicescript"
	"go.uber.org/zap"
)

func mmapTestDice(t *testing.T) *Dice {
	t.Helper()
	t.Chdir(t.TempDir())
	temporary := t.TempDir()
	for _, key := range []string{"TMP", "TEMP", "TMPDIR"} {
		t.Setenv(key, temporary)
	}
	if err := os.MkdirAll("data/decks", 0o755); err != nil {
		t.Fatal(err)
	}
	d := &Dice{Logger: zap.NewNop().Sugar()}
	d.Config = NewConfig(d)
	t.Cleanup(d.CloseDecks)
	return d
}

func mmapWrite(t *testing.T, filename, text string) {
	t.Helper()
	if err := os.WriteFile(filename, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
}

func mmapDraw(t *testing.T, ctx *MsgContext, name string, shuffle bool) string {
	t.Helper()
	ctx.deckDepth = 0
	ok, got, err := deckDraw(ctx, name, shuffle)
	if err != nil || !ok {
		t.Fatalf("draw %q: exists=%v err=%v", name, ok, err)
	}
	return got
}

func TestMappedDeckRecursiveReloadUpdateDelete(t *testing.T) {
	d := mmapTestDice(t)
	filename := "data/decks/no-title.json"
	mmapWrite(t, filename, `{"_export":["root"],"root":["{_private}/{%_private}/#{DRAW-other}"],"_private":["A"]}`)
	mmapWrite(t, "data/decks/other.json", `{"other":["X"]}`)
	DeckReload(d)
	deck := d.DeckListSnapshot()[0]
	if deck.Name != "no-title.json" || deck.storage == nil || deck.DeckItems != nil || deck.RawData != nil {
		t.Fatalf("unexpected loaded deck: %+v", deck)
	}
	ctx := &MsgContext{Dice: d, diceRandSrc: newPCGDiceSource(7)}
	if got := mmapDraw(t, ctx, "root", true); got != "A/A/X" {
		t.Fatalf("recursive result %q", got)
	}
	mmapWrite(t, "update.tmp", `{"root":["{_private}"],"_private":["::0::BAD","::9::B"]}`)
	if err := d.DeckUpdate(deck, "update.tmp"); err != nil {
		t.Fatal(err)
	}
	for range 20 {
		if got := mmapDraw(t, ctx, "root", true); got != "B" {
			t.Fatalf("updated result %q", got)
		}
	}
	if got := mmapDraw(t, &MsgContext{Dice: d}, "root", false); got != "B" {
		t.Fatalf("new context after update: %q", got)
	}
	mmapWrite(t, "bad.tmp", `{"root":`)
	if err := d.DeckUpdate(d.DeckListSnapshot()[0], "bad.tmp"); err == nil {
		t.Fatal("invalid update succeeded")
	}
	if got := mmapDraw(t, ctx, "root", false); got != "B" {
		t.Fatalf("failed update replaced version: %q", got)
	}
	mmapWrite(t, filename, `{"root":["A"]}`)
	DeckReload(d)
	if got := mmapDraw(t, ctx, "root", true); got != "A" {
		t.Fatalf("A-B-A result %q", got)
	}
	DeckDelete(d, d.DeckListSnapshot()[0])
	if len(d.DeckListSnapshot()) != 1 {
		t.Fatal("partial deletion")
	}
	DeckDelete(d, d.DeckListSnapshot()[0])
	if len(d.DeckListSnapshot()) != 0 {
		t.Fatal("last deletion")
	}
}

func TestMappedDeckReloadResetsContextPool(t *testing.T) {
	d := mmapTestDice(t)
	filename := "data/decks/pool.json"
	mmapWrite(t, filename, `{"g":["o0","o1","o2","o3","o4","o5","o6","o7"]}`)
	DeckReload(d)
	ctx := &MsgContext{Dice: d, diceRandSrc: newPCGDiceSource(41)}
	_ = mmapDraw(t, ctx, "g", true)
	mmapWrite(t, filename, `{"g":["new"]}`)
	DeckReload(d)
	for i := range 8 {
		if got := mmapDraw(t, ctx, "g", true); got != "new" {
			t.Fatalf("draw %d used a stale pool: %q", i, got)
		}
	}
}

func TestMappedDeckSixteenKiBBlocks(t *testing.T) {
	items := make([]string, 800)
	for i := range items {
		items[i] = fmt.Sprintf("entry-%d-%s", i, strings.Repeat("x", 300))
	}
	s, err := buildMappedDeck(t.TempDir(), map[string][]string{"g": items})
	if err != nil {
		t.Fatal(err)
	}
	defer s.close()
	g, _ := s.group("g")
	if g.blockCount() < 2 {
		t.Fatal("expected multiple blocks")
	}
	for i := range g.blockCount() {
		b := s.blockAt(&g, i)
		if b.rawSize > 16<<10 || b.count > 256 {
			t.Fatalf("block %d exceeds production limits: %+v", i, b)
		}
		for _, index := range []int{b.first, b.first + b.count - 1} {
			got, err := s.item(&g, index)
			if err != nil || got != items[index] {
				t.Fatalf("boundary item %d: %q, %v", index, got, err)
			}
		}
	}
}

func TestMappedDeckHashCollision(t *testing.T) {
	s, err := buildMappedDeck(t.TempDir(), map[string][]string{"alpha": {"A"}, "beta": {"B"}, "nil": nil, "empty": {}})
	if err != nil {
		t.Fatal(err)
	}
	defer s.close()
	for name, want := range map[string][]string{"alpha": {"A"}, "beta": {"B"}, "nil": nil, "empty": {}} {
		got, err := s.materialize(name)
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("%s: %v %v", name, got, err)
		}
	}
	// Force all directory entries into one hash bucket. Lookup must still
	// compare each original key, including a missing key in that bucket.
	for i := range s.hashes {
		s.hashes[i].hash = 42
	}
	for _, name := range []string{"alpha", "beta"} {
		g, ok := s.groupWithHash(name, 42)
		if !ok {
			t.Fatal("collision hid key", name)
		}
		got, err := s.item(&g, 0)
		if err != nil || got != strings.ToUpper(name[:1]) {
			t.Fatal("collision returned wrong value", got, err)
		}
	}
	if _, ok := s.groupWithHash("missing", 42); ok {
		t.Fatal("hash equality accepted without key equality")
	}
}

func TestMappedDeckParserAndWeightCompatibility(t *testing.T) {
	d := mmapTestDice(t)
	fixtures := map[string]string{
		"x.json":       "\ufeff" + `{"extension":{"normal":1},"_keys":["g"],"g":["a",1,null,{},"::0::zero","::4294967296::large"],"_private":["p"]}`,
		"x.jsonc":      `{/* comment */"g":["a","b",],}`,
		"no-extension": `{"g":["a","b"]}`,
		"x.yaml":       "name: Y\ncommand: g\ndefault: [a, b]\nprivate: [p]\n",
		"x.toml":       "[meta]\ntitle='T'\n[decks]\ng=['a','b']\n__private=['p']\n",
	}
	for filename, body := range fixtures {
		t.Run(filename, func(t *testing.T) {
			// Compare to the actual existing parsers, not a second new decoder.
			old := newDeckInfo()
			if !parseDeck(d, filename, []byte(body), old) {
				t.Fatal("baseline parse failed")
			}
			mmapWrite(t, filepath.Join("data/decks", filename), body)
			mapped := d.readDeck(filepath.Join("data/decks", filename), "")
			if mapped.storage == nil {
				t.Fatal("mapping missing")
			}
			defer closeDecks(d, []*DeckInfo{mapped})
			if !reflect.DeepEqual(old.Command, mapped.Command) || old.Format != mapped.Format {
				t.Fatal("metadata differs")
			}
			for name, want := range old.DeckItems {
				got, err := mapped.storage.materialize(name)
				if err != nil || !reflect.DeepEqual(want, got) {
					t.Fatalf("group %s differs: %v", name, err)
				}
			}
		})
	}
	for _, tt := range []struct {
		name string
		body string
		want string
	}{
		{"object-extension", `{"extension":{"normal":1},"g":["ok"]}`, "ok"},
		{"mixed-array", `{"g":[1,null,{},"ok",[2]]}`, "ok"},
		{"zero-and-default", `{"g":["::0::BAD","ok"]}`, "ok"},
		{"wide-weight", `{"g":["::4294967296::WIDE","::1::small"]}`, "WIDE"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			fixtureDice := mmapTestDice(t)
			mmapWrite(t, "data/decks/compat.json", tt.body)
			DeckReload(fixtureDice)
			if decks := fixtureDice.DeckListSnapshot(); len(decks) != 1 || !decks[0].Enable {
				t.Fatalf("load rejected: %+v", decks)
			}
			ctx := &MsgContext{Dice: fixtureDice, diceRandSrc: newPCGDiceSource(41)}
			for i := range 32 {
				if got := mmapDraw(t, ctx, "g", false); got != tt.want {
					t.Fatalf("draw %d: got %q, want %q", i, got, tt.want)
				}
			}
		})
	}
	for _, text := range []string{"plain", "::0::zero", "::1::one", "::4294967296::large", "::9223372036854775808::overflow", "::x::bad", "::::", "::12:x"} {
		w, s := extractWeight(text)
		w2, s2 := deckEntryWeight(text)
		if w != w2 || s != s2 {
			t.Fatalf("weight parser mismatch: %q", text)
		}
	}
	items := []string{"::0::BAD", "default", "::3::heavy"}
	s, err := buildMappedDeck(t.TempDir(), map[string][]string{"g": items})
	if err != nil {
		t.Fatal(err)
	}
	defer s.close()
	deck := &DeckInfo{storage: s, contentVersion: 1}
	for _, shuffle := range []bool{false, true} {
		ctx := &MsgContext{Dice: d, diceRandSrc: newPCGDiceSource(19)}
		seen := map[string]int{}
		for range 2000 {
			got, err := drawStoredDeck(ctx, deck, "g", shuffle)
			if err != nil {
				t.Fatal(err)
			}
			seen[got]++
		}
		if seen["BAD"] != 0 || seen["default"] == 0 || seen["heavy"] == 0 {
			t.Fatalf("weights: %v", seen)
		}
		if !shuffle && (seen["heavy"] < 1300 || seen["heavy"] > 1700) {
			t.Fatalf("weighted distribution: %v", seen)
		}
		if shuffle && seen["heavy"] != 1000 {
			t.Fatalf("without replacement: %v", seen)
		}
	}
}

func TestMappedDeckBlocksOversizeEviction(t *testing.T) {
	items := make([]string, 901)
	for i := range 900 {
		items[i] = fmt.Sprintf("entry-%d-%s", i, strings.Repeat("abcdefgh", 1024))
	}
	items[900] = strings.Repeat("long合法文本", 700000)
	s, err := buildMappedDeck(t.TempDir(), map[string][]string{"g": items})
	if err != nil {
		t.Fatal(err)
	}
	defer s.close()
	g, _ := s.group("g")
	for _, i := range []int{0, 255, 256, 899, 900, 0, 256} {
		got, err := s.item(&g, i)
		if err != nil || got != items[i] {
			t.Fatalf("item %d: %v", i, err)
		}
	}
	for i := range 900 {
		if _, err := s.item(&g, i); err != nil {
			t.Fatal(err)
		}
	}
	if got, err := s.item(&g, 0); err != nil || got != items[0] {
		t.Fatal("reread after eviction", err)
	}
	deckBlockReader.Lock()
	used := deckBlockReader.bytes
	deckBlockReader.Unlock()
	if used > deckCacheBytes {
		t.Fatalf("cache exceeded budget: %d", used)
	}
}

func TestMappedDeckReloadWhileRecursiveCloudRequest(t *testing.T) {
	d := mmapTestDice(t)
	entered, release := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	unblock := func() { releaseOnce.Do(func() { close(release) }) }
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(entered)
		<-release
		_, _ = w.Write([]byte(`["{_leaf}"]`))
	}))
	defer server.Close()
	defer unblock()
	filename := "data/decks/cloud.toml"
	mmapWrite(t, filename, fmt.Sprintf("[decks]\n_leaf=['OLD']\n[root]\nexport=true\nvisible=true\ncloud_extra=true\noptions_urls=[%q]\n", server.URL))
	DeckReload(d)
	result := make(chan struct {
		text string
		err  error
	}, 1)
	go func() {
		_, text, err := deckDraw(&MsgContext{Dice: d}, "root", false)
		result <- struct {
			text string
			err  error
		}{text, err}
	}()
	select {
	case <-entered:
	case got := <-result:
		t.Fatalf("draw returned before entering the cloud request: %+v", got)
	}
	old := d.DeckListSnapshot()[0].storage
	mmapWrite(t, filename, "[decks]\nroot=['NEW']\n_leaf=['NEW']\n")
	DeckReload(d) // Must finish while the old network request is blocked.
	if old.closed || !cacheTestFileExists(t, old.filename) {
		t.Fatal("in-flight mapping closed")
	}
	unblock()
	got := <-result
	if got.err != nil || got.text != "OLD" {
		t.Fatalf("mixed snapshot: %+v", got)
	}
	if !old.closed || cacheTestFileExists(t, old.filename) {
		t.Fatal("retired mapping not released")
	}
	if got := mmapDraw(t, &MsgContext{Dice: d}, "root", false); got != "NEW" {
		t.Fatalf("new snapshot: %q", got)
	}
}

func TestMappedDeckShallowCopyAfterDraw(t *testing.T) {
	d := mmapTestDice(t)
	d.Config.VMVersionForDeck = "v2"
	filename := "data/decks/copy.json"
	mmapWrite(t, filename, `{"root":["[capture()]"],"g":["A","A"]}`)
	DeckReload(d)
	old := d.DeckListSnapshot()[0].storage
	ctx := &MsgContext{Dice: d}
	ctx.CreateVmIfNotExists()
	var copied *MsgContext
	ctx.vm.Attrs.Store("capture", ds.NewNativeFunctionVal(&ds.NativeFunctionData{
		Name: "capture",
		NativeFunc: func(_ *ds.Context, _ *ds.VMValue, _ []*ds.VMValue) *ds.VMValue {
			copied = ctx.ShallowCopy()
			// The copied context may outlive this VM invocation.
			copied.vm = nil
			return ds.NewStrVal("captured")
		},
	}))
	if got := mmapDraw(t, ctx, "g", true); got != "A" {
		t.Fatal(got)
	}
	if got := mmapDraw(t, ctx, "root", false); got != "captured" || copied == nil {
		t.Fatalf("context was not copied during draw: %q", got)
	}
	mmapWrite(t, filename, `{"root":["B"],"g":["::0::BAD","::9::B"]}`)
	DeckReload(d)
	if !old.closed || cacheTestFileExists(t, old.filename) {
		t.Fatal("an idle context copy must not retain the old mapping")
	}
	for range 4 {
		if got := mmapDraw(t, copied, "g", true); got != "B" {
			t.Fatalf("copied context used a retired version or pool: %q", got)
		}
	}
}

func TestMappedDeckShallowCopyInFlight(t *testing.T) {
	d := mmapTestDice(t)
	d.Config.VMVersionForDeck = "v2"
	entered, release := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	unblock := func() { releaseOnce.Do(func() { close(release) }) }
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(entered)
		<-release
		_, _ = w.Write([]byte(`["#{DRAW-leaf}"]`))
	}))
	defer server.Close()
	defer unblock()
	filename := "data/decks/copy.toml"
	mmapWrite(t, filename, fmt.Sprintf("[decks]\nroot=['[capture()]']\n[cloud]\nexport=true\nvisible=true\ncloud_extra=true\noptions_urls=[%q]\n", server.URL))
	mmapWrite(t, "data/decks/leaf.json", `{"leaf":["OLD"]}`)
	DeckReload(d)
	old := d.DeckListSnapshot()[0].storage
	ctx := &MsgContext{Dice: d}
	ctx.SetSplitKey("###deck-copy-test###")
	ctx.CreateVmIfNotExists()
	type drawResult struct {
		exists bool
		text   string
		err    error
	}
	result := make(chan drawResult, 1)
	ctx.vm.Attrs.Store("capture", ds.NewNativeFunctionVal(&ds.NativeFunctionData{
		Name: "capture",
		NativeFunc: func(_ *ds.Context, _ *ds.VMValue, _ []*ds.VMValue) *ds.VMValue {
			copied := ctx.ShallowCopy()
			copied.vm = nil
			mmapWrite(t, filename, "[decks]\nroot=['NEW']\n")
			mmapWrite(t, "data/decks/leaf.json", `{"leaf":["NEW"]}`)
			DeckReload(d)
			go func() {
				ok, text, err := deckDraw(copied, "cloud", false)
				result <- drawResult{ok, text, err}
			}()
			select {
			case <-entered:
				return ds.NewStrVal("captured")
			case got := <-result:
				t.Fatalf("active copy failed to inherit the parent version: %+v", got)
				return nil
			}
		},
	}))
	if got := mmapDraw(t, ctx, "root", false); got != "captured" {
		t.Fatal(got)
	}
	// The parent has returned, but the child still owns the old version. Its
	// subsequent cross-deck recursive draw must use OLD, not the published NEW.
	if old.closed || !cacheTestFileExists(t, old.filename) {
		t.Fatal("parent released the child's mapping")
	}
	if got := mmapDraw(t, &MsgContext{Dice: d}, "leaf", false); got != "NEW" {
		t.Fatal(got)
	}
	unblock()
	if got := <-result; !got.exists || got.err != nil || got.text != "OLD" {
		t.Fatalf("child mixed versions after its parent returned: %+v", got)
	}
	if !old.closed || cacheTestFileExists(t, old.filename) {
		t.Fatal("child did not release the retired mapping")
	}
}

func TestMappedDeckConcurrentABA(t *testing.T) {
	d := mmapTestDice(t)
	filename := "data/decks/a.json"
	mmapWrite(t, filename, `{"root":["A"]}`)
	DeckReload(d)
	for _, value := range []string{"B", "A", "B", "A"} {
		var wg sync.WaitGroup
		start := make(chan struct{})
		errs := make(chan error, 8)
		for range 8 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-start
				ctx := &MsgContext{Dice: d}
				// InitSplitKey's process-global RNG has a pre-existing race.
				// Supply the message delimiter independently of deck concurrency.
				ctx.SetSplitKey("###deck-concurrency-test###")
				for range 30 {
					ctx.deckDepth = 0
					ok, text, err := deckDraw(ctx, "root", false)
					if err != nil || !ok || (text != "A" && text != "B") {
						errs <- fmt.Errorf("draw: %q %v: %w", text, ok, err)
						return
					}
				}
			}()
		}
		close(start)
		body, _ := json.Marshal(map[string][]string{"root": {value}})
		mmapWrite(t, filename, string(body))
		DeckReload(d)
		wg.Wait()
		close(errs)
		for err := range errs {
			t.Fatal(err)
		}
		for range 30 {
			if got := mmapDraw(t, &MsgContext{Dice: d}, "root", false); got != value {
				t.Fatalf("after publish %q != %q", got, value)
			}
		}
	}
}

func TestMappedDeckBuildFailureIsClean(t *testing.T) {
	dir := t.TempDir()
	blocker := filepath.Join(dir, "regular-file")
	mmapWrite(t, blocker, "block")
	if _, err := buildMappedDeck(blocker, map[string][]string{"g": {"old"}}); err == nil {
		t.Fatal("expected deterministic failure")
	}
	if _, err := buildMappedDeck(dir, map[string][]string{"g": {"::9223372036854775807::a", "::9223372036854775807::b", "::2::c"}}); err == nil {
		t.Fatal("overflow silently narrowed")
	}
	files, err := filepath.Glob(filepath.Join(dir, "*.blocks"))
	if err != nil || len(files) != 0 {
		t.Fatalf("partial files: %v %v", files, err)
	}
}

func TestMappedDeckMatchesLegacyDrawSequence(t *testing.T) {
	d := mmapTestDice(t)
	for _, items := range [][]string{
		{"a", "b", "c"},
		{"::0::never", "::3::a", "default", "::9::b", "::4294967296::wide"},
	} {
		s, err := buildMappedDeck(t.TempDir(), map[string][]string{"g": items})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if err := s.close(); err != nil {
				t.Error(err)
			}
		})
		for _, shuffle := range []bool{false, true} {
			legacy := &DeckInfo{DeckItems: map[string][]string{"g": items}}
			mapped := &DeckInfo{storage: s, contentVersion: 1}
			oldCtx := &MsgContext{Dice: d, diceRandSrc: newPCGDiceSource(41)}
			newCtx := &MsgContext{Dice: d, diceRandSrc: newPCGDiceSource(41)}
			for range 40 {
				want, e1 := executeDeck(oldCtx, legacy, "g", shuffle)
				got, e2 := executeDeck(newCtx, mapped, "g", shuffle)
				if e1 != nil || e2 != nil || want != got {
					t.Fatalf("shuffle=%v old=%q/%v new=%q/%v", shuffle, want, e1, got, e2)
				}
			}
		}
	}
	// The legacy replacement chooser accepts totals wider than a signed int.
	items := []string{"::9223372036854775807::a", "::1::b"}
	s, err := buildMappedDeck(t.TempDir(), map[string][]string{"g": items})
	if err != nil {
		t.Fatal(err)
	}
	defer s.close()
	for range 10 {
		if _, err := drawStoredDeck(&MsgContext{Dice: d}, &DeckInfo{storage: s}, "g", false); err != nil {
			t.Fatal(err)
		}
	}
}

func TestMappedDeckDownloadedCorpus(t *testing.T) {
	dir := os.Getenv("SEAL_DECK_CORPUS")
	if dir == "" {
		t.Skip("set SEAL_DECK_CORPUS to the fetch.py output directory")
	}
	body, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var files []struct {
		File   string `json:"file"`
		Member string `json:"member"`
	}
	if err := json.Unmarshal(body, &files); err != nil {
		t.Fatal(err)
	}
	if len(files) < 3 {
		t.Fatal("not enough corpus decks")
	}
	d := mmapTestDice(t)
	for _, file := range files {
		t.Run(file.Member, func(t *testing.T) {
			content, err := os.ReadFile(filepath.Join(dir, file.File))
			if err != nil {
				t.Fatal(err)
			}
			old := newDeckInfo()
			if !parseDeck(d, file.File, content, old) {
				// The downloaded example.toml is not a valid legacy deck. Assert
				// identical rejection, rather than counting it as a successful load.
				if file.File != "143273f78d3d.toml" {
					t.Fatalf("unexpected legacy rejection: %s", old.ErrText)
				}
				next := d.readDeck(filepath.Join(dir, file.File), "")
				if next == nil || next.Enable || next.storage != nil || next.ErrText != old.ErrText {
					t.Fatal("invalid fixture rejection differs")
				}
				t.Logf("both reject invalid fixture: %s", old.ErrText)
				return
			}
			s, err := buildMappedDeck(t.TempDir(), old.DeckItems)
			if err != nil {
				t.Fatal(err)
			}
			defer s.close()
			checked := 0
			for name, items := range old.DeckItems {
				g, exists := s.group(name)
				if !exists || g.count != len(items) {
					t.Fatalf("group %s count", name)
				}
				for i, want := range items {
					got, err := s.item(&g, i)
					if err != nil || want != got {
						t.Fatalf("%s[%d] mismatch: %v", name, i, err)
					}
					checked++
				}
			}
			t.Logf("verified %d groups, %d entries, %d source bytes", s.groupCount(), checked, len(content))
			var command string
			switch file.File {
			case "1ade496f16f5.json":
				command = "匕首心角色"
			case "21db2db7e762.json":
				command = "lor接待来宾"
			case "15c74ebfc6da.toml":
				command = "来一杯奶茶"
			case "f1ae8ea77fc7.toml":
				command = "旅游地点抽取"
			case "fe10b7fa2746.json":
				command = "coc法术"
			case "3d00ce865e1a.json":
				command = "宝可梦作成"
			case "69e4ed53bc20.json":
				command = "挥指"
			case "17a5bc472f8e.json":
				command = "宝可梦钓鱼"
			case "abfea7132815.json":
				command = "附近的宝可梦"
			case "355cc99b8e5c.json":
				command = "随机宝可梦六连"
			case "9e1d9d34d160.json":
				command = "诺弗兰炼金"
			case "5bad5566ce8e.json":
				command = "卢恩符文"
			case "0e2254386d1f.json":
				command = "六版神话生物"
			case "e0a1e869c4a9.json":
				command = "旷野之息图鉴"
			case "16045caa67bc.json":
				command = "酷刑"
			case "44b13b5e816c.json":
				command = "考研单词"
			case "0a7ec3e394f2.json":
				command = "哈利波特魔杖"
			case "79f623789f19.json":
				command = "古希腊神祗"
			case "3b1e103e72a8.json":
				command = "本草纲目"
			case "d504fe3f15bf.json":
				command = "宝石"
			case "cdef7cca870e.json":
				command = "司辰的注意"
			}
			if command != "" {
				mapped := *old
				mapped.DeckItems = nil
				mapped.RawData = nil
				mapped.storage = s
				c1 := &MsgContext{Dice: d, Player: &GroupPlayerInfo{}, EndPoint: &EndPointInfo{}, diceRandSrc: newPCGDiceSource(2026)}
				c2 := &MsgContext{Dice: d, Player: &GroupPlayerInfo{}, EndPoint: &EndPointInfo{}, diceRandSrc: newPCGDiceSource(2026)}
				for range 32 {
					c1.deckDepth, c2.deckDepth = 0, 0
					want, e1 := executeDeck(c1, old, command, true)
					got, e2 := executeDeck(c2, &mapped, command, true)
					if e1 != nil || e2 != nil || want != got {
						t.Fatalf("recursive legacy comparison: old=%q/%v new=%q/%v", want, e1, got, e2)
					}
				}
			}
		})
	}
}

func TestMappedDeckFilesystemFailureFallsBack(t *testing.T) {
	d := mmapTestDice(t)
	blocked := filepath.Join(t.TempDir(), "file")
	mmapWrite(t, blocked, "not a directory")
	d.BaseConfig.DataDir = blocked
	t.Setenv("TMP", blocked)
	t.Setenv("TEMP", blocked)
	t.Setenv("TMPDIR", blocked)
	deck := newDeckInfo()
	deck.Enable = true
	deck.DeckItems["g"] = []string{"::0::BAD", "::5::ok"}
	d.compactDeck(deck)
	if deck.storage != nil || len(deck.DeckItems["g"]) != 2 {
		t.Fatal("fallback lost original deck")
	}
	for _, shuffle := range []bool{false, true} {
		ctx := &MsgContext{Dice: d}
		for range 10 {
			got, err := executeDeck(ctx, deck, "g", shuffle)
			if err != nil || got != "ok" {
				t.Fatalf("fallback draw %q/%v", got, err)
			}
		}
	}
}

func TestMappedDeckUpdateStoreIdentity(t *testing.T) {
	d := mmapTestDice(t)
	d.StoreManager = &StoreManager{InstalledDecks: map[string]bool{}}
	mmapWrite(t, "data/decks/store.json", `{"_storeID":["store-id"],"_etag":["etag"],"g":["old"]}`)
	DeckReload(d)
	mmapWrite(t, "update.tmp", `{"g":["new"]}`)
	if err := d.DeckUpdate(d.DeckListSnapshot()[0], "update.tmp"); err != nil {
		t.Fatal(err)
	}
	deck := d.DeckListSnapshot()[0]
	if deck.StoreID != "store-id" || deck.Etag != "etag" || !d.StoreManager.InstalledDecks["store-id"] {
		t.Fatal("omitted identity metadata was cleared")
	}
	mmapWrite(t, "update.tmp", `{"_storeID":[""],"_etag":[""],"g":["new"]}`)
	if err := d.DeckUpdate(deck, "update.tmp"); err != nil {
		t.Fatal(err)
	}
	deck = d.DeckListSnapshot()[0]
	if deck.StoreID != "" || deck.Etag != "" || len(d.StoreManager.InstalledDecks) != 0 {
		t.Fatal("explicit identity clearing was ignored")
	}
}
