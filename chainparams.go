package lnd

import (
	"github.com/litecoinfinance/btcd/chaincfg"
	bitcoinCfg "github.com/litecoinfinance/btcd/chaincfg"
	"github.com/litecoinfinance/btcd/chaincfg/chainhash"
	bitcoinWire "github.com/litecoinfinance/btcd/wire"
	"github.com/litecoinfinance/lnd/keychain"
	litecoinfinanceCfg "github.com/litecoinfinance/ltfnd/chaincfg"
	litecoinfinanceWire "github.com/litecoinfinance/ltfnd/wire"
)

// activeNetParams is a pointer to the parameters specific to the currently
// active bitcoin network.
var activeNetParams = bitcoinTestNetParams

// bitcoinNetParams couples the p2p parameters of a network with the
// corresponding RPC port of a daemon running on the particular network.
type bitcoinNetParams struct {
	*bitcoinCfg.Params
	rpcPort  string
	CoinType uint32
}

// litecoinfinanceNetParams couples the p2p parameters of a network with the
// corresponding RPC port of a daemon running on the particular network.
type litecoinfinanceNetParams struct {
	*litecoinfinanceCfg.Params
	rpcPort  string
	CoinType uint32
}

// bitcoinTestNetParams contains parameters specific to the 3rd version of the
// test network.
var bitcoinTestNetParams = bitcoinNetParams{
	Params:   &bitcoinCfg.TestNet3Params,
	rpcPort:  "18334",
	CoinType: keychain.CoinTypeTestnet,
}

// bitcoinMainNetParams contains parameters specific to the current Bitcoin
// mainnet.
var bitcoinMainNetParams = bitcoinNetParams{
	Params:   &bitcoinCfg.MainNetParams,
	rpcPort:  "8334",
	CoinType: keychain.CoinTypeBitcoin,
}

// bitcoinSimNetParams contains parameters specific to the simulation test
// network.
var bitcoinSimNetParams = bitcoinNetParams{
	Params:   &bitcoinCfg.SimNetParams,
	rpcPort:  "18556",
	CoinType: keychain.CoinTypeTestnet,
}

// litecoinfinanceSimNetParams contains parameters specific to the simulation test
// network.
var litecoinfinanceSimNetParams = litecoinfinanceNetParams{
	Params:   &litecoinfinanceCfg.SimNetParams,
	rpcPort:  "18556",
	CoinType: keychain.CoinTypeTestnet,
}

// litecoinfinanceTestNetParams contains parameters specific to the 4th version of the
// test network.
var litecoinfinanceTestNetParams = litecoinfinanceNetParams{
	Params:   &litecoinfinanceCfg.TestNet4Params,
	rpcPort:  "19334",
	CoinType: keychain.CoinTypeTestnet,
}

// litecoinfinanceMainNetParams contains the parameters specific to the current
// Litecoinfinance mainnet.
var litecoinfinanceMainNetParams = litecoinfinanceNetParams{
	Params:   &litecoinfinanceCfg.MainNetParams,
	rpcPort:  "39329",
	CoinType: keychain.CoinTypeLitecoinfinance,
}

// litecoinfinanceRegTestNetParams contains parameters specific to a local litecoinfinance
// regtest network.
var litecoinfinanceRegTestNetParams = litecoinfinanceNetParams{
	Params:   &litecoinfinanceCfg.RegressionNetParams,
	rpcPort:  "18334",
	CoinType: keychain.CoinTypeTestnet,
}

// bitcoinRegTestNetParams contains parameters specific to a local bitcoin
// regtest network.
var bitcoinRegTestNetParams = bitcoinNetParams{
	Params:   &bitcoinCfg.RegressionNetParams,
	rpcPort:  "18334",
	CoinType: keychain.CoinTypeTestnet,
}

// applyLitecoinfinanceParams applies the relevant chain configuration parameters that
// differ for litecoinfinance to the chain parameters typed for btcsuite derivation.
// This function is used in place of using something like interface{} to
// abstract over _which_ chain (or fork) the parameters are for.
func applyLitecoinfinanceParams(params *bitcoinNetParams, litecoinfinanceParams *litecoinfinanceNetParams) {
	params.Name = litecoinfinanceParams.Name
	params.Net = bitcoinWire.BitcoinNet(litecoinfinanceParams.Net)
	params.DefaultPort = litecoinfinanceParams.DefaultPort
	params.CoinbaseMaturity = litecoinfinanceParams.CoinbaseMaturity

	copy(params.GenesisHash[:], litecoinfinanceParams.GenesisHash[:])

	// Address encoding magics
	params.PubKeyHashAddrID = litecoinfinanceParams.PubKeyHashAddrID
	params.ScriptHashAddrID = litecoinfinanceParams.ScriptHashAddrID
	params.PrivateKeyID = litecoinfinanceParams.PrivateKeyID
	params.WitnessPubKeyHashAddrID = litecoinfinanceParams.WitnessPubKeyHashAddrID
	params.WitnessScriptHashAddrID = litecoinfinanceParams.WitnessScriptHashAddrID
	params.Bech32HRPSegwit = litecoinfinanceParams.Bech32HRPSegwit

	copy(params.HDPrivateKeyID[:], litecoinfinanceParams.HDPrivateKeyID[:])
	copy(params.HDPublicKeyID[:], litecoinfinanceParams.HDPublicKeyID[:])

	params.HDCoinType = litecoinfinanceParams.HDCoinType

	checkPoints := make([]chaincfg.Checkpoint, len(litecoinfinanceParams.Checkpoints))
	for i := 0; i < len(litecoinfinanceParams.Checkpoints); i++ {
		var chainHash chainhash.Hash
		copy(chainHash[:], litecoinfinanceParams.Checkpoints[i].Hash[:])

		checkPoints[i] = chaincfg.Checkpoint{
			Height: litecoinfinanceParams.Checkpoints[i].Height,
			Hash:   &chainHash,
		}
	}
	params.Checkpoints = checkPoints

	params.rpcPort = litecoinfinanceParams.rpcPort
	params.CoinType = litecoinfinanceParams.CoinType
}

// isTestnet tests if the given params correspond to a testnet
// parameter configuration.
func isTestnet(params *bitcoinNetParams) bool {
	switch params.Params.Net {
	case bitcoinWire.TestNet3, bitcoinWire.BitcoinNet(litecoinfinanceWire.TestNet4):
		return true
	default:
		return false
	}
}
