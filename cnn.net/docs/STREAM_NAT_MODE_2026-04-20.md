# Stream NAT Mode (Optional) - 2026-04-20

## Summary

- `Cnn.Agent` now supports two L4 runtime modes:
  - `userspace` (default): existing `TcpListener/TcpClient` proxy path.
  - `nat` (optional): Linux kernel NAT path using `iptables` DNAT rules.
- Safety behavior:
  - default keeps current behavior (`userspace`) unchanged.
  - when `nat` is enabled and apply fails, runtime can automatically fall back to `userspace`.

## Configuration (`src/Cnn.Agent/appsettings.json`)

```json
"Stream": {
  "Mode": "userspace",
  "FallbackToUserspaceOnNatFailure": true,
  "IptablesBinary": "iptables",
  "CommandTimeoutMs": 3000
}
```

## Current NAT constraints

1. Linux only.
2. Requires executable `iptables` and enough privileges (`CAP_NET_ADMIN` / root).
3. Per stream listen key currently requires exactly one enabled target in NAT mode.
4. NAT target must be an IP address (not hostname).

## Code references

- runtime orchestrator: `src/Cnn.Agent/Stream/StreamRuntime.cs`
- NAT executor: `src/Cnn.Agent/Stream/KernelNatRuntime.cs`
- options model: `src/Cnn.Agent/Stream/StreamRuntimeOptions.cs`
- runtime debug endpoint: `GET /debug/stream/runtime`

## Verification

- Build verified:
  - `dotnet build src/Cnn.Agent/Cnn.Agent.csproj`
  - `dotnet build src/Cnn.Api/Cnn.Api.csproj`
- Script:
  - `./scripts/verify_stream_nat_mode.sh`
  - non-Linux / missing `iptables` / missing privilege will print `SKIP` and exit 0.
- Runtime NAT dataplane verification still requires privileged Linux environment.
