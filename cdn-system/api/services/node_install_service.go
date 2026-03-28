package services

import (
	"bytes"
	"cdn-api/models"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

type NodeInstallConfig struct {
	Host      string
	Port      int
	User      string
	AuthType  string
	Password  string
	Key       string
	WorkDir   string
	APIBase   string
	NodeID    int64
	NodeToken string
}

const nodeBaseDir = "/www/node"

func InstallNodeAgent(node *models.Node, apiBase string) error {
	if node == nil {
		return errors.New("node is nil")
	}
	cfg, err := buildInstallConfig(node, apiBase)
	if err != nil {
		return err
	}
	agentBinary := ResolveAgentBinaryPath()
	if agentBinary == "" {
		return errors.New("cdn-agent binary not found")
	}
	info, err := os.Stat(agentBinary)
	if err == nil {
		_ = UpdateInstallProgress(node.ID, "starting", 0, 0, info.Size(), "")
	} else {
		_ = UpdateInstallProgress(node.ID, "starting", 0, 0, 0, "")
	}
	log.Printf("[Install] ssh connect start node=%d host=%s port=%d user=%s", node.ID, cfg.Host, cfg.Port, cfg.User)
	client, err := dialSSH(cfg)
	if err != nil {
		_ = UpdateInstallProgress(node.ID, "failed", 0, 0, 0, err.Error())
		log.Printf("[Install] ssh connect failed node=%d err=%v", node.ID, err)
		return err
	}
	defer client.Close()
	log.Printf("[Install] ssh connected node=%d", node.ID)
	_ = UpdateInstallProgress(node.ID, "connected", 0, 0, 0, "")

	agentDir := path.Join(nodeBaseDir, "app")
	if err := runRemoteCommand(client, fmt.Sprintf("mkdir -p %s", shellQuote(agentDir))); err != nil {
		return fmt.Errorf("prepare work dir failed: %w", err)
	}

	agentPath := path.Join(agentDir, "cdn-agent")
	log.Printf("[Install] upload start node=%d src=%s dst=%s", node.ID, agentBinary, agentPath)
	if err := uploadFileWithProgress(client, node.ID, agentBinary, agentPath, 0o755, info); err != nil {
		return fmt.Errorf("upload agent failed: %w", err)
	}
	log.Printf("[Install] upload done node=%d dst=%s", node.ID, agentPath)

	agentJSON, err := buildAgentJSON(cfg)
	if err != nil {
		return err
	}
	configPath := path.Join(agentDir, "agent.json")
	log.Printf("[Install] upload config start node=%d dst=%s", node.ID, configPath)
	if err := uploadBytes(client, agentJSON, configPath, 0o644); err != nil {
		return fmt.Errorf("upload agent.json failed: %w", err)
	}
	log.Printf("[Install] upload config done node=%d dst=%s", node.ID, configPath)
	_ = UpdateInstallProgress(node.ID, "config", 100, 0, 0, "")

	isRoot := remoteIsRoot(client)
	hasSystemd := remoteHasCommand(client, "systemctl")
	logPath := path.Join(agentDir, "agent.log")
	startMode := "systemd"
	if isRoot && hasSystemd {
		servicePath := path.Join(agentDir, "cdn-agent.service")
		servicePayload := buildRemoteAgentService(agentDir, agentPath, configPath)
		if err := uploadBytes(client, servicePayload, servicePath, 0o644); err != nil {
			return fmt.Errorf("upload service file failed: %w", err)
		}
		if err := runRemoteCommand(client, fmt.Sprintf("mv %s /etc/systemd/system/cdn-agent.service", shellQuote(servicePath))); err != nil {
			return fmt.Errorf("install service failed: %w", err)
		}
		if err := runRemoteCommand(client, "systemctl daemon-reload"); err != nil {
			return fmt.Errorf("systemctl daemon-reload failed: %w", err)
		}
		if err := runRemoteCommand(client, "systemctl enable --now cdn-agent"); err != nil {
			return fmt.Errorf("systemctl enable/start failed: %w", err)
		}
		if err := runRemoteCommand(client, "systemctl is-active --quiet cdn-agent"); err != nil {
			return fmt.Errorf("cdn-agent service not active: %w", err)
		}
	} else {
		startCmd := fmt.Sprintf("nohup %s -config %s > %s 2>&1 &", shellQuote(agentPath), shellQuote(configPath), shellQuote(logPath))
		if err := runRemoteCommand(client, startCmd); err != nil {
			return fmt.Errorf("start agent failed: %w", err)
		}
		startMode = "nohup"
		log.Printf("[Install] systemd unavailable (root=%v systemd=%v), fallback to nohup", isRoot, hasSystemd)
	}

	if err := verifyRemoteInstall(client, agentDir, agentPath); err != nil {
		return fmt.Errorf("verify deploy failed: %w", err)
	}
	log.Printf("[Install] agent started node=%d mode=%s log=%s", node.ID, startMode, logPath)
	_ = UpdateInstallProgress(node.ID, "running", 100, 0, 0, "")
	return nil
}

func buildInstallConfig(node *models.Node, apiBase string) (*NodeInstallConfig, error) {
	host := strings.TrimSpace(node.SSHHost)
	if host == "" {
		host = strings.TrimSpace(node.IP)
	}
	if host == "" {
		return nil, errors.New("ssh host is required")
	}
	port := node.SSHPort
	if port <= 0 {
		port = 22
	}
	user := strings.TrimSpace(node.SSHUser)
	if user == "" {
		return nil, errors.New("ssh user is required")
	}
	authType := strings.TrimSpace(node.SSHAuthType)
	if authType == "" {
		authType = "password"
	}
	authType = strings.ToLower(authType)
	if authType != "password" && authType != "key" {
		return nil, errors.New("ssh auth type must be password or key")
	}
	if authType == "password" && strings.TrimSpace(node.SSHPassword) == "" {
		return nil, errors.New("ssh password is required")
	}
	if authType == "key" && strings.TrimSpace(node.SSHKey) == "" {
		return nil, errors.New("ssh key is required")
	}
	workDir := nodeBaseDir
	if strings.TrimSpace(apiBase) == "" {
		return nil, errors.New("api base url is required")
	}
	if node.ID == 0 {
		return nil, errors.New("node id is required")
	}
	if strings.TrimSpace(node.Token) == "" {
		return nil, errors.New("node token is required")
	}
	return &NodeInstallConfig{
		Host:      host,
		Port:      port,
		User:      user,
		AuthType:  authType,
		Password:  node.SSHPassword,
		Key:       node.SSHKey,
		WorkDir:   workDir,
		APIBase:   strings.TrimRight(apiBase, "/"),
		NodeID:    node.ID,
		NodeToken: node.Token,
	}, nil
}

func buildAgentJSON(cfg *NodeInstallConfig) ([]byte, error) {
	payload := map[string]interface{}{
		"api":                cfg.APIBase,
		"token":              cfg.NodeToken,
		"node_id":            fmt.Sprintf("%d", cfg.NodeID),
		"geneva_enable":      false,
		"geneva_window_size": 4,
		"reset_resources":    true,
		"bootstrap_sync":     true,
		"bootstrap_start":    true,
		"auto_disable_firewall": true,
	}
	return json.MarshalIndent(payload, "", "  ")
}

func normalizeWorkDir(dir string) string {
	trimmed := strings.TrimSpace(dir)
	if trimmed == "" {
		trimmed = "/www"
	}
	return strings.TrimRight(trimmed, "/")
}

func dialSSH(cfg *NodeInstallConfig) (*ssh.Client, error) {
	var authMethod ssh.AuthMethod
	switch cfg.AuthType {
	case "key":
		key := strings.TrimSpace(cfg.Key)
		signer, err := ssh.ParsePrivateKey([]byte(key))
		if err != nil {
			return nil, fmt.Errorf("parse ssh key failed: %w", err)
		}
		authMethod = ssh.PublicKeys(signer)
	default:
		authMethod = ssh.Password(cfg.Password)
	}
	clientConfig := &ssh.ClientConfig{
		User:            cfg.User,
		Auth:            []ssh.AuthMethod{authMethod},
		Timeout:         12 * time.Second,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	dialer := net.Dialer{Timeout: 12 * time.Second}
	conn, err := dialer.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	sshConn, chans, reqs, err := ssh.NewClientConn(conn, addr, clientConfig)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	return ssh.NewClient(sshConn, chans, reqs), nil
}

func runRemoteCommand(client *ssh.Client, cmd string) error {
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()
	return session.Run(cmd)
}

func runRemoteCommandOutput(client *ssh.Client, cmd string) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()
	var out bytes.Buffer
	session.Stdout = &out
	session.Stderr = &out
	err = session.Run(cmd)
	return strings.TrimSpace(out.String()), err
}

func remoteHasCommand(client *ssh.Client, name string) bool {
	err := runRemoteCommand(client, fmt.Sprintf("command -v %s >/dev/null 2>&1", shellQuote(name)))
	return err == nil
}

func remoteIsRoot(client *ssh.Client) bool {
	out, err := runRemoteCommandOutput(client, "id -u")
	return err == nil && strings.TrimSpace(out) == "0"
}

func buildRemoteAgentService(agentDir, agentPath, configPath string) []byte {
	libDir := path.Join(agentDir, "edge-node", "openresty", "luajit", "lib")
	content := fmt.Sprintf(`[Unit]
Description=CDN Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
WorkingDirectory=%s
Environment=LD_LIBRARY_PATH=%s
ExecStart=%s -config %s
Restart=always
RestartSec=3
LimitNOFILE=1048576

[Install]
WantedBy=multi-user.target
`, agentDir, libDir, agentPath, configPath)
	return []byte(content)
}

func verifyRemoteInstall(client *ssh.Client, agentDir, agentPath string) error {
	if _, err := runRemoteCommandOutput(client, fmt.Sprintf("%s -version", shellQuote(agentPath))); err != nil {
		return fmt.Errorf("agent version check failed: %w", err)
	}
	edgeRoot := path.Join(agentDir, "edge-node")
	nginxBin := path.Join(edgeRoot, "openresty", "nginx", "sbin", "nginx")
	luajitLib := path.Join(edgeRoot, "openresty", "luajit", "lib")
	nginxConf := path.Join(edgeRoot, "conf", "nginx.conf")
	testCmd := fmt.Sprintf("if [ -x %s ]; then LD_LIBRARY_PATH=%s %s -t -p %s -c %s; else echo 'nginx-not-ready'; fi",
		shellQuote(nginxBin),
		shellQuote(luajitLib),
		shellQuote(nginxBin),
		shellQuote(edgeRoot+"/"),
		shellQuote(nginxConf),
	)
	if out, err := runRemoteCommandOutput(client, testCmd); err != nil {
		return fmt.Errorf("nginx test failed: %w output=%s", err, out)
	}
	return nil
}

func uploadFile(client *ssh.Client, localPath, remotePath string, perm os.FileMode) error {
	file, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer file.Close()
	return uploadStream(client, file, remotePath, perm)
}

func uploadBytes(client *ssh.Client, data []byte, remotePath string, perm os.FileMode) error {
	return uploadStream(client, bytes.NewReader(data), remotePath, perm)
}

func uploadStream(client *ssh.Client, reader io.Reader, remotePath string, perm os.FileMode) error {
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()
	stdin, err := session.StdinPipe()
	if err != nil {
		return err
	}
	cmd := fmt.Sprintf("cat > %s && chmod %o %s", shellQuote(remotePath), perm, shellQuote(remotePath))
	if err := session.Start(cmd); err != nil {
		_ = stdin.Close()
		return err
	}
	_, copyErr := io.Copy(stdin, reader)
	_ = stdin.Close()
	waitErr := session.Wait()
	if copyErr != nil {
		return copyErr
	}
	return waitErr
}

func uploadFileWithProgress(client *ssh.Client, nodeID int64, localPath, remotePath string, perm os.FileMode, info os.FileInfo) error {
	file, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer file.Close()
	var total int64
	if info != nil {
		total = info.Size()
	}
	_ = UpdateInstallProgress(nodeID, "uploading", 0, 0, total, "")
	lastPercent := -1
	lastReport := time.Time{}
	reader := newProgressReader(file, remotePath, total, func(sent int64) {
		if total <= 0 {
			return
		}
		percent := int(float64(sent) / float64(total) * 100)
		now := time.Now()
		if percent != lastPercent && (lastReport.IsZero() || now.Sub(lastReport) >= time.Second || percent == 100) {
			_ = UpdateInstallProgress(nodeID, "uploading", percent, sent, total, "")
			lastPercent = percent
			lastReport = now
		}
	})
	return uploadStream(client, reader, remotePath, perm)
}

const uploadProgressStep = int64(5 * 1024 * 1024)

type progressReader struct {
	reader     io.Reader
	label      string
	total      int64
	totalSize  int64
	nextLog    int64
	lastReport time.Time
	onProgress func(sent int64)
}

func newProgressReader(reader io.Reader, label string, totalSize int64, onProgress func(sent int64)) io.Reader {
	return &progressReader{
		reader:     reader,
		label:      label,
		totalSize:  totalSize,
		nextLog:    uploadProgressStep,
		onProgress: onProgress,
	}
}

func (p *progressReader) Read(buf []byte) (int, error) {
	n, err := p.reader.Read(buf)
	if n > 0 {
		p.total += int64(n)
		if p.total >= p.nextLog {
			now := time.Now()
			if p.lastReport.IsZero() || now.Sub(p.lastReport) >= time.Second {
				log.Printf("[Install] upload progress %s bytes=%d", p.label, p.total)
				p.lastReport = now
			}
			p.nextLog = p.total + uploadProgressStep
		}
		if p.onProgress != nil {
			p.onProgress(p.total)
		}
	}
	return n, err
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
