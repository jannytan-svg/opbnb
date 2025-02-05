package main

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/flags"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum-optimism/optimism/op-service/txmgr/metrics"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/urfave/cli/v2"
)

type ContractCreator[T any] func(common.Address, *batching.MultiCaller) (T, error)

// NewContractWithTxMgr creates a new contract and a transaction manager.
func NewContractWithTxMgr[T any](ctx *cli.Context, flagName string, creator ContractCreator[T]) (T, txmgr.TxManager, error) {
	var contract T
	caller, txMgr, err := newClientsFromCLI(ctx)
	if err != nil {
		return contract, nil, err
	}

	created, err := newContractFromCLI(ctx, flagName, caller, creator)
	if err != nil {
		return contract, nil, err
	}

	return created, txMgr, nil
}

// newContractFromCLI creates a new contract from the CLI context.
func newContractFromCLI[T any](ctx *cli.Context, flagName string, caller *batching.MultiCaller, creator ContractCreator[T]) (T, error) {
	var contract T
	gameAddr, err := opservice.ParseAddress(ctx.String(flagName))
	if err != nil {
		return contract, err
	}

	created, err := creator(gameAddr, caller)
	if err != nil {
		return contract, fmt.Errorf("failed to create dispute game bindings: %w", err)
	}

	return created, nil
}

// newClientsFromCLI creates a new caller and transaction manager from the CLI context.
func newClientsFromCLI(ctx *cli.Context) (*batching.MultiCaller, txmgr.TxManager, error) {
	logger, err := setupLogging(ctx)
	if err != nil {
		return nil, nil, err
	}

	rpcUrl := ctx.String(flags.L1EthRpcFlag.Name)
	if rpcUrl == "" {
		return nil, nil, fmt.Errorf("missing %v", flags.L1EthRpcFlag.Name)
	}

	l1Client, err := dial.DialEthClientWithTimeout(ctx.Context, dial.DefaultDialTimeout, logger, rpcUrl)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to dial L1: %w", err)
	}
	defer l1Client.Close()

	caller := batching.NewMultiCaller(l1Client.(*ethclient.Client).Client(), batching.DefaultBatchSize)
	txMgrConfig := txmgr.ReadCLIConfig(ctx)
	txMgr, err := txmgr.NewSimpleTxManager("challenger", logger, &metrics.NoopTxMetrics{}, txMgrConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create the transaction manager: %w", err)
	}

	return caller, txMgr, nil
}
