// Copyright (C) 2019-2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package executor

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ava-labs/avalanchego/chains/atomic"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/constants"
	"github.com/ava-labs/avalanchego/utils/set"
	"github.com/ava-labs/avalanchego/utils/timer/mockable"
	"github.com/ava-labs/avalanchego/vms/components/avax"
	"github.com/ava-labs/avalanchego/vms/components/verify"
	"github.com/ava-labs/avalanchego/vms/platformvm/reward"
	"github.com/ava-labs/avalanchego/vms/platformvm/state"
	"github.com/ava-labs/avalanchego/vms/platformvm/txs"
)

var (
	_ txs.Visitor = (*StandardTxExecutor)(nil)

	errEmptyNodeID              = errors.New("validator nodeID cannot be empty")
	errMaxStakeDurationTooLarge = errors.New("max stake duration must be less than or equal to the global max stake duration")
)

type StandardTxExecutor struct {
	// inputs, to be filled before visitor methods are called
	*Backend
	State state.Diff // state is expected to be modified
	Tx    *txs.Tx

	// outputs of visitor execution
	OnAccept       func() // may be nil
	Inputs         set.Set[ids.ID]
	AtomicRequests map[ids.ID]*atomic.Requests // may be nil
}

func (*StandardTxExecutor) AdvanceTimeTx(*txs.AdvanceTimeTx) error {
	return ErrWrongTxType
}

func (*StandardTxExecutor) RewardValidatorTx(*txs.RewardValidatorTx) error {
	return ErrWrongTxType
}

func (e *StandardTxExecutor) CreateChainTx(tx *txs.CreateChainTx) error {
	if err := e.Tx.SyntacticVerify(e.Ctx); err != nil {
		return err
	}

	baseTxCreds, err := verifyPoASubnetAuthorization(e.Backend, e.State, e.Tx, tx.SubnetID, tx.SubnetAuth)
	if err != nil {
		return err
	}

	// Verify the flowcheck
	timestamp := e.State.GetTimestamp()
	createBlockchainTxFee := e.Config.GetCreateBlockchainTxFee(timestamp)
	if err := e.FlowChecker.VerifySpend(
		e.Tx.Version(),
		tx,
		e.State,
		tx.Ins,
		tx.Outs,
		baseTxCreds,
		map[ids.ID]uint64{
			e.Ctx.AVAXAssetID: createBlockchainTxFee,
		},
	); err != nil {
		return err
	}

	txID := e.Tx.ID()

	// Consume the UTXOS
	avax.Consume(e.State, tx.Ins)
	// Produce the UTXOS
	avax.Produce(e.State, txID, tx.Outs)
	// Add the new chain to the database
	e.State.AddChain(e.Tx)

	// If this proposal is committed and this node is a member of the subnet
	// that validates the blockchain, create the blockchain
	e.OnAccept = func() {
		e.Config.CreateChain(txID, tx)
	}
	return nil
}

func (e *StandardTxExecutor) CreateSubnetTx(tx *txs.CreateSubnetTx) error {
	// Make sure this transaction is well formed.
	if err := e.Tx.SyntacticVerify(e.Ctx); err != nil {
		return err
	}

	// Verify the flowcheck
	timestamp := e.State.GetTimestamp()
	createSubnetTxFee := e.Config.GetCreateSubnetTxFee(timestamp)
	if err := e.FlowChecker.VerifySpend(
		e.Tx.Version(),
		tx,
		e.State,
		tx.Ins,
		tx.Outs,
		e.Tx.Creds,
		map[ids.ID]uint64{
			e.Ctx.AVAXAssetID: createSubnetTxFee,
		},
	); err != nil {
		return err
	}

	txID := e.Tx.ID()

	// Consume the UTXOS
	avax.Consume(e.State, tx.Ins)
	// Produce the UTXOS
	avax.Produce(e.State, txID, tx.Outs)
	// Add the new subnet to the database
	e.State.AddSubnet(e.Tx)
	return nil
}

func (e *StandardTxExecutor) ImportTx(tx *txs.ImportTx) error {
	if err := e.Tx.SyntacticVerify(e.Ctx); err != nil {
		return err
	}

	e.Inputs = set.NewSet[ids.ID](len(tx.ImportedInputs))
	utxoIDs := make([][]byte, len(tx.ImportedInputs))
	for i, in := range tx.ImportedInputs {
		utxoID := in.UTXOID.InputID()

		e.Inputs.Add(utxoID)
		utxoIDs[i] = utxoID[:]
	}

	if e.Bootstrapped.Get() {
		if err := verify.SameSubnet(context.TODO(), e.Ctx, tx.SourceChain); err != nil {
			return err
		}

		allUTXOBytes, err := e.Ctx.SharedMemory.Get(tx.SourceChain, utxoIDs)
		if err != nil {
			return fmt.Errorf("failed to get shared memory: %w", err)
		}

		utxos := make([]*avax.UTXO, len(tx.Ins)+len(tx.ImportedInputs))
		for index, input := range tx.Ins {
			utxo, err := e.State.GetUTXO(input.InputID())
			if err != nil {
				return fmt.Errorf("failed to get UTXO %s: %w", &input.UTXOID, err)
			}
			utxos[index] = utxo
		}
		for i, utxoBytes := range allUTXOBytes {
			utxo := &avax.UTXO{}
			if _, err := txs.Codec.Unmarshal(utxoBytes, utxo); err != nil {
				return fmt.Errorf("failed to unmarshal UTXO: %w", err)
			}
			utxos[i+len(tx.Ins)] = utxo
		}

		ins := make([]*avax.TransferableInput, len(tx.Ins)+len(tx.ImportedInputs))
		copy(ins, tx.Ins)
		copy(ins[len(tx.Ins):], tx.ImportedInputs)

		if err := e.FlowChecker.VerifySpendUTXOs(
			e.Tx.Version(),
			tx,
			utxos,
			ins,
			tx.Outs,
			e.Tx.Creds,
			map[ids.ID]uint64{
				e.Ctx.AVAXAssetID: e.Config.TxFee,
			},
		); err != nil {
			return err
		}
	}

	txID := e.Tx.ID()

	// Consume the UTXOS
	avax.Consume(e.State, tx.Ins)
	// Produce the UTXOS
	avax.Produce(e.State, txID, tx.Outs)

	e.AtomicRequests = map[ids.ID]*atomic.Requests{
		tx.SourceChain: {
			RemoveRequests: utxoIDs,
		},
	}
	return nil
}

func (e *StandardTxExecutor) ExportTx(tx *txs.ExportTx) error {
	if err := e.Tx.SyntacticVerify(e.Ctx); err != nil {
		return err
	}

	outs := make([]*avax.TransferableOutput, len(tx.Outs)+len(tx.ExportedOutputs))
	copy(outs, tx.Outs)
	copy(outs[len(tx.Outs):], tx.ExportedOutputs)

	if e.Bootstrapped.Get() {
		if err := verify.SameSubnet(context.TODO(), e.Ctx, tx.DestinationChain); err != nil {
			return err
		}
	}

	// Verify the flowcheck
	if err := e.FlowChecker.VerifySpend(
		e.Tx.Version(),
		tx,
		e.State,
		tx.Ins,
		outs,
		e.Tx.Creds,
		map[ids.ID]uint64{
			e.Ctx.AVAXAssetID: e.Config.TxFee,
		},
	); err != nil {
		return fmt.Errorf("failed verifySpend: %w", err)
	}

	txID := e.Tx.ID()

	// Consume the UTXOS
	avax.Consume(e.State, tx.Ins)
	// Produce the UTXOS
	avax.Produce(e.State, txID, tx.Outs)

	elems := make([]*atomic.Element, len(tx.ExportedOutputs))
	for i, out := range tx.ExportedOutputs {
		utxo := &avax.UTXO{
			UTXOID: avax.UTXOID{
				TxID:        txID,
				OutputIndex: uint32(len(tx.Outs) + i),
			},
			Asset: avax.Asset{ID: out.AssetID()},
			Out:   out.Out,
		}

		utxoBytes, err := txs.Codec.Marshal(e.Tx.Version(), utxo)
		if err != nil {
			return fmt.Errorf("failed to marshal UTXO: %w", err)
		}
		utxoID := utxo.InputID()
		elem := &atomic.Element{
			Key:   utxoID[:],
			Value: utxoBytes,
		}
		if out, ok := utxo.Out.(avax.Addressable); ok {
			elem.Traits = out.Addresses()
		}

		elems[i] = elem
	}
	e.AtomicRequests = map[ids.ID]*atomic.Requests{
		tx.DestinationChain: {
			PutRequests: elems,
		},
	}
	return nil
}

func (e *StandardTxExecutor) AddValidatorTx(tx *txs.AddValidatorTx) error {
	if tx.Validator.NodeID == ids.EmptyNodeID {
		return errEmptyNodeID
	}

	if _, err := verifyAddValidatorTx(
		e.Backend,
		e.State,
		e.Tx,
		tx,
	); err != nil {
		return err
	}

	txID := e.Tx.ID()
	if err := e.addStakerFromStakerTx(tx, e.State.GetTimestamp(), mockable.MaxTime); err != nil {
		return err
	}

	avax.Consume(e.State, tx.Ins)
	avax.Produce(e.State, txID, tx.Outs)
	return nil
}

func (e *StandardTxExecutor) AddSubnetValidatorTx(tx *txs.AddSubnetValidatorTx) error {
	err := verifyAddSubnetValidatorTx(
		e.Backend,
		e.State,
		e.Tx,
		tx,
	)
	if err != nil {
		return err
	}

	var (
		txID      = e.Tx.ID()
		chainTime = e.State.GetTimestamp()
	)

	err = e.addStakerFromStakerTx(tx, chainTime, chainTime.Add(tx.StakingPeriod()))
	if err != nil {
		return err
	}

	avax.Produce(e.State, txID, tx.Outs)
	return nil
}

func (e *StandardTxExecutor) AddDelegatorTx(tx *txs.AddDelegatorTx) error {
	_, primaryValidatorEndTime, err := verifyAddDelegatorTx(
		e.Backend,
		e.State,
		e.Tx,
		tx,
	)
	if err != nil {
		return err
	}

	txID := e.Tx.ID()
	if err := e.addStakerFromStakerTx(tx, e.State.GetTimestamp(), primaryValidatorEndTime); err != nil {
		return err
	}

	avax.Consume(e.State, tx.Ins)
	avax.Produce(e.State, txID, tx.Outs)
	return nil
}

// Verifies a [*txs.RemoveSubnetValidatorTx] and, if it passes, executes it on
// [e.State]. For verification rules, see [removeSubnetValidatorValidation].
// This transaction will result in [tx.NodeID] being removed as a validator of
// [tx.SubnetID].
// Note: [tx.NodeID] may be either a current or pending validator.
func (e *StandardTxExecutor) RemoveSubnetValidatorTx(tx *txs.RemoveSubnetValidatorTx) error {
	staker, isCurrentValidator, err := removeSubnetValidatorValidation(
		e.Backend,
		e.State,
		e.Tx,
		tx,
	)
	if err != nil {
		return err
	}

	if isCurrentValidator {
		e.State.DeleteCurrentValidator(staker)
	} else {
		e.State.DeletePendingValidator(staker)
	}

	// Invariant: There are no permissioned subnet delegators to remove.

	txID := e.Tx.ID()
	avax.Consume(e.State, tx.Ins)
	avax.Produce(e.State, txID, tx.Outs)

	return nil
}

func (e *StandardTxExecutor) TransformSubnetTx(tx *txs.TransformSubnetTx) error {
	if err := e.Tx.SyntacticVerify(e.Ctx); err != nil {
		return err
	}

	// Note: math.MaxInt32 * time.Second < math.MaxInt64 - so this can never
	// overflow.
	if time.Duration(tx.MaxStakeDuration)*time.Second > e.Backend.Config.MaxStakeDuration {
		return errMaxStakeDurationTooLarge
	}

	baseTxCreds, err := verifyPoASubnetAuthorization(e.Backend, e.State, e.Tx, tx.Subnet, tx.SubnetAuth)
	if err != nil {
		return err
	}

	totalRewardAmount := tx.MaximumSupply - tx.InitialSupply
	if err := e.Backend.FlowChecker.VerifySpend(
		e.Tx.Version(),
		tx,
		e.State,
		tx.Ins,
		tx.Outs,
		baseTxCreds,
		// Invariant: [tx.AssetID != e.Ctx.AVAXAssetID]. This prevents the first
		//            entry in this map literal from being overwritten by the
		//            second entry.
		map[ids.ID]uint64{
			e.Ctx.AVAXAssetID: e.Config.TransformSubnetTxFee,
			tx.AssetID:        totalRewardAmount,
		},
	); err != nil {
		return err
	}

	txID := e.Tx.ID()

	// Consume the UTXOS
	avax.Consume(e.State, tx.Ins)
	// Produce the UTXOS
	avax.Produce(e.State, txID, tx.Outs)
	// Transform the new subnet in the database
	e.State.AddSubnetTransformation(e.Tx)
	e.State.SetCurrentSupply(tx.Subnet, tx.InitialSupply)
	return nil
}

func (e *StandardTxExecutor) AddPermissionlessValidatorTx(tx *txs.AddPermissionlessValidatorTx) error {
	err := verifyAddPermissionlessValidatorTx(
		e.Backend,
		e.State,
		e.Tx,
		tx,
	)
	if err != nil {
		return err
	}

	var (
		txID      = e.Tx.ID()
		chainTime = e.State.GetTimestamp()
	)

	if tx.SubnetID() == constants.PrimaryNetworkID {
		err = e.addStakerFromStakerTx(tx, chainTime, mockable.MaxTime)
	} else {
		err = e.addStakerFromStakerTx(tx, chainTime, chainTime.Add(tx.StakingPeriod()))
	}

	if err != nil {
		return err
	}

	avax.Consume(e.State, tx.Ins)
	avax.Produce(e.State, txID, tx.Outs)
	return nil
}

func (e *StandardTxExecutor) AddPermissionlessDelegatorTx(tx *txs.AddPermissionlessDelegatorTx) error {
	primaryValidatorEndTime, err := verifyAddPermissionlessDelegatorTx(
		e.Backend,
		e.State,
		e.Tx,
		tx,
	)
	if err != nil {
		return err
	}

	txID := e.Tx.ID()
	if err := e.addStakerFromStakerTx(tx, e.State.GetTimestamp(), primaryValidatorEndTime); err != nil {
		return err
	}

	avax.Consume(e.State, tx.Ins)
	avax.Produce(e.State, txID, tx.Outs)

	return nil
}

func (e *StandardTxExecutor) StopStakerTx(tx *txs.StopStakerTx) error {
	stakers, stopTime, err := verifyStopStakerTx(
		e.Backend,
		e.State,
		e.Tx,
		tx,
	)
	if err != nil {
		return err
	}

	for _, toStop := range stakers {
		state.MarkStakerForRemovalInPlaceBeforeTime(toStop, stopTime)
		if toStop.Priority.IsValidator() {
			err = e.State.UpdateCurrentValidator(toStop)
		} else {
			err = e.State.UpdateCurrentDelegator(toStop)
		}
		if err != nil {
			return err
		}
	}

	txID := e.Tx.ID()
	avax.Consume(e.State, tx.Ins)
	avax.Produce(e.State, txID, tx.Outs)

	return nil
}

// addStakerFromStakerTx creates the staker and adds it to state.
// Post Continuous Staking fork activation it has updates current supply in state
func (e *StandardTxExecutor) addStakerFromStakerTx(
	stakerTx txs.Staker,
	chainTime time.Time,
	endTimeBound time.Time,
) error {
	// Pre Continuous Staking fork, stakers are added as pending first, then promoted
	// to current when chainTime reaches their start time.
	// Post Continuous Staking fork, stakers are immediately marked as current.
	// Their start time is current chain time.

	var (
		txID   = e.Tx.ID()
		staker *state.Staker
		err    error
	)

	if !e.Config.IsContinuousStakingActivated(chainTime) {
		preContinuousStakingStakerTx, ok := stakerTx.(txs.PreContinuousStakingStaker)
		if !ok {
			return fmt.Errorf("expected tx type txs.PreContinuousStakingStaker but got %T", stakerTx)
		}
		staker, err = state.NewPendingStaker(txID, preContinuousStakingStakerTx)
	} else {
		var (
			potentialReward = uint64(0)
			stakeDuration   = stakerTx.StakingPeriod()
		)
		if stakerTx.CurrentPriority() != txs.SubnetPermissionedValidatorCurrentPriority {
			subnetID := stakerTx.SubnetID()
			currentSupply, err := e.State.GetCurrentSupply(subnetID)
			if err != nil {
				return err
			}

			rewardCfg, err := e.State.GetRewardConfig(subnetID)
			if err != nil {
				return err
			}
			rewards := reward.NewCalculator(rewardCfg)

			potentialReward = rewards.Calculate(
				stakeDuration,
				stakerTx.Weight(),
				currentSupply,
			)

			updatedSupply := currentSupply + potentialReward
			e.State.SetCurrentSupply(subnetID, updatedSupply)
		}
		staker, err = state.NewCurrentStaker(
			txID,
			stakerTx,
			chainTime,
			endTimeBound,
			potentialReward,
		)
	}

	if err != nil {
		return err
	}

	switch priority := staker.Priority; {
	case priority.IsCurrentValidator():
		e.State.PutCurrentValidator(staker)
	case priority.IsCurrentDelegator():
		e.State.PutCurrentDelegator(staker)
	case priority.IsPendingValidator():
		e.State.PutPendingValidator(staker)
	case priority.IsPendingDelegator():
		e.State.PutPendingDelegator(staker)
	}
	return nil
}
