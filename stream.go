package bootstrap

import (
	"bufio"

	ggio "github.com/gogo/protobuf/io"
	proto "github.com/golang/protobuf/proto"
	inet "github.com/libp2p/go-libp2p-net"
	protocol "github.com/upperwal/go-libp2p-bootstrap/protocol"
)

type streamWrapper struct {
	s  *inet.Stream
	bw *bufio.Writer
	r  *ggio.ReadCloser
	w  *ggio.WriteCloser
}

func (sw streamWrapper) writeMsg(msg proto.Message) error {
	w := *sw.w
	bw := sw.bw

	err := w.WriteMsg(msg)
	if err != nil {
		return err
	}

	return bw.Flush()
}

func (sw streamWrapper) readMsg(incoming chan PacketWPeer) error {
	r := *sw.r
	s := *sw.s

	protocolPacket := &protocol.Protocol{}

	for {
		err := r.ReadMsg(protocolPacket)
		if err != nil {
			return err
		}

		incoming <- PacketWPeer{
			peer:   s.Conn().RemotePeer(),
			packet: protocolPacket,
		}
	}
}
