package discovery

import (
	"errors"
	"sync"
	"time"

	"github.com/litecoinfinance/btcd/chaincfg/chainhash"
	"github.com/litecoinfinance/lnd/lnpeer"
	"github.com/litecoinfinance/lnd/lnwire"
	"github.com/litecoinfinance/lnd/routing/route"
	"github.com/litecoinfinance/lnd/ticker"
)

const (
	// DefaultSyncerRotationInterval is the default interval in which we'll
	// rotate a single active syncer.
	DefaultSyncerRotationInterval = 20 * time.Minute

	// DefaultHistoricalSyncInterval is the default interval in which we'll
	// force a historical sync to ensure we have as much of the public
	// network as possible.
	DefaultHistoricalSyncInterval = time.Hour
)

var (
	// ErrSyncManagerExiting is an error returned when we attempt to
	// start/stop a gossip syncer for a connected/disconnected peer, but the
	// SyncManager has already been stopped.
	ErrSyncManagerExiting = errors.New("sync manager exiting")
)

// newSyncer in an internal message we'll use within the SyncManager to signal
// that we should create a GossipSyncer for a newly connected peer.
type newSyncer struct {
	// peer is the newly connected peer.
	peer lnpeer.Peer

	// doneChan serves as a signal to the caller that the SyncManager's
	// internal state correctly reflects the stale active syncer.
	doneChan chan struct{}
}

// staleSyncer is an internal message we'll use within the SyncManager to signal
// that a peer has disconnected and its GossipSyncer should be removed.
type staleSyncer struct {
	// peer is the peer that has disconnected.
	peer route.Vertex

	// doneChan serves as a signal to the caller that the SyncManager's
	// internal state correctly reflects the stale active syncer. This is
	// needed to ensure we always create a new syncer for a flappy peer
	// after they disconnect if they happened to be an active syncer.
	doneChan chan struct{}
}

// SyncManagerCfg contains all of the dependencies required for the SyncManager
// to carry out its duties.
type SyncManagerCfg struct {
	// ChainHash is a hash that indicates the specific network of the active
	// chain.
	ChainHash chainhash.Hash

	// ChanSeries is an interface that provides access to a time series view
	// of the current known channel graph. Each GossipSyncer enabled peer
	// will utilize this in order to create and respond to channel graph
	// time series queries.
	ChanSeries ChannelGraphTimeSeries

	// NumActiveSyncers is the number of peers for which we should have
	// active syncers with. After reaching NumActiveSyncers, any future
	// gossip syncers will be passive.
	NumActiveSyncers int

	// RotateTicker is a ticker responsible for notifying the SyncManager
	// when it should rotate its active syncers. A single active syncer with
	// a chansSynced state will be exchanged for a passive syncer in order
	// to ensure we don't keep syncing with the same peers.
	RotateTicker ticker.Ticker

	// HistoricalSyncTicker is a ticker responsible for notifying the
	// SyncManager when it should attempt a historical sync with a gossip
	// sync peer.
	HistoricalSyncTicker ticker.Ticker
}

// SyncManager is a subsystem of the gossiper that manages the gossip syncers
// for peers currently connected. When a new peer is connected, the manager will
// create its accompanying gossip syncer and determine whether it should have an
// ActiveSync or PassiveSync sync type based on how many other gossip syncers
// are currently active. Any ActiveSync gossip syncers are started in a
// round-robin manner to ensure we're not syncing with multiple peers at the
// same time. The first GossipSyncer registered with the SyncManager will
// attempt a historical sync to ensure we have as much of the public channel
// graph as possible.
type SyncManager struct {
	start sync.Once
	stop  sync.Once

	cfg SyncManagerCfg

	// historicalSync allows us to perform an initial historical sync only
	// _once_ with a peer during the SyncManager's startup.
	historicalSync sync.Once

	// newSyncers is a channel we'll use to process requests to create
	// GossipSyncers for newly connected peers.
	newSyncers chan *newSyncer

	// staleSyncers is a channel we'll use to process requests to tear down
	// GossipSyncers for disconnected peers.
	staleSyncers chan *staleSyncer

	// syncersMu guards the read and write access to the activeSyncers and
	// inactiveSyncers maps below.
	syncersMu sync.Mutex

	// activeSyncers is the set of all syncers for which we are currently
	// receiving graph updates from. The number of possible active syncers
	// is bounded by NumActiveSyncers.
	activeSyncers map[route.Vertex]*GossipSyncer

	// inactiveSyncers is the set of all syncers for which we are not
	// currently receiving new graph updates from.
	inactiveSyncers map[route.Vertex]*GossipSyncer

	wg   sync.WaitGroup
	quit chan struct{}
}

// newSyncManager constructs a new SyncManager backed by the given config.
func newSyncManager(cfg *SyncManagerCfg) *SyncManager {
	return &SyncManager{
		cfg:          *cfg,
		newSyncers:   make(chan *newSyncer),
		staleSyncers: make(chan *staleSyncer),
		activeSyncers: make(
			map[route.Vertex]*GossipSyncer, cfg.NumActiveSyncers,
		),
		inactiveSyncers: make(map[route.Vertex]*GossipSyncer),
		quit:            make(chan struct{}),
	}
}

// Start starts the SyncManager in order to properly carry out its duties.
func (m *SyncManager) Start() {
	m.start.Do(func() {
		m.wg.Add(1)
		go m.syncerHandler()
	})
}

// Stop stops the SyncManager from performing its duties.
func (m *SyncManager) Stop() {
	m.stop.Do(func() {
		close(m.quit)
		m.wg.Wait()

		for _, syncer := range m.inactiveSyncers {
			syncer.Stop()
		}
		for _, syncer := range m.activeSyncers {
			syncer.Stop()
		}
	})
}

// syncerHandler is the SyncManager's main event loop responsible for:
//
// 1. Creating and tearing down GossipSyncers for connected/disconnected peers.

// 2. Finding new peers to receive graph updates from to ensure we don't only
//    receive them from the same set of peers.

// 3. Finding new peers to force a historical sync with to ensure we have as
//    much of the public network as possible.
//
// NOTE: This must be run as a goroutine.
func (m *SyncManager) syncerHandler() {
	defer m.wg.Done()

	m.cfg.RotateTicker.Resume()
	defer m.cfg.RotateTicker.Stop()

	m.cfg.HistoricalSyncTicker.Resume()
	defer m.cfg.HistoricalSyncTicker.Stop()

	var (
		// attemptInitialHistoricalSync determines whether we should
		// attempt an initial historical sync when a new peer connects.
		attemptInitialHistoricalSync = true

		// initialHistoricalSyncCompleted serves as a barrier when
		// initializing new active GossipSyncers. If false, the initial
		// historical sync has not completed, so we'll defer
		// initializing any active GossipSyncers. If true, then we can
		// transition the GossipSyncer immediately. We set up this
		// barrier to ensure we have most of the graph before attempting
		// to accept new updates at tip.
		initialHistoricalSyncCompleted = false

		// initialHistoricalSyncer is the syncer we are currently
		// performing an initial historical sync with.
		initialHistoricalSyncer *GossipSyncer

		// initialHistoricalSyncSignal is a signal that will fire once
		// the intiial historical sync has been completed. This is
		// crucial to ensure that another historical sync isn't
		// attempted just because the initialHistoricalSyncer was
		// disconnected.
		initialHistoricalSyncSignal chan struct{}
	)

	for {
		select {
		// A new peer has been connected, so we'll create its
		// accompanying GossipSyncer.
		case newSyncer := <-m.newSyncers:
			// If we already have a syncer, then we'll exit early as
			// we don't want to override it.
			if _, ok := m.GossipSyncer(newSyncer.peer.PubKey()); ok {
				close(newSyncer.doneChan)
				continue
			}

			s := m.createGossipSyncer(newSyncer.peer)

			m.syncersMu.Lock()
			switch {
			// If we've exceeded our total number of active syncers,
			// we'll initialize this GossipSyncer as passive.
			case len(m.activeSyncers) >= m.cfg.NumActiveSyncers:
				fallthrough

			// Otherwise, it should be initialized as active. If the
			// initial historical sync has yet to complete, then
			// we'll declare is as passive and attempt to transition
			// it when the initial historical sync completes.
			case !initialHistoricalSyncCompleted:
				s.setSyncType(PassiveSync)
				m.inactiveSyncers[s.cfg.peerPub] = s

			// The initial historical sync has completed, so we can
			// immediately start the GossipSyncer as active.
			default:
				s.setSyncType(ActiveSync)
				m.activeSyncers[s.cfg.peerPub] = s
			}
			m.syncersMu.Unlock()

			s.Start()

			// Once we create the GossipSyncer, we'll signal to the
			// caller that they can proceed since the SyncManager's
			// internal state has been updated.
			close(newSyncer.doneChan)

			// We'll force a historical sync with the first peer we
			// connect to, to ensure we get as much of the graph as
			// possible.
			if !attemptInitialHistoricalSync {
				continue
			}

			log.Debugf("Attempting initial historical sync with "+
				"GossipSyncer(%x)", s.cfg.peerPub)

			if err := s.historicalSync(); err != nil {
				log.Errorf("Unable to attempt initial "+
					"historical sync with "+
					"GossipSyncer(%x): %v", s.cfg.peerPub,
					err)
				continue
			}

			// Once the historical sync has started, we'll get a
			// keep track of the corresponding syncer to properly
			// handle disconnects. We'll also use a signal to know
			// when the historical sync completed.
			attemptInitialHistoricalSync = false
			initialHistoricalSyncer = s
			initialHistoricalSyncSignal = s.ResetSyncedSignal()

		// An existing peer has disconnected, so we'll tear down its
		// corresponding GossipSyncer.
		case staleSyncer := <-m.staleSyncers:
			// Once the corresponding GossipSyncer has been stopped
			// and removed, we'll signal to the caller that they can
			// proceed since the SyncManager's internal state has
			// been updated.
			m.removeGossipSyncer(staleSyncer.peer)
			close(staleSyncer.doneChan)

			// If we don't have an initialHistoricalSyncer, or we do
			// but it is not the peer being disconnected, then we
			// have nothing left to do and can proceed.
			switch {
			case initialHistoricalSyncer == nil:
				fallthrough
			case staleSyncer.peer != initialHistoricalSyncer.cfg.peerPub:
				continue
			}

			// Otherwise, our initialHistoricalSyncer corresponds to
			// the peer being disconnected, so we'll have to find a
			// replacement.
			log.Debug("Finding replacement for intitial " +
				"historical sync")

			s := m.forceHistoricalSync()
			if s == nil {
				log.Debug("No eligible replacement found " +
					"for initial historical sync")
				attemptInitialHistoricalSync = true
				continue
			}

			log.Debugf("Replaced initial historical "+
				"GossipSyncer(%v) with GossipSyncer(%x)",
				staleSyncer.peer, s.cfg.peerPub)

			initialHistoricalSyncer = s
			initialHistoricalSyncSignal = s.ResetSyncedSignal()

		// Our initial historical sync signal has completed, so we'll
		// nil all of the relevant fields as they're no longer needed.
		case <-initialHistoricalSyncSignal:
			initialHistoricalSyncer = nil
			initialHistoricalSyncSignal = nil
			initialHistoricalSyncCompleted = true

			log.Debug("Initial historical sync completed")

			// With the initial historical sync complete, we can
			// begin receiving new graph updates at tip. We'll
			// determine whether we can have any more active
			// GossipSyncers. If we do, we'll randomly select some
			// that are currently passive to transition.
			m.syncersMu.Lock()
			numActiveLeft := m.cfg.NumActiveSyncers - len(m.activeSyncers)
			if numActiveLeft <= 0 {
				m.syncersMu.Unlock()
				continue
			}

			log.Debugf("Attempting to transition %v passive "+
				"GossipSyncers to active", numActiveLeft)

			for i := 0; i < numActiveLeft; i++ {
				chooseRandomSyncer(
					m.inactiveSyncers, m.transitionPassiveSyncer,
				)
			}

			m.syncersMu.Unlock()

		// Our RotateTicker has ticked, so we'll attempt to rotate a
		// single active syncer with a passive one.
		case <-m.cfg.RotateTicker.Ticks():
			m.rotateActiveSyncerCandidate()

		// Our HistoricalSyncTicker has ticked, so we'll randomly select
		// a peer and force a historical sync with them.
		case <-m.cfg.HistoricalSyncTicker.Ticks():
			m.forceHistoricalSync()

		case <-m.quit:
			return
		}
	}
}

// createGossipSyncer creates the GossipSyncer for a newly connected peer.
func (m *SyncManager) createGossipSyncer(peer lnpeer.Peer) *GossipSyncer {
	nodeID := route.Vertex(peer.PubKey())
	log.Infof("Creating new GossipSyncer for peer=%x", nodeID[:])

	encoding := lnwire.EncodingSortedPlain
	s := newGossipSyncer(gossipSyncerCfg{
		chainHash:     m.cfg.ChainHash,
		peerPub:       nodeID,
		channelSeries: m.cfg.ChanSeries,
		encodingType:  encoding,
		chunkSize:     encodingTypeToChunkSize[encoding],
		batchSize:     requestBatchSize,
		sendToPeer: func(msgs ...lnwire.Message) error {
			return peer.SendMessageLazy(false, msgs...)
		},
		sendToPeerSync: func(msgs ...lnwire.Message) error {
			return peer.SendMessageLazy(true, msgs...)
		},
	})

	// Gossip syncers are initialized by default in a PassiveSync type
	// and chansSynced state so that they can reply to any peer queries or
	// handle any sync transitions.
	s.setSyncState(chansSynced)
	s.setSyncType(PassiveSync)
	return s
}

// removeGossipSyncer removes all internal references to the disconnected peer's
// GossipSyncer and stops it. In the event of an active GossipSyncer being
// disconnected, a passive GossipSyncer, if any, will take its place.
func (m *SyncManager) removeGossipSyncer(peer route.Vertex) {
	m.syncersMu.Lock()
	defer m.syncersMu.Unlock()

	s, ok := m.gossipSyncer(peer)
	if !ok {
		return
	}

	log.Infof("Removing GossipSyncer for peer=%v", peer)

	// We'll stop the GossipSyncer for the disconnected peer in a goroutine
	// to prevent blocking the SyncManager.
	go s.Stop()

	// If it's a non-active syncer, then we can just exit now.
	if _, ok := m.inactiveSyncers[peer]; ok {
		delete(m.inactiveSyncers, peer)
		return
	}

	// Otherwise, we'll need find a new one to replace it, if any.
	delete(m.activeSyncers, peer)
	newActiveSyncer := chooseRandomSyncer(
		m.inactiveSyncers, m.transitionPassiveSyncer,
	)
	if newActiveSyncer == nil {
		return
	}

	log.Debugf("Replaced active GossipSyncer(%x) with GossipSyncer(%x)",
		peer, newActiveSyncer.cfg.peerPub)
}

// rotateActiveSyncerCandidate rotates a single active syncer. In order to
// achieve this, the active syncer must be in a chansSynced state in order to
// process the sync transition.
func (m *SyncManager) rotateActiveSyncerCandidate() {
	m.syncersMu.Lock()
	defer m.syncersMu.Unlock()

	// If we couldn't find an eligible active syncer to rotate, we can
	// return early.
	activeSyncer := chooseRandomSyncer(m.activeSyncers, nil)
	if activeSyncer == nil {
		log.Debug("No eligible active syncer to rotate")
		return
	}

	// Similarly, if we don't have a candidate to rotate with, we can return
	// early as well.
	candidate := chooseRandomSyncer(m.inactiveSyncers, nil)
	if candidate == nil {
		log.Debug("No eligible candidate to rotate active syncer")
		return
	}

	// Otherwise, we'll attempt to transition each syncer to their
	// respective new sync type.
	log.Debugf("Rotating active GossipSyncer(%x) with GossipSyncer(%x)",
		activeSyncer.cfg.peerPub, candidate.cfg.peerPub)

	if err := m.transitionActiveSyncer(activeSyncer); err != nil {
		log.Errorf("Unable to transition active GossipSyncer(%x): %v",
			activeSyncer.cfg.peerPub, err)
		return
	}

	if err := m.transitionPassiveSyncer(candidate); err != nil {
		log.Errorf("Unable to transition passive GossipSyncer(%x): %v",
			activeSyncer.cfg.peerPub, err)
		return
	}
}

// transitionActiveSyncer transitions an active syncer to a passive one.
//
// NOTE: This must be called with the syncersMu lock held.
func (m *SyncManager) transitionActiveSyncer(s *GossipSyncer) error {
	log.Debugf("Transitioning active GossipSyncer(%x) to passive",
		s.cfg.peerPub)

	if err := s.ProcessSyncTransition(PassiveSync); err != nil {
		return err
	}

	delete(m.activeSyncers, s.cfg.peerPub)
	m.inactiveSyncers[s.cfg.peerPub] = s

	return nil
}

// transitionPassiveSyncer transitions a passive syncer to an active one.
//
// NOTE: This must be called with the syncersMu lock held.
func (m *SyncManager) transitionPassiveSyncer(s *GossipSyncer) error {
	log.Debugf("Transitioning passive GossipSyncer(%x) to active",
		s.cfg.peerPub)

	if err := s.ProcessSyncTransition(ActiveSync); err != nil {
		return err
	}

	delete(m.inactiveSyncers, s.cfg.peerPub)
	m.activeSyncers[s.cfg.peerPub] = s

	return nil
}

// forceHistoricalSync chooses a syncer with a remote peer at random and forces
// a historical sync with it.
func (m *SyncManager) forceHistoricalSync() *GossipSyncer {
	m.syncersMu.Lock()
	defer m.syncersMu.Unlock()

	// We'll sample from both sets of active and inactive syncers in the
	// event that we don't have any inactive syncers.
	return chooseRandomSyncer(m.gossipSyncers(), func(s *GossipSyncer) error {
		return s.historicalSync()
	})
}

// chooseRandomSyncer iterates through the set of syncers given and returns the
// first one which was able to successfully perform the action enclosed in the
// function closure.
//
// NOTE: It's possible for a nil value to be returned if there are no eligible
// candidate syncers.
func chooseRandomSyncer(syncers map[route.Vertex]*GossipSyncer,
	action func(*GossipSyncer) error) *GossipSyncer {

	for _, s := range syncers {
		// Only syncers in a chansSynced state are viable for sync
		// transitions, so skip any that aren't.
		if s.syncState() != chansSynced {
			continue
		}

		if action != nil {
			if err := action(s); err != nil {
				log.Debugf("Skipping eligible candidate "+
					"GossipSyncer(%x): %v", s.cfg.peerPub,
					err)
				continue
			}
		}

		return s
	}

	return nil
}

// InitSyncState is called by outside sub-systems when a connection is
// established to a new peer that understands how to perform channel range
// queries. We'll allocate a new GossipSyncer for it, and start any goroutines
// needed to handle new queries. The first GossipSyncer registered with the
// SyncManager will attempt a historical sync to ensure we have as much of the
// public channel graph as possible.
//
// TODO(wilmer): Only mark as ActiveSync if this isn't a channel peer.
func (m *SyncManager) InitSyncState(peer lnpeer.Peer) error {
	done := make(chan struct{})

	select {
	case m.newSyncers <- &newSyncer{
		peer:     peer,
		doneChan: done,
	}:
	case <-m.quit:
		return ErrSyncManagerExiting
	}

	select {
	case <-done:
		return nil
	case <-m.quit:
		return ErrSyncManagerExiting
	}
}

// PruneSyncState is called by outside sub-systems once a peer that we were
// previously connected to has been disconnected. In this case we can stop the
// existing GossipSyncer assigned to the peer and free up resources.
func (m *SyncManager) PruneSyncState(peer route.Vertex) {
	done := make(chan struct{})

	// We avoid returning an error when the SyncManager is stopped since the
	// GossipSyncer will be stopped then anyway.
	select {
	case m.staleSyncers <- &staleSyncer{
		peer:     peer,
		doneChan: done,
	}:
	case <-m.quit:
		return
	}

	select {
	case <-done:
	case <-m.quit:
	}
}

// GossipSyncer returns the associated gossip syncer of a peer. The boolean
// returned signals whether there exists a gossip syncer for the peer.
func (m *SyncManager) GossipSyncer(peer route.Vertex) (*GossipSyncer, bool) {
	m.syncersMu.Lock()
	defer m.syncersMu.Unlock()
	return m.gossipSyncer(peer)
}

// gossipSyncer returns the associated gossip syncer of a peer. The boolean
// returned signals whether there exists a gossip syncer for the peer.
func (m *SyncManager) gossipSyncer(peer route.Vertex) (*GossipSyncer, bool) {
	syncer, ok := m.inactiveSyncers[peer]
	if ok {
		return syncer, true
	}
	syncer, ok = m.activeSyncers[peer]
	if ok {
		return syncer, true
	}
	return nil, false
}

// GossipSyncers returns all of the currently initialized gossip syncers.
func (m *SyncManager) GossipSyncers() map[route.Vertex]*GossipSyncer {
	m.syncersMu.Lock()
	defer m.syncersMu.Unlock()
	return m.gossipSyncers()
}

// gossipSyncers returns all of the currently initialized gossip syncers.
func (m *SyncManager) gossipSyncers() map[route.Vertex]*GossipSyncer {
	numSyncers := len(m.inactiveSyncers) + len(m.activeSyncers)
	syncers := make(map[route.Vertex]*GossipSyncer, numSyncers)

	for _, syncer := range m.inactiveSyncers {
		syncers[syncer.cfg.peerPub] = syncer
	}
	for _, syncer := range m.activeSyncers {
		syncers[syncer.cfg.peerPub] = syncer
	}

	return syncers
}
