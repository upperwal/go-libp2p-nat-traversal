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

	bootstrap "github.com/upperwal/go-libp2p-bootstrap"
)

func main() {
	logging.SetLogLevel("bootstrap", "DEBUG")

	port := flag.Int("p", 3000, "port number")
	rp := flag.String("r", "", "remote peer id")
	flag.Parse()

	ctx := context.Background()

	// libp2p.New constructs a new libp2p Host.
	// Other options can be added here.
	sourceMultiAddr, _ := ma.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", *port))

	host, err := libp2p.New(ctx, libp2p.ListenAddrs(sourceMultiAddr))
	if err != nil {
		panic(err)
	}

	fmt.Println("This node: ", host.ID().Pretty(), " ", host.Addrs())

	_, err = dht.New(ctx, host)
	if err != nil {
		panic(err)
	}

	b, _ := bootstrap.NewBootstrap(ctx, &host)

	/* ma, _ := ma.NewMultiaddr("/ip4/127.0.0.1/tcp/3000/p2p/QmSHQpWVzoGWiYRyBrikFp6tr8MAwm6RnUxPsu1NC2y8iJ")
	pi, _ := pstore.InfoFromP2pAddr(ma) */
	b.ConnectToServiceNodes(ctx, []string{"/ip4/127.0.0.1/tcp/3001/p2p/QmQ3jP79BhHUyVmobyQtMEZmVSe7ceBHKs4HoCw8Ep7zzA"})

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
