package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	logging "github.com/ipfs/go-log"
	libp2p "github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	peer "github.com/libp2p/go-libp2p-peer"
	ma "github.com/multiformats/go-multiaddr"
	mplex "github.com/whyrusleeping/go-smux-multiplex"

	ntraversal "github.com/upperwal/go-libp2p-nat-traversal"
)

type arrayFlags []string

func (i *arrayFlags) String() string {
	return ""
}

func (i *arrayFlags) Set(v string) error {
	*i = append(*i, v)
	return nil
}

var bootNodeFlags arrayFlags

func main() {
	logging.SetLogLevel("nat-traversal", "DEBUG")
	/* logging.SetLogLevel("swarm2", "DEBUG") */
	logging.SetLogLevel("tcp-tpt", "DEBUG")
	logging.SetLogLevel("reuseport-transport", "DEBUG")

	port := flag.Int("p", 0, "port number")
	rp := flag.String("r", "", "remote peer id")
	flag.Var(&bootNodeFlags, "b", "bootnode multiaddr")
	flag.Parse()

	if len(bootNodeFlags) == 0 {
		fmt.Println("Set a bootnode multiaddr")
		os.Exit(1)
	}

	ctx := context.Background()

	sourceMultiAddr, _ := ma.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", *port))

	host, err := libp2p.New(
		ctx,
		libp2p.ListenAddrs(sourceMultiAddr),
		libp2p.Muxer("/mplex/6.7.0", mplex.DefaultTransport),
	)
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
	b.ConnectToServiceNodes(ctx, bootNodeFlags)

	/* ma, _ := ma.NewMultiaddr("/ip4/127.0.0.1/tcp/3000/p2p/QmVvYUj13isfoP4p9ppDZgboX9QwUDKkefP2nTGxVwfYBz")
	pi, _ := pstore.InfoFromP2pAddr(ma) */

	time.Sleep(6 * time.Second)
	if *rp != "" {

		p, err := peer.IDB58Decode(string(*rp))
		fmt.Println(err)
		fmt.Println("Trying connection to: ", p)

		cerr, err := b.ConnectThroughHolePunching(ctx, p)
		if err != nil {
			fmt.Println(err)
		}

		err = <-cerr
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println("Connected to: ", p, "using hole punching")
		}

		/* pi, err := d.FindPeer(ctx, p)
		if err != nil {
			fmt.Println(err)
		}
		if err := host.Connect(ctx, pi); err != nil {
			fmt.Println("Expecting: peers are behind non-full cone nat. Now trying hole punching")
			host.Network().(*swarm.Swarm).Backoff().Clear(pi.ID)
			host.Peerstore().ClearAddrs(pi.ID)

			cerr, err := b.ConnectThroughHolePunching(ctx, p)
			if err != nil {
				fmt.Println(err)
			}

			err = <-cerr
			if err != nil {
				fmt.Println(err)
			} else {
				fmt.Println("Connected to: ", p, "using hole punching")
			}
		} else {
			fmt.Println("Connected to: ", p, "without Hole Punching")
		} */

	}

	select {}
}
