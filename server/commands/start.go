package commands

import (
	"fmt"
	"os"
	"path"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/cosmos/cosmos-sdk"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/tendermint/tmlibs/cli"
	cmn "github.com/tendermint/tmlibs/common"

	"github.com/CyberMiles/travis/app"
	"github.com/CyberMiles/travis/genesis"
	ethapp "github.com/CyberMiles/travis/vm/app"
	"github.com/ethereum/go-ethereum/eth"
)

// GetTickStartCmd - initialize a command as the start command with tick
func GetTickStartCmd(ticker sdk.Ticker) *cobra.Command {
	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Start this full node",
		RunE:  tickStartCmd(ticker),
	}
	return startCmd
}

// nolint TODO: move to config file
const EyesCacheSize = 10000

//returns the start command which uses the tick
func tickStartCmd(ticker sdk.Ticker) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		rootDir := viper.GetString(cli.HomeFlag)

		cmdName := cmd.Root().Name()
		appName := fmt.Sprintf("%s v%v", cmdName, version.Version)
		storeApp, err := app.NewStoreApp(
			appName,
			path.Join(rootDir, "data", "merkleeyes.db"),
			EyesCacheSize,
			logger.With("module", "app"))
		if err != nil {
			return err
		}

		return start(rootDir, storeApp, ticker)
	}
}

func start(rootDir string, storeApp *app.StoreApp, ticker sdk.Ticker) error {
	srvs, err := startServices(rootDir, storeApp, ticker)
	if err != nil {
		return errors.Errorf("Error in start services: %v\n", err)
	}

	// wait forever
	cmn.TrapSignal(func() {
		srvs.tmNode.Stop()
	})

	return nil
}

func createBaseCoinApp(rootDir string, storeApp *app.StoreApp, ethApp *ethapp.EthermintApplication, ticker sdk.Ticker, ethereum *eth.Ethereum) (*app.BaseApp, error) {
	travisApp, err := app.NewBaseApp(storeApp, ethApp, ticker, ethereum)
	if err != nil {
		return nil, err
	}
	// if chain_id has not been set yet, load the genesis.
	// else, assume it's been loaded
	if travisApp.GetChainID() == "" {
		// If genesis file exists, set key-value options
		genesisFile := path.Join(rootDir, "genesis.json")
		if _, err := os.Stat(genesisFile); err == nil {
			err = genesis.Load(travisApp, genesisFile)
			if err != nil {
				return nil, errors.Errorf("Error in LoadGenesis: %v\n", err)
			}
		} else {
			fmt.Printf("No genesis file at %s, skipping...\n", genesisFile)
		}
	}

	chainID := travisApp.GetChainID()
	logger.Info("Starting Travis", "chain_id", chainID)

	return travisApp, nil
}
