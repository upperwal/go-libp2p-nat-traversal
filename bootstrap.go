package bootstrap

import (
	"bufio"
	"context"
	"fmt"
	"sync"

	"github.com/libp2p/go-libp2p-kad-dht"

	protocol "github.com/upperwal/go-libp2p-bootstrap/protocol"

	ggio "github.com/gogo/protobuf/io"

	iaddr "github.com/ipfs/go-ipfs-addr"
	logging "github.com/ipfs/go-log"
	host "github.com/libp2p/go-libp2p-host"
	inet "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
)

var log = logging.Logger("bootstrap")

const (
	protocolBootstrap = "/bootstrap/1.0.0"
)

type StreamContainer struct {
	mux      *sync.Mutex
	peerList map[peer.ID]*streamWrapper
}

type PacketWPeer struct {
	peer   peer.ID
	packet *protocol.Protocol
}

// Bootstrap <TODO>
type Bootstrap struct {
	host           *host.Host
	serviceNodes   []peer.ID
	bootstrapPeers StreamContainer
	incoming       chan PacketWPeer
	outgoing       chan PacketWPeer
	dht            *dht.IpfsDHT
	connMap        map[peer.ID]chan error
}

// NewBootstrap creates a new bootstraper node.
func NewBootstrap(ctx context.Context, host *host.Host, opt ...Option) (*Bootstrap, error) {

	sc := StreamContainer{
		mux:      &sync.Mutex{},
		peerList: make(map[peer.ID]*streamWrapper),
	}

	d, err := dht.New(ctx, *host)
	if err != nil {
		return nil, err
	}

	b := &Bootstrap{
		host:           host,
		serviceNodes:   make([]peer.ID, 0),
		bootstrapPeers: sc,
		incoming:       make(chan PacketWPeer, 10),
		outgoing:       make(chan PacketWPeer, 10),
		dht:            d,
		connMap:        make(map[peer.ID]chan error),
	}

	(*host).SetStreamHandler(protocolBootstrap, b.streamHandler)

	go b.messageHandler()

	return b, nil
}

// ConnectToServiceNodes connects to bootstrap service nodes.
// "/ip4/35.196.131.102/tcp/3001/p2p/QmQnAZsyiJSovuqg8zjP3nKdm6Pwb75Mpn8HnGyD5WYZ15"
func (b *Bootstrap) ConnectToServiceNodes(ctx context.Context, listPeers []string) {
	for _, peerAddr := range listPeers {
		addr, _ := iaddr.ParseString(peerAddr)
		peerinfo, _ := pstore.InfoFromP2pAddr(addr.Multiaddr())

		(*b.host).Peerstore().AddAddrs(peerinfo.ID, peerinfo.Addrs, pstore.PermanentAddrTTL)
		log.Info("Connecting to: ", peerinfo.ID)
		if s, err := (*b.host).NewStream(ctx, peerinfo.ID, protocolBootstrap); err == nil {
			log.Info("Connection established with bootstrap node: ", *peerinfo)

			b.setStreamWrapper(s)
			b.serviceNodes = append(b.serviceNodes, peerinfo.ID)
		} else {
			log.Error(err)
		}
	}
}

func (b *Bootstrap) setStreamWrapper(s inet.Stream) {

	bw := bufio.NewWriter(s)

	r := ggio.NewDelimitedReader(s, 1<<20)
	w := ggio.NewDelimitedWriter(bw)

	sm := &streamWrapper{
		s:  &s,
		bw: bw,
		r:  &r,
		w:  &w,
	}

	b.bootstrapPeers.mux.Lock()
	b.bootstrapPeers.peerList[s.Conn().RemotePeer()] = sm
	b.bootstrapPeers.mux.Unlock()

	go sm.readMsg(b.incoming)
}

// ConnectThroughHolePunching uses a stun server to coordinate a hole punching.
func (b *Bootstrap) ConnectThroughHolePunching(ctx context.Context, p peer.ID) (chan error, error) {
	if len(b.serviceNodes) == 0 {
		log.Error("not connected to any service node")
		return nil, fmt.Errorf("not connected to any service node")
	}

	log.Info("Conn to peer: ", p)

	res := make(chan error, 1)
	b.connMap[p] = res

	b.outgoing <- PacketWPeer{
		peer: b.serviceNodes[0],
		packet: &protocol.Protocol{
			Type: protocol.Protocol_CONNECTION_REQUEST,
			PeerID: &protocol.Protocol_PeerID{
				Id: []byte(peer.IDHexEncode(p)),
			},
		},
	}
	return res, nil
}

func (b *Bootstrap) messageHandler() {
	for {
		select {
		case m := <-b.incoming:
			log.Info("incoming packet")
			switch m.packet.Type {
			case protocol.Protocol_CONNECTION_REQUEST:
				go b.handleConnectionRequest(m)
			case protocol.Protocol_HOLE_PUNCH_REQUEST:
				go b.handleHolePunchRequest(m)
			}
		case o := <-b.outgoing:
			log.Info("sending out: ", o.peer, o.packet)
			b.bootstrapPeers.peerList[o.peer].writeMsg(o.packet)
		}
	}
}

func (b *Bootstrap) handleConnectionRequest(m PacketWPeer) {
	id, _ := peer.IDHexDecode(string(m.packet.PeerID.Id))
	log.Info("Got a connection request to: ", id)

	//host := *b.host

	piNonInit, err := b.findPeerInfo(id)
	if err != nil {
		log.Error(err)
		b.sendErrMessage(err)
		return
	}
	piInitiator, err := b.findPeerInfo(m.peer)
	if err != nil {
		log.Error(err)
		b.sendErrMessage(err)
		return
	}

	b.sendPunchRequest(id, piInitiator)
	b.sendPunchRequest(m.peer, piNonInit)
}

func (b *Bootstrap) findPeerInfo(p peer.ID) ([]byte, error) {
	pi, err := b.dht.FindPeer(context.Background(), p)
	if err != nil {
		log.Error(err)
		return nil, fmt.Errorf("Could not find a peer")
	}
	data, err := pi.MarshalJSON()
	if err != nil {
		log.Error(err)
		return nil, fmt.Errorf("cannot marshal peerinfo")
	}

	return data, nil
}

func (b *Bootstrap) sendPunchRequest(to peer.ID, pi []byte) {
	b.outgoing <- PacketWPeer{
		peer: to,
		packet: &protocol.Protocol{
			Type: protocol.Protocol_HOLE_PUNCH_REQUEST,
			PeerInfo: &protocol.Protocol_PeerInfo{
				Info: pi,
			},
		},
	}
}

func (b *Bootstrap) sendErrMessage(err error) {}

func (b *Bootstrap) handleHolePunchRequest(m PacketWPeer) {
	pi := pstore.PeerInfo{}
	pi.UnmarshalJSON(m.packet.PeerInfo.Info)

	log.Info("Got punch request to: ", pi)

	if err := (*b.host).Connect(context.Background(), pi); err != nil {
		log.Error(err)
		b.connMap[pi.ID] <- fmt.Errorf("could not open a connection")
	} else {
		b.connMap[pi.ID] <- nil
		close(b.connMap[pi.ID])
		delete(b.connMap, pi.ID)
	}
}

func (b *Bootstrap) streamHandler(s inet.Stream) {
	log.Info("Connected to: ", s.Conn().RemotePeer())
	b.setStreamWrapper(s)
}
