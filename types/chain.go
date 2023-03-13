package types

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type ChainStatus int
type CounterChannelInfo struct {
	ChainId     string
	ChannelName string
}

type Chain struct {
	ChainId              string
	AccPrefix            string
	StoreFile            string
	LogFile              string
	TraceStartTime       time.Time
	WsRpcEndpoint        string
	GRPCEndpoint         string
	Status               ChainStatus
	LastStoreUpdated     time.Time
	LastRPCUpdated       time.Time
	LastRPCRestUpdated   time.Time
	LastTraceHeight      int64
	LastRPCHeight        int64
	LastRPCRestHeight    int64
	TimeoutSec           int // if a store or newBlock event is not occured in timeoutsec, alert
	CounterChannel       map[string]CounterChannelInfo
	MainSide             int
	SubscriptionDisabled int
}

type OrderInfo struct {
}

type OrderbookInfo struct {
}

type PairInfo struct {
	PairId        uint64
	BaseDenom     string // move to pair
	QuoteDenom    string
	PoolInfo      []*PoolInfo
	LastPrice     sdk.Dec
	PrevLastPrice sdk.Dec // use to bar. updated last of flushing in dbproc
	//LastPoolOrderbook *amm.OrderBook
	LastBarTs       int64 // last bar ts(at start of bar timestamp)
	Whitelisted     int
	PriceMultiplier sdk.Dec // base denom prec / quote denom prec
	VolPrec         int64   // base denom precesion. int

}

type PoolInfo struct {
	PoolId                 uint64
	PairId                 uint64
	PoolDenom              string
	PoolReserveAddr        string
	PoolReserveHexAddr     string
	StakingReserveHexAddr  string
	BaseAmount             sdk.Int
	QuoteAmount            sdk.Int
	TotalSupply            sdk.Int
	PoolAmountUpdateHeight int64
	NeedRefresh            bool
	PoolType               int
	Creator                string
	TranslationX           *sdk.Dec
	TranslationY           *sdk.Dec
	MinPrice               *sdk.Dec
	MaxPrice               *sdk.Dec
	PoolPrice              sdk.Dec
	PoolPricePush          bool // need to flush price to db
	Disabled               bool
}
