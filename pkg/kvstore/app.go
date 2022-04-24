package kvstore

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/zeroFruit/cosmos-ab/pkg/code"

	"github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/version"
	dbm "github.com/tendermint/tm-db"
)

type Application struct {
	types.BaseApplication
	mu     sync.Mutex
	logger log.Logger
	state  State
	// RetainBlocks is blocks to retain after commit (via ResponseCommit.RetainHeight)
	RetainBlocks int64
}

func NewApplication() *Application {
	logger, err := log.NewDefaultLogger(log.LogFormatJSON, log.LogLevelInfo, false)
	if err != nil {
		panic(err)
	}
	return &Application{
		logger: logger,
		state:  loadState(dbm.NewMemDB()),
	}
}

func (app *Application) Info(types.RequestInfo) types.ResponseInfo {
	app.mu.Lock()
	defer app.mu.Unlock()
	return types.ResponseInfo{
		Data:             fmt.Sprintf("{\"size\":%v}", app.state.Size),
		Version:          version.ABCIVersion,
		AppVersion:       ProtocolVersion,
		LastBlockHeight:  app.state.Height,
		LastBlockAppHash: app.state.AppHash,
	}

}

// InitChain initialize blockchain w validators/other info from TendermintCore
func (app *Application) InitChain(req types.RequestInitChain) types.ResponseInitChain {
	app.logger.Info("init chain requested", "chainId", req.ChainId, "initHeight", req.InitialHeight)
	return app.BaseApplication.InitChain(req)
}

// CheckTx validates a tx for the mempool
func (app *Application) CheckTx(req types.RequestCheckTx) types.ResponseCheckTx {
	app.logger.Info("checkTx requested", "tx", string(req.Tx), "type", req.Type.String())
	return types.ResponseCheckTx{Code: code.CodeTypeOK, GasWanted: 1}
}
func (app *Application) BeginBlock(req types.RequestBeginBlock) types.ResponseBeginBlock {
	app.logger.Info("beginBlock requested", "hash", hex.EncodeToString(req.Hash), "round", req.LastCommitInfo.Round)
	return app.BaseApplication.BeginBlock(req)
}
func (app *Application) DeliverTx(req types.RequestDeliverTx) types.ResponseDeliverTx {

	var key, value string

	parts := bytes.Split(req.Tx, []byte("="))
	if len(parts) == 2 {
		key, value = string(parts[0]), string(parts[1])
	} else {
		key, value = string(req.Tx), string(req.Tx)
	}

	err := app.state.db.Set(prefixKey([]byte(key)), []byte(value))
	if err != nil {
		panic(err)
	}
	app.state.Size++

	events := []types.Event{
		{
			Type: "app",
			Attributes: []types.EventAttribute{
				{Key: "key", Value: key, Index: true},
			},
		},
	}
	app.logger.Info("deliverTx requested", "events", events[0].String())
	return types.ResponseDeliverTx{Code: code.CodeTypeOK, Events: events}
}
func (app *Application) EndBlock(req types.RequestEndBlock) types.ResponseEndBlock {
	return app.BaseApplication.EndBlock(req)
}
func (app *Application) Commit() types.ResponseCommit {
	appHash := make([]byte, 8)
	binary.PutVarint(appHash, app.state.Size)
	app.state.AppHash = appHash
	app.state.Height++
	saveState(app.state)

	resp := types.ResponseCommit{Data: appHash}
	if app.RetainBlocks > 0 && app.state.Height >= app.RetainBlocks {
		resp.RetainHeight = app.state.Height - app.RetainBlocks + 1
	}
	app.logger.Info("commit requested", "appHash", hex.EncodeToString(appHash), "stateHeight", app.state.Height)
	return resp
}

func (app *Application) Query(reqQuery types.RequestQuery) (resQuery types.ResponseQuery) {
	app.logger.Info("query requested", "data", string(reqQuery.Data), "path", reqQuery.Path, "height", reqQuery.Height)

	value, err := app.state.db.Get(prefixKey(reqQuery.Data))
	if err != nil {
		panic(err)
	}
	if value == nil {
		resQuery.Log = "does not exist"
	} else {
		resQuery.Log = "exists"
	}
	resQuery.Index = -1 // TODO make Proof return index
	resQuery.Key = reqQuery.Data
	resQuery.Value = value
	resQuery.Height = app.state.Height

	return
}
