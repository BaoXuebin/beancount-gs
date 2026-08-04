package ledger

import (
	"sync"
	"time"

	"github.com/beancount-gs/api/internal/ai"
	"github.com/beancount-gs/api/internal/beancount"
	"github.com/beancount-gs/api/internal/db"
)

type Amount struct {
	Number   string
	Currency string
}

type Cost struct {
	Number   string
	Currency string
	Date     string
	Label    string
}

type Posting struct {
	Account string
	Units   *Amount
	Cost    *Cost
	Price   *Amount
}

type Transaction struct {
	ID        string
	Date      string
	Flag      string
	Payee     string
	Narration string
	Tags      []string
	Links     []string
	Postings  []Posting
}

type Filters struct {
	From    string
	To      string
	Month   string
	Account string
	Tag     string
	Q       string
	Order   string // asc | desc
}

type Actor struct {
	UserID string
	Login  string
}

type Service struct {
	Store  *db.Store
	Engine beancount.QueryEngine
	AI     *ai.Client
	Now    func() time.Time
	FX     FXProvider // 汇率拉取（测试可注入；nil 时用默认公开接口）
	locks  sync.Map   // ledgerID -> *sync.Mutex，串行化同一账本的文件写入
}
