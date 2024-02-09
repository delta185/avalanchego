// Copyright (C) 2019-2024, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package executor

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/crypto/secp256k1"
	"github.com/ava-labs/avalanchego/vms/platformvm/state"
)

func TestNewExportTx(t *testing.T) {
	env := newEnvironment(t, banffFork)
	env.ctx.Lock.Lock()
	defer env.ctx.Lock.Unlock()

	type test struct {
		description        string
		destinationChainID ids.ID
		sourceKeys         []*secp256k1.PrivateKey
		timestamp          time.Time
	}

	sourceKey := preFundedKeys[0]

	tests := []test{
		{
			description:        "P->X export",
			destinationChainID: env.ctx.XChainID,
			sourceKeys:         []*secp256k1.PrivateKey{sourceKey},
			timestamp:          defaultValidateStartTime,
		},
		{
			description:        "P->C export",
			destinationChainID: env.ctx.CChainID,
			sourceKeys:         []*secp256k1.PrivateKey{sourceKey},
			timestamp:          env.config.ApricotPhase5Time,
		},
	}

	to := ids.GenerateTestShortID()
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			require := require.New(t)

			tx, err := env.txBuilder.NewExportTx(
				defaultBalance-defaultTxFee, // Amount of tokens to export
				tt.destinationChainID,
				to,
				tt.sourceKeys,
				ids.ShortEmpty, // Change address
			)
			require.NoError(err)

			stateDiff, err := state.NewDiff(lastAcceptedID, env)
			require.NoError(err)

			stateDiff.SetTimestamp(tt.timestamp)

			verifier := StandardTxExecutor{
				Backend: &env.backend,
				State:   stateDiff,
				Tx:      tx,
			}
			require.NoError(tx.Unsigned.Visit(&verifier))
		})
	}
}
