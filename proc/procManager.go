package proc

import (
	"context"
	"cremon-liquidity/config"
	"cremon-liquidity/global"
	"cremon-liquidity/proc/dbproc"
	"cremon-liquidity/proc/monproc"
	"cremon-liquidity/proc/rpcproc"
	"cremon-liquidity/proc/traceproc"
	"cremon-liquidity/types"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

var GlobalInst *ProcManager

type ProcManager struct {
	StartTime time.Time
	RP        []*rpcproc.RpcProc
	TP        *traceproc.TraceProc // only for trace crescent blockchain
	DP        *dbproc.DbProc
	MP        *monproc.MonProc
	TGP       *monproc.TgBotProc
	//Datahandler
	DbDataCh chan interface{}
	LogCh    chan types.LogMsg

	LogicCtx    context.Context
	LogicCancel context.CancelFunc // cancel all logic proc. ex) rpc, trace, datahandler, db

	MgrCtx    context.Context
	MgrCancel context.CancelFunc // cancel managing proc. ex) ha, monitor
}

func NewProcManager(ctx context.Context, cfg *config.Config) (*ProcManager, error) {
	// if singleton instance
	if GlobalInst != nil {
		return GlobalInst, nil
	}
	log.Info().Msg("======== Process Manager  =========")

	mhdCfg, _ := json.Marshal(global.Config)
	fmt.Println("pm 에서 바라보는 global.Config", string(mhdCfg))

	// init global variables
	global.OrderNestMap = make(map[uint64](map[uint64]*types.OrderInfo))
	global.PairOrderbook = make(map[uint64]*types.OrderbookInfo)
	global.PairInfos = make(map[uint64]*types.PairInfo)
	global.Hex2PoolInfo = make(map[string]*types.PoolInfo)
	global.Denom2PoolInfo = make(map[string]*types.PoolInfo)
	global.AckCheckMap = make(map[string][]string)
	global.AckCheckMapMutex = make(map[string]*sync.Mutex)
	global.UpdateCh = make(chan global.UpdateReq, 100)

	chainNum := len(cfg.Chains)
	if chainNum == 0 {
		log.Panic().Msg("No Chain configs. ProcManager halt")
	}

	//make contexts for child proc
	logicCtx, logicCancel := context.WithCancel(ctx)
	mgrCtx, mgrCancel := context.WithCancel(ctx)

	//TODO: tuning channel size. have to set large enough for genesis with airdrop
	//[!] set rpc buffer enough in APP node config not to disconnect from TM rpc.
	pm := &ProcManager{
		StartTime:   time.Now(),
		DbDataCh:    make(chan interface{}, 100100),
		LogCh:       make(chan types.LogMsg, 1000),
		LogicCtx:    logicCtx,
		LogicCancel: logicCancel,
		MgrCtx:      mgrCtx,
		MgrCancel:   mgrCancel,
	}

	log.Info().Msg("======== Process Manager : initing DB, Monitoring, ... process managers ")

	// trace proc
	tp, err := traceproc.NewTraceProc(cfg.Chains[cfg.MainIdx], pm.DbDataCh)
	if err != nil {
		return nil, err
	}
	pm.TP = tp

	GlobalInst = pm
	return pm, nil
}

func (p *ProcManager) Start(traceType string) {

	log.Info().Msg("===== Process Manager Start() invoked =====")
	defer func() {
		log.Warn().Str("time", time.Now().Format(time.RFC3339)).Msg("================= procMgr finish ================")
	}()

	// 하 시발...

	//0-0 set monitor first
	if p.MP != nil {
		log.Info().Msg("모니터 프로세스 없음")
		//go p.MP.Start(p.MgrCtx)
	}

	// 3. check tracestore
	if traceType == "file" {
		fmt.Println("file type started", global.Config.TraceFile)
		go p.TP.TraceFile(p.LogicCtx)
	} else {
		go p.TP.TraceStore(p.LogicCtx)
	}
	go p.TP.DataHandler(p.LogicCtx)

	// infinity wait
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	tick := time.NewTicker(5 * time.Second)

	for {
		select {
		case <-signalChan:
			return
		case <-tick.C:
			//log.Info().Int("#DbChan", len(p.DbDataCh)).Msg("Channel Status")
			p.DbDataCh = nil
		}
	}
}
