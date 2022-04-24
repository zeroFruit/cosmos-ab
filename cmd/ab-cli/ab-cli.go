package main

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/zeroFruit/cosmos-ab/pkg/kvstore"

	abciclient "github.com/tendermint/tendermint/abci/client"
	"github.com/tendermint/tendermint/abci/types"
	"github.com/zeroFruit/cosmos-ab/pkg/code"

	"github.com/tendermint/tendermint/abci/server"

	"github.com/spf13/cobra"
	"github.com/tendermint/tendermint/libs/log"
)

var (
	// global

	// client is a global variable so it can be reused by the console
	client       abciclient.Client
	ctx          = context.Background()
	transportTyp = "grpc" // either socket or grpc

	// query
	flagPath   string
	flagHeight int
)

var (
	flagAddress string
)

func RootCommand(logger log.Logger) *cobra.Command {
	return &cobra.Command{
		Use: "ab",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {

			switch cmd.Use {
			case "kv":
				return nil
			}

			if logger == nil {
				logger = log.MustNewDefaultLogger(log.LogFormatPlain, log.LogLevelInfo, false)
			}

			if client == nil {
				var err error
				client, err = abciclient.NewClient(flagAddress, transportTyp, false)
				if err != nil {
					return err
				}
				client.SetLogger(logger.With("module", "ab-client"))
				if err := client.Start(); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

func Execute() error {
	logger, err := log.NewDefaultLogger(log.LogFormatJSON, log.LogLevelInfo, false)
	if err != nil {
		return err
	}

	cmd := RootCommand(logger)
	addGlobalFlags(cmd)
	addCommands(cmd)
	return cmd.Execute()
}

func addGlobalFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVarP(&flagAddress,
		"address",
		"",
		"tcp://0.0.0.0:26658",
		"address of application socket")
}

func addQueryFlags(queryCmd *cobra.Command) {
	queryCmd.PersistentFlags().StringVarP(&flagPath, "path", "", "/store", "path to prefix query with")
	queryCmd.PersistentFlags().IntVarP(&flagHeight, "height", "", 0, "height to query the blockchain at")
}

func addCommands(cmd *cobra.Command) {
	cmd.AddCommand(getInfoCmd())
	cmd.AddCommand(getDeliverTxCmd())
	cmd.AddCommand(getKVStoreCmd())
	cmd.AddCommand(getCommitCmd())
	queryCmd := getQueryCmd()
	addQueryFlags(queryCmd)
	cmd.AddCommand(queryCmd)
}

func getInfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "info",
		Short: "get some info about the application",
		Long:  "get some info about the application",
		Args:  cobra.ExactArgs(0),
		RunE:  makeInfoCmd,
	}

}

func getDeliverTxCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tx",
		Short: "deliver a new transaction to the application",
		Long:  "deliver a new transaction to the application",
		Args:  cobra.ExactArgs(1),
		RunE:  makeDeliverTxCmd,
	}
}

func getKVStoreCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kv",
		Short: "run the application server",
		Long:  "run the application server",
		Args:  cobra.ExactArgs(0),
		RunE:  makeKVStoreCmd(),
	}
	return cmd
}

func getQueryCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "query",
		Short: "query the application state",
		Long:  "query the application state",
		Args:  cobra.ExactArgs(1),
		RunE:  makeQueryCmd,
	}
}

func getCommitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "commit",
		Short: "commit the application state and return the Merkle root hash",
		Long:  "commit the application state and return the Merkle root hash",
		Args:  cobra.ExactArgs(0),
		RunE:  makeCommitCmd,
	}
}

// Get some info from the application
func makeInfoCmd(cmd *cobra.Command, args []string) error {
	var version string
	if len(args) == 1 {
		version = args[0]
	}
	res, err := client.InfoSync(ctx, types.RequestInfo{Version: version})
	if err != nil {
		return err
	}
	printResponse(cmd, args, response{
		Data: []byte(res.Data),
	})
	return nil
}

// Append a new tx to application
func makeDeliverTxCmd(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		printResponse(cmd, args, response{
			Code: code.CodeTypeOK,
			Log:  "want the tx",
		})
		return nil
	}
	txBytes, err := stringOrHexToBytes(args[0])
	if err != nil {
		return err
	}
	res, err := client.DeliverTxSync(ctx, types.RequestDeliverTx{Tx: txBytes})
	if err != nil {
		return err
	}
	printResponse(cmd, args, response{
		Code: res.Code,
		Data: res.Data,
		Info: res.Info,
		Log:  res.Log,
	})
	return nil
}

func makeKVStoreCmd() func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		// Create the application - in memory or persisted to disk
		app := kvstore.NewApplication()

		// Start the listener
		srv, err := server.NewServer(flagAddress, transportTyp, app)
		if err != nil {
			return err
		}

		ctx, cancel := signal.NotifyContext(cmd.Context(), syscall.SIGTERM)
		defer cancel()

		if err := srv.Start(); err != nil {
			return err
		}

		// Run forever.
		<-ctx.Done()
		return nil
	}
}

func makeQueryCmd(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		printResponse(cmd, args, response{
			Code: code.CodeTypeArgsBad,
			Info: "want the query",
			Log:  "",
		})
		return nil
	}
	queryBytes, err := stringOrHexToBytes(args[0])
	if err != nil {
		return err
	}

	resQuery, err := client.QuerySync(ctx, types.RequestQuery{
		Data:   queryBytes,
		Path:   flagPath,
		Height: int64(flagHeight),
		Prove:  false,
	})
	if err != nil {
		return err
	}
	printResponse(cmd, args, response{
		Code: resQuery.Code,
		Info: resQuery.Info,
		Log:  resQuery.Log,
		Query: &queryResponse{
			Key:      resQuery.Key,
			Value:    resQuery.Value,
			Height:   resQuery.Height,
			ProofOps: resQuery.ProofOps,
		},
	})
	return nil
}

// Get application Merkle root hash
func makeCommitCmd(cmd *cobra.Command, args []string) error {
	res, err := client.CommitSync(ctx)
	if err != nil {
		return err
	}
	printResponse(cmd, args, response{
		Data: res.Data,
	})
	return nil
}
