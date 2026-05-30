package explorer

// Internal types for unmarshaling Bitcoin Core RPC responses.

type rpcBlockchainInfo struct {
	Chain      string  `json:"chain"`
	Blocks     int64   `json:"blocks"`
	Headers    int64   `json:"headers"`
	BestHash   string  `json:"bestblockhash"`
	Difficulty float64 `json:"difficulty"`
	MedianTime int64   `json:"mediantime"`
	SizeOnDisk int64   `json:"size_on_disk"`
	Pruned     bool    `json:"pruned"`
	IBD        bool    `json:"initialblockdownload"`
	Progress   float64 `json:"verificationprogress"`
}

// rpcBlockHeader holds fields common to all getblock verbosity levels.
type rpcBlockHeader struct {
	Hash          string  `json:"hash"`
	Height        int64   `json:"height"`
	Time          int64   `json:"time"`
	MedianTime    int64   `json:"mediantime"`
	NTx           int     `json:"nTx"`
	Size          int     `json:"size"`
	Weight        int     `json:"weight"`
	StrippedSize  int     `json:"strippedsize"`
	Difficulty    float64 `json:"difficulty"`
	MerkleRoot    string  `json:"merkleroot"`
	PrevHash      string  `json:"previousblockhash"`
	NextHash      string  `json:"nextblockhash"`
	Nonce         uint32  `json:"nonce"`
	Bits          string  `json:"bits"`
	Version       int32   `json:"version"`
	Confirmations int64   `json:"confirmations"`
}

// rpcBlockV1 is the getblock verbosity-1 response (tx = txid strings).
type rpcBlockV1 struct {
	rpcBlockHeader
	Tx []string `json:"tx"`
}

// rpcBlockV2 is the getblock verbosity-2 response (tx = full objects).
type rpcBlockV2 struct {
	rpcBlockHeader
	Tx []rpcTx `json:"tx"`
}

type rpcBlockStats struct {
	TotalFee      int64 `json:"totalfee"`
	AvgFeeRate    int64 `json:"avgfeerate"`
	MedianFeeRate int64 `json:"medianfeerate"`
}

type rpcTx struct {
	Txid          string    `json:"txid"`
	Hash          string    `json:"hash"`
	Version       int32     `json:"version"`
	Size          int       `json:"size"`
	VSize         int       `json:"vsize"`
	Weight        int       `json:"weight"`
	LockTime      uint32    `json:"locktime"`
	BlockHash     string    `json:"blockhash"`
	BlockTime     int64     `json:"blocktime"`
	Time          int64     `json:"time"`
	Confirmations int64     `json:"confirmations"`
	Vin           []rpcVin  `json:"vin"`
	Vout          []rpcVout `json:"vout"`
}

type rpcVin struct {
	Txid     string      `json:"txid"`
	Vout     uint32      `json:"vout"`
	Coinbase string      `json:"coinbase"`
	Sequence uint32      `json:"sequence"`
	Prevout  *rpcPrevout `json:"prevout"`
}

type rpcPrevout struct {
	Value        float64   `json:"value"`
	ScriptPubKey rpcScript `json:"scriptPubKey"`
}

type rpcVout struct {
	Value        float64   `json:"value"`
	N            int       `json:"n"`
	ScriptPubKey rpcScript `json:"scriptPubKey"`
}

type rpcScript struct {
	Address   string   `json:"address"`
	Addresses []string `json:"addresses"` // legacy multi-address format
	Type      string   `json:"type"`
	ASM       string   `json:"asm"`
	Hex       string   `json:"hex"`
}

type rpcMempoolEntry struct {
	Fees struct {
		Base float64 `json:"base"`
	} `json:"fees"`
	VSize  int   `json:"vsize"`
	Weight int   `json:"weight"`
	Time   int64 `json:"time"`
	Height int64 `json:"height"`
}

type rpcNetworkInfo struct {
	Version         int    `json:"version"`
	Subversion      string `json:"subversion"`
	ProtocolVersion int    `json:"protocolversion"`
	Connections     int    `json:"connections"`
	ConnectionsIn   int    `json:"connections_in"`
	ConnectionsOut  int    `json:"connections_out"`
}

type rpcValidateAddress struct {
	IsValid   bool   `json:"isvalid"`
	Address   string `json:"address"`
	IsScript  bool   `json:"isscript"`
	IsWitness bool   `json:"iswitness"`
}
