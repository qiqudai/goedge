package main

import (
	"archive/zip"
	"bytes"
	fsutil "cdn-common/io"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func initEnvironment() {
	rootDir := runtimeRoot()
	workDirMissing := !dirExists(rootDir) || ResetResources

	// Create work directories
	dirs := []string{
		rootDir,
		filepath.Join(rootDir, "conf"),
		filepath.Join(rootDir, "conf", "dynamic"),
		filepath.Join(rootDir, "logs"),
		filepath.Join(rootDir, "cache"),
		filepath.Join(rootDir, "cert"),
		filepath.Join(rootDir, "cert", "acme"),
		filepath.Join(rootDir, "data"),
	}
	for _, d := range dirs {
		os.MkdirAll(d, 0755)
	}

	if workDirMissing {
		restoreDir("assets/conf", filepath.Join(rootDir, "conf"))
		restoreDir("assets/lua", filepath.Join(rootDir, "lua"))
		restoreDir("assets/data", filepath.Join(rootDir, "data"))
		restoreDir("assets/cert", filepath.Join(rootDir, "cert"))
	}

	// 1. Unpack Binary / Runtime
	binName := "nginx"

	if runtime.GOOS == "windows" {
		binName = "nginx.exe"
		NginxBinPath = filepath.Join(rootDir, binName)
		if _, err := os.Stat(NginxBinPath); err != nil {
			restoreDir("assets/nginx-win", rootDir)
		}
	} else {
		// Set generic path, adjust if structure differs
		// Based on user provided zip: openresty/nginx/sbin/nginx
		NginxBinPath = filepath.Join(rootDir, "openresty", "nginx", "sbin", "nginx")
		if _, err := os.Stat(NginxBinPath); err != nil {
			// Linux: Unzip the embedded openresty.zip
			// Expect structure: openresty/nginx/sbin/nginx
			zipPath := "assets/openresty.zip"
			destDir := rootDir
			if err := unzipEmbedded(zipPath, destDir); err != nil {
				log.Printf("[Warn] Failed to unzip %s: %v (Ignore if not Linux or file missing)", zipPath, err)
			}
		}
		os.Chmod(NginxBinPath, 0755)
	}

	// 3. Patch nginx.conf
	confFile := filepath.Join(rootDir, "conf", "nginx.conf")
	if data, err := ioutil.ReadFile(confFile); err == nil {
		content := string(data)
		absCache, _ := filepath.Abs(filepath.Join(rootDir, "cache"))
		// Dynamically resolve data dir for ip2region
		absData, _ := filepath.Abs(filepath.Join(rootDir, "data", "ip2region.xdb"))
		content = strings.ReplaceAll(content, "/var/cache/nginx", filepath.ToSlash(absCache))
		// Patch ip2region path
		content = strings.ReplaceAll(content, "/opt/cdn-agent/data/ip2region.xdb", filepath.ToSlash(absData))
		ioutil.WriteFile(confFile, []byte(content), 0644)
	}
	ensureDynamicConf(filepath.Join(rootDir, "conf", "dynamic", "http.conf"))
	ensureDynamicConf(filepath.Join(rootDir, "conf", "dynamic", "http_global.conf"))
	ensureDynamicConf(filepath.Join(rootDir, "conf", "dynamic", "main.conf"))
	ensureDynamicConf(filepath.Join(rootDir, "conf", "dynamic", "events.conf"))
	ensureDynamicConf(filepath.Join(rootDir, "conf", "dynamic", "stream.conf"))
	ensureDynamicConf(filepath.Join(rootDir, "conf", "dynamic", "stream_global.conf"))

	// 4. Generate Fallback Certs
	generateFallbackCert(filepath.Join(rootDir, "cert"))

	// 5. Set Global Config Path
	if abs, err := filepath.Abs(NginxBinPath); err == nil {
		NginxBinPath = abs
	}
	if runtime.GOOS != "windows" {
		if err := ensureLinuxNginxWrapper(NginxBinPath); err != nil {
			log.Printf("[Warn] Ensure nginx wrapper failed: %v", err)
		}
	}

	confPath := filepath.Join(rootDir, "conf", "cdn_config.json")
	if abs, err := filepath.Abs(confPath); err == nil {
		CONFIG_PATH = abs
	} else {
		CONFIG_PATH = confPath
	}
	CONFIG_BAK = CONFIG_PATH + ".bak"

	loadPersistedConfigs()
	loadPersistedPackages()

	log.Printf("[Init] Environment Setup: Bin=%s, Config=%s", NginxBinPath, CONFIG_PATH)
}

func unzipEmbedded(zipPath, dest string) error {
	// 1. Read from Embed FS
	f, err := assetsFS.Open(zipPath)
	if err != nil {
		return err
	}
	defer f.Close()

	// 2. Access file stat to get size (needed for zip.NewReader)
	stat, err := f.Stat()
	if err != nil {
		return err
	}

	// 3. Read entire content into memory for ReaderAt
	// (embed.FS doesn't support ReaderAt directly efficiently without copy?)
	// Actually typical zip usage requires ReaderAt.
	// Since file maps to memory in embed, we can read it.
	// But zip.NewReader takes ReaderAt. bytes.NewReader impl ReaderAt.

	content, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}

	r, err := zip.NewReader(bytes.NewReader(content), stat.Size())
	if err != nil {
		return err
	}

	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)

		// Check for ZipSlip (Directory traversal)
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			continue
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}
	return nil
}

func restoreDir(embedPath, localPath string) {
	restoreDirWithOptions(embedPath, localPath, true)
}

func restoreDirIfMissing(embedPath, localPath string) {
	restoreDirWithOptions(embedPath, localPath, false)
}

func restoreDirWithOptions(embedPath, localPath string, overwrite bool) {
	entries, err := assetsFS.ReadDir(embedPath)
	if err != nil {
		// Embed dir might not exist if user didn't put anything, safe to ignore for optional dirs
		return
	}
	os.MkdirAll(localPath, 0755)
	for _, entry := range entries {
		fp := path.Join(embedPath, entry.Name())
		lp := filepath.Join(localPath, entry.Name())
		if entry.IsDir() {
			restoreDirWithOptions(fp, lp, overwrite)
		} else {
			restoreFile(fp, lp, overwrite)
		}
	}
}

func generateFallbackCert(certDir string) {
	pemPath := filepath.Join(certDir, "fallback.pem")
	keyPath := filepath.Join(certDir, "fallback.key")

	if _, err := os.Stat(pemPath); err == nil {
		return // already exists
	}

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Printf("[Warn] Failed to generate fallback cert key: %v", err)
		return
	}

	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "fallback"},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(10 * 365 * 24 * time.Hour),
	}
	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		log.Printf("[Warn] Failed to generate fallback cert: %v", err)
		return
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		log.Printf("[Warn] Failed to marshal fallback key: %v", err)
		return
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	if err := os.WriteFile(pemPath, certPEM, 0644); err != nil {
		log.Printf("[Warn] Failed to write fallback cert: %v", err)
		return
	}
	if err := os.WriteFile(keyPath, keyPEM, 0600); err != nil {
		log.Printf("[Warn] Failed to write fallback key: %v", err)
		return
	}
	log.Printf("[Info] Generated fallback self-signed cert at %s", pemPath)
}

func restoreFile(embedPath, localPath string, overwrite bool) {
	data, err := assetsFS.ReadFile(embedPath)
	if err != nil {
		// Log but don't crash, maybe user put binary manually
		log.Printf("[Warn] Embedded asset not found: %s (This is expected if you haven't replaced placeholders yet)", embedPath)
		return
	}
	os.MkdirAll(filepath.Dir(localPath), 0755)
	if !overwrite {
		if _, err := os.Stat(localPath); err == nil {
			return
		}
	}
	if err := ioutil.WriteFile(localPath, data, 0755); err != nil {
		log.Printf("[Error] Failed to extract %s: %v", localPath, err)
	}
}

func ensureDynamicConf(path string) {
	if _, err := os.Stat(path); err == nil {
		return
	}
	if err := fsutil.WriteFileAtomic(path, []byte(""), 0o644); err != nil {
		log.Printf("[Error] Ensure dynamic conf failed: %v", err)
	}
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func ensureLinuxNginxWrapper(binPath string) error {
	if runtime.GOOS == "windows" || strings.TrimSpace(binPath) == "" {
		return nil
	}
	realPath := binPath + ".real"
	hasReal := false
	if info, err := os.Stat(realPath); err == nil && !info.IsDir() {
		hasReal = true
	}

	switch {
	case hasReal:
		if fileLooksLikeELF(binPath) {
			_ = os.Remove(realPath)
			if err := os.Rename(binPath, realPath); err != nil {
				return err
			}
		}
		return writeNginxWrapper(
			binPath,
			realPath,
			filepath.Join(runtimeRoot(), "openresty", "luajit", "lib"),
			filepath.Join(runtimeRoot(), "openresty", "lualib"),
		)
	case fileLooksLikeELF(binPath):
		if err := os.Rename(binPath, realPath); err != nil {
			return err
		}
		return writeNginxWrapper(
			binPath,
			realPath,
			filepath.Join(runtimeRoot(), "openresty", "luajit", "lib"),
			filepath.Join(runtimeRoot(), "openresty", "lualib"),
		)
	default:
		if isGeneratedNginxWrapper(binPath) {
			return os.Chmod(binPath, 0o755)
		}
	}
	return nil
}

func fileLooksLikeELF(path string) bool {
	fp, err := os.Open(path)
	if err != nil {
		return false
	}
	defer fp.Close()
	header := make([]byte, 4)
	if _, err := io.ReadFull(fp, header); err != nil {
		return false
	}
	return bytes.Equal(header, []byte{0x7f, 'E', 'L', 'F'})
}

func isGeneratedNginxWrapper(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), "cdn-agent-nginx-wrapper")
}

func writeNginxWrapper(path, realPath, libDir, lualibDir string) error {
	quotedReal := "'" + strings.ReplaceAll(realPath, "'", "'\\''") + "'"
	quotedLib := "'" + strings.ReplaceAll(libDir, "'", "'\\''") + "'"
	quotedLua := "'" + strings.ReplaceAll(lualibDir, "'", "'\\''") + "'"
	content := "#!/usr/bin/env sh\n" +
		"# cdn-agent-nginx-wrapper\n" +
		"LIB_DIR=" + quotedLib + "\n" +
		"LUALIB_DIR=" + quotedLua + "\n" +
		"if [ -d \"$LIB_DIR\" ]; then\n" +
		"  case \":${LD_LIBRARY_PATH:-}:\" in\n" +
		"    *\":$LIB_DIR:\"*) ;;\n" +
		"    *) export LD_LIBRARY_PATH=\"$LIB_DIR${LD_LIBRARY_PATH:+:$LD_LIBRARY_PATH}\" ;;\n" +
		"  esac\n" +
		"fi\n" +
		"if [ -d \"$LUALIB_DIR\" ]; then\n" +
		"  export LUA_PATH=\"$LUALIB_DIR/?.lua;$LUALIB_DIR/?/init.lua;${LUA_PATH:-};;\"\n" +
		"  export LUA_CPATH=\"$LUALIB_DIR/?.so;${LUA_CPATH:-};;\"\n" +
		"fi\n" +
		"exec " + quotedReal + " \"$@\"\n"
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		return err
	}
	return os.Chmod(realPath, 0o755)
}
