package main

import (
	"context"
	"fmt"

	cid "github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log"
	libp2p "github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	mh "github.com/multiformats/go-multihash"

	ntraversal "github.com/upperwal/go-libp2p-nat-traversal"
)

func main() {
	logging.SetLogLevel("nat-traversal", "DEBUG")

	ctx := context.Background()

	host, err := libp2p.New(ctx)
	if err != nil {
		panic(err)
	}

	fmt.Println("This node: ", host.ID().Pretty(), " ", host.Addrs())

	d, err := dht.New(ctx, host)
	if err != nil {
		panic(err)
	}

	b, _ := ntraversal.NewNatTraversal(ctx, &host, d)
	b.ConnectToServiceNodes(ctx, []string{"/ip4/127.0.0.1/tcp/3001/p2p/Qmc5mVjNN6n8DG4ky2wxQTY3tWks4Wufgqhz9PbevadKBW"})

	v1b := cid.V1Builder{Codec: cid.Raw, MhType: mh.SHA2_256}
	rendezvousPoint, _ := v1b.Sum([]byte("hey"))
	err = d.Provide(ctx, rendezvousPoint, true)
	if err != nil {
		fmt.Println(err)
	}

	pis, err := d.FindProviders(ctx, rendezvousPoint)
	if err != nil {
		fmt.Println(err)
	}

	for _, pi := range pis {
		if err := host.Connect(ctx, pi); err != nil {
			cerr, err := b.ConnectThroughHolePunching(ctx, pi.ID)
			if err != nil {
				fmt.Println(err)
			} else {
				err = <-cerr
				if err != nil {
					fmt.Println(err)
				}
			}
		}
	}

	select {}
}
