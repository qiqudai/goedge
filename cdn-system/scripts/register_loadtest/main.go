// 本地注册接口压测工具，仅用于 127.0.0.1 自测环境。
//
// 用法:
//
//	go run . -url http://127.0.0.1:8080/api/login/register -c 100 -n 10000
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

type registerPayload struct {
	CountryCode      string `json:"country_code"`
	Account          string `json:"account"`
	Password         string `json:"password"`
	PasswordConfirm  string `json:"password_confirm"`
	InviteCode       string `json:"invite_code"`
	Channel          int    `json:"channel"`
	LanguageID       int    `json:"language_id"`
}

var userAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:133.0) Gecko/20100101 Firefox/133.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.2 Safari/605.1.15",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
}

type stats struct {
	ok       atomic.Int64
	fail     atomic.Int64
	latencyMu sync.Mutex
	latencies []time.Duration
}

func (s *stats) record(ok bool, d time.Duration) {
	if ok {
		s.ok.Add(1)
	} else {
		s.fail.Add(1)
	}
	s.latencyMu.Lock()
	s.latencies = append(s.latencies, d)
	s.latencyMu.Unlock()
}

func (s *stats) snapshot() (ok, fail int64, p50, p95, p99 time.Duration) {
	ok = s.ok.Load()
	fail = s.fail.Load()
	s.latencyMu.Lock()
	ds := append([]time.Duration(nil), s.latencies...)
	s.latencyMu.Unlock()
	if len(ds) == 0 {
		return ok, fail, 0, 0, 0
	}
	sort.Slice(ds, func(i, j int) bool { return ds[i] < ds[j] })
	p50 = ds[len(ds)*50/100]
	p95 = ds[len(ds)*95/100]
	p99 = ds[len(ds)*99/100]
	return
}

func randomAccount(rng *rand.Rand) string {
	// 8 位数字账号，与示例格式一致
	return fmt.Sprintf("%08d", rng.Intn(100_000_000))
}

func randomPassword(rng *rand.Rand) string {
	return fmt.Sprintf("%08d", rng.Intn(100_000_000))
}

func setBrowserHeaders(req *http.Request, rng *rand.Rand, origin string) {
	ua := userAgents[rng.Intn(len(userAgents))]
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("Origin", origin)
	req.Header.Set("Referer", origin+"/")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("sec-ch-ua", `"Google Chrome";v="131", "Chromium";v="131", "Not_A Brand";v="24"`)
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("sec-ch-ua-platform", `"Windows"`)
}

func main() {
	var (
		targetURL   = flag.String("url", "http://127.0.0.1/api/login/register", "注册接口地址")
		origin      = flag.String("origin", "http://127.0.0.1", "Origin / Referer 前缀，模拟浏览器")
		concurrency = flag.Int("c", 50, "并发 worker 数")
		total       = flag.Int("n", 1000, "总请求数，0 表示不限直到 Ctrl+C")
		timeout     = flag.Duration("timeout", 10*time.Second, "单次请求超时")
		inviteCode  = flag.String("invite", "idnoclui", "invite_code")
		channel     = flag.Int("channel", 3, "channel")
		languageID  = flag.Int("language", 3, "language_id")
		countryCode = flag.String("country", "+235", "country_code")
	)
	flag.Parse()

	if *concurrency <= 0 {
		fmt.Fprintln(os.Stderr, "concurrency must be > 0")
		os.Exit(1)
	}

	transport := &http.Transport{
		MaxIdleConns:        *concurrency * 2,
		MaxIdleConnsPerHost: *concurrency * 2,
		MaxConnsPerHost:     *concurrency * 2,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  false,
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   *timeout,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\n收到中断信号，停止中...")
		cancel()
	}()

	st := &stats{}
	var wg sync.WaitGroup
	sem := make(chan struct{}, *concurrency)

	fmt.Printf("目标: %s\n", *targetURL)
	fmt.Printf("并发: %d | 总量: ", *concurrency)
	if *total > 0 {
		fmt.Printf("%d\n", *total)
	} else {
		fmt.Println("不限 (Ctrl+C 停止)")
	}

	startAll := time.Now()
	var sent atomic.Int64

	for i := 0; i < *concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(workerID)*7919))

			for {
				if *total > 0 {
					n := sent.Add(1)
					if n > int64(*total) {
						return
					}
				} else {
					select {
					case <-ctx.Done():
						return
					default:
					}
				}

				select {
				case <-ctx.Done():
					return
				case sem <- struct{}{}:
				}

				account := randomAccount(rng)
				password := randomPassword(rng)
				body, err := json.Marshal(registerPayload{
					CountryCode:     *countryCode,
					Account:         account,
					Password:        password,
					PasswordConfirm: password,
					InviteCode:      *inviteCode,
					Channel:         *channel,
					LanguageID:      *languageID,
				})
				if err != nil {
					st.record(false, 0)
					<-sem
					continue
				}

				req, err := http.NewRequestWithContext(ctx, http.MethodPost, *targetURL, bytes.NewReader(body))
				if err != nil {
					st.record(false, 0)
					<-sem
					continue
				}
				setBrowserHeaders(req, rng, *origin)

				t0 := time.Now()
				resp, err := client.Do(req)
				latency := time.Since(t0)
				if err != nil {
					st.record(false, latency)
					<-sem
					continue
				}
				_, _ = io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
				st.record(resp.StatusCode >= 200 && resp.StatusCode < 300, latency)
				<-sem
			}
		}(i)
	}

	ticker := time.NewTicker(2 * time.Second)
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	for {
		select {
		case <-done:
			ticker.Stop()
			elapsed := time.Since(startAll)
			ok, fail, p50, p95, p99 := st.snapshot()
			totalDone := ok + fail
			rps := float64(totalDone) / elapsed.Seconds()
			fmt.Printf("\n完成 | 耗时: %s | 总计: %d | 成功: %d | 失败: %d | RPS: %.1f\n",
				elapsed.Round(time.Millisecond), totalDone, ok, fail, rps)
			fmt.Printf("延迟 P50: %s | P95: %s | P99: %s\n", p50, p95, p99)
			return
		case <-ticker.C:
			ok, fail, p50, p95, _ := st.snapshot()
			totalDone := ok + fail
			elapsed := time.Since(startAll)
			rps := float64(totalDone) / elapsed.Seconds()
			fmt.Printf("[进度] 已发: %d | 成功: %d | 失败: %d | RPS: %.1f | P50: %s | P95: %s\n",
				totalDone, ok, fail, rps, p50, p95)
		case <-ctx.Done():
			<-done
			return
		}
	}
}
