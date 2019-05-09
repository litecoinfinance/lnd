package wtpolicy

import (
	"errors"
	"fmt"

	"github.com/litecoinfinance/btcd/wire"
	"github.com/litecoinfinance/btcutil"
	"github.com/litecoinfinance/lnd/lnwallet"
	"github.com/litecoinfinance/lnd/watchtower/blob"
)

const (
	// RewardScale is the denominator applied when computing the
	// proportional component for a tower's reward output. The current scale
	// is in millionths.
	RewardScale = 1000000

	// DefaultMaxUpdates specifies the number of encrypted blobs a client
	// can send to the tower in a single session.
	DefaultMaxUpdates = 1024

	// DefaultRewardRate specifies the fraction of the channel that the
	// tower takes if it successfully sweeps a breach. The value is
	// expressed in millionths of the channel capacity.
	DefaultRewardRate = 10000

	// DefaultSweepFeeRate specifies the fee rate used to construct justice
	// transactions. The value is expressed in satoshis per kilo-weight.
	DefaultSweepFeeRate = 3000
)

var (
	// ErrFeeExceedsInputs signals that the total input value of breaching
	// commitment txn is insufficient to cover the fees required to sweep
	// it.
	ErrFeeExceedsInputs = errors.New("sweep fee exceeds input value")

	// ErrRewardExceedsInputs signals that the reward given to the tower (in
	// addition to the transaction fees) is more than the input amount.
	ErrRewardExceedsInputs = errors.New("reward amount exceeds input value")

	// ErrCreatesDust signals that the session's policy would create a dust
	// output for the victim.
	ErrCreatesDust = errors.New("justice transaction creates dust at fee rate")
)

// DefaultPolicy returns a Policy containing the default parameters that can be
// used by clients or servers.
func DefaultPolicy() Policy {
	return Policy{
		BlobType:   blob.TypeDefault,
		MaxUpdates: DefaultMaxUpdates,
		RewardRate: DefaultRewardRate,
		SweepFeeRate: lnwallet.SatPerKWeight(
			DefaultSweepFeeRate,
		),
	}
}

// Policy defines the negotiated parameters for a session between a client and
// server. The parameters specify the format of encrypted blobs sent to the
// tower, the reward schedule for the tower, and the number of encrypted blobs a
// client can send in one session.
type Policy struct {
	// BlobType specifies the blob format that must be used by all updates sent
	// under the session key used to negotiate this session.
	BlobType blob.Type

	// MaxUpdates is the maximum number of updates the watchtower will honor
	// for this session.
	MaxUpdates uint16

	// RewardBase is the fixed amount allocated to the tower when the
	// policy's blob type specifies a reward for the tower. This is taken
	// before adding the proportional reward.
	RewardBase uint32

	// RewardRate is the fraction of the total balance of the revoked
	// commitment that the watchtower is entitled to. This value is
	// expressed in millionths of the total balance.
	RewardRate uint32

	// SweepFeeRate expresses the intended fee rate to be used when
	// constructing the justice transaction. All sweep transactions created
	// for this session must use this value during construction, and the
	// signatures must implicitly commit to the resulting output values.
	SweepFeeRate lnwallet.SatPerKWeight
}

// String returns a human-readable description of the current policy.
func (p Policy) String() string {
	return fmt.Sprintf("(blob-type=%b max-updates=%d reward-rate=%d "+
		"sweep-fee-rate=%d)", p.BlobType, p.MaxUpdates, p.RewardRate,
		p.SweepFeeRate)
}

// ComputeAltruistOutput computes the lone output value of a justice transaction
// that pays no reward to the tower. The value is computed using the weight of
// of the justice transaction and subtracting an amount that satisfies the
// policy's fee rate.
func (p *Policy) ComputeAltruistOutput(totalAmt btcutil.Amount,
	txWeight int64) (btcutil.Amount, error) {

	txFee := p.SweepFeeRate.FeeForWeight(txWeight)
	if txFee > totalAmt {
		return 0, ErrFeeExceedsInputs
	}

	sweepAmt := totalAmt - txFee

	// TODO(conner): replace w/ configurable dust limit
	dustLimit := lnwallet.DefaultDustLimit()

	// Check that the created outputs won't be dusty.
	if sweepAmt <= dustLimit {
		return 0, ErrCreatesDust
	}

	return sweepAmt, nil
}

// ComputeRewardOutputs splits the total funds in a breaching commitment
// transaction between the victim and the tower, according to the sweep fee rate
// and reward rate. The reward to he tower is subtracted first, before
// splitting the remaining balance amongst the victim and fees.
func (p *Policy) ComputeRewardOutputs(totalAmt btcutil.Amount,
	txWeight int64) (btcutil.Amount, btcutil.Amount, error) {

	txFee := p.SweepFeeRate.FeeForWeight(txWeight)
	if txFee > totalAmt {
		return 0, 0, ErrFeeExceedsInputs
	}

	// Apply the reward rate to the remaining total, specified in millionths
	// of the available balance.
	rewardAmt := ComputeRewardAmount(totalAmt, p.RewardBase, p.RewardRate)
	if rewardAmt+txFee > totalAmt {
		return 0, 0, ErrRewardExceedsInputs
	}

	// The sweep amount for the victim constitutes the remainder of the
	// input value.
	sweepAmt := totalAmt - rewardAmt - txFee

	// TODO(conner): replace w/ configurable dust limit
	dustLimit := lnwallet.DefaultDustLimit()

	// Check that the created outputs won't be dusty.
	if sweepAmt <= dustLimit {
		return 0, 0, ErrCreatesDust
	}

	return sweepAmt, rewardAmt, nil
}

// ComputeRewardAmount computes the amount rewarded to the tower using the
// proportional rate expressed in millionths, e.g. one million is equivalent to
// one hundred percent of the total amount. The amount is rounded up to the
// nearest whole satoshi.
func ComputeRewardAmount(total btcutil.Amount, base, rate uint32) btcutil.Amount {
	rewardBase := btcutil.Amount(base)
	rewardRate := btcutil.Amount(rate)

	// If the base reward exceeds the total, there is no more funds left
	// from which to derive the proportional fee. We simply return the base,
	// the caller should detect that this exceeds the total amount input.
	if rewardBase > total {
		return rewardBase
	}

	// Otherwise, subtract the base from the total and compute the
	// proportional reward from the remaining total.
	afterBase := total - rewardBase
	proportional := (afterBase*rewardRate + RewardScale - 1) / RewardScale

	return rewardBase + proportional
}

// ComputeJusticeTxOuts constructs the justice transaction outputs for the given
// policy. If the policy specifies a reward for the tower, there will be two
// outputs paying to the victim and the tower. Otherwise there will be a single
// output sweeping funds back to the victim. The totalAmt should be the sum of
// any inputs used in the transaction. The passed txWeight should include the
// weight of the outputs for the justice transaction, which is dependent on
// whether the justice transaction has a reward. The sweepPkScript should be the
// pkScript of the victim to which funds will be recovered. The rewardPkScript
// is the pkScript of the tower where its reward will be deposited, and will be
// ignored if the blob type does not specify a reward.
func (p *Policy) ComputeJusticeTxOuts(totalAmt btcutil.Amount, txWeight int64,
	sweepPkScript, rewardPkScript []byte) ([]*wire.TxOut, error) {

	var outputs []*wire.TxOut

	// If the policy specifies a reward for the tower, compute a split of
	// the funds based on the policy's parameters. Otherwise, we will use an
	// the altruist output computation and sweep as much of the funds back
	// to the victim as possible.
	if p.BlobType.Has(blob.FlagReward) {
		// Using the total input amount and the transaction's weight,
		// compute the sweep and reward amounts. This corresponds to the
		// amount returned to the victim and the amount paid to the
		// tower, respectively. To do so, the required transaction fee
		// is subtracted from the total, and the remaining amount is
		// divided according to the prenegotiated reward rate from the
		// client's session info.
		sweepAmt, rewardAmt, err := p.ComputeRewardOutputs(
			totalAmt, txWeight,
		)
		if err != nil {
			return nil, err
		}

		// Add the sweep and reward outputs to the list of txouts.
		outputs = append(outputs, &wire.TxOut{
			PkScript: sweepPkScript,
			Value:    int64(sweepAmt),
		})
		outputs = append(outputs, &wire.TxOut{
			PkScript: rewardPkScript,
			Value:    int64(rewardAmt),
		})
	} else {
		// Using the total input amount and the transaction's weight,
		// compute the sweep amount, which corresponds to the amount
		// returned to the victim. To do so, the required transaction
		// fee is subtracted from the total input amount.
		sweepAmt, err := p.ComputeAltruistOutput(
			totalAmt, txWeight,
		)
		if err != nil {
			return nil, err
		}

		// Add the sweep output to the list of txouts.
		outputs = append(outputs, &wire.TxOut{
			PkScript: sweepPkScript,
			Value:    int64(sweepAmt),
		})
	}

	return outputs, nil
}
