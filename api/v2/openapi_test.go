package v2

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/danielgtaylor/huma/v2"

	storym "sealdice-core/api/v2/model/story"
)

func TestBuildOpenAPIIncludesCurrentV2Routes(t *testing.T) {
	spec := BuildOpenAPI()
	if spec == nil {
		t.Fatal("expected OpenAPI spec")
	}

	data, err := json.Marshal(spec)
	if err != nil {
		t.Fatalf("marshal OpenAPI spec: %v", err)
	}
	if !json.Valid(data) {
		t.Fatal("OpenAPI spec is not valid JSON")
	}

	for _, path := range []string{
		"/sd-api/v2/base/health",
		"/sd-api/v2/base/login",
		"/sd-api/v2/backup/list",
		"/sd-api/v2/group/list",
		"/sd-api/v2/group/platforms",
		"/sd-api/v2/ban/list",
		"/sd-api/v2/config/reply",
		"/sd-api/v2/config/advanced",
		"/sd-api/v2/deck/list",
		"/sd-api/v2/deck/reload",
		"/sd-api/v2/deck/upload",
		"/sd-api/v2/deck/delete",
		"/sd-api/v2/deck/check-update",
		"/sd-api/v2/deck/update",
		"/sd-api/v2/story/info",
		"/sd-api/v2/story/logs/page",
		"/sd-api/v2/story/items/page",
		"/sd-api/v2/story/log",
		"/sd-api/v2/story/upload-log",
		"/sd-api/v2/story/backup/list",
		"/sd-api/v2/story/backup/download",
		"/sd-api/v2/story/backup/batch-delete",
		"/sd-api/v2/story/cleanup/preview",
		"/sd-api/v2/story/cleanup",
		"/sd-api/v2/js/status",
		"/sd-api/v2/js/list",
		"/sd-api/v2/js/reload",
		"/sd-api/v2/js/shutdown",
		"/sd-api/v2/js/enable",
		"/sd-api/v2/js/disable",
		"/sd-api/v2/js/delete",
		"/sd-api/v2/js/execute",
		"/sd-api/v2/js/record",
		"/sd-api/v2/js/check-update",
		"/sd-api/v2/js/update",
		"/sd-api/v2/js/upload",
		"/sd-api/v2/js/configs",
		"/sd-api/v2/js/configs/reset",
		"/sd-api/v2/js/dead-configs",
		"/sd-api/v2/js/dead-configs/delete",
		"/sd-api/v2/js/{name}/data",
		"/sd-api/v2/js/{name}/data/list",
		"/sd-api/v2/js/{name}/data/delete",
		"/sd-api/v2/js/{name}/data/shrink",
		"/sd-api/v2/js/{name}/data/info",
		"/sd-api/v2/custom-reply/files",
		"/sd-api/v2/custom-reply/files/{filename}",
		"/sd-api/v2/custom-reply/files/{filename}/conditions",
		"/sd-api/v2/custom-reply/files/{filename}/download",
		"/sd-api/v2/custom-reply/files/{filename}/rules",
		"/sd-api/v2/custom-reply/files/upload",
		"/sd-api/v2/custom-reply/debug-mode",
		"/sd-api/v2/custom-text/",
		"/sd-api/v2/custom-text/{category}",
		"/sd-api/v2/custom-text/{category}/preview-refresh",
		"/sd-api/v2/imconnection/protocols",
		"/sd-api/v2/imconnection/{id}/enable",
		"/sd-api/v2/imconnection/{id}/workflow",
		"/sd-api/v2/imconnection/{id}/qrcode",
		"/sd-api/v2/imconnection/sign-info",
	} {
		if spec.Paths[path] == nil {
			t.Fatalf("expected path %s in OpenAPI spec", path)
		}
	}
}

func TestStoryWriteSchemasDisallowLegacyFields(t *testing.T) {
	spec := BuildOpenAPI()
	if spec == nil {
		t.Fatal("expected OpenAPI spec")
	}

	deleteSchemaName := huma.SchemaFromType(spec.Components.Schemas, reflect.TypeOf(storym.DeleteLogReqBody{})).Ref
	uploadSchemaName := huma.SchemaFromType(spec.Components.Schemas, reflect.TypeOf(storym.UploadLogReqBody{})).Ref

	deleteSchema := huma.SchemaFromType(spec.Components.Schemas, reflect.TypeOf(storym.DeleteLogReqBody{}))
	if deleteSchemaName != "" {
		deleteSchema = spec.Components.Schemas.SchemaFromRef(deleteSchemaName)
	}
	if addl, ok := deleteSchema.AdditionalProperties.(bool); !ok || addl {
		t.Fatalf("delete schema additionalProperties = %#v, want false", deleteSchema.AdditionalProperties)
	}

	uploadSchema := huma.SchemaFromType(spec.Components.Schemas, reflect.TypeOf(storym.UploadLogReqBody{}))
	if uploadSchemaName != "" {
		uploadSchema = spec.Components.Schemas.SchemaFromRef(uploadSchemaName)
	}
	if addl, ok := uploadSchema.AdditionalProperties.(bool); !ok || addl {
		t.Fatalf("upload schema additionalProperties = %#v, want false", uploadSchema.AdditionalProperties)
	}
}
