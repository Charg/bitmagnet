package dht

import "net/netip"

// DiscoveredHash is an info hash passively observed in an incoming query to the DHT server.
type DiscoveredHash struct {
	InfoHash ID
	// Source is the query type the hash was observed in (QAnnouncePeer or QGetPeers).
	Source string
	// From is the address of the node that sent the query.
	From netip.AddrPort
	// Peer is the address of a peer known to have the torrent (from announce_peer); zero if unknown.
	Peer netip.AddrPort
}
