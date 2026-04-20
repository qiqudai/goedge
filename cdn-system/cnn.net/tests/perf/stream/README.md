# Stream L4 Perf Scripts

This folder provides two baseline scripts required by `STREAM_L4_PROXY_AI_SPEC.md`.

## 1) 10k connection stability

Run:

```bash
./tests/perf/stream/stream_10k_stability.sh 127.0.0.1 10000 600 10000
```

Args:
- `host` default `127.0.0.1`
- `port` default `10000`
- `duration_seconds` default `600`
- `concurrency` default `10000`

## 2) Hot-reload survival

Run:

```bash
CONFIG_PUSH_CMD="./scripts/push_stream_config.sh" \
./tests/perf/stream/stream_hot_reload_survival.sh 127.0.0.1 10000 300 2000 30 5
```

Args:
- `host` default `127.0.0.1`
- `port` default `10000`
- `duration_seconds` default `300`
- `concurrency` default `2000`
- `reload_count` default `30`
- `reload_interval_seconds` default `5`

`CONFIG_PUSH_CMD` must point to a command that pushes an updated edge config.
