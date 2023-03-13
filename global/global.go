package global

import (
	"cremon-liquidity/config"
	"cremon-liquidity/types"
	"sync"
)

var Config *config.Config
var OrderNestMap map[uint64](map[uint64]*types.OrderInfo)
var PairOrderbook map[uint64]*types.OrderbookInfo
var PairInfos map[uint64]*types.PairInfo
var Hex2PoolInfo map[string]*types.PoolInfo
var Denom2PoolInfo map[string]*types.PoolInfo
var AckCheckMap map[string][]string
var AckCheckMapMutex map[string]*sync.Mutex
var TgMsg chan string // global channel to send msg

type UpdateReq struct {
	ReqType   string
	VarIn64   int64
	VarString string
}

var UpdateCh chan UpdateReq

type HProfRec struct {
	Height int64 // current height

	TsTracestoreBegin  int64 // start ts
	TsTracestoreEnd    int64 // end tracestore
	TsDBBegin          int64
	TsDBEnd            int64
	TsRPCNewBlockBegin int64
	TsRPCNewBlockEnd   int64
	TsRPCFirstTXBegin  int64
	TsRPCLastTXEnd     int64
	TraceEventCnt      uint32 // # of trace operation

	M sync.Mutex
}

// global Height Profile Record
var GHPR HProfRec
