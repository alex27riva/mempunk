package explorer

// OverviewVM is the view model for the overview page.
type OverviewVM struct {
	Chain        string
	Blocks       int64
	Headers      int64
	BestHash     string
	Difficulty   float64
	IBD          bool
	RecentBlocks []BlockSummaryVM
}

// BlockSummaryVM is a single row in the overview block list.
type BlockSummaryVM struct {
	Height  int64
	Hash    string
	Time    int64
	Age     string
	TxCount int
	Size    int
	Weight  int
	SizeFmt string
}

// BlockVM is the view model for the block detail page.
type BlockVM struct {
	Hash          string
	Height        int64
	Time          int64
	Age           string
	Confirmations int64
	TxCount       int
	Size          int
	Weight        int
	StrippedSize  int
	Difficulty    float64
	MerkleRoot    string
	PrevHash      string
	NextHash      string
	Nonce         uint32
	Bits          string
	Version       int32
	TotalFeeSats  int64
	TotalFeeBTC   string
	AvgFeeRate    int64
	Txs           []TxSummaryVM // nil when ShowTxDetails is false; capped to TxShown
	TxShown       int           // number of txs included in Txs
	HasMoreTxs    bool          // true when TxCount > TxShown
}

// TxSummaryVM is a row in the block transaction list.
type TxSummaryVM struct {
	Txid    string
	VSize   int
	Weight  int
	Outputs []OutputVM
}

// TxVM is the view model for the transaction detail page.
type TxVM struct {
	Txid          string
	Hash          string
	Version       int32
	Size          int
	VSize         int
	Weight        int
	LockTime      uint32
	Confirmed     bool
	BlockHash     string
	BlockTime     int64
	Age           string
	Confirmations int64
	Coinbase      bool
	TotalIn       int64 // satoshis; -1 for coinbase
	TotalOut      int64
	Fee           int64   // TotalIn - TotalOut; -1 for coinbase
	FeeRate       float64 // sat/vB
	Inputs        []InputVM
	Outputs       []OutputVM
	RawJSON       string // pretty-printed for <details>
	// Mempool-only fields
	MempoolTime int64
	MempoolFee  int64
}

// InputVM is a single transaction input.
type InputVM struct {
	Index      int
	Txid       string
	Vout       uint32
	Sequence   uint32
	IsCoinbase bool
	ValueSats  int64
	ValueBTC   string
	Address    string
	ScriptType string
}

// OutputVM is a single transaction output.
type OutputVM struct {
	Index      int
	ValueSats  int64
	ValueBTC   string
	Address    string
	ScriptType string
	Spent      bool
}

// AddressVM is the view model for the address page (validateaddress only).
type AddressVM struct {
	Address   string
	IsValid   bool
	Network   string
	IsScript  bool
	IsWitness bool
}

// NodeVM is the view model for the node info page.
type NodeVM struct {
	Version         int
	VersionFmt      string // e.g. "27.1.0"
	Subversion      string
	ProtocolVersion int
	Connections     int
	ConnectionsIn   int
	ConnectionsOut  int
	Chain           string
	Blocks          int64
	Headers         int64
	BestHash        string
	Difficulty      float64
	SizeOnDisk      int64
	SizeFmt         string
	Pruned          bool
	IBD             bool
}
