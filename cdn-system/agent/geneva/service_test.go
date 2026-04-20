package geneva

import (
	"net"
	"testing"

	"github.com/florianl/go-nfqueue"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

func TestProcessPacketOnlyTouchesEarlyTLSControlPackets(t *testing.T) {
	svc := New(Config{
		WindowSize:  512,
		PacketLimit: 2,
		Ports:       []int{443},
	})

	synAck := buildIPv4TCPPacket(t, packetSpec{
		srcIP:   "10.0.0.1",
		dstIP:   "10.0.0.2",
		srcPort: 443,
		dstPort: 55000,
		syn:     true,
		ack:     true,
		window:  65535,
	})
	modified, verdict := svc.processPacket(synAck)
	if verdict != nfqueue.NfAccept {
		t.Fatalf("unexpected verdict: %d", verdict)
	}
	if modified == nil {
		t.Fatal("expected SYN-ACK to be modified")
	}
	if got := tcpWindow(t, modified); got != 512 {
		t.Fatalf("unexpected window after SYN-ACK: %d", got)
	}

	pureAck := buildIPv4TCPPacket(t, packetSpec{
		srcIP:   "10.0.0.1",
		dstIP:   "10.0.0.2",
		srcPort: 443,
		dstPort: 55000,
		ack:     true,
		window:  65535,
	})
	modified, verdict = svc.processPacket(pureAck)
	if verdict != nfqueue.NfAccept {
		t.Fatalf("unexpected verdict: %d", verdict)
	}
	if modified == nil {
		t.Fatal("expected early pure ACK to be modified")
	}
	if got := tcpWindow(t, modified); got != 512 {
		t.Fatalf("unexpected window after pure ACK: %d", got)
	}

	// Packet limit reached, no more rewrites.
	modified, verdict = svc.processPacket(pureAck)
	if verdict != nfqueue.NfAccept {
		t.Fatalf("unexpected verdict after limit: %d", verdict)
	}
	if modified != nil {
		t.Fatal("expected no modification after packet limit")
	}
}

func TestProcessPacketStopsOnServerPayload(t *testing.T) {
	svc := New(Config{
		WindowSize:  512,
		PacketLimit: 4,
		Ports:       []int{443},
	})

	synAck := buildIPv4TCPPacket(t, packetSpec{
		srcIP:   "10.0.0.1",
		dstIP:   "10.0.0.2",
		srcPort: 443,
		dstPort: 55000,
		syn:     true,
		ack:     true,
		window:  65535,
	})
	if modified, _ := svc.processPacket(synAck); modified == nil {
		t.Fatal("expected SYN-ACK to be modified")
	}

	serverTLSData := buildIPv4TCPPacket(t, packetSpec{
		srcIP:   "10.0.0.1",
		dstIP:   "10.0.0.2",
		srcPort: 443,
		dstPort: 55000,
		ack:     true,
		psh:     true,
		window:  65535,
		payload: []byte{0x16, 0x03, 0x03, 0x00, 0x2a},
	})
	if modified, _ := svc.processPacket(serverTLSData); modified != nil {
		t.Fatal("expected server payload packet to pass through untouched")
	}

	pureAck := buildIPv4TCPPacket(t, packetSpec{
		srcIP:   "10.0.0.1",
		dstIP:   "10.0.0.2",
		srcPort: 443,
		dstPort: 55000,
		ack:     true,
		window:  65535,
	})
	if modified, _ := svc.processPacket(pureAck); modified != nil {
		t.Fatal("expected flow to stop being modified after server payload")
	}
}

func TestProcessPacketIgnoresNonTLSPorts(t *testing.T) {
	svc := New(Config{
		WindowSize:  512,
		PacketLimit: 4,
		Ports:       []int{443},
	})

	httpSynAck := buildIPv4TCPPacket(t, packetSpec{
		srcIP:   "10.0.0.1",
		dstIP:   "10.0.0.2",
		srcPort: 80,
		dstPort: 55000,
		syn:     true,
		ack:     true,
		window:  65535,
	})
	if modified, _ := svc.processPacket(httpSynAck); modified != nil {
		t.Fatal("expected non-443 traffic to be ignored")
	}
}

type packetSpec struct {
	srcIP   string
	dstIP   string
	srcPort uint16
	dstPort uint16
	syn     bool
	ack     bool
	psh     bool
	window  uint16
	payload []byte
}

func buildIPv4TCPPacket(t *testing.T, spec packetSpec) []byte {
	t.Helper()

	ip := &layers.IPv4{
		Version:  4,
		IHL:      5,
		TTL:      64,
		Protocol: layers.IPProtocolTCP,
		SrcIP:    net.ParseIP(spec.srcIP).To4(),
		DstIP:    net.ParseIP(spec.dstIP).To4(),
	}
	tcp := &layers.TCP{
		SrcPort: layers.TCPPort(spec.srcPort),
		DstPort: layers.TCPPort(spec.dstPort),
		SYN:     spec.syn,
		ACK:     spec.ack,
		PSH:     spec.psh,
		Window:  spec.window,
		Seq:     1,
		Ack:     1,
	}
	if err := tcp.SetNetworkLayerForChecksum(ip); err != nil {
		t.Fatalf("set checksum layer: %v", err)
	}

	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{ComputeChecksums: true, FixLengths: true}
	payload := gopacket.Payload(spec.payload)
	if err := gopacket.SerializeLayers(buf, opts, ip, tcp, payload); err != nil {
		t.Fatalf("serialize layers: %v", err)
	}
	return buf.Bytes()
}

func tcpWindow(t *testing.T, raw []byte) uint16 {
	t.Helper()
	packet := gopacket.NewPacket(raw, layers.LayerTypeIPv4, gopacket.DecodeOptions{Lazy: true, NoCopy: true})
	tcpLayer := packet.Layer(layers.LayerTypeTCP)
	if tcpLayer == nil {
		t.Fatal("missing TCP layer")
	}
	return tcpLayer.(*layers.TCP).Window
}
