// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package cpu

var NoUsage User = noUsage{}

type noUsage struct{}

func (noUsage) Usage() float64 { return 0 }
