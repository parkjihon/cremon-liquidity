package traceproc

import (
	"bytes"
	"context"
	"cremon-liquidity/global"
	"cremon-liquidity/models"
	"cremon-liquidity/types"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	farmingtypes "github.com/crescent-network/crescent/v4/x/farming/types"
	liquiditytypes "github.com/crescent-network/crescent/v4/x/liquidity/types"
	"github.com/nxadm/tail"
	"github.com/rs/zerolog/log"
)

type TraceProc struct {
	Chain          *types.Chain // config chain
	DataSourcePath string
	WatchedOps     []TraceOpBytes
	Cdc            codec.Codec
	LastHeight     uint64
	IOChan         chan TraceOperation // from trace file
	ErrorChan      chan error
	DbProcCh       chan interface{}
	BypassHeight   int64
	BypassPoolId   map[uint64]string // poolkey stored every withdraw and deposit. if poolkey already inserted. skip it
	BypassBankAddr map[string]string // hex_addr -> addr. skip it if already added. check only 32byte hex_address(including module, pool reserved addr )
	StartRpcCh     chan interface{}
	BurstCounter   int // set limit of flush. force push to prevent max packet drop
	CountPair      struct {
		Write   uint64
		Delete  uint64
		Read    uint64
		Iterate uint64
	}
	CountPool struct {
		Write   uint64
		Delete  uint64
		Read    uint64
		Iterate uint64
	}
	CountDeposit struct {
		Write   uint64
		Delete  uint64
		Read    uint64
		Iterate uint64
	}
	CountWithdraw struct {
		Write   uint64
		Delete  uint64
		Read    uint64
		Iterate uint64
	}
	CountOrder struct {
		Write   uint64
		Delete  uint64
		Read    uint64
		Iterate uint64
	}
}

type TraceOpBytes []byte

const (
	WriteOpStr     = "write"
	DeleteOpStr    = "delete"
	ReadOpStr      = "read"
	IterRangeOpStr = "iterRange"
	IterKeyOpStr   = "iterKey"
	IterValueOpStr = "iterValue"
)

var (
	WriteOpBytes     TraceOpBytes = []byte(WriteOpStr)
	DeleteOpBytes    TraceOpBytes = []byte(DeleteOpStr)
	ReadOpBytes      TraceOpBytes = []byte(ReadOpStr)
	IterRangeOpBytes TraceOpBytes = []byte(IterRangeOpStr)
	IterKeyOpBytes   TraceOpBytes = []byte(IterKeyOpStr)
	IterValueOpBytes TraceOpBytes = []byte(IterValueOpStr)
)

func NewTraceProc(chainInfo *types.Chain, dbCh chan interface{}) (*TraceProc, error) {
	//trigger to start RPC procs
	startRpcCh := make(chan interface{})
	// 탐색할 operation
	// JIHON: 여기에 ReadOpBytes 및 IterRangeOpBytes 넣어야 함
	//watchOps := []TraceOpBytes{WriteOpBytes, DeleteOpBytes}
	//watchOps := []TraceOpBytes{DeleteOpBytes}
	watchOps := []TraceOpBytes{IterRangeOpBytes, IterKeyOpBytes, IterValueOpBytes, ReadOpBytes, WriteOpBytes, DeleteOpBytes}
	//watchOps := []TraceOpBytes{IterRangeOpBytes, WriteOpBytes, DeleteOpBytes}

	t := &TraceProc{
		Chain:          chainInfo,
		DataSourcePath: chainInfo.StoreFile,
		WatchedOps:     watchOps,
		IOChan:         make(chan TraceOperation, 20000), //TODO optimize buf size
		ErrorChan:      make(chan error),
		//DataProcCh:     toDataProcCh,
		DbProcCh:       dbCh,
		BypassPoolId:   make(map[uint64]string),
		BypassBankAddr: make(map[string]string),
		StartRpcCh:     startRpcCh,
	}

	return t, nil
}

func (tp *TraceProc) TraceFile(ctx context.Context) {

	errCnt := 0

	log.Info().Str("DataSource File", global.Config.TraceFile).Msg("TraceProc start")

	if errCnt > 0 {
		// when error limit is reached(ex. 3 times), stop tracing
		log.Warn().Msg("Read Storefile error")
		if errCnt > 3 {
			//global.TgMsg <- "[PANIC] can't open store-file"
			log.Panic().Msg("TraceStore open error")

		}
		time.Sleep(10 * time.Millisecond)
	}

	t, err := tail.TailFile(
		global.Config.TraceFile, tail.Config{Follow: false, ReOpen: false, Pipe: true, Logger: tail.DiscardingLogger})
	if err != nil {
		log.Warn().Str("err", err.Error()).Msg("tail creation error")
		global.TgMsg <- "TailFile Error"
		errCnt++
		//break
	}
	errCnt = 0

	//atomic.StoreInt64(&global.GHPR.Height, to.BlockHeight)

	to := TraceOperation{}
	for line := range t.Lines {

		if line.Err != nil {
			fmt.Printf("line reading error, line %v, error %s", line, err.Error())
			//line error is critical. stop tracing
			global.TgMsg <- "[EMERGENCY] Line ERROR!!!"
			log.Warn().Str("err", line.Err.Error()).Msg("line reading error")
			panic("line. ERR")
		}

		if len(line.Text) == 0 {
			//continue
			break
		}
		lineBytes := []byte(line.Text)

		// if tp.StartRpcCh != nil {
		// 	//Start all rpc processes by closing trigger channel
		// 	close(tp.StartRpcCh)
		// 	fmt.Print("=================== START RPC triggered by trace =============\n")
		// 	tp.StartRpcCh = nil
		// }

		if len(lineBytes) < 25 {
			fmt.Println("라인 끝?")
			global.TgMsg <- fmt.Sprintf("[PANIC]line reading error len=%d, line %s, error:too short", len(line.Text), line.Text)

			//line error is critical. stop tracing
			panic("line.ERR")
		}

		// contains operation string
		cmpBytes := lineBytes[13:23]
		cmpMatch := false
		for _, op := range tp.WatchedOps {
			if bytes.Contains(cmpBytes, op) {
				cmpMatch = true
				break
			}
		}

		if !cmpMatch {
			continue
		}

		//fmt.Println("&")

		to = TraceOperation{}
		if err := json.Unmarshal(lineBytes, &to); err != nil {
			global.TgMsg <- fmt.Sprintf("=====failed unmarshaling, %s=========\ndata: %s", err.Error(), line.Text)
			continue
			//return
		}

		// data: {"operation":"write","key":"AA==","value":"","metadata":null}
		if to.Operation == WriteOpStr && len(to.Value) == 0 {
			// Some kv doesn't have any value even if the operation mode is write.
			// TODO. Check again there is no missing data with no-value
			// limit order write with no data
			// global.TgMsg <- "no data in write o, data:" + line.Text
			continue
		}

		to.SysTsRead = time.Now().UnixNano()

		// start of new height
		if to.BlockHeight != 0 && to.BlockHeight != global.GHPR.Height {
			if global.GHPR.Height == 0 {
				fmt.Println("탐색 시작 블록:", to.BlockHeight)
			}
			if to.BlockHeight%100 == 0 {
				fmt.Println("현재 블록:", to.BlockHeight, "Pair#:", tp.CountPair, "Pool#:", tp.CountPool, "Deposit#:", tp.CountDeposit, "Withdraw#:", tp.CountWithdraw, "Order#:", tp.CountOrder)
			}

			//reset burst count
			tp.BurstCounter = 0

			if global.GHPR.TsRPCNewBlockBegin != 0 {
				// export data
				//(&types.GHPR).Print()
			}

			atomic.StoreInt64(&global.GHPR.Height, to.BlockHeight)
			atomic.StoreInt64(&global.GHPR.TsTracestoreBegin, to.SysTsRead)
			atomic.StoreInt64(&global.GHPR.TsTracestoreEnd, to.SysTsRead) //set same as start timestamp, update in every line
			atomic.StoreInt64(&global.GHPR.TsDBBegin, 0)
			atomic.StoreInt64(&global.GHPR.TsDBEnd, 0)
			atomic.StoreInt64(&global.GHPR.TsRPCFirstTXBegin, 0)
			atomic.StoreInt64(&global.GHPR.TsRPCLastTXEnd, 0)
			atomic.StoreInt64(&global.GHPR.TsRPCNewBlockBegin, 0)
			atomic.StoreInt64(&global.GHPR.TsRPCNewBlockEnd, 0)
			atomic.StoreUint32(&global.GHPR.TraceEventCnt, 1)

			// pass trace block height at the start of new height
			h := models.TimeHeightRow{
				ChainId:   tp.Chain.ChainId,
				TraceYN:   "Y",
				Height:    to.BlockHeight,
				Timestamp: time.Now().UnixMicro(),
				Timeout:   0,
			}
			tp.DbProcCh <- h
			//fmt.Print("***Push TraceHeight***:", to.BlockHeight)

			// update global vars
			atomic.StoreInt64(&tp.Chain.LastTraceHeight, to.BlockHeight)
			tp.Chain.LastStoreUpdated = time.Now()

		} else if to.BlockHeight == global.GHPR.Height {

			atomic.StoreInt64(&global.GHPR.TsTracestoreEnd, to.SysTsRead)
			atomic.AddUint32(&global.GHPR.TraceEventCnt, 1)

		}
		//fmt.Printf("\nLine:%d %d ==> len %d \n", to.BlockHeight, to.SysTsRead, len(tp.IOChan))

		if tp.BypassHeight != 0 && tp.BypassHeight >= to.BlockHeight {
			if tp.BypassHeight == to.BlockHeight {
				tp.BypassHeight = 0
			}
			continue
		}

		//fmt.Println(to)

		tp.IOChan <- to
	} // end line loop

	time.Sleep(time.Second)
	fmt.Println("탐색 종료 블록:", to.BlockHeight, "Pair#:", tp.CountPair, "Pool#:", tp.CountPool, "Deposit#:", tp.CountDeposit, "Withdraw#:", tp.CountWithdraw, "Order#:", tp.CountOrder)
	fmt.Println("-------+------------+---------------+---------------+---------------+---------------+")
	fmt.Printf("%-20s|%15s|%15s|%15s|%15s|\n", "prefix", "write", "delete", "read", "iterate")
	fmt.Printf("%-20s|%15d|%15d|%15d|%15d|\n", "pair", tp.CountPair.Write, tp.CountPair.Delete, tp.CountPair.Read, tp.CountPair.Iterate)
	fmt.Printf("%-20s|%15d|%15d|%15d|%15d|\n", "pool", tp.CountPool.Write, tp.CountPool.Delete, tp.CountPool.Read, tp.CountPool.Iterate)
	fmt.Printf("%-20s|%15d|%15d|%15d|%15d|\n", "deposit", tp.CountDeposit.Write, tp.CountDeposit.Delete, tp.CountDeposit.Read, tp.CountDeposit.Iterate)
	fmt.Printf("%-20s|%15d|%15d|%15d|%15d|\n", "withdraw", tp.CountWithdraw.Write, tp.CountWithdraw.Delete, tp.CountWithdraw.Read, tp.CountWithdraw.Iterate)
	fmt.Printf("%-20s|%15d|%15d|%15d|%15d|\n", "order", tp.CountOrder.Write, tp.CountOrder.Delete, tp.CountOrder.Read, tp.CountOrder.Iterate)
	fmt.Println("-------+------------+---------------+---------------+---------------+---------------+")
	os.Exit(1)
}

func (tp *TraceProc) TraceStore(ctx context.Context) {

	errCnt := 0

	log.Info().Str("DataSource", tp.DataSourcePath).Msg("TraceProc start")

	for { // infinite cycle

		if errCnt > 0 {
			// when error limit is reached(ex. 3 times), stop tracing
			log.Warn().Msg("Read Storefile error")
			if errCnt > 3 {
				//global.TgMsg <- "[PANIC] can't open store-file"
				log.Panic().Msg("TraceStore open error")

			}
			time.Sleep(10 * time.Millisecond)
		}

		t, err := tail.TailFile(
			tp.DataSourcePath, tail.Config{Follow: true, ReOpen: true, Pipe: true, Logger: tail.DiscardingLogger})
		if err != nil {
			log.Warn().Str("err", err.Error()).Msg("tail creation error")
			global.TgMsg <- "TailFile Error"
			errCnt++
			break
		}
		errCnt = 0

		for line := range t.Lines {
			if line.Err != nil {
				fmt.Printf("line reading error, line %v, error %s", line, err.Error())
				//line error is critical. stop tracing
				global.TgMsg <- "[EMERGENCY] Line ERROR!!!"
				log.Warn().Str("err", line.Err.Error()).Msg("line reading error")
				panic("line. ERR")
			}

			if len(line.Text) == 0 {
				continue
			}
			lineBytes := []byte(line.Text)

			if tp.StartRpcCh != nil {
				//Start all rpc processes by closing trigger channel
				close(tp.StartRpcCh)
				fmt.Print("=================== START RPC triggered by trace =============\n")
				tp.StartRpcCh = nil
			}

			if len(lineBytes) < 25 {

				global.TgMsg <- fmt.Sprintf("[PANIC]line reading error len=%d, line %s, error:too short", len(line.Text), line.Text)

				//line error is critical. stop tracing
				panic("line.ERR")
			}

			// contains operation string
			cmpBytes := lineBytes[13:23]
			cmpMatch := false
			for _, op := range tp.WatchedOps {
				if bytes.Contains(cmpBytes, op) {
					cmpMatch = true
					break
				}
			}

			if !cmpMatch {
				continue
			}

			//fmt.Println("&")

			to := TraceOperation{}
			if err := json.Unmarshal(lineBytes, &to); err != nil {
				global.TgMsg <- fmt.Sprintf("=====failed unmarshaling, %s=========\ndata: %s", err.Error(), line.Text)
				continue
				//return
			}

			// data: {"operation":"write","key":"AA==","value":"","metadata":null}
			if to.Operation == WriteOpStr && len(to.Value) == 0 {
				// Some kv doesn't have any value even if the operation mode is write.
				// TODO. Check again there is no missing data with no-value
				// limit order write with no data
				// global.TgMsg <- "no data in write o, data:" + line.Text
				continue
			}

			to.SysTsRead = time.Now().UnixNano()

			// start of new height
			if to.BlockHeight != 0 && to.BlockHeight != global.GHPR.Height {

				//reset burst count
				tp.BurstCounter = 0

				if global.GHPR.TsRPCNewBlockBegin != 0 {
					// export data
					//(&types.GHPR).Print()
				}

				atomic.StoreInt64(&global.GHPR.Height, to.BlockHeight)
				atomic.StoreInt64(&global.GHPR.TsTracestoreBegin, to.SysTsRead)
				atomic.StoreInt64(&global.GHPR.TsTracestoreEnd, to.SysTsRead) //set same as start timestamp, update in every line
				atomic.StoreInt64(&global.GHPR.TsDBBegin, 0)
				atomic.StoreInt64(&global.GHPR.TsDBEnd, 0)
				atomic.StoreInt64(&global.GHPR.TsRPCFirstTXBegin, 0)
				atomic.StoreInt64(&global.GHPR.TsRPCLastTXEnd, 0)
				atomic.StoreInt64(&global.GHPR.TsRPCNewBlockBegin, 0)
				atomic.StoreInt64(&global.GHPR.TsRPCNewBlockEnd, 0)
				atomic.StoreUint32(&global.GHPR.TraceEventCnt, 1)

				// pass trace block height at the start of new height
				h := models.TimeHeightRow{
					ChainId:   tp.Chain.ChainId,
					TraceYN:   "Y",
					Height:    to.BlockHeight,
					Timestamp: time.Now().UnixMicro(),
					Timeout:   0,
				}
				tp.DbProcCh <- h
				//fmt.Print("***Push TraceHeight***:", to.BlockHeight)

				// update global vars
				atomic.StoreInt64(&tp.Chain.LastTraceHeight, to.BlockHeight)
				tp.Chain.LastStoreUpdated = time.Now()

			} else if to.BlockHeight == global.GHPR.Height {

				atomic.StoreInt64(&global.GHPR.TsTracestoreEnd, to.SysTsRead)
				atomic.AddUint32(&global.GHPR.TraceEventCnt, 1)

			}
			//fmt.Printf("\nLine:%d %d ==> len %d \n", to.BlockHeight, to.SysTsRead, len(tp.IOChan))

			if tp.BypassHeight != 0 && tp.BypassHeight >= to.BlockHeight {
				if tp.BypassHeight == to.BlockHeight {
					tp.BypassHeight = 0
				}
				continue
			}

			//fmt.Println(to)

			tp.IOChan <- to

		} // end line loop

		select {
		//		case <-time.After(time.Duration(tp.Chain.TimeoutSec) * time.Second):
		//			tp.ErrorChan <- fmt.Errorf("read timeout")
		//			return
		case <-ctx.Done():
			tp.ErrorChan <- fmt.Errorf("canceled")
			return
		default:
		}

	}
}

var prevOpStr string = ""
var prevKey []byte = nil

func (tp *TraceProc) DataHandler(ctx context.Context) {

	for {

		//fmt.Println("@")

		select {
		case <-ctx.Done():
			//log.Warn().Msg("TraceProc DataHander cancel")
			log.Panic().Msg("TraceProc DataHander cancel")
			return
		case data := <-tp.IOChan:

			//debug Print
			debugPrint := true

			data.SysTsHandlerStart = time.Now().UnixNano()

			//TODO check the first prefix

			// if iterValue
			// if len(data.Key) == 0 {
			// 	continue
			// } else {
			// 	fmt.Println("data.Key", data.Key, "data.BlockHeight", data.BlockHeight)
			// }
			// prefix dispatch

			// If prev OP was IterKey & cur OP is IterValue, key of cur IterValue is prev IterKey
			if data.Operation == IterValueOpStr {
				data.Key = []byte{0x00}
			}
			if prevOpStr == IterKeyOpStr && data.Operation == IterValueOpStr {
				data.Key = prevKey
			}
			prevOpStr = data.Operation
			prevKey = data.Key

			// count
			pKey := &tp.CountPair
			switch data.Key[0] {
			case 0xa5: //liquidity/pairKey
				pKey = &tp.CountPair
			case 0xab:
				pKey = &tp.CountPool
			case 0xb0:
				pKey = &tp.CountDeposit
			case 0xb1:
				pKey = &tp.CountWithdraw
			case 0xb2:
				pKey = &tp.CountOrder
			default:
				continue
			}
			switch data.Operation {
			case WriteOpStr:
				atomic.AddUint64(&pKey.Write, 1)
			case DeleteOpStr:
				atomic.AddUint64(&pKey.Delete, 1)
			case ReadOpStr:
				atomic.AddUint64(&pKey.Read, 1)
			case IterKeyOpStr:
				atomic.AddUint64(&pKey.Iterate, 1)
			case IterValueOpStr:
				atomic.AddUint64(&pKey.Iterate, 1)
			}

			// filter for each prefix case. you need to change prefix value for a specific case
			if data.Key[0] != 0x00 {
				continue
			}

			switch data.Key[0] {
			case 0xa5: //liquidity/pairKey
				// if data.Operation != WriteOpStr {
				// 	fmt.Println("[0xa5]", data.Operation)
				// }
				//fmt.Println("0xa5 liquidity", data.Operation)
				// iterKey print
				fmt.Println("0xa5 liquidity", data.Operation, "data.BlockHeight", data.BlockHeight, "key", sdk.BigEndianToUint64(data.Key[1:]))

				var pair liquiditytypes.Pair
				//0xa5 + 8byte(uint64) => Pair( Id, baseDenom, QuoteDenom, EscrowAddr, lastswapREqid, lastCancelReqId, LastPrice, CurrBatId)
				// LastPrice can be nil if no swap before
				//pairId := sdk.BigEndianToUint64(data.Key[1:])
				//fmt.Print("PairKey:", pairId) // a5 + pairID
				if err := global.Config.Cdc.Unmarshal(data.Value, &pair); err != nil {
					fmt.Print("lqerr:", err)
					continue
				}
				//fmt.Println(pair)

				if err := tp.LiquidityProcess(data); err != nil {
					fmt.Print(err)
				} else {
					debugPrint = false
				}
			case 0xab: //liquidity/poolKey
				//fmt.Println("0xab liquidity", data.Operation)
				fmt.Println("0xa5 liquidity", data.Operation, "data.BlockHeight", data.BlockHeight, "key", sdk.BigEndianToUint64(data.Key[1:]))

				// if data.Operation != WriteOpStr {
				// 	continue
				// }

				var pool liquiditytypes.Pool
				// 0xab + 8byte(uint64) => Pool( Id, PairId, Reserve Addr, poolcoin denom, lastDepositReqid, lastDepositReqId, Disabled)
				// LastPrice can be nil if no swap before
				//poolId := sdk.BigEndianToUint64(data.Key[1:])
				//fmt.Print("PoolKey:", poolId) // a5 + pairID
				if err := global.Config.Cdc.Unmarshal(data.Value, &pool); err != nil {
					fmt.Print("lqerr:", err)
					continue
				}
				//fmt.Print(pool)
				if err := tp.LiquidityProcess(data); err != nil {
					fmt.Print(err)
				} else {
					debugPrint = false
				}
			case 0xb0: //deposit
				//fmt.Println("0xb0 liquidity", data.Operation)
				fmt.Println("0xb0", data.Operation, "data.BlockHeight", data.BlockHeight, "key", sdk.BigEndianToUint64(data.Key[1:]), "tx", data.TxHash)

				// if data.Operation != WriteOpStr {
				// 	continue
				// }
				if err := tp.LiquidityProcess(data); err != nil {
					fmt.Print(err)
				}
			case 0xb1: //withdraw
				//fmt.Println("0xb1 liquidity", data.Operation)
				fmt.Println("0xb1", data.Operation, "data.BlockHeight", data.BlockHeight, "key", sdk.BigEndianToUint64(data.Key[1:]))

				// if data.Operation != WriteOpStr {
				// 	continue
				// }
				if err := tp.LiquidityProcess(data); err != nil {
					fmt.Print(err)
				}
			case 0xb2: //swap
				//fmt.Println("0xb2 liquidity", data.Operation)
				fmt.Println(data.Key[0], data.Operation, "data.BlockHeight", data.BlockHeight, "pair", sdk.BigEndianToUint64(data.Key[1:9]), "req", sdk.BigEndianToUint64(data.Key[9:17]))

				// if data.Operation != WriteOpStr {
				// 	//fmt.Print("[DEL]0xb2")
				// 	continue
				// }
				if err := tp.LiquidityProcess(data); err != nil {
					fmt.Print("[W]", err)
				}
			case 0xb3: // liquidity order index key
				continue
			default:

				//found but not handling keys
				/*
					currentEpochDays
					farming/DelayedStakingGasFee
					farming/FarmingFeeCollector
					farming/MaxNumPrivatePlans
					farming/NextEpochDays
					farming/PrivatePlanCreationFee
					auth/SigVerifyCostED25519
					auth/SigVerifyCostSecp256k1
					auth/TxSigLimit
					auth/TxSizeCostPerByte
					bank/DefaultSendEnabled
					bank/SendEnabled
					baseapp/BlockParams
					baseapp/EvidenceParams
					baseapp/ValidatorParams
					budget/Budgets
					budget/EpochBlocks
					crisis/ConstantFee
					distribution/baseproposerreward
					distribution/bonusproposerreward
					distribution/communitytax
					distribution/withdrawaddrenabled
					gov/depositparams
					gov/tallyparams
					gov/votingparams
					ibc/AllowedClients
					ibc/MaxExpectedTimePerBlock
					liquidity/DepositExtraGas
					liquidity/FeeCollectorAddress
					liquidity/MaxOrderLifespan
					liquidity/MaxPriceLimitRatio
					liquidity/SwapFeeRate
					liquidstaking/LiquidBondDenom
					liquidstaking/MinLiquidStakingAmount
					mint/BlockTimeThreshold
					mint/InflationSchedules
					mint/MintDenom
					mint/params(custom)
					slashing/DowntimeJailDuration
					slashing/MinSignedPerWindow
					slashing/SignedBlocksWindow
					slashing/SlashFractionDoubleSign
					slashing/SlashFractionDowntime
					staking/BondDenom
					staking/HistoricalEntries
					staking/MaxEntries
					staking/MaxValidators
					staking/UnbondingTime
					transfer/ReceiveEnabled
					transfer/SendEnabled
					capability_index
					index


				*/

			}

			// force flush
			tp.BurstCounter++
			if tp.BurstCounter > 20000 {
				f := models.DBFlush{FlushType: 1}
				tp.DbProcCh <- f

				tp.BurstCounter = 0

				global.TgMsg <- "Force Flushing"
			}

			//for debug. disable heavy load. manullay
			if debugPrint {
				keyBase64 := make([]byte, base64.StdEncoding.EncodedLen(len(data.Key)))
				ValueBase64 := make([]byte, base64.StdEncoding.EncodedLen(len(data.Value)))
				base64.StdEncoding.Encode(keyBase64, data.Key)
				base64.StdEncoding.Encode(ValueBase64, data.Value)

				//fmt.Printf("// %s => %s\n", string(keyBase64), string(ValueBase64))
			}

		}

	}
}

func (tp *TraceProc) LiquidityProcess(data TraceOperation) error {

	switch data.Key[0] {
	case liquiditytypes.PairKeyPrefix[0]:

		id := sdk.BigEndianToUint64(data.Key[1:])
		pair := liquiditytypes.Pair{}
		//hAddr := hex.EncodeToString(denom)

		if err := global.Config.Cdc.Unmarshal(data.Value, &pair); err != nil {
			return err
		}

		decInt := sdk.ZeroDec()
		dec := pair.LastPrice
		if dec != nil {
			decInt = *pair.LastPrice
		}

		d := models.PairRow{
			PairId:       id,
			BaseDenom:    pair.BaseCoinDenom,
			QuoteDenom:   pair.QuoteCoinDenom,
			EscrowAddr:   pair.EscrowAddress,
			LastPrice:    decInt.String(),
			CurrentBatch: pair.CurrentBatchId,
			Whitelisted:  false,
			Deleted:      false,
		}

		//if len(data.TxHash) > 0 {
		tp.DbProcCh <- d
		//}

		//update lastprice
		if pi, ok := global.PairInfos[id]; ok {
			pi.LastPrice = decInt
		}

		if data.Operation != IterKeyOpStr {
			fmt.Println(data.Value)
			//fmt.Printf("height: %d liqudity pair %s id:%d (%s/%s) escrow:%s price:%s currentBatch:%d, txhash:%s \n\n", data.BlockHeight, data.Operation, id, pair.QuoteCoinDenom, pair.BaseCoinDenom, pair.EscrowAddress, decInt.String(), pair.CurrentBatchId, data.TxHash)
			// Pair defines a coin pair.
			// type Pair struct {
			// 	Id             uint64                                  `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
			// 	BaseCoinDenom  string                                  `protobuf:"bytes,2,opt,name=base_coin_denom,json=baseCoinDenom,proto3" json:"base_coin_denom,omitempty"`
			// 	QuoteCoinDenom string                                  `protobuf:"bytes,3,opt,name=quote_coin_denom,json=quoteCoinDenom,proto3" json:"quote_coin_denom,omitempty"`
			// 	EscrowAddress  string                                  `protobuf:"bytes,4,opt,name=escrow_address,json=escrowAddress,proto3" json:"escrow_address,omitempty"`
			// 	LastOrderId    uint64                                  `protobuf:"varint,5,opt,name=last_order_id,json=lastOrderId,proto3" json:"last_order_id,omitempty"`
			// 	LastPrice      *github_com_cosmos_cosmos_sdk_types.Dec `protobuf:"bytes,6,opt,name=last_price,json=lastPrice,proto3,customtype=github.com/cosmos/cosmos-sdk/types.Dec" json:"last_price,omitempty"`
			// 	CurrentBatchId uint64                                  `protobuf:"varint,7,opt,name=current_batch_id,json=currentBatchId,proto3" json:"current_batch_id,omitempty"`
			// }
		}

	case liquiditytypes.PoolKeyPrefix[0]:

		id := sdk.BigEndianToUint64(data.Key[1:])

		if data.Operation == DeleteOpStr {
			// pool is not deleted. only disabled
			//} else if data.Operation == WriteOpStr {
		} else {
			var pool liquiditytypes.Pool
			// 0xab + 8byte(uint64) => Pool( Id, PairId, Reserve Addr, poolcoin denom, lastDepositReqid, lastDepositReqId, Disabled)
			if err := global.Config.Cdc.Unmarshal(data.Value, &pool); err != nil {
				fmt.Print("lqerr:", err)
				return err
			}

			if id != pool.Id {
				fmt.Printf("Pool ID error :%d %d\n", id, pool.Id)
				return errors.New("Pool ID error")
			}

			stAddr := farmingtypes.StakingReserveAcc(pool.PoolCoinDenom)
			hexStakingAddr := hex.EncodeToString(stAddr)

			//fmt.Println(pool)
			if data.Operation != IterKeyOpStr {
				fmt.Println(pool)
				fmt.Println("")
				// type Pool struct {
				// 	Type                  PoolType                                `protobuf:"varint,1,opt,name=type,proto3,enum=crescent.liquidity.v1beta1.PoolType" json:"type,omitempty"`
				// 	Id                    uint64                                  `protobuf:"varint,2,opt,name=id,proto3" json:"id,omitempty"`
				// 	PairId                uint64                                  `protobuf:"varint,3,opt,name=pair_id,json=pairId,proto3" json:"pair_id,omitempty"`
				// 	Creator               string                                  `protobuf:"bytes,4,opt,name=creator,proto3" json:"creator,omitempty"`
				// 	ReserveAddress        string                                  `protobuf:"bytes,5,opt,name=reserve_address,json=reserveAddress,proto3" json:"reserve_address,omitempty"`
				// 	PoolCoinDenom         string                                  `protobuf:"bytes,6,opt,name=pool_coin_denom,json=poolCoinDenom,proto3" json:"pool_coin_denom,omitempty"`
				// 	MinPrice              *github_com_cosmos_cosmos_sdk_types.Dec `protobuf:"bytes,7,opt,name=min_price,json=minPrice,proto3,customtype=github.com/cosmos/cosmos-sdk/types.Dec" json:"min_price,omitempty"`
				// 	MaxPrice              *github_com_cosmos_cosmos_sdk_types.Dec `protobuf:"bytes,8,opt,name=max_price,json=maxPrice,proto3,customtype=github.com/cosmos/cosmos-sdk/types.Dec" json:"max_price,omitempty"`
				// 	LastDepositRequestId  uint64                                  `protobuf:"varint,9,opt,name=last_deposit_request_id,json=lastDepositRequestId,proto3" json:"last_deposit_request_id,omitempty"`
				// 	LastWithdrawRequestId uint64                                  `protobuf:"varint,10,opt,name=last_withdraw_request_id,json=lastWithdrawRequestId,proto3" json:"last_withdraw_request_id,omitempty"`
				// 	Disabled              bool                                    `protobuf:"varint,11,opt,name=disabled,proto3" json:"disabled,omitempty"`
				// }
			}

			minp := ""
			maxp := ""
			if pool.MinPrice != nil {
				minp = pool.MinPrice.String()
			}
			if pool.MaxPrice != nil {
				maxp = pool.MaxPrice.String()
			}

			d := models.PoolRow{
				PoolId:                id,
				PairId:                pool.PairId,
				PoolDenom:             pool.PoolCoinDenom,
				ReserveAddr:           pool.ReserveAddress,
				Disabled:              pool.Disabled,
				CreatedHeight:         data.BlockHeight,
				Status:                0,
				StakingReserveHexAddr: hexStakingAddr,
				PoolType:              int(pool.Type),
				MinPrice:              minp,
				MaxPrice:              maxp,
				Creator:               pool.Creator,
			}

			//if len(data.TxHash) > 0 {
			tp.DbProcCh <- d
			//}

			if pool.Disabled {

				if pool, OK := global.Denom2PoolInfo[pool.PoolCoinDenom]; OK {
					pool.Disabled = true
				}

				fmt.Printf("pool Disabled id:%d pair=%d denom=%s resevAddr:%s height:%d txhash:%s\n\n", id, pool.PairId, pool.PoolCoinDenom, pool.ReserveAddress, data.BlockHeight, data.TxHash)
			} else {
				//fmt.Printf("new pool id:%d pair=%d denom=%s resevAddr:%s height:%d txhash:%s\n\n", id, pool.PairId, pool.PoolCoinDenom, pool.ReserveAddress, data.BlockHeight, data.TxHash)
			}

			// check if pair of pool already whitelisted, add denom as whitelisted immediately
			// for pair plan
			pair, OK := global.PairInfos[pool.PairId]
			if OK {

				sDenom := pool.PoolCoinDenom

				// denom can be added with total supply key. (remove)
				//new denom or be overrided
				d := models.DenomRow{
					Denom:       sDenom,
					OrgChainId:  global.Config.Chains[global.Config.MainIdx].ChainId,
					Ticker:      strings.ToUpper(sDenom),
					BaseDenom:   sDenom,
					IbcPath:     "",
					Exponent:    12,
					Whitelisted: pair.Whitelisted, // whitelisting pool denom if pair whitelisted
				}
				tp.DbProcCh <- d

				//if the new pool added, all pools will be loaded at th end of flushing

			}

		}
	case liquiditytypes.OrderKeyPrefix[0]:

		// key prefix + pairId(Uint64) + requestId(Uint64)
		pairId := sdk.BigEndianToUint64(data.Key[1:9])
		reqId := sdk.BigEndianToUint64(data.Key[9:17])

		var req liquiditytypes.Order
		if err := global.Config.Cdc.Unmarshal(data.Value, &req); err != nil {
			fmt.Print("lqerr:", err)
			return err
		}

		//fmt.Printf("Order req height:%d pairId:%d reqId:%d data:%v\n\n", data.BlockHeight, pairId, reqId, req)
		//fmt.Println("len of Key", len(data.Key))
		//fmt.Println("len of Value", len(data.Value))
		if len(data.Value) == 0 {
			return nil
		}
		// Order defines an order.
		// type Order struct {
		// 	Type OrderType `protobuf:"varint,1,opt,name=type,proto3,enum=crescent.liquidity.v1beta1.OrderType" json:"type,omitempty"`
		// 	Id uint64 `protobuf:"varint,2,opt,name=id,proto3" json:"id,omitempty"`
		// 	PairId uint64 `protobuf:"varint,3,opt,name=pair_id,json=pairId,proto3" json:"pair_id,omitempty"`
		// 	MsgHeight int64 `protobuf:"varint,4,opt,name=msg_height,json=msgHeight,proto3" json:"msg_height,omitempty"`
		// 	Orderer string `protobuf:"bytes,5,opt,name=orderer,proto3" json:"orderer,omitempty"`
		// 	Direction OrderDirection `protobuf:"varint,6,opt,name=direction,proto3,enum=crescent.liquidity.v1beta1.OrderDirection" json:"direction,omitempty"`
		// 	OfferCoin types.Coin     `protobuf:"bytes,7,opt,name=offer_coin,json=offerCoin,proto3" json:"offer_coin"`
		// 	RemainingOfferCoin types.Coin `protobuf:"bytes,8,opt,name=remaining_offer_coin,json=remainingOfferCoin,proto3" json:"remaining_offer_coin"`
		// 	ReceivedCoin types.Coin `protobuf:"bytes,9,opt,name=received_coin,json=receivedCoin,proto3" json:"received_coin"`
		// 	Price      github_com_cosmos_cosmos_sdk_types.Dec `protobuf:"bytes,10,opt,name=price,proto3,customtype=github.com/cosmos/cosmos-sdk/types.Dec" json:"price"`
		// 	Amount     github_com_cosmos_cosmos_sdk_types.Int `protobuf:"bytes,11,opt,name=amount,proto3,customtype=github.com/cosmos/cosmos-sdk/types.Int" json:"amount"`
		// 	OpenAmount github_com_cosmos_cosmos_sdk_types.Int `protobuf:"bytes,12,opt,name=open_amount,json=openAmount,proto3,customtype=github.com/cosmos/cosmos-sdk/types.Int" json:"open_amount"`
		// 	BatchId  uint64      `protobuf:"varint,13,opt,name=batch_id,json=batchId,proto3" json:"batch_id,omitempty"`
		// 	ExpireAt time.Time   `protobuf:"bytes,14,opt,name=expire_at,json=expireAt,proto3,stdtime" json:"expire_at"`
		// 	Status   OrderStatus `protobuf:"varint,15,opt,name=status,proto3,enum=crescent.liquidity.v1beta1.OrderStatus" json:"status,omitempty"`
		// }

		if pairId != req.PairId || reqId != req.Id {
			//fmt.Printf("order key-value mismatch :%d %d => %d %d\n", pairId, req.PairId, reqId, req.Id)
			//panic("key-value mismatch")
			//return errors.New("Pool ID error")
		}

		m := types.MsgSwap{
			Order:     req,
			Height:    data.BlockHeight,
			Timestamp: time.Now().UnixMicro(),
			TxHash:    data.TxHash,
		}

		tp.DbProcCh <- m

		//fmt.Printf("\nSwapReq height:%d txhash:%s\n%v\n", data.BlockHeight, data.TxHash, req)

	case liquiditytypes.DepositRequestKeyPrefix[0]:
		// if data.Operation != WriteOpStr {
		// 	return nil
		// }

		// key prefix + pairId(Uint64) + requestId(Uint64)
		poolId := sdk.BigEndianToUint64(data.Key[1:9])
		reqId := sdk.BigEndianToUint64(data.Key[9:17])

		var req liquiditytypes.DepositRequest
		if err := global.Config.Cdc.Unmarshal(data.Value, &req); err != nil {
			fmt.Print("lqerr:", err)
			return err
		}

		//fmt.Printf("\nDeposit req height:%d txhash:%s\n%v\n", data.BlockHeight, data.TxHash, req)
		fmt.Printf("Withdraw req height:%d poolId:%d reqId:%d data:%v\n\n", data.BlockHeight, poolId, reqId, req)
		if len(data.Value) == 0 {
			return nil
		}
		//fmt.Println(req)
		//fmt.Println("")
		// type DepositRequest struct {
		// 	// id specifies the id for the request
		// 	Id uint64 `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
		// 	// pool_id specifies the pool id
		// 	PoolId uint64 `protobuf:"varint,2,opt,name=pool_id,json=poolId,proto3" json:"pool_id,omitempty"`
		// 	// msg_height specifies the block height when the request is stored for the batch execution
		// 	MsgHeight int64 `protobuf:"varint,3,opt,name=msg_height,json=msgHeight,proto3" json:"msg_height,omitempty"`
		// 	// depositor specifies the bech32-encoded address that makes a deposit to the pool
		// 	Depositor string `protobuf:"bytes,4,opt,name=depositor,proto3" json:"depositor,omitempty"`
		// 	// deposit_coins specifies the amount of coins to deposit.
		// 	DepositCoins github_com_cosmos_cosmos_sdk_types.Coins `protobuf:"bytes,5,rep,name=deposit_coins,json=depositCoins,proto3,castrepeated=github.com/cosmos/cosmos-sdk/types.Coins" json:"deposit_coins"`
		// 	// accepted_coins specifies the amount of coins that are accepted.
		// 	AcceptedCoins  github_com_cosmos_cosmos_sdk_types.Coins `protobuf:"bytes,6,rep,name=accepted_coins,json=acceptedCoins,proto3,castrepeated=github.com/cosmos/cosmos-sdk/types.Coins" json:"accepted_coins"`
		// 	MintedPoolCoin types.Coin                               `protobuf:"bytes,7,opt,name=minted_pool_coin,json=mintedPoolCoin,proto3" json:"minted_pool_coin"`
		// 	Status         RequestStatus                            `protobuf:"varint,8,opt,name=status,proto3,enum=crescent.liquidity.v1beta1.RequestStatus" json:"status,omitempty"`
		// }

		if poolId != req.PoolId || reqId != req.Id || len(req.DepositCoins) == 0 || len(req.DepositCoins) > 2 {
			fmt.Printf("deopsit key-value mismatch :%d %d => %d %d, len=%d\n", poolId, req.PoolId, reqId, req.Id, len(req.DepositCoins))
			//panic("key-value mismatch")
			//return errors.New("Pool ID error")
		}

		// single asset
		if len(req.DepositCoins) == 1 {
			d := models.DepositRow{
				PoolId:         req.PoolId,
				ReqId:          req.Id,
				MsgHeight:      req.MsgHeight,
				Owner:          req.Depositor,
				DepositAAmount: req.DepositCoins[0].Amount.String(),
				DenomA:         req.DepositCoins[0].Denom,
				Status:         int(req.Status),
				TxHash:         data.TxHash,
				Timestamp:      time.Now().UnixMicro(),
			}

			if req.Status == 2 { // executed
				d.AcceptedAAmount = req.AcceptedCoins[0].Amount.String()
				d.MintedPoolAmount = req.MintedPoolCoin.Amount.String()
			}

			if len(data.TxHash) > 0 {
				tp.DbProcCh <- d
			}

		} else {
			d := models.DepositRow{
				PoolId:         req.PoolId,
				ReqId:          req.Id,
				MsgHeight:      req.MsgHeight,
				Owner:          req.Depositor,
				DepositAAmount: req.DepositCoins[0].Amount.String(),
				DepositBAmount: req.DepositCoins[1].Amount.String(),
				DenomA:         req.DepositCoins[0].Denom,
				DenomB:         req.DepositCoins[1].Denom,
				Status:         int(req.Status),
				TxHash:         data.TxHash,
				Timestamp:      time.Now().UnixMicro(),
			}

			if req.Status == 2 { // executed
				d.AcceptedAAmount = req.AcceptedCoins[0].Amount.String()
				d.AcceptedBAmount = req.AcceptedCoins[1].Amount.String()
				d.MintedPoolAmount = req.MintedPoolCoin.Amount.String()
			}

			if len(data.TxHash) > 0 {
				tp.DbProcCh <- d
			}

		}

	case liquiditytypes.WithdrawRequestKeyPrefix[0]:

		// key prefix + pairId(Uint64) + requestId(Uint64)
		poolId := sdk.BigEndianToUint64(data.Key[1:9])
		reqId := sdk.BigEndianToUint64(data.Key[9:17])

		var req liquiditytypes.WithdrawRequest
		if err := global.Config.Cdc.Unmarshal(data.Value, &req); err != nil {
			global.TgMsg <- fmt.Sprintf("Withdraw Unmarshal err. Pool:%d reqId:%d err:%s", poolId, reqId, err.Error())
			return err
		}

		fmt.Printf("Withdraw req height:%d poolId:%d reqId:%d data:%v\n\n", data.BlockHeight, poolId, reqId, req)
		if len(data.Value) == 0 {
			return nil
		}

		//fmt.Printf("\nWithdraw req height:%d txhash:%s\n%v\n", data.BlockHeight, data.TxHash, req)
		if poolId != req.PoolId || reqId != req.Id {
			fmt.Printf("withdraw key-value mismatch :%d %d => %d %d\n", poolId, req.PoolId, reqId, req.Id)
			//panic("key-value mismatch")
			//return errors.New("Pool ID error")
		}

		d := models.WithdrawRow{
			PoolId:          req.PoolId,
			ReqId:           req.Id,
			Height:          req.MsgHeight,
			Owner:           req.Withdrawer,
			OfferPoolAmount: req.PoolCoin.Amount.String(),
			Status:          int(req.Status),
			TxHash:          data.TxHash,
			Timestamp:       time.Now().UnixMicro(),
		}

		switch int(req.Status) {
		case 2: // success
			if len(req.WithdrawnCoins) == 2 {
				//TODO. core use base 1st.
				d.WithdrawAAmount = req.WithdrawnCoins[0].Amount.String()
				d.WithdrawBAmount = req.WithdrawnCoins[1].Amount.String()
				d.DenomA = req.WithdrawnCoins[0].Denom
				d.DenomB = req.WithdrawnCoins[1].Denom

			} else if len(req.WithdrawnCoins) == 1 {
				// ranged pool with out-of-limit
				d.WithdrawAAmount = req.WithdrawnCoins[0].Amount.String()
				d.DenomA = req.WithdrawnCoins[0].Denom
				//fmt.Print(req.WithdrawnCoins.Len())
				//fmt.Print(req.WithdrawnCoins)
				//fmt.Print("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
			} else {
				global.TgMsg <- "withdraw err. zero coin"
				global.TgMsg <- fmt.Sprintf("withdraw pool:%d reqId:%d\n", poolId, reqId)
				//fmt.Print(req.WithdrawnCoins)
				//fmt.Print("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")

			}

		}

		if len(data.TxHash) > 0 {
			tp.DbProcCh <- d

		}

	}

	return nil
}
