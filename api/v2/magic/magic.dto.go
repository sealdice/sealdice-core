package magic

type SourceKind string

const (
	SourceKindSQLite     SourceKind = "sqlite"
	SourceKindMySQL      SourceKind = "mysql"
	SourceKindPostgreSQL SourceKind = "postgres"
)

type SourceStage string

const (
	SourceStageCurrent       SourceStage = "current"
	SourceStageLegacyV146    SourceStage = "legacy_v146"
	SourceStageUnknownLegacy SourceStage = "unknown_legacy"
)

type SourceProfile struct {
	Kind       SourceKind `json:"kind"`
	SQLitePath string     `json:"sqlitePath,omitempty"`
	DSN        string     `json:"dsn,omitempty"`
}

type InspectReq struct {
	Body struct {
		Source SourceProfile `json:"source"`
	} `json:"body"`
}

type SchemaFingerprint struct {
	HasAttrs                bool     `json:"hasAttrs"`
	HasLegacyAttrsUser      bool     `json:"hasLegacyAttrsUser"`
	HasLegacyAttrsGroup     bool     `json:"hasLegacyAttrsGroup"`
	HasLegacyAttrsGroupUser bool     `json:"hasLegacyAttrsGroupUser"`
	HasGroupInfo            bool     `json:"hasGroupInfo"`
	HasGroupPlayerInfo      bool     `json:"hasGroupPlayerInfo"`
	HasBanInfo              bool     `json:"hasBanInfo"`
	HasEndpointInfo         bool     `json:"hasEndpointInfo"`
	HasLogs                 bool     `json:"hasLogs"`
	HasLogItems             bool     `json:"hasLogItems"`
	HasCensorLog            bool     `json:"hasCensorLog"`
	Tables                  []string `json:"tables"`
}

type InspectResult struct {
	Kind                 SourceKind        `json:"kind"`
	Stage                SourceStage       `json:"stage"`
	CanDirectMigrate     bool              `json:"canDirectMigrate"`
	RequiresSQLiteRepair bool              `json:"requiresSqliteRepair"`
	RequiresV150Upgrade  bool              `json:"requiresV150Upgrade"`
	Messages             []string          `json:"messages"`
	Fingerprint          SchemaFingerprint `json:"fingerprint"`
}
