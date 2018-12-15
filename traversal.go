package ntraversal

import (
	"bufio"
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	protocol "github.com/upperwal/go-libp2p-nat-traversal/protocol"

	ggio "github.com/gogo/protobuf/io"

	iaddr "github.com/ipfs/go-ipfs-addr"
	logging "github.com/ipfs/go-log"
	host "github.com/libp2p/go-libp2p-host"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	inet "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	swarm "github.com/libp2p/go-libp2p-swarm"
)

var log = logging.Logger("nat-traversal")

const (
	protocolBootstrap = "/ntraversal/1.0.0"
)

type StreamContainer struct {
	mux      *sync.Mutex
	peerList map[peer.ID]*streamWrapper
}

type PacketWPeer struct {
	peer   peer.ID
	packet *protocol.Protocol
}

// NatTraversal <TODO>
type NatTraversal struct {
	host           *host.Host
	serviceNodes   []peer.ID
	bootstrapPeers StreamContainer
	incoming       chan PacketWPeer
	outgoing       chan PacketWPeer
	dht            *dht.IpfsDHT
	connMap        map[peer.ID]chan error
}

// NewNatTraversal creates a new bootstraper node.
func NewNatTraversal(ctx context.Context, host *host.Host, dht *dht.IpfsDHT, opt ...Option) (*NatTraversal, error) {

	sc := StreamContainer{
		mux:      &sync.Mutex{},
		peerList: make(map[peer.ID]*streamWrapper),
	}

	b := &NatTraversal{
		host:           host,
		serviceNodes:   make([]peer.ID, 0),
		bootstrapPeers: sc,
		incoming:       make(chan PacketWPeer, 10),
		outgoing:       make(chan PacketWPeer, 10),
		dht:            dht,
		connMap:        make(map[peer.ID]chan error),
	}

	(*host).SetStreamHandler(protocolBootstrap, b.streamHandler)

	go b.messageHandler()

	return b, nil
}

// ConnectToServiceNodes connects to bootstrap service nodes.
// "/ip4/35.196.131.102/tcp/3001/p2p/QmQnAZsyiJSovuqg8zjP3nKdm6Pwb75Mpn8HnGyD5WYZ15"
func (b *NatTraversal) ConnectToServiceNodes(ctx context.Context, listPeers []string) {
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

func (b *NatTraversal) setStreamWrapper(s inet.Stream) {

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
func (b *NatTraversal) ConnectThroughHolePunching(ctx context.Context, p peer.ID) (chan error, error) {
	if len(b.serviceNodes) == 0 {
		log.Error("not connected to any service node")
		return nil, fmt.Errorf("not connected to any service node")
	}

	log.Info("Conn to peer: ", p)

	// TODO: timer logic: should have a timeout
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

func (b *NatTraversal) messageHandler() {
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
			go b.bootstrapPeers.peerList[o.peer].writeMsg(o.packet)
		}
	}
}

func (b *NatTraversal) handleConnectionRequest(m PacketWPeer) {
	id, _ := peer.IDHexDecode(string(m.packet.PeerID.Id))
	log.Info("Got a connection request to: ", id)

	//host := *b.host

	piInitiator, err := b.findPeerInfo(m.peer)
	if err != nil {
		log.Error(err)
		b.sendErrMessage(err)
		return
	}
	b.sendPunchRequest(id, piInitiator)

	time.Sleep(time.Millisecond * 500)

	piNonInit, err := b.findPeerInfo(id)
	if err != nil {
		log.Error(err)
		b.sendErrMessage(err)
		return
	}
	b.sendPunchRequest(m.peer, piNonInit)
}

func (b *NatTraversal) findPeerInfo(p peer.ID) ([]byte, error) {
	pi, err := b.dht.FindPeer(context.Background(), p)
	if err != nil {
		log.Error(err)
		return nil, fmt.Errorf("Could not find a peer")
	}
	piPublic := pstore.PeerInfo{}
	for _, addr := range pi.Addrs {
		// hacky way to remove all loopback and private addresses
		// should be removed
		if strings.Contains(addr.String(), "127.") ||
			strings.Contains(addr.String(), "192.") ||
			strings.Contains(addr.String(), "10.") ||
			strings.Contains(addr.String(), "p2p-circuit") {
			continue
		}
		piPublic.Addrs = append(piPublic.Addrs, addr)
	}
	piPublic.ID = pi.ID
	data, err := piPublic.MarshalJSON()
	if err != nil {
		log.Error(err)
		return nil, fmt.Errorf("cannot marshal peerinfo")
	}

	return data, nil
}

func (b *NatTraversal) sendPunchRequest(to peer.ID, pi []byte) {
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

func (b *NatTraversal) sendErrMessage(err error) {}

func (b *NatTraversal) handleHolePunchRequest(m PacketWPeer) {
	pi := pstore.PeerInfo{}
	pi.UnmarshalJSON(m.packet.PeerInfo.Info)

	log.Info("Got punch request to: ", pi)

	cleanup := func() {
		close(b.connMap[pi.ID])
		delete(b.connMap, pi.ID)
	}

	cnt := 3
	var err error
	for i := 0; i < cnt; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err = (*b.host).Connect(ctx, pi)
		cancel()
		if err == nil {
			log.Info(i+1, "trial succeeded.", err)
			break
		}
		(*b.host).Network().(*swarm.Swarm).Backoff().Clear(pi.ID)

		if strings.Contains(err.Error(), "no route to host") {
			log.Info("Delay because of", err)
			time.Sleep(time.Second * 5)
		}

		log.Error(i+1, "Failed")
	}

	if err != nil {
		log.Error("All attempts Failed")
		b.connMap[pi.ID] <- err
	} else {
		b.connMap[pi.ID] <- nil
	}

	cleanup()
}

func (b *NatTraversal) streamHandler(s inet.Stream) {
	log.Info("Connected to: ", s.Conn().RemotePeer())
	b.setStreamWrapper(s)
}
