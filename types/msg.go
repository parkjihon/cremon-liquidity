package types

import (
	liquiditytypes "github.com/crescent-network/crescent/v4/x/liquidity/types"
)

// struct as a message for rpc, trace channel communication
type MsgSwap struct {
	Order     liquiditytypes.Order
	TxHash    string
	Height    int64
	Timestamp int64
}

var (
	_ liquiditytypes.Order
)

// LogMsg type
const (
	TraceLog = iota
	InfoLog
	WarningLog
	ErrorLog
	CriticalLog
)

// struct to monitor msg
type LogMsg struct {
	LogType   int
	Timestamp int64
	Body      string
	Details   string
}
