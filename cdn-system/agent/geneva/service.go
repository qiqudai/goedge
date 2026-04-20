package geneva

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/florianl/go-nfqueue"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

// Config defines the parameters for the Geneva service.
//
// This implementation intentionally avoids global sysctl changes and MSS
// rewriting. It only nudges early outbound TLS control packets by shrinking the
// advertised TCP window for the first few packets on port 443.
type Config struct {
	QueueNum    int // NFQUEUE ID (default: 100)
	WindowSize  uint16
	PacketLimit int
	Ports       []int // Ports to intercept (default: [443])
	Debug       bool
}

// Service controls the Geneva logic.
type Service struct {
	config  Config
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	mu      sync.Mutex // Protects running state and flow state
	running bool
	flows   map[flowKey]*flowState
}

type flowKey struct {
	srcIP   string
	dstIP   string
	srcPort uint16
	dstPort uint16
}

type flowState struct {
	modifiedPackets int
}

// New creates a new Geneva service instance.
func New(cfg Config) *Service {
	if cfg.QueueNum == 0 {
		cfg.QueueNum = 100
	}
	if cfg.WindowSize == 0 {
		cfg.WindowSize = 512
	}
	if cfg.PacketLimit == 0 {
		cfg.PacketLimit = 6
	}
	if len(cfg.Ports) == 0 {
		cfg.Ports = []int{443}
	}
	return &Service{
		config: cfg,
		flows:  map[flowKey]*flowState{},
	}
}

// Start enables the packet interception rule and begins packet processing.
func (s *Service) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("geneva service is already running")
	}

	if err := s.setupNetwork(); err != nil {
		return fmt.Errorf("failed to setup geneva interception: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.running = true

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		if err := s.runNFQueue(ctx); err != nil {
			log.Printf("[GENEVA] Error in NFQueue loop: %v", err)
		}
	}()

	log.Printf("[GENEVA] Safe mode started. Queue=%d Window=%d PacketLimit=%d Ports=%v", s.config.QueueNum, s.config.WindowSize, s.config.PacketLimit, s.config.Ports)
	return nil
}

// Stop disables the service, stopping packet processing and removing rules.
func (s *Service) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	log.Println("[GENEVA] Stopping service...")

	if s.cancel != nil {
		s.cancel()
	}
	s.wg.Wait()

	if err := s.cleanupNetwork(); err != nil {
		return fmt.Errorf("failed to cleanup geneva interception: %v", err)
	}

	s.flows = map[flowKey]*flowState{}
	s.running = false
	log.Println("[GENEVA] Service stopped.")
	return nil
}

func (s *Service) runNFQueue(ctx context.Context) error {
	config := nfqueue.Config{
		NfQueue:      uint16(s.config.QueueNum),
		MaxPacketLen: 0xFFFF,
		MaxQueueLen:  1024,
		Copymode:     nfqueue.NfQnlCopyPacket,
		WriteTimeout: 100 * time.Millisecond,
	}

	nfq, err := nfqueue.Open(&config)
	if err != nil {
		return err
	}
	defer nfq.Close()

	fn := func(a nfqueue.Attribute) int {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[GENEVA-PANIC] Recovered packet: %v", r)
			}
		}()

		id := *a.PacketID
		payload := *a.Payload

		modifiedPayload, verdict := s.processPacket(payload)

		if verdict == nfqueue.NfDrop {
			nfq.SetVerdict(id, nfqueue.NfDrop)
		} else if modifiedPayload != nil {
			nfq.SetVerdictModPacket(id, nfqueue.NfAccept, modifiedPayload)
		} else {
			nfq.SetVerdict(id, nfqueue.NfAccept)
		}
		return 0
	}

	if err := nfq.RegisterWithErrorFunc(ctx, fn, func(e error) int {
		if s.config.Debug {
			log.Printf("[GENEVA-DEBUG] NFQueue callback error: %v", e)
		}
		return 0
	}); err != nil {
		return err
	}

	<-ctx.Done()
	return nil
}

func (s *Service) processPacket(data []byte) ([]byte, int) {
	packet := gopacket.NewPacket(data, layers.LayerTypeIPv4, gopacket.DecodeOptions{Lazy: true, NoCopy: true})

	ipLayer := packet.Layer(layers.LayerTypeIPv4)
	if ipLayer == nil {
		return nil, nfqueue.NfAccept
	}
	ip, _ := ipLayer.(*layers.IPv4)

	tcpLayer := packet.Layer(layers.LayerTypeTCP)
	if tcpLayer == nil {
		return nil, nfqueue.NfAccept
	}
	tcp, _ := tcpLayer.(*layers.TCP)

	if !s.shouldTargetPort(uint16(tcp.SrcPort)) {
		return nil, nfqueue.NfAccept
	}

	key := flowKey{
		srcIP:   ip.SrcIP.String(),
		dstIP:   ip.DstIP.String(),
		srcPort: uint16(tcp.SrcPort),
		dstPort: uint16(tcp.DstPort),
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if tcp.RST || tcp.FIN {
		delete(s.flows, key)
		return nil, nfqueue.NfAccept
	}

	// Once the server starts sending TLS/application data, stop interference.
	if len(tcp.Payload) > 0 {
		delete(s.flows, key)
		return nil, nfqueue.NfAccept
	}

	state, ok := s.flows[key]
	if tcp.SYN && tcp.ACK {
		state = &flowState{}
		s.flows[key] = state
	} else if !ok {
		return nil, nfqueue.NfAccept
	}

	if !shouldModifyAckWindow(tcp) {
		return nil, nfqueue.NfAccept
	}
	if state.modifiedPackets >= s.config.PacketLimit {
		delete(s.flows, key)
		return nil, nfqueue.NfAccept
	}

	if s.config.Debug {
		log.Printf("[GENEVA-DEBUG] Adjusting early TLS ACK window: %s:%d -> %s:%d packet=%d/%d", ip.SrcIP, tcp.SrcPort, ip.DstIP, tcp.DstPort, state.modifiedPackets+1, s.config.PacketLimit)
	}

	tcp.Window = s.config.WindowSize
	state.modifiedPackets++

	if err := tcp.SetNetworkLayerForChecksum(ip); err != nil {
		return nil, nfqueue.NfAccept
	}

	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{ComputeChecksums: true, FixLengths: true}
	if err := gopacket.SerializePacket(buffer, opts, packet); err != nil {
		return nil, nfqueue.NfAccept
	}

	return buffer.Bytes(), nfqueue.NfAccept
}

func (s *Service) shouldTargetPort(port uint16) bool {
	for _, allowed := range s.config.Ports {
		if uint16(allowed) == port {
			return true
		}
	}
	return false
}

func shouldModifyAckWindow(tcp *layers.TCP) bool {
	if tcp == nil {
		return false
	}
	if tcp.RST || tcp.FIN || tcp.PSH || tcp.URG || tcp.ECE || tcp.CWR {
		return false
	}
	if tcp.SYN && tcp.ACK {
		return true
	}
	return tcp.ACK && !tcp.SYN
}

func (s *Service) setupNetwork() error {
	return s.setupNFQueueRule()
}

func (s *Service) cleanupNetwork() error {
	return s.cleanupNFQueueRule()
}

func (s *Service) setupNFQueueRule() error {
	portsStr := intSliceToString(s.config.Ports)
	rule := []string{
		"-p", "tcp",
		"-m", "multiport", "--sports", portsStr,
		"-j", "NFQUEUE",
		"--queue-num", strconv.Itoa(s.config.QueueNum),
	}
	return ensureIptablesRule("filter", "OUTPUT", rule)
}

func (s *Service) cleanupNFQueueRule() error {
	portsStr := intSliceToString(s.config.Ports)
	rule := []string{
		"-p", "tcp",
		"-m", "multiport", "--sports", portsStr,
		"-j", "NFQUEUE",
		"--queue-num", strconv.Itoa(s.config.QueueNum),
	}
	return deleteIptablesRule("filter", "OUTPUT", rule)
}

func ensureIptablesRule(table string, chain string, rule []string) error {
	checkArgs := append([]string{"-t", table, "-C", chain}, rule...)
	if err := runCmd("iptables", checkArgs...); err == nil {
		return nil
	}

	insertArgs := append([]string{"-t", table, "-I", chain}, rule...)
	return runCmd("iptables", insertArgs...)
}

func deleteIptablesRule(table string, chain string, rule []string) error {
	checkArgs := append([]string{"-t", table, "-C", chain}, rule...)
	if err := runCmd("iptables", checkArgs...); err != nil {
		return nil
	}

	deleteArgs := append([]string{"-t", table, "-D", chain}, rule...)
	return runCmd("iptables", deleteArgs...)
}

func runCmd(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s %s: %v (out: %s)", name, strings.Join(args, " "), err, string(out))
	}
	return nil
}

func intSliceToString(ints []int) string {
	strs := make([]string, len(ints))
	for i, v := range ints {
		strs[i] = fmt.Sprintf("%d", v)
	}
	return strings.Join(strs, ",")
}
