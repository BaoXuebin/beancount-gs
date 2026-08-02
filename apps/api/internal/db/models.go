package db

// 时间戳统一以 UTC RFC3339 字符串存储，避免 SQLite 对 time.Time 扫描的不确定性。

type User struct {
	ID          string
	GitHubID    string
	GitHubLogin string
	Email       string
	DisplayName string
	CreatedAt   string
}

type Team struct {
	ID          string
	Name        string
	OwnerUserID string
	Role        string
	MemberCount int
	LedgerCount int
	CreatedAt   string
}

type Ledger struct {
	ID                string
	TeamID            string
	Name              string
	DataPath          string
	OperatingCurrency string
	StartDate         string
	OpeningBalances   string
	IsBak             bool
	Revision          int64
	Role              string
	MemberCount       int
	CreatedAt         string
}

type AuditLog struct {
	ID        int64
	LedgerID  string
	UserID    string
	Actor     string
	Action    string
	Object    string
	Detail    string
	CreatedAt string
}
