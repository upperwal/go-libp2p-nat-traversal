package main

import (
	"context"
	"fmt"

	libp2p "github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/multiformats/go-multiaddr"
	logging "github.com/ipfs/go-log"
)

func main() {
	logging.SetLogLevel("dht", "DEBUG")
	ctx := context.Background()

	// libp2p.New constructs a new libp2p Host.
	// Other options can be added here.
	sourceMultiAddr, _ := multiaddr.NewMultiaddr("/ip4/0.0.0.0/tcp/3001")

	host, err := libp2p.New(ctx, libp2p.ListenAddrs(sourceMultiAddr))
	if err != nil {
		panic(err)
	}

	fmt.Println("This node: ", host.ID().Pretty(), " ", host.Addrs())

	_, err = dht.New(ctx, host)
	if err != nil {
		panic(err)
	}

	select {}
}

