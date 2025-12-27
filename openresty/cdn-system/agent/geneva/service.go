package geneva

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/florianl/go-nfqueue"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

// Config defines the parameters for the Geneva service
type Config struct {
	QueueNum   int // NFQUEUE ID (default: 100)
	WindowSize uint16
	Ports      []int // Ports to intercept (e.g., [80, 443])
	Debug      bool
}

// Service controls the Geneva logic
type Service struct {
	config Config
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.Mutex // Protects running state
	running bool
}

// New creates a new Geneva service instance
func New(cfg Config) *Service {
	if cfg.QueueNum == 0 {
		cfg.QueueNum = 100
	}
	if cfg.WindowSize == 0 {
		cfg.WindowSize = 4
	}
	if len(cfg.Ports) == 0 {
		cfg.Ports = []int{80, 443}
	}
	return &Service{
		config: cfg,
	}
}

// Start enables the firewall rules and begins packet processing.
// It is non-blocking (runs in background).
func (s *Service) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("geneva service is already running")
	}

	// 1. Setup IPTables
	if err := s.setupIptables(); err != nil {
		return fmt.Errorf("failed to setup iptables: %v", err)
	}

	// 2. Start NFQueue
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.running = true

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		if err := s.runNFQueue(ctx); err != nil {
			// If runtime error occurs, we log it.
			// Real-world code might want a channel to report errors back.
			log.Printf("[GENEVA] Error in NFQueue loop: %v", err)
		}
	}()

	log.Printf("[GENEVA] Service started. Queue: %d, Window: %d", s.config.QueueNum, s.config.WindowSize)
	return nil
}

// Stop disables the service, stopping packet processing and removing firewall rules.
func (s *Service) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil // Already stopped
	}

	log.Println("[GENEVA] Stopping service...")

	// 1. Cancel context to stop NFQueue loop
	if s.cancel != nil {
		s.cancel()
	}

	// 2. Wait for goroutine to finish
	s.wg.Wait()

	// 3. Cleanup IPTables
	if err := s.cleanupIptables(); err != nil {
		return fmt.Errorf("failed to cleanup iptables: %v", err)
	}

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
		} else {
			if modifiedPayload != nil {
				nfq.SetVerdictModPacket(id, nfqueue.NfAccept, modifiedPayload)
			} else {
				nfq.SetVerdict(id, nfqueue.NfAccept)
			}
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

	// Logic: Modifying Window Size for Control/Data Packets
	isTarget := (tcp.SYN && tcp.ACK) || (tcp.FIN && tcp.ACK) || (tcp.PSH && tcp.ACK) || (tcp.ACK && !tcp.SYN && !tcp.FIN && !tcp.RST)

	if isTarget {
		if s.config.Debug {
			log.Printf("[GENEVA-DEBUG] Modifying: %s:%d -> %s:%d", ip.SrcIP, tcp.SrcPort, ip.DstIP, tcp.DstPort)
		}

		tcp.Window = s.config.WindowSize

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

	return nil, nfqueue.NfAccept
}

// --- IPTables Utilities ---

const (
	iptCheck  = "iptables -C OUTPUT -p tcp -m multiport --sports %s -j NFQUEUE --queue-num %d"
	iptInsert = "iptables -I OUTPUT -p tcp -m multiport --sports %s -j NFQUEUE --queue-num %d"
	iptDelete = "iptables -D OUTPUT -p tcp -m multiport --sports %s -j NFQUEUE --queue-num %d"
)

func (s *Service) setupIptables() error {
	portsStr := intSliceToString(s.config.Ports)
	
	// Check first
	checkCmd := fmt.Sprintf(iptCheck, portsStr, s.config.QueueNum)
	if err := runCmd(checkCmd); err == nil {
		return nil // Already exists
	}

	insertCmd := fmt.Sprintf(iptInsert, portsStr, s.config.QueueNum)
	return runCmd(insertCmd)
}

func (s *Service) cleanupIptables() error {
	portsStr := intSliceToString(s.config.Ports)
	delCmd := fmt.Sprintf(iptDelete, portsStr, s.config.QueueNum)
	return runCmd(delCmd)
}

func runCmd(cmdStr string) error {
	parts := strings.Fields(cmdStr)
	cmd := exec.Command(parts[0], parts[1:]...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%v (out: %s)", err, string(out))
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
