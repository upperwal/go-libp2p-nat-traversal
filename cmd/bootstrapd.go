package main

import (
	"context"
	"flag"
	"fmt"

	logging "github.com/ipfs/go-log"
	libp2p "github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	inet "github.com/libp2p/go-libp2p-net"
	"github.com/multiformats/go-multiaddr"

	ntraversal "github.com/upperwal/go-libp2p-nat-traversal"
)

func handleStream(stream inet.Stream) {

}

func main() {
	logging.SetLogLevel("nat-traversal", "DEBUG")

	port := flag.Int("p", 3000, "port number")
	flag.Parse()

	ctx := context.Background()

	// libp2p.New constructs a new libp2p Host.
	// Other options can be added here.
	sourceMultiAddr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", *port))

	host, err := libp2p.New(ctx, libp2p.ListenAddrs(sourceMultiAddr))
	if err != nil {
		panic(err)
	}

	fmt.Println("This node: ", host.ID().Pretty(), " ", host.Addrs())

	d, err := dht.New(ctx, host)
	if err != nil {
		panic(err)
	}

	ntraversal.NewNatTraversal(ctx, &host, d)

	select {}
}
