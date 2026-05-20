package helpdoc_test

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	helpdoc "sealdice-core/api/v2/helpdoc"
	helpdocm "sealdice-core/api/v2/model/helpdoc"
	"sealdice-core/dice"
	"sealdice-core/logger"
	"sealdice-core/model/common/request"
)

func newTestHelpDocService(t *testing.T) *helpdoc.Service {
	t.Helper()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Chdir(%q): %v", tempDir, err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(cwd)
	})
	if err := os.MkdirAll(filepath.Join("data", "helpdoc"), 0o755); err != nil {
		t.Fatalf("mkdir helpdoc dir: %v", err)
	}

	d := &dice.Dice{
		Logger: logger.M(),
	}
	d.BaseConfig.Name = "helpdoc-test"
	d.BaseConfig.DataDir = tempDir
	d.Config = dice.NewConfig(d)
	dm := &dice.DiceManager{
		Dice: []*dice.Dice{d},
		Help: &dice.HelpManager{
			HelpDocTree: []*dice.HelpDoc{
				{
					Key:   "file-1",
					Name:  "sample.json",
					Path:  "data/helpdoc/sample.json",
					Group: "default",
					Type:  ".json",
				},
			},
			Config: &dice.HelpConfig{Aliases: map[string][]string{
				"default": {"d"},
			}},
		},
	}
	d.Parent = dm
	return helpdoc.NewService(dm)
}

func TestGetTreeReturnsHelpDocTree(t *testing.T) {
	svc := newTestHelpDocService(t)

	resp, err := svc.GetTree(t.Context(), &request.Empty{})
	if err != nil {
		t.Fatalf("GetTree returned error: %v", err)
	}
	if len(resp.Body.Item.Data) != 1 {
		t.Fatalf("tree length = %d, want 1", len(resp.Body.Item.Data))
	}
	if resp.Body.Item.Data[0].Name != "sample.json" {
		t.Fatalf("tree first name = %q, want sample.json", resp.Body.Item.Data[0].Name)
	}
}

func TestSaveConfigPersistsAliases(t *testing.T) {
	svc := newTestHelpDocService(t)

	resp, err := svc.SetConfig(t.Context(), &helpdocm.ConfigReq{
		Body: request.RequestWrapper[helpdocm.HelpConfigBody]{
			Body: helpdocm.HelpConfigBody{
				Aliases: map[string][]string{"default": {"d", "main"}},
			},
		},
	})
	if err != nil {
		t.Fatalf("SetConfig returned error: %v", err)
	}
	if !resp.Body.Item.Success {
		t.Fatalf("SetConfig success = false, want true")
	}

	data, err := os.ReadFile(filepath.Join("data", "helpdoc", dice.HelpConfigFilename))
	if err != nil {
		t.Fatalf("read saved config: %v", err)
	}
	if string(data) == "" {
		t.Fatal("saved config is empty")
	}
}

func TestInitUploadRejectsBuiltinGroup(t *testing.T) {
	svc := newTestHelpDocService(t)

	_, err := svc.InitUpload(t.Context(), &helpdocm.UploadInitReq{
		Body: request.RequestWrapper[helpdocm.HelpDocUploadInitReqBody]{
			Body: helpdocm.HelpDocUploadInitReqBody{
				Group:     "builtin",
				Filename:  "sample.json",
				FileSize:  2,
				FileHash:  "abc",
				ChunkSize: 1,
			},
		},
	})
	if err == nil {
		t.Fatal("InitUpload returned nil error, want builtin group rejection")
	}
}

func TestChunkUploadCompletesIntoSelectedGroup(t *testing.T) {
	svc := newTestHelpDocService(t)
	content := []byte(`{"mod":"demo","helpdoc":{"hello":"world"}}`)
	hash := sha256.Sum256(content)

	initResp, err := svc.InitUpload(t.Context(), &helpdocm.UploadInitReq{
		Body: request.RequestWrapper[helpdocm.HelpDocUploadInitReqBody]{
			Body: helpdocm.HelpDocUploadInitReqBody{
				Group:     "team",
				Filename:  "team.json",
				FileSize:  int64(len(content)),
				FileHash:  hex.EncodeToString(hash[:]),
				ChunkSize: 16,
			},
		},
	})
	if err != nil {
		t.Fatalf("InitUpload returned error: %v", err)
	}
	sessionID := initResp.Body.Item.SessionID

	chunks := [][]byte{content[:16], content[16:32], content[32:]}
	for index, chunk := range chunks {
		if _, err := svc.UploadChunk(t.Context(), &helpdocm.UploadChunkReq{
			SessionID: sessionID,
			Index:     index,
			RawBody:   chunk,
		}); err != nil {
			t.Fatalf("UploadChunk(%d) returned error: %v", index, err)
		}
	}

	completeResp, err := svc.CompleteUpload(t.Context(), &helpdocm.UploadCompleteReq{
		Body: request.RequestWrapper[helpdocm.HelpDocUploadCompleteReqBody]{
			Body: helpdocm.HelpDocUploadCompleteReqBody{SessionID: sessionID},
		},
	})
	if err != nil {
		t.Fatalf("CompleteUpload returned error: %v", err)
	}
	if !completeResp.Body.Item.Success {
		t.Fatalf("CompleteUpload success = false, want true")
	}
	if _, err := os.Stat(filepath.Join("data", "helpdoc", "team", "team.json")); err != nil {
		t.Fatalf("uploaded helpdoc not found: %v", err)
	}
}
