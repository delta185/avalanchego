// Copyright (C) 2019-2024, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package fee

import (
	"errors"
	"fmt"
	"time"

	"github.com/ava-labs/avalanchego/codec"
	"github.com/ava-labs/avalanchego/utils/constants"
	"github.com/ava-labs/avalanchego/utils/wrappers"
	"github.com/ava-labs/avalanchego/vms/components/avax"
	"github.com/ava-labs/avalanchego/vms/components/fees"
	"github.com/ava-labs/avalanchego/vms/components/verify"
	"github.com/ava-labs/avalanchego/vms/platformvm/txs"
	"github.com/ava-labs/avalanchego/vms/platformvm/upgrade"
	"github.com/ava-labs/avalanchego/vms/secp256k1fx"
)

var (
	_ txs.Visitor = (*calculator)(nil)

	errFailedFeeCalculation       = errors.New("failed fee calculation")
	errFailedComplexityCumulation = errors.New("failed cumulating complexity")
)

type Calculator struct {
	c *calculator
}

func (c *Calculator) GetFee() uint64 {
	return c.c.fee
}

func (c *Calculator) ResetFee(newFee uint64) {
	c.c.fee = newFee
}

func (c *Calculator) GetTipPercentage() fees.TipPercentage {
	return c.c.tipPercentage
}

func (c *Calculator) ResetTipPercentage(tip fees.TipPercentage) {
	c.c.tipPercentage = tip
}

func (c *Calculator) ComputeFee(tx txs.UnsignedTx) (uint64, error) {
	err := tx.Visit(c.c)
	return c.c.fee, err
}

func (c *Calculator) AddFeesFor(complexity fees.Dimensions) (uint64, error) {
	return c.c.addFeesFor(complexity)
}

func (c *Calculator) RemoveFeesFor(unitsToRm fees.Dimensions) (uint64, error) {
	return c.c.removeFeesFor(unitsToRm)
}

// CalculateTipPercentage calculates and sets the tip percentage, given the fees actually paid
// and the fees required to accept the target transaction.
// [CalculateTipPercentage] requires that c.Visit has been called for the target transaction.
func (c *Calculator) CalculateTipPercentage(feesPaid uint64) error {
	if feesPaid < c.c.fee {
		return fmt.Errorf("fees paid are less the required fees: fees paid %v, fees required %v",
			feesPaid,
			c.c.fee,
		)
	}

	if c.c.fee == 0 {
		return nil
	}

	tip := feesPaid - c.c.fee
	c.c.tipPercentage = fees.TipPercentage(tip * fees.TipDenonimator / c.c.fee)
	return c.c.tipPercentage.Validate()
}

type calculator struct {
	// setup
	isEActive bool

	// Pre E-upgrade inputs
	upgrades  upgrade.Config
	staticCfg StaticConfig
	time      time.Time

	// Post E-upgrade inputs
	feeManager         *fees.Manager
	blockMaxComplexity fees.Dimensions
	credentials        []verify.Verifiable

	// tipPercentage can either be an input (e.g. when building a transaction)
	// or an output (once a transaction is verified)
	tipPercentage fees.TipPercentage

	// outputs of visitor execution
	fee uint64
}

func NewStaticCalculator(cfg StaticConfig, ut upgrade.Config, chainTime time.Time) *Calculator {
	return &Calculator{
		c: &calculator{
			upgrades:  ut,
			staticCfg: cfg,
			time:      chainTime,
		},
	}
}

// NewDynamicCalculator must be used post E upgrade activation
func NewDynamicCalculator(
	cfg StaticConfig,
	feeManager *fees.Manager,
	blockMaxComplexity fees.Dimensions,
	creds []verify.Verifiable,
) *Calculator {
	return &Calculator{
		c: &calculator{
			isEActive:          true,
			staticCfg:          cfg,
			feeManager:         feeManager,
			blockMaxComplexity: blockMaxComplexity,
			credentials:        creds,
		},
	}
}

func (c *calculator) AddValidatorTx(*txs.AddValidatorTx) error {
	// AddValidatorTx is banned following Durango activation, so we
	// only return the pre EUpgrade fee here
	c.fee = c.staticCfg.AddPrimaryNetworkValidatorFee
	return nil
}

func (c *calculator) AddSubnetValidatorTx(tx *txs.AddSubnetValidatorTx) error {
	if !c.isEActive {
		c.fee = c.staticCfg.AddSubnetValidatorFee
		return nil
	}

	complexity, err := c.meterTx(tx, tx.Outs, tx.Ins)
	if err != nil {
		return err
	}

	_, err = c.addFeesFor(complexity)
	return err
}

func (c *calculator) AddDelegatorTx(*txs.AddDelegatorTx) error {
	// AddValidatorTx is banned following Durango activation, so we
	// only return the pre EUpgrade fee here
	c.fee = c.staticCfg.AddPrimaryNetworkDelegatorFee
	return nil
}

func (c *calculator) CreateChainTx(tx *txs.CreateChainTx) error {
	if !c.isEActive {
		if c.upgrades.IsApricotPhase3Activated(c.time) {
			c.fee = c.staticCfg.CreateBlockchainTxFee
		} else {
			c.fee = c.staticCfg.CreateAssetTxFee
		}
		return nil
	}

	complexity, err := c.meterTx(tx, tx.Outs, tx.Ins)
	if err != nil {
		return err
	}

	_, err = c.addFeesFor(complexity)
	return err
}

func (c *calculator) CreateSubnetTx(tx *txs.CreateSubnetTx) error {
	if !c.isEActive {
		if c.upgrades.IsApricotPhase3Activated(c.time) {
			c.fee = c.staticCfg.CreateSubnetTxFee
		} else {
			c.fee = c.staticCfg.CreateAssetTxFee
		}
		return nil
	}

	complexity, err := c.meterTx(tx, tx.Outs, tx.Ins)
	if err != nil {
		return err
	}

	_, err = c.addFeesFor(complexity)
	return err
}

func (c *calculator) AdvanceTimeTx(*txs.AdvanceTimeTx) error {
	c.fee = 0
	return nil // no fees
}

func (c *calculator) RewardValidatorTx(*txs.RewardValidatorTx) error {
	c.fee = 0
	return nil // no fees
}

func (c *calculator) RemoveSubnetValidatorTx(tx *txs.RemoveSubnetValidatorTx) error {
	if !c.isEActive {
		c.fee = c.staticCfg.TxFee
		return nil
	}

	complexity, err := c.meterTx(tx, tx.Outs, tx.Ins)
	if err != nil {
		return err
	}

	_, err = c.addFeesFor(complexity)
	return err
}

func (c *calculator) TransformSubnetTx(tx *txs.TransformSubnetTx) error {
	if !c.isEActive {
		c.fee = c.staticCfg.TransformSubnetTxFee
		return nil
	}

	complexity, err := c.meterTx(tx, tx.Outs, tx.Ins)
	if err != nil {
		return err
	}

	_, err = c.addFeesFor(complexity)
	return err
}

func (c *calculator) TransferSubnetOwnershipTx(tx *txs.TransferSubnetOwnershipTx) error {
	if !c.isEActive {
		c.fee = c.staticCfg.TxFee
		return nil
	}

	complexity, err := c.meterTx(tx, tx.Outs, tx.Ins)
	if err != nil {
		return err
	}

	_, err = c.addFeesFor(complexity)
	return err
}

func (c *calculator) AddPermissionlessValidatorTx(tx *txs.AddPermissionlessValidatorTx) error {
	if !c.isEActive {
		if tx.Subnet != constants.PrimaryNetworkID {
			c.fee = c.staticCfg.AddSubnetValidatorFee
		} else {
			c.fee = c.staticCfg.CreateAssetTxFee
		}
		return nil
	}

	complexity, err := c.meterTx(tx, tx.Outs, tx.Ins)
	if err != nil {
		return err
	}

	_, err = c.addFeesFor(complexity)
	return err
}

func (c *calculator) AddPermissionlessDelegatorTx(tx *txs.AddPermissionlessDelegatorTx) error {
	if !c.isEActive {
		if tx.Subnet != constants.PrimaryNetworkID {
			c.fee = c.staticCfg.AddSubnetDelegatorFee
		} else {
			c.fee = c.staticCfg.AddPrimaryNetworkDelegatorFee
		}
		return nil
	}

	outs := make([]*avax.TransferableOutput, len(tx.Outs)+len(tx.StakeOuts))
	copy(outs, tx.Outs)
	copy(outs[len(tx.Outs):], tx.StakeOuts)

	complexity, err := c.meterTx(tx, outs, tx.Ins)
	if err != nil {
		return err
	}

	_, err = c.addFeesFor(complexity)
	return err
}

func (c *calculator) BaseTx(tx *txs.BaseTx) error {
	if !c.isEActive {
		c.fee = c.staticCfg.TxFee
		return nil
	}

	complexity, err := c.meterTx(tx, tx.Outs, tx.Ins)
	if err != nil {
		return err
	}

	_, err = c.addFeesFor(complexity)
	return err
}

func (c *calculator) ImportTx(tx *txs.ImportTx) error {
	if !c.isEActive {
		c.fee = c.staticCfg.TxFee
		return nil
	}

	ins := make([]*avax.TransferableInput, len(tx.Ins)+len(tx.ImportedInputs))
	copy(ins, tx.Ins)
	copy(ins[len(tx.Ins):], tx.ImportedInputs)

	complexity, err := c.meterTx(tx, tx.Outs, ins)
	if err != nil {
		return err
	}

	_, err = c.addFeesFor(complexity)
	return err
}

func (c *calculator) ExportTx(tx *txs.ExportTx) error {
	if !c.isEActive {
		c.fee = c.staticCfg.TxFee
		return nil
	}

	outs := make([]*avax.TransferableOutput, len(tx.Outs)+len(tx.ExportedOutputs))
	copy(outs, tx.Outs)
	copy(outs[len(tx.Outs):], tx.ExportedOutputs)

	complexity, err := c.meterTx(tx, outs, tx.Ins)
	if err != nil {
		return err
	}

	_, err = c.addFeesFor(complexity)
	return err
}

func (c *calculator) meterTx(
	uTx txs.UnsignedTx,
	allOuts []*avax.TransferableOutput,
	allIns []*avax.TransferableInput,
) (fees.Dimensions, error) {
	var complexity fees.Dimensions

	uTxSize, err := txs.Codec.Size(txs.CodecVersion, uTx)
	if err != nil {
		return complexity, fmt.Errorf("couldn't calculate UnsignedTx marshal length: %w", err)
	}
	complexity[fees.Bandwidth] = uint64(uTxSize)

	// meter credentials, one by one. Then account for the extra bytes needed to
	// serialize a slice of credentials (codec version bytes + slice size bytes)
	for i, cred := range c.credentials {
		c, ok := cred.(*secp256k1fx.Credential)
		if !ok {
			return complexity, fmt.Errorf("don't know how to calculate complexity of %T", cred)
		}
		credDimensions, err := fees.MeterCredential(txs.Codec, txs.CodecVersion, len(c.Sigs))
		if err != nil {
			return complexity, fmt.Errorf("failed adding credential %d: %w", i, err)
		}
		complexity, err = fees.Add(complexity, credDimensions)
		if err != nil {
			return complexity, fmt.Errorf("failed adding credentials: %w", err)
		}
	}
	complexity[fees.Bandwidth] += wrappers.IntLen // length of the credentials slice
	complexity[fees.Bandwidth] += codec.VersionSize

	for _, in := range allIns {
		inputDimensions, err := fees.MeterInput(txs.Codec, txs.CodecVersion, in)
		if err != nil {
			return complexity, fmt.Errorf("failed retrieving size of inputs: %w", err)
		}
		inputDimensions[fees.Bandwidth] = 0 // inputs bandwidth is already accounted for above, so we zero it
		complexity, err = fees.Add(complexity, inputDimensions)
		if err != nil {
			return complexity, fmt.Errorf("failed adding inputs: %w", err)
		}
	}

	for _, out := range allOuts {
		outputDimensions, err := fees.MeterOutput(txs.Codec, txs.CodecVersion, out)
		if err != nil {
			return complexity, fmt.Errorf("failed retrieving size of outputs: %w", err)
		}
		outputDimensions[fees.Bandwidth] = 0 // outputs bandwidth is already accounted for above, so we zero it
		complexity, err = fees.Add(complexity, outputDimensions)
		if err != nil {
			return complexity, fmt.Errorf("failed adding outputs: %w", err)
		}
	}

	return complexity, nil
}

func (c *calculator) addFeesFor(complexity fees.Dimensions) (uint64, error) {
	if c.feeManager == nil || complexity == fees.Empty {
		return 0, nil
	}

	boundBreached, dimension := c.feeManager.CumulateComplexity(complexity, c.blockMaxComplexity)
	if boundBreached {
		return 0, fmt.Errorf("%w: breached dimension %d", errFailedComplexityCumulation, dimension)
	}

	fee, err := c.feeManager.CalculateFee(complexity, c.tipPercentage)
	if err != nil {
		return 0, fmt.Errorf("%w: %w", errFailedFeeCalculation, err)
	}

	c.fee += fee
	return fee, nil
}

func (c *calculator) removeFeesFor(unitsToRm fees.Dimensions) (uint64, error) {
	if c.feeManager == nil || unitsToRm == fees.Empty {
		return 0, nil
	}

	if err := c.feeManager.RemoveComplexity(unitsToRm); err != nil {
		return 0, fmt.Errorf("failed removing units: %w", err)
	}

	fee, err := c.feeManager.CalculateFee(unitsToRm, c.tipPercentage)
	if err != nil {
		return 0, fmt.Errorf("%w: %w", errFailedFeeCalculation, err)
	}

	c.fee -= fee
	return fee, nil
}