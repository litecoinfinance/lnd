package lnwallet

import (
	"github.com/litecoinfinance/btcutil"
	"github.com/litecoinfinance/btcwallet/wallet/txrules"
	"github.com/litecoinfinance/lnd/input"
)

// DefaultDustLimit is used to calculate the dust HTLC amount which will be
// send to other node during funding process.
func DefaultDustLimit() btcutil.Amount {
	return txrules.GetDustThreshold(input.P2WSHSize, txrules.DefaultRelayFeePerKb)
}
