// Copyright (C) 2019-2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package network

import (
	"fmt"
	"sync"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/wrappers"
)

// GossipTracker tracks the peers that we're currently aware of, as well as the
// peers we've told other peers about. This data is stored in a bitset to
// optimize space, where only N (num peers) bits will be used.
//
// This is done by recording some state information of both what peers this node
// is aware of, and what peers we've told each peer about.
//
//
// As an example, say we track three peers (most-significant-bit first):
// 	local: 		[1, 1, 1] // [p3, p2, p1] we always know about everyone
// 	knownPeers:	{
// 		p1: [1, 1, 1] // p1 knows about everyone
// 		p2: [0, 1, 1] // p2 doesn't know about p3
// 		p3: [0, 0, 1] // p3 knows only about p3
// 	}
//
// GetUnknown computes the information we haven't sent to a given peer
// (using the bitwise AND NOT operator). Ex:
// 	GetUnknown(p1) -  [0, 0, 0]
// 	GetUnknown(p2) -  [1, 0, 0]
// 	GetUnknown(p3) -  [1, 1, 0]
//
// Using the GossipTracker, we can quickly compute the peers each peer doesn't
// know about using GetUnknown so that in subsequent PeerList gossip messages
// we only send information that this peer (most likely) doesn't already know
// about. The only edge-case where we'll send a redundant set of bytes is if
// another remote peer gossips to the same peer we're trying to gossip to first.
type GossipTracker struct {
	// a bitset of the peers that we are aware of
	local ids.BigBitSet

	// a mapping of peer => the peers we know we sent to them
	knownPeers map[ids.NodeID]ids.BigBitSet
	// a mapping of peers => the index they occupy in the bitsets
	peersToIndices map[ids.NodeID]int
	// a mapping of indices in the bitsets => the peer they correspond to
	indicesToPeers map[int]ids.NodeID

	lock    sync.RWMutex
	metrics gossipTrackerMetrics
}

// NewGossipTracker returns an instance of GossipTracker
func NewGossipTracker(registerer prometheus.Registerer, namespace string) (*GossipTracker, error) {
	m, err := newGossipTrackerMetrics(registerer, fmt.Sprintf("%s_gossip_tracker", namespace))
	if err != nil {
		return nil, err
	}

	return &GossipTracker{
		local:          ids.NewBigBitSet(),
		knownPeers:     make(map[ids.NodeID]ids.BigBitSet),
		peersToIndices: make(map[ids.NodeID]int),
		indicesToPeers: make(map[int]ids.NodeID),
		metrics:        m,
	}, nil
}

// Contains returns if a peer is being tracked
func (g *GossipTracker) Contains(id ids.NodeID) bool {
	g.lock.RLock()
	defer g.lock.RUnlock()

	_, ok := g.knownPeers[id]
	return ok
}

// Add starts tracking a peer
func (g *GossipTracker) Add(id ids.NodeID) bool {
	g.lock.Lock()
	defer g.lock.Unlock()

	// Don't add the peer if it's already being tracked
	if _, ok := g.peersToIndices[id]; ok {
		return false
	}

	// Add the peer to the MSB of the bitset.
	// NOTE: strict ordering is not guaranteed due to invariants with [Remove].
	// TODO: consider adding to the LSB instead, so that every time a new peer
	// is added the resulting unknown isn't [1, 0,..., 0] (high sparsity),
	// and is instead [1].
	tail := len(g.peersToIndices)
	g.peersToIndices[id] = tail
	g.knownPeers[id] = ids.NewBigBitSet()
	g.indicesToPeers[tail] = id

	g.local.Add(tail)

	g.metrics.localPeersSize.Set(float64(g.local.Len()))
	g.metrics.peersToIndicesSize.Set(float64(len(g.peersToIndices)))
	g.metrics.indicesToPeersSize.Set(float64(len(g.indicesToPeers)))

	return true
}

// Remove stops tracking a given peer
func (g *GossipTracker) Remove(id ids.NodeID) bool {
	g.lock.Lock()
	defer g.lock.Unlock()

	// Only remove peers that are actually being tracked
	idx, ok := g.peersToIndices[id]
	if !ok {
		return false
	}

	evicted := g.indicesToPeers[idx]
	// swap the peer-to-be-removed with the tail peer
	// if the element we're swapping with is ourselves, we can skip this swap
	// since we only need to delete instead
	tail := len(g.peersToIndices) - 1
	if idx != tail {
		lastPeer := g.indicesToPeers[tail]

		g.indicesToPeers[idx] = lastPeer
		g.peersToIndices[lastPeer] = idx
	}

	delete(g.knownPeers, evicted)
	delete(g.peersToIndices, evicted)
	delete(g.indicesToPeers, tail)

	g.local.Remove(tail)

	// remove the peer from everyone else's peer lists
	for _, knownPeers := range g.knownPeers {
		// swap the element to be removed with the tail
		if idx != tail {
			if knownPeers.Contains(tail) {
				knownPeers.Add(idx)
			} else {
				knownPeers.Remove(idx)
			}
		}
		knownPeers.Remove(tail)
	}

	g.metrics.localPeersSize.Set(float64(g.local.Len()))
	g.metrics.peersToIndicesSize.Set(float64(len(g.peersToIndices)))
	g.metrics.indicesToPeersSize.Set(float64(len(g.indicesToPeers)))

	return true
}

// UpdateKnown adds to the peers that a peer knows about
// invariants:
// 1. [id] and [learned] should only contain nodeIDs that have been tracked with
// 	  Add(). Trying to add nodeIDs that aren't tracked yet will result in a noop
// 	  and this will return [false].
func (g *GossipTracker) UpdateKnown(id ids.NodeID, learned []ids.NodeID) bool {
	g.lock.Lock()
	defer g.lock.Unlock()

	known, ok := g.knownPeers[id]
	if !ok {
		return false
	}

	bs := ids.NewBigBitSet()
	for _, nodeID := range learned {
		idx, ok := g.peersToIndices[nodeID]
		if !ok {
			return false
		}

		bs.Add(idx)
	}

	known.Union(bs)

	return true
}

// GetUnknown returns the peers that we haven't sent to this peer
// [limit] should be >= 0
func (g *GossipTracker) GetUnknown(id ids.NodeID, limit int) ([]ids.NodeID, bool) {
	if limit <= 0 {
		return nil, false
	}

	g.lock.RLock()
	defer g.lock.RUnlock()

	// Calculate the unknown information we need to send to this peer.
	// We do this by computing the [local] information we know,
	// computing what the peer knows in its [knownPeers], and sending over
	// the difference.
	unknown := ids.NewBigBitSet()
	unknown.Union(g.local)

	knownPeers, ok := g.knownPeers[id]
	if !ok {
		return nil, false
	}

	unknown.Difference(knownPeers)

	result := make([]ids.NodeID, 0, limit)

	// We iterate from the LSB -> MSB when computing our diffs.
	// This is because [Add] always inserts at the MSB, so we retrieve the
	// unknown peers starting at the oldest unknown peer to avoid complications
	// where a subset of nodes might be "flickering" offline/online, resulting
	// in the same diff being sent over each time.
	for i := 0; i < unknown.Len(); i++ {
		// skip the bits that aren't set
		if !unknown.Contains(i) {
			continue
		}
		// stop if we exceed the max specified elements to return
		if len(result) >= limit {
			break
		}

		result = append(result, g.indicesToPeers[i])
	}

	return result, true
}

type gossipTrackerMetrics struct {
	localPeersSize     prometheus.Gauge
	peersToIndicesSize prometheus.Gauge
	indicesToPeersSize prometheus.Gauge
}

func newGossipTrackerMetrics(registerer prometheus.Registerer, namespace string) (gossipTrackerMetrics, error) {
	m := gossipTrackerMetrics{
		localPeersSize: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "local_peers_size",
				Help:      "amount of peers this node is tracking gossip for",
			},
		),
		peersToIndicesSize: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "peers_to_indices_size",
				Help:      "amount of peers this node is tracking in peersToIndices",
			},
		),
		indicesToPeersSize: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "indices_to_peers_size",
				Help:      "amount of peers this node is tracking in indicesToPeers",
			},
		),
	}

	errs := wrappers.Errs{}
	errs.Add(
		registerer.Register(m.localPeersSize),
		registerer.Register(m.peersToIndicesSize),
		registerer.Register(m.indicesToPeersSize),
	)

	return m, errs.Err
}
