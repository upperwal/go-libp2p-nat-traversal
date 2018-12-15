package main

import (
	"context"
	"flag"
	"fmt"

	logging "github.com/ipfs/go-log"
	libp2p "github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	peer "github.com/libp2p/go-libp2p-peer"
	ma "github.com/multiformats/go-multiaddr"

	ntraversal "github.com/upperwal/go-libp2p-nat-traversal"
)

func main() {
	logging.SetLogLevel("nat-traversal", "DEBUG")

	port := flag.Int("p", 3000, "port number")
	rp := flag.String("r", "", "remote peer id")
	flag.Parse()

	ctx := context.Background()

	sourceMultiAddr, _ := ma.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", *port))

	host, err := libp2p.New(ctx, libp2p.ListenAddrs(sourceMultiAddr))
	if err != nil {
		panic(err)
	}

	fmt.Println("This node: ", host.ID().Pretty(), " ", host.Addrs())

	d, err := dht.New(ctx, host)
	if err != nil {
		panic(err)
	}

	b, _ := ntraversal.NewNatTraversal(ctx, &host, d)

	/* ma, _ := ma.NewMultiaddr("/ip4/127.0.0.1/tcp/3000/p2p/QmSHQpWVzoGWiYRyBrikFp6tr8MAwm6RnUxPsu1NC2y8iJ")
	pi, _ := pstore.InfoFromP2pAddr(ma) */
	b.ConnectToServiceNodes(ctx, []string{"/ip4/127.0.0.1/tcp/3001/p2p/Qmc5mVjNN6n8DG4ky2wxQTY3tWks4Wufgqhz9PbevadKBW"})

	/* ma, _ := ma.NewMultiaddr("/ip4/127.0.0.1/tcp/3000/p2p/QmVvYUj13isfoP4p9ppDZgboX9QwUDKkefP2nTGxVwfYBz")
	pi, _ := pstore.InfoFromP2pAddr(ma) */
	if *rp != "" {

		p, err := peer.IDB58Decode(string(*rp))
		fmt.Println(err)
		fmt.Println("Conn to: ", p)

		cerr, err := b.ConnectThroughHolePunching(ctx, p)
		if err != nil {
			fmt.Println(err)
		}

		err = <-cerr
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println("Connected to: ", p)
		}
	}

	select {}
}
