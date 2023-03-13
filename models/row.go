package models

//Based on the database scheme 2022-05-13
//Check updates of crescent-backend repository
//Recovery field used for updating last commit state by recovery program

type DBFlush struct {
	FlushType int //0: normal  1: too many events. early flushing. 2: timeout. late flushing
}

type AccountRow struct {
	Address         string `db:"address" json:"address"`
	HexAddr         string `db:"hex_addr" json:"hexAddress"`
	CreateTimestamp int64  `db:"created_ts" json:"createdTimestamp"`
	UpdateTimestamp int64  `db:"update_ts" json:"updateTimestamp"`
}

type BalanceRow struct {
	Address      string `db:"hex_addr" json:"address"`
	Denom        string `db:"denom" json:"denom"`
	Amount       string `db:"amount" json:"amount"` //dec
	UpdateHeight int64  `db:"update_height" json:"update_height"`
	Timestamp    int64  `db:"-" json:"-"`
	Recovery     int    `db:"-" json:"-"`
}

type TotalSupplyRow struct {
	Denom        string `db:"denom" json:"denom"`
	Amount       string `db:"amount" json:"amount"` //dec
	UpdateHeight int64  `db:"update_height" json:"update_height"`
	Recovery     int    `db:"-" json:"-"`
}

type DenomRow struct {
	Denom       string `json:"denom" db:"denom"`
	OrgChainId  string `json:"chain_id" db:"chain_id"`
	Ticker      string `json:"ticker" db:"ticker_nm"`
	LogoUrl     string `json:"logoUrl" db:"logo_url"`
	BaseDenom   string `json:"base_denom" db:"base_denom"`
	IbcPath     string `json:"ibc_path" db:"ibc_path"` // transfer/orgSideChannel/denom
	Whitelisted int    `json:"whitelisted" db:"whitelisted"`
	Exponent    int    `json:"exponent" db:"exponent"`
}

type PairRow struct {
	PairId       uint64 `json:"pairId" db:"pair_id"`
	BaseDenom    string `json:"baseDenom" db:"base_denom"`
	QuoteDenom   string `json:"quoteDenom" db:"quote_denom"`
	EscrowAddr   string `json:"escrowAddr" db:"escrow_addr"`
	LastPrice    string `json:"lastPrice" db:"last_price"`
	CurrentBatch uint64 `json:"currentBatch" db:"current_batch"`
	Whitelisted  bool   `json:"whitelisted" db:"whitelisted"`
	Deleted      bool   `json:"deleted" db:"deleted"`
}

type OrderbookRow struct {
	PairId    uint64 `json:"pairId" db:"pair_id"`
	Prec      int    `json:"precision" db:"prec"`
	Json      string `json:"orderbook" db:"json"`
	Timestamp int64  `json:"timestamp" db:"timestamp"`
	BasePrice string `json:"basePrice" db:"base_price"`
}

type PoolRow struct {
	PoolId                uint64 `json:"poolId" db:"pool_id"`
	PairId                uint64 `json:"pailId" db:"pair_id"`
	PoolDenom             string `json:"poolDenom" db:"pool_denom"`
	ReserveAddr           string `json:"reserveAddr" db:"reserve_address"`
	Disabled              bool   `json:"disabled" db:"disabled"`
	CreatedHeight         int64  `json:"createdHeight" db:"created_height"`
	Status                int    `json:"status" db:"status"`
	StakingReserveHexAddr string `json:"-" db:"staking_reserve_hex_addr"`
	PoolType              int    `json:"poolType" db:"pool_type"`
	MinPrice              string `json:"minPrice" db:"min_price"`
	MaxPrice              string `json:"maxPrice" db:"max_price"`
	Creator               string `json:"creator" db:"creator"`
}

type PoolPriceRow struct {
	PoolId          uint64 `json:"poolId" db:"pool_id"`
	Price           string `json:"price" db:"price"`
	UpdateHeight    int64  `json:"updateHeight" db:"update_height"`
	UpdateTimestamp int64  `json:"updateTimestamp" db:"update_ts"`
}

type TimeHeightRow struct {
	ChainId   string `json:"chainId" db:"chain_id"`
	TraceYN   string `json:"traceYN" db:"trace_yn"`
	Height    int64  `json:"height" db:"height"`
	Timestamp int64  `json:"timestamp" db:"ts"`
	Timeout   int64  `json:"timeout" db:"timeout"`
}

type ClaimRecordRow struct {
	Owner             string `json:"owner" db:"owner"` // our-side
	AirdropId         int    `json:"airdropId" db:"airdrop_id"`
	InitCoins         string `json:"initCoins" db:"initial_claimable_coins"`
	ClaimableCoins    string `json:"claimableCoins" db:"claimable_coins"`
	ClaimedConditions string `json:"claimedConditions" db:"claimed_conditions"`
	Height            int64  `json:"height" db:"height"`
	Timestamp         int64  `json:"timestamp" db:"timestamp"`
	TxHash            string `json:"txhash" db:"txhash"`
}

// status
// 0 send-checked
// 1 commited(sent)
// 2 recved at other side
// 3 acked  - finish
// 4 timeouted  - finish

type TXIbcTransferRow struct {
	TxHash             string `json:"txhash" db:"txhash"`
	Owner              string `json:"owner" db:"owner"`                // our-side
	OwnerChannel       string `json:"ownerChannel" db:"owner_channel"` // our-side
	Denom              string `json:"denom" db:"denom"`                // our-side
	Amount             string `json:"amount" db:"amount"`              //dec
	Receiver           string `db:"receiver" json:"receiver"`
	Sender             string `db:"sender" json:"sender"`
	Status             int    `json:"status" db:"status"`
	CommitHeight       int64  `json:"commitHeight" db:"commit_height"`
	RecvHeight         int64  `json:"recvHeight" db:"recv_height"`
	ResultHeight       int64  `json:"resultHeight" db:"rst_height"`
	PacketSeq          string `json:"pktSeq" db:"pkt_seq"`
	SendChannel        string `json:"sendChannel" db:"send_ch"`
	RecvChannel        string `json:"recvChannel" db:"recv_ch"`
	BroadcastTimestamp int64  `json:"broadcastTimestamp" db:"broadcast_ts"`
	CommitTimestamp    int64  `json:"sendTimestamp" db:"send_ts"`
	RecvTimestamp      int64  `json:"recvTimestamp" db:"recv_ts"`
	ResultTimestamp    int64  `json:"resultTimestamp" db:"result_ts"`
	SendDenom          string `json:"sendDenom" db:"send_denom"`      // denom of token at send-side chain
	SendChainId        string `json:"sendChainId" db:"send_chain_id"` // denom of token at send-side chain
	Direction          int    `json:"direction" db:"direction"`
}

type DepositRow struct {
	PoolId           uint64 `db:"pool_id" json:"pool_id,omitempty"`
	ReqId            uint64 `db:"req_id" json:"id,omitempty"`
	MsgHeight        int64  `db:"height" json:"msg_height,omitempty"`
	Owner            string `db:"owner" json:"depositor,omitempty"`
	DepositAAmount   string `db:"deposit_a_amount" json:"depositAAmount"`
	DepositBAmount   string `db:"deposit_b_amount" json:"depositBAmount"`
	AcceptedAAmount  string `db:"accepted_a_amount" json:"acceptedAAmount"`
	AcceptedBAmount  string `db:"accepted_b_amount" json:"acceptedBAmount"`
	DenomA           string `json:"denomA" db:"denom_a"`
	DenomB           string `json:"denomB" db:"denom_b"`
	MintedPoolAmount string `json:"mintedPoolAmount" db:"minted_pool_amount"`
	Status           int    `json:"status" db:"status"`
	TxHash           string `json:"txhash" db:"txhash"`
	Timestamp        int64  `db:"timestamp" json:"timestamp,omitempty"`
}
type WithdrawRow struct {
	PoolId          uint64 `db:"pool_id" json:"pool_id,omitempty"`
	ReqId           uint64 `db:"req_id" json:"id,omitempty"`
	Height          int64  `db:"height" json:"msg_height,omitempty"`
	Owner           string `db:"owner" json:"depositor,omitempty"`
	OfferPoolAmount string `db:"offer_pool_amount" json:"offerPoolAmount"`
	WithdrawAAmount string `db:"withdraw_a_amount" json:"withdrawAAmount"`
	WithdrawBAmount string `db:"withdraw_b_amount" json:"withdrawBAmount"`
	DenomA          string `json:"denomA" db:"denom_a"`
	DenomB          string `json:"denomB" db:"denom_b"`

	Status    int    `json:"status" db:"status"`
	TxHash    string `json:"txhash" db:"txhash"`
	Timestamp int64  `db:"timestamp" json:"timestamp,omitempty"`
}

// more than one possibility for one pair/req
// batch, height indicate the original value of request created.
// json for record tx event
type SwapFilledRow struct {
	PairId             uint64 `db:"pair_id" json:"pairId,omitempty"`
	ReqId              uint64 `db:"req_id" json:"reqId,omitempty"`
	BatchId            uint64 `db:"batch_id" json:"batchId,omitempty"`
	Status             int    `db:"status" json:"-"`
	Height             int64  `db:"height" json:"-"`
	Timestamp          int64  `db:"timestamp" json:"-"`
	OfferDenom         string `db:"offer_denom" json:"offerDenom"`
	DemandDenom        string `db:"demand_denom" json:"demandDenom"`
	FilledOfferAmount  string `db:"offer_amount" json:"offerAmount"`   // for this batch
	FilledDemandAmount string `db:"demand_amount" json:"demandAmount"` // for this batch
	Price              string `db:"price" json:"price"`
	SwappedBaseAmount  string `db:"swapped_base_amount" json:"swappedBaseAmount"` // for this batch
	Owner              string `db:"owner" json:"-"`
	UpdateHeight       int64  `db:"update_height" json:"-"` // updated
}

// json for record tx event
type SwapReqRow struct {
	PairId               uint64 `db:"pair_id" json:"pairId,omitempty"`
	ReqId                uint64 `db:"req_id" json:"reqId,omitempty"`
	Height               int64  `db:"height" json:"-"` // order tx height
	Timestamp            int64  `db:"timestamp" json:"-"`
	Price                string `db:"order_price" json:"orderPrice"`
	FilledBaseAmount     string `db:"filled_base_amount" json:"-"`             // accumulated
	OpenBaseAmount       string `db:"open_base_amount" json:"orderBaseAmount"` // accumulated
	ExpireTimestamp      int64  ` db:"expire_ts" json:"expireTimestamp"`
	Direction            int    `db:"direction" json:"direction,omitempty"`
	Status               int    `db:"status" json:"-"`
	OfferAmount          string `db:"offer_amount" json:"offerAmount"`
	RemainOfferAmount    string `db:"remain_offer_amount" json:"remainOfferAmount"`
	OfferDenom           string `db:"offer_denom" json:"offerDenom"`
	DemandDenom          string `db:"demand_denom" json:"demandDenom"`
	DemandReceivedAmount string `db:"demand_received_amount" json:"receivedDemandAmount"` // acc
	Owner                string `db:"owner" json:"-"`
	TxHash               string `json:"txhash" db:"txhash"`
	UpdateHeight         int64  `db:"update_height" json:"-"` // req state updated
	OrderType            int64  `db:"order_type" json:"orderType"`
}

type StakingRow struct {
	Owner           string `db:"owner" json:"-"`
	Denom           string `db:"denom" json:"denom"`
	StartEpoch      uint64 `db:"start_epoch" json:"-"`
	QueueAmount     string `db:"queue_amount" json:"queueAmount"`
	Amount          string `db:"amount" json:"-"`
	UpdateHeight    int64  `db:"update_height" json:"updateHeight"`
	UpdateTimestamp int64  `db:"update_ts" json:"-"`
	TxHash          string `json:"txhash" db:"txhash"`
}

type TotalStakeRow struct {
	Denom           string `db:"denom" json:"denom"`
	Amount          string `db:"amount" json:"amount"`
	UpdateHeight    int64  `db:"update_height" json:"updateHeight"`
	UpdateTimestamp int64  `db:"update_ts" json:"timestamp"`
	ReserveHexAddr  string `db:"reserve_hex_addr" json:"-"`
}

type HistoricalRewardRow struct {
	Denom           string `db:"denom" json:"denom"`
	Epoch           uint64 `db:"epoch" json:"epoch"`
	RewardDenom     string `db:"reward_denom"`
	RewardAmount    string `db:"reward_amount"`
	UpdateHeight    int64  `db:"update_height" json:"updateHeight"`
	UpdateTimestamp int64  `db:"update_ts" json:"timestamp"`
}

type RewardDistRow struct {
	Owner      string `db:"owner"`
	PoolDenom  string `db:"pool_denom"`
	DistDenom  string `db:"dist_denom"`
	DistAmount string `db:"dist_amount"`
	Height     int64  `db:"height"`
	Timestamp  int64  `db:"timestamp"`
}

type UnhavestRewardsRow struct {
	Height       int64  `db:"height"`
	Timestamp    int64  `db:"timestamp"`
	Owner        string `db:"owner"`
	PoolDenom    string `db:"pool_denom"`
	RewardDenom  string `db:"reward_denom"`
	RewardAmount string `db:"reward_amount"`
}

type FarmPlanRow struct {
	PlanId            uint64 `db:"plan_id"`
	PoolDenom         string `db:"pool_denom"`
	RewardDenom       string `db:"reward_denom"`
	RewardAmount      int64  `db:"reward_amount"`
	FarmingPoolAddr   string `db:"farming_pool_addr"`
	TerminationAddr   string `db:"termination_addr"`
	PlanType          int    `db:"plan_type"`
	StartTimestamp    int64  `db:"start_timestamp"`
	EndTimestamp      int64  `db:"end_timestamp"`
	LastDistTimestamp int64  `db:"last_dist_timestamp"`
	Terminated        int    `db:"terminated"`
	Height            int64  `db:"height"`
}

// only one of pair id or pooldenom should be set
type LpFarmPlanRow struct {
	PlanId             uint64 `db:"plan_id"`
	PairId             uint64 `db:"target_pair_id"`
	PoolDenom          string `db:"target_pool_denom"`
	RewardDenom        string `db:"reward_denom"`
	RewardAmountPerDay string `db:"reward_amount_per_day"` //decimal
	FarmingPoolAddr    string `db:"farming_src_addr"`      // source of farming rewards
	TerminationAddr    string `db:"termination_addr"`      // destination addr of farming rewards when terminated
	StartTimestamp     int64  `db:"start_ts"`
	EndTimestamp       int64  `db:"end_ts"`
	Private            int    `db:"private_yn"`
	Terminated         int    `db:"terminated"`
	UpdateHeight       int64  `db:"update_height"`
}

type LpFarmPositionRow struct {
	Farmer              string `db:"farmer"`
	StakeDenom          string `db:"stake_denom"`
	StakeAmount         string `db:"stake_amount"`
	StartingBlockHeight int64  `db:"start_height"`
	PreviousPeriod      uint64 `db:"previous_period"`
	UpdateHeight        int64  `db:"update_height"`
	UpdateTimestamp     int64  `db:"update_ts"`
}

type LpFarmHistoricalRewardRow struct {
	Denom           string `db:"denom"`
	RewardDenom     string `db:"reward_denom"`
	Period          uint64 `db:"period"`
	RewardAmount    string `db:"reward_amount"`
	UpdateHeight    int64  `db:"update_height"`
	UpdateTimestamp int64  `db:"update_ts"`
}

type LpFarmRow struct {
	FarmDenom               string `db:"farm_denom"`
	RewardDenom             string `db:"reward_denom"`
	TotalFarmingAmount      string `db:"total_farming_amount"`
	Period                  uint64 `db:"period"`
	RewardCurrentAmount     string `db:"reward_current_amt"`
	RewardOutstandingAmount string `db:"reward_outstanding_amt"`
	Height                  int64  `db:"update_height"`
	Timestamp               int64  `db:"update_ts"`
	PairFarmingRatio        string `db:"pair_farming_ratio"` // if no farmer exists, this value is 0. ignore if 0 is received
}

type VoteRow struct {
	Owner      string `db:"owner"`
	ProposalId string `db:"proposal_id"`
	VoteOption string `db:"vote_option"`
	Height     int64  `db:"height"`
	Timestamp  int64  `db:"timestamp"`
	TxHash     string `db:"txhash" json:"txhash"`
}

type ProposalRow struct {
	ProposalId   string `db:"proposal_id"`
	ProposalObj  string `db:"proposal_obj"`
	ProposalType string `db:"proposal_type"`

	Height    int64  `db:"height"`
	Timestamp int64  `db:"timestamp"`
	Proposer  string `db:"proposer"`
	TxHash    string `db:"txhash" json:"txhash"`
	Deleted   int    `db:"deleted" json:"-"`
	Recovery  int    `db:"-" json:"-"`
}

type LiquidStakingRow struct {
	Owner        string `db:"owner" json:"-"`
	Amount       string `db:"amount" json:"amount"`
	BondedAmount string `db:"bonded_amount" json:"bondedAmount"`
	Height       int64  `db:"height" json:"-"`
	Timestamp    int64  `db:"timestamp" json:"-"`
	TxHash       string `db:"txhash" json:"txhash"`
}

type LiquidUnstakingRow struct {
	Owner           string `db:"owner" json:"-"`
	UnbondingAmount string `db:"unbonding_amount"  json:"unbondingAmount"`
	Amount          string `db:"amount" json:"amount" `
	CompleteTsSec   int64  `db:"complete_ts_sec"  json:"completeTimestamp"`
	Height          int64  `db:"height" json:"-"`
	Timestamp       int64  `db:"timestamp" json:"-"`
	TxHash          string `db:"txhash"  json:"txhash"`
}

// Liquid Farm Module
type LiquidFarmRow struct {
	PoolId           uint64 `db:"pool_id" json:"poolId"`
	PoolDenom        string `db:"pool_denom" json:"poolDenom"`
	LfDenom          string `db:"lf_denom" json:"lfDenom"`
	MinDepositAmount string `db:"min_deposit_amount" json:"minDepositAmount"`
	MinBidAmount     string `db:"min_bid_amount" json:"minBidAmount"`
	FarmingAddr      string `db:"farming_addr" json:"farmingAddr"`
	FeeRate          string `db:"fee_rate" json:"feeRate"`
	ActiveTimestamp  int64  `db:"active_ts" json:"-"`
}

type LiquidFarmAuction struct {
	AuctionId      uint64 `db:"auction_id" json:"auctionId"`
	PoolId         uint64 `db:"pool_id" json:"poolId"`
	BidCoinDenom   string `db:"bid_coin_denom" json:"bidCoinDenom"`
	PayReserveAddr string `db:"pay_reserve_addr" json:"payReserveAddr"`
	WinningAddr    string `db:"winning_addr" json:"winningAddr"`
	WinningAmount  string `db:"winning_amount" json:"winningAmount"`
	StartTimestamp int64  `db:"start_ts" json:"startTimestamp"`
	EndTimestamp   int64  `db:"end_ts" json:"endTimestamp"`
	Status         int    `db:"status" json:"status"`
	FeeRate        string `db:"fee_rate" `
}

type LiquidFarmAuctionReward struct {
	PoolId       uint64 `db:"pool_id" json:"poolId"`
	AuctionId    uint64 `db:"auction_id" json:"auctionId"`
	RewardDenom  string `db:"reward_denom" json:"rewardDenom"`
	RewardAmount string `db:"reward_amount" json:"rewardAmount"`
}

type LiquidFarmAuctionFee struct {
	PoolId    uint64 `db:"pool_id" json:"poolId"`
	AuctionId uint64 `db:"auction_id" json:"auctionId"`
	FeeDenom  string `db:"fee_denom" json:"feeDenom"`
	FeeAmount string `db:"fee_amount" json:"feeAmount"`
}

type LiquidFarmCompoundReward struct {
	PoolId          uint64 `db:"pool_id" json:"poolId"`
	Amount          string `db:"amount" json:"amount"`
	UpdateHeight    int64  `db:"update_height"`
	UpdateTimestamp int64  `db:"update_ts"`
}

//rpc event
type LiquidFarmFarm struct {
	PoolId          uint64 `db:"pool_id" json:"poolId"`
	Farmer          string `db:"farmer" json:"farmer"`
	FarmingCoin     string `db:"farming_coin" json:"farmingCoin"`
	MintedCoin      string `db:"minted_coin" json:"mintedCoin"`
	Height          int64  `db:"height" json:"-"`
	Timestamp       int64  `db:"timestamp" json:"-"`
	TxHash          string `db:"txhash" json:"txhash"`
}

//rpc event
type LiquidFarmUnfarm struct {
	PoolId        uint64 `db:"pool_id" json:"poolId"`
	Farmer        string `db:"farmer" json:"farmer"`
	UnfarmingCoin string `db:"unfarming_coin" json:"unfarmingCoin"`
	UnfarmedCoin  string `db:"unfarmed_coin" json:"unfarmedCoin"`
	Height        int64  `db:"height" json:"-"`
	Timestamp     int64  `db:"timestamp" json:"-"`
	TxHash        string `db:"txhash" json:"txhash"`
}
type EventRow struct {
	Owner         string `db:"owner_addr" `
	Height        int64  `db:"height" `
	TimestampUsec int64  `db:"timestamp_us"`
	EvtType       string `db:"evt_type"`
	EvtGroup      string `db:"evt_group"`
	TxType        string `db:"tx_type"`
	Result        []byte `db:"result"`
	TxHash        string `db:"txhash"`
}
