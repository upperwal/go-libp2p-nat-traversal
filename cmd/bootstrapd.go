package main

import (
	"context"
	"flag"
	"fmt"

	logging "github.com/ipfs/go-log"
	libp2p "github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	inet "github.com/libp2p/go-libp2p-net"
	ma "github.com/multiformats/go-multiaddr"

	ntraversal "github.com/upperwal/go-libp2p-nat-traversal"
)

var log = logging.Logger("nat-traversal")

type netNotifiee struct{}

func (nn *netNotifiee) Connected(n inet.Network, c inet.Conn) {
	fmt.Printf("Connected to: %s/p2p/%s\n", c.RemoteMultiaddr(), c.RemotePeer().Pretty())
}

func (nn *netNotifiee) Disconnected(n inet.Network, v inet.Conn)   {}
func (nn *netNotifiee) OpenedStream(n inet.Network, v inet.Stream) {}
func (nn *netNotifiee) ClosedStream(n inet.Network, v inet.Stream) {}
func (nn *netNotifiee) Listen(n inet.Network, a ma.Multiaddr)      {}
func (nn *netNotifiee) ListenClose(n inet.Network, a ma.Multiaddr) {}

func main() {
	logging.SetLogLevel("nat-traversal", "DEBUG")

	port := flag.Int("p", 3000, "port number")
	flag.Parse()

	ctx := context.Background()

	// libp2p.New constructs a new libp2p Host.
	// Other options can be added here.
	sourceMultiAddr, _ := ma.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", *port))

	host, err := libp2p.New(ctx, libp2p.ListenAddrs(sourceMultiAddr))
	if err != nil {
		panic(err)
	}

	no := &netNotifiee{}
	host.Network().Notify(no)

	fmt.Println("This node: ", host.ID().Pretty(), " ", host.Addrs())

	d, err := dht.New(ctx, host)
	if err != nil {
		panic(err)
	}

	ntraversal.NewNatTraversal(ctx, &host, d)

	select {}
}
