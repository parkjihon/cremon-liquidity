package traceproc

import (
	"encoding/json"
	"fmt"
)

const (
	metadataBlockHeight = "blockHeight"
	metadataTxHash      = "txHash"
)

//decode type.
//to be deprecated
const (
	DTNotSet = iota
	DTBankSupply
	DTBankBalance
	DTBankDenomMetadata
	DTIbcAppTransferPort
	DTIbcAppTransferDenomTrace
)

type TraceOperation struct {
	Operation         string `json:"operation"`
	Key               []byte `json:"key"`
	Value             []byte `json:"value"`
	BlockHeight       int64  `json:"block_height"`
	TxHash            string `json:"tx_hash"`
	SysTsRead         int64
	SysTsHandlerStart int64
	SysTsHandlerEnd   int64
	DecodeType        int
	Recovery          int // if 1. recover data from last store. not used anymore
}

func (t TraceOperation) String() string {
	return fmt.Sprintf(`[%s] "%v" -> "%v"`, t.Operation, string(t.Key), string(t.Value))
}

type traceOperationInter struct {
	Operation string                 `json:"operation"`
	Key       []byte                 `json:"key"`
	Value     []byte                 `json:"value"`
	Metadata  map[string]interface{} `json:"metadata"`
}

func (t *TraceOperation) UnmarshalJSON(bytes []byte) error {
	toi := traceOperationInter{}

	if err := json.Unmarshal(bytes, &toi); err != nil {
		return err
	}

	if toi.Metadata == nil {
		t.BlockHeight = 0
	} else {
		if data, ok := toi.Metadata[metadataBlockHeight]; ok {
			t.BlockHeight = int64(data.(float64))
		}

		if data, ok := toi.Metadata[metadataTxHash]; ok {
			t.TxHash = data.(string)
		}
	}

	t.Operation = toi.Operation
	t.Key = toi.Key
	t.Value = toi.Value

	return nil
}
