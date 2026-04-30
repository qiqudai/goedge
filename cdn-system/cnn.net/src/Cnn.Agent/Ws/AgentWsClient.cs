using System.Collections.Concurrent;
using System.Formats.Tar;
using System.IO.Compression;
using System.Net;
using System.Net.Http.Headers;
using System.Net.Http.Json;
using System.Net.WebSockets;
using System.Net.Sockets;
using System.Security.Cryptography;
using System.Text;
using System.Text.Json;
using System.Text.Json.Serialization;
using Directory = System.IO.Directory;
using Cnn.Common.Contracts;
using Cnn.Common.Contracts.Agent;
using Cnn.Agent.Acme;
using Cnn.Agent.Cache;
using Cnn.Agent.Config;
using Cnn.Agent.Diagnostics;
using Cnn.Agent.Network;
using Cnn.Agent.Proxy;
using Cnn.Agent.Security;
using Cnn.Agent.Sync;
using Cnn.Agent.Stream;
using Cnn.Agent.Tasks;
using Certes;
using Certes.Acme;
using Certes.Acme.Resource;
using Microsoft.AspNetCore.Http;
using Microsoft.Extensions.Hosting;
using Microsoft.Extensions.Logging;
using Microsoft.Extensions.Options;

namespace Cnn.Agent.Ws;

public sealed class AgentWsClient : BackgroundService
{
    private readonly IConfiguration _configuration;
    private readonly ILogger<AgentWsClient> _logger;
    private readonly CacheRuntimeStore _cacheStore;
    private readonly CacheOptions _cacheOptions;
    private readonly EdgeConfigStore _edgeConfigStore;
    private readonly IEdgeProxyRuntime _proxyRuntime;
    private readonly IStreamRuntime _streamRuntime;
    private readonly IConfigVersionTracker _configVersionTracker;
    private readonly ISyncStateStore _syncStateStore;
    private readonly ITaskIdempotencyStore _taskIdempotencyStore;
    private readonly ITaskAckOutbox _taskAckOutbox;
    private readonly ITlsCertificateStore _tlsCertificateStore;
    private readonly ITlsRuntimePolicyStore _tlsRuntimePolicyStore;
    private readonly IDebugSwitchStore _debugSwitchStore;
    private readonly IDebugSessionService _debugSessionService;
    private readonly IDebugAuditLogger _debugAuditLogger;
    private readonly IManualDebugLogWriter _manualDebugLogWriter;
    private readonly AcmeTokenStore _tokenStore;
    private readonly AgentRuntimePaths _runtimePaths;
    private readonly AgentNodeState _nodeState;
    private readonly IHttpClientFactory _httpClientFactory;
    private readonly IPackageBandwidthLimiter _packageBandwidthLimiter;
    private readonly SemaphoreSlim _sendLock = new(1, 1);
    private readonly ConcurrentDictionary<string, TaskCompletionSource<L2NodesResponse>> _l2Waiters = new(StringComparer.OrdinalIgnoreCase);
    private readonly object _nodeSyncLock = new();
    private readonly List<NodeSyncAck> _pendingNodeSyncs = new();
    private readonly ConcurrentDictionary<long, AgentPackageConfigDto> _localPackages = new();
    private readonly ConcurrentDictionary<long, int> _deployCertAttempts = new();
    private readonly object _l2Lock = new();
    private readonly Dictionary<long, L2HealthState> _l2States = new();
    private Dictionary<string, bool> _l2Snapshot = new();
    private static readonly TimeSpan HeartbeatInterval = TimeSpan.FromSeconds(3);
    private static readonly TimeSpan LogShipInterval = TimeSpan.FromSeconds(5);
    private static readonly TimeSpan MetricsInterval = TimeSpan.FromSeconds(10);
    private static readonly TimeSpan L2CheckInterval = TimeSpan.FromSeconds(10);
    private static readonly JsonSerializerOptions JsonOptions = new(JsonSerializerDefaults.Web)
    {
        PropertyNameCaseInsensitive = true
    };

    public AgentWsClient(
        IConfiguration configuration,
        ILogger<AgentWsClient> logger,
        CacheRuntimeStore cacheStore,
        IOptions<CacheOptions> cacheOptions,
        EdgeConfigStore edgeConfigStore,
        IEdgeProxyRuntime proxyRuntime,
        IStreamRuntime streamRuntime,
        IConfigVersionTracker configVersionTracker,
        ISyncStateStore syncStateStore,
        ITaskIdempotencyStore taskIdempotencyStore,
        ITaskAckOutbox taskAckOutbox,
        ITlsCertificateStore tlsCertificateStore,
        ITlsRuntimePolicyStore tlsRuntimePolicyStore,
        IDebugSwitchStore debugSwitchStore,
        IDebugSessionService debugSessionService,
        IDebugAuditLogger debugAuditLogger,
        IManualDebugLogWriter manualDebugLogWriter,
        AcmeTokenStore tokenStore,
        AgentRuntimePaths runtimePaths,
        AgentNodeState nodeState,
        IHttpClientFactory httpClientFactory,
        IPackageBandwidthLimiter packageBandwidthLimiter)
    {
        _configuration = configuration;
        _logger = logger;
        _cacheStore = cacheStore;
        _cacheOptions = cacheOptions.Value ?? new CacheOptions();
        _edgeConfigStore = edgeConfigStore;
        _proxyRuntime = proxyRuntime;
        _streamRuntime = streamRuntime;
        _configVersionTracker = configVersionTracker;
        _syncStateStore = syncStateStore;
        _taskIdempotencyStore = taskIdempotencyStore;
        _taskAckOutbox = taskAckOutbox;
        _tlsCertificateStore = tlsCertificateStore;
        _tlsRuntimePolicyStore = tlsRuntimePolicyStore;
        _debugSwitchStore = debugSwitchStore;
        _debugSessionService = debugSessionService;
        _debugAuditLogger = debugAuditLogger;
        _manualDebugLogWriter = manualDebugLogWriter;
        _tokenStore = tokenStore;
        _runtimePaths = runtimePaths;
        _nodeState = nodeState;
        _httpClientFactory = httpClientFactory;
        _packageBandwidthLimiter = packageBandwidthLimiter;
    }

    protected override async Task ExecuteAsync(CancellationToken stoppingToken)
    {
        var baseUrl = _configuration["Api:BaseUrl"]?.TrimEnd('/') ?? "http://localhost:5000";
        var wsUrl = baseUrl.Replace("http://", "ws://").Replace("https://", "wss://") + "/ws/agent";
        var nodeId = _configuration["Node:Id"] ?? "node-1";
        var token = _configuration["Node:Token"] ?? "token";

        EnsureRuntimeDirectories();
        LoadPersistedPackages();
        TryLoadPersistedEdgeConfig();

        var agentVersion = ResolveAgentVersion();
        var capabilities = ResolveCapabilities();

        while (!stoppingToken.IsCancellationRequested)
        {
            using var socket = new ClientWebSocket();
            try
            {
                await socket.ConnectAsync(new Uri(wsUrl), stoppingToken);
                await SendAsync(socket, new
                {
                    kind = "agent_hello",
                    node_id = nodeId,
                    token,
                    agent_version = agentVersion,
                    capabilities
                }, stoppingToken);

                using var connectionCts = CancellationTokenSource.CreateLinkedTokenSource(stoppingToken);

                var heartbeatTask = RunHeartbeatAsync(socket, connectionCts.Token);
                var logTask = RunLogShipAsync(socket, nodeId, connectionCts.Token);
                var metricsTask = RunMetricsAsync(socket, nodeId, connectionCts.Token);
                var l2Task = RunL2MonitorAsync(socket, connectionCts.Token);

                await ReceiveLoopAsync(socket, connectionCts.Token);

                connectionCts.Cancel();
                await Task.WhenAll(heartbeatTask, logTask, metricsTask, l2Task);
            }
            catch (Exception ex)
            {
                _logger.LogWarning(ex, "WS connection failed, retrying...");
            }

            await Task.Delay(TimeSpan.FromSeconds(5), stoppingToken);
        }
    }

    private async Task<bool> SendAsync(ClientWebSocket socket, object payload, CancellationToken token)
    {
        var json = JsonSerializer.Serialize(payload, JsonOptions);
        return await SendRawJsonAsync(socket, json, token);
    }

    private async Task<bool> SendRawJsonAsync(ClientWebSocket socket, string json, CancellationToken token)
    {
        if (socket.State != WebSocketState.Open)
        {
            return false;
        }

        var bytes = Encoding.UTF8.GetBytes(json);
        await _sendLock.WaitAsync(token);
        try
        {
            await socket.SendAsync(bytes, WebSocketMessageType.Text, true, token);
            return true;
        }
        catch (Exception ex)
        {
            _logger.LogWarning(ex, "WS send failed");
            return false;
        }
        finally
        {
            _sendLock.Release();
        }
    }

    private void EnsureRuntimeDirectories()
    {
        Directory.CreateDirectory(_runtimePaths.RuntimeRoot);
        Directory.CreateDirectory(_runtimePaths.ConfDir);
        Directory.CreateDirectory(_runtimePaths.CacheDir);
        Directory.CreateDirectory(_runtimePaths.PackagesDir);
        Directory.CreateDirectory(_runtimePaths.CertDir);
        Directory.CreateDirectory(_runtimePaths.LogsDir);
        Directory.CreateDirectory(_runtimePaths.PluginsDir);
    }

    private void LoadPersistedPackages()
    {
        if (!Directory.Exists(_runtimePaths.PackagesDir))
        {
            return;
        }

        foreach (var path in Directory.EnumerateFiles(_runtimePaths.PackagesDir, "*.json", SearchOption.TopDirectoryOnly))
        {
            try
            {
                var name = Path.GetFileNameWithoutExtension(path);
                if (!long.TryParse(name, out var packageId) || packageId <= 0)
                {
                    continue;
                }

                var json = File.ReadAllText(path);
                var config = JsonSerializer.Deserialize<AgentPackageConfigDto>(json, JsonOptions);
                if (config != null)
                {
                    _localPackages[packageId] = config;
                }
            }
            catch
            {
                // ignore corrupted package file
            }
        }
    }

    private void TryLoadPersistedEdgeConfig()
    {
        try
        {
            if (!File.Exists(_runtimePaths.ConfigPath))
            {
                return;
            }

            var raw = File.ReadAllText(_runtimePaths.ConfigPath);
            if (string.IsNullOrWhiteSpace(raw))
            {
                return;
            }

            var config = JsonSerializer.Deserialize<EdgeConfigDto>(raw, JsonOptions);
            if (config == null)
            {
                return;
            }

            _edgeConfigStore.Update(config);
            _tlsCertificateStore.Reload(config);
            _tlsRuntimePolicyStore.Reload(config);
            var result = _proxyRuntime.TryApply(config, force: true);
            var streamResult = _streamRuntime.Apply(config);
            if (!result.Success)
            {
                _syncStateStore.MarkApplyError(config.Version, result.Error, Guid.NewGuid().ToString("N"));
                _logger.LogWarning("apply persisted config failed version={Version} error={Error}", result.Version, result.Error ?? "unknown");
            }
            if (!streamResult.Success)
            {
                var streamError = streamResult.Errors.Count > 0 ? streamResult.Errors[0] : "stream apply failed";
                _syncStateStore.MarkApplyError(config.Version, streamError, Guid.NewGuid().ToString("N"));
                _logger.LogWarning(
                    "apply persisted stream config failed version={Version} received={Received} planned={Planned} applied={Applied} skipped={Skipped} error={Error}",
                    config.Version,
                    streamResult.Received,
                    streamResult.Planned,
                    streamResult.Applied,
                    streamResult.Skipped,
                    streamError);
            }
            if (result.Success && streamResult.Success)
            {
                _configVersionTracker.MarkApplied(config.Version);
            }
        }
        catch (Exception ex)
        {
            _logger.LogWarning(ex, "load persisted edge config failed");
        }
    }

    private string ResolveAgentVersion()
    {
        var version = _configuration["Agent:Version"]?.Trim();
        if (!string.IsNullOrWhiteSpace(version))
        {
            return version;
        }

        return typeof(AgentWsClient).Assembly.GetName().Version?.ToString() ?? "unknown";
    }

    private static IReadOnlyList<string> ResolveCapabilities()
    {
        return new[]
        {
            "\u5957\u9910\u540c\u6b65",
            "ACL\u53d1\u5e03",
            "CC\u53d1\u5e03"
        };
    }

    private async Task ReceiveLoopAsync(ClientWebSocket socket, CancellationToken cancellationToken)
    {
        while (socket.State == WebSocketState.Open && !cancellationToken.IsCancellationRequested)
        {
            var json = await ReceiveMessageAsync(socket, cancellationToken);
            if (json == null)
            {
                break;
            }
            if (string.IsNullOrWhiteSpace(json))
            {
                continue;
            }

            await HandleMessageAsync(socket, json, cancellationToken);
        }
    }

    private static async Task<string?> ReceiveMessageAsync(ClientWebSocket socket, CancellationToken cancellationToken)
    {
        var buffer = new byte[8 * 1024];
        using var stream = new MemoryStream();

        while (true)
        {
            WebSocketReceiveResult result;
            try
            {
                result = await socket.ReceiveAsync(buffer, cancellationToken);
            }
            catch (OperationCanceledException)
            {
                return null;
            }

            if (result.MessageType == WebSocketMessageType.Close)
            {
                return null;
            }

            if (result.Count > 0)
            {
                stream.Write(buffer, 0, result.Count);
            }

            if (result.EndOfMessage)
            {
                break;
            }
        }

        if (stream.Length == 0)
        {
            return string.Empty;
        }

        return Encoding.UTF8.GetString(stream.ToArray());
    }

    private async Task RunHeartbeatAsync(ClientWebSocket socket, CancellationToken cancellationToken)
    {
        var timer = new PeriodicTimer(HeartbeatInterval);
        try
        {
            while (await timer.WaitForNextTickAsync(cancellationToken))
            {
                await SendAsync(socket, new
                {
                    kind = "heartbeat",
                    timestamp = DateTimeOffset.UtcNow.ToUnixTimeSeconds(),
                    status = "active"
                }, cancellationToken);

                await RetryPendingNodeSyncAsync(socket, cancellationToken);
            }
        }
        catch (OperationCanceledException)
        {
            // ignore
        }
        finally
        {
            timer.Dispose();
        }
    }

    private async Task RunLogShipAsync(ClientWebSocket socket, string nodeId, CancellationToken cancellationToken)
    {
        var timer = new PeriodicTimer(LogShipInterval);
        try
        {
            while (await timer.WaitForNextTickAsync(cancellationToken))
            {
                if (_debugSwitchStore.IsEnabled(DebugSwitchKeys.ShipAccessLogs))
                {
                    await ShipAccessLogsAsync(socket, nodeId, cancellationToken);
                }

                if (_debugSwitchStore.IsEnabled(DebugSwitchKeys.ShipStreamLogs))
                {
                    await ShipStreamLogsAsync(socket, nodeId, cancellationToken);
                }

                if (_debugSwitchStore.IsEnabled(DebugSwitchKeys.ShipSecurityLogs))
                {
                    await ShipSecurityLogsAsync(socket, nodeId, cancellationToken);
                    await ShipManualDebugLogsAsync(socket, nodeId, cancellationToken);
                }

                if (_debugSwitchStore.IsEnabled(DebugSwitchKeys.ShipSystemLogs))
                {
                    await ShipSystemLogsAsync(socket, nodeId, cancellationToken);
                }
            }
        }
        catch (OperationCanceledException)
        {
            // ignore
        }
        finally
        {
            timer.Dispose();
        }
    }

    private async Task RunMetricsAsync(ClientWebSocket socket, string nodeId, CancellationToken cancellationToken)
    {
        var timer = new PeriodicTimer(MetricsInterval);
        try
        {
            while (await timer.WaitForNextTickAsync(cancellationToken))
            {
                if (_debugSwitchStore.IsEnabled(DebugSwitchKeys.ShipMetrics))
                {
                    await ShipMetricsAsync(socket, nodeId, cancellationToken);
                }
            }
        }
        catch (OperationCanceledException)
        {
            // ignore
        }
        finally
        {
            timer.Dispose();
        }
    }

    private async Task RunL2MonitorAsync(ClientWebSocket socket, CancellationToken cancellationToken)
    {
        var timer = new PeriodicTimer(L2CheckInterval);
        try
        {
            while (await timer.WaitForNextTickAsync(cancellationToken))
            {
                await CheckL2NodesAsync(socket, cancellationToken);
            }
        }
        catch (OperationCanceledException)
        {
            // ignore
        }
        finally
        {
            timer.Dispose();
        }
    }

    private async Task HandleIssueCertAsync(ClientWebSocket socket, TaskDispatchMessage message, CancellationToken cancellationToken)
    {
        var payload = string.IsNullOrWhiteSpace(message.Task?.Payload)
            ? null
            : JsonSerializer.Deserialize<IssueCertTaskPayload>(message.Task.Payload, JsonOptions);
        var item = payload?.Items.FirstOrDefault();
        if (payload == null || item == null || item.Domains.Count == 0)
        {
            await SendTaskAckAsync(socket, message, "fail", null, null, "invalid payload", cancellationToken);
            return;
        }

        try
        {
            var issued = await IssueHttp01Async(payload, item, cancellationToken);
            var sent = await SendCertIssuedAsync(socket, item.CertId, message.Task!.TaskId, issued, false, 0, cancellationToken);
            if (!sent)
            {
                var ok = await SendIssuedCertAsync(item.CertId, message.Task.TaskId, issued, cancellationToken);
                if (!ok)
                {
                    await SendTaskAckAsync(socket, message, "fail", null, null, "cert upload failed", cancellationToken);
                    return;
                }
            }

            await SendTaskAckAsync(socket, message, "success", null, null, null, cancellationToken);
        }
        catch (Exception ex)
        {
            await SendTaskAckAsync(socket, message, "fail", null, null, ex.Message, cancellationToken);
        }
    }

    private async Task<IssuedCertResult> IssueHttp01Async(
        IssueCertTaskPayload payload,
        IssueCertItem item,
        CancellationToken cancellationToken)
    {
        var email = payload.Email?.Trim();
        if (string.IsNullOrWhiteSpace(email))
        {
            email = _configuration["Acme:Email"]?.Trim() ?? _configuration["App:AcmeEmail"]?.Trim();
        }
        if (string.IsNullOrWhiteSpace(email))
        {
            throw new InvalidOperationException("acme_email is required");
        }

        var ca = NormalizeCa(payload.Ca);
        var caDirUrl = string.IsNullOrWhiteSpace(payload.CaDirUrl) ? BuildCaDirUrl(ca) : payload.CaDirUrl!;

        var ctx = await CreateAcmeContextAsync(caDirUrl, email, cancellationToken);
        var order = await ctx.NewOrder(item.Domains.ToList());
        var authz = await order.Authorizations();

        foreach (var auth in authz)
        {
            var httpChallenge = await auth.Http();
            var token = httpChallenge.Token;
            var keyAuth = httpChallenge.KeyAuthz;
            _tokenStore.Put(token, keyAuth, TimeSpan.FromMinutes(15));
            try
            {
                await httpChallenge.Validate();
                await WaitAuthorizationValidAsync(auth, cancellationToken);
            }
            finally
            {
                _tokenStore.Delete(token);
            }
        }

        var key = KeyFactory.NewKey(KeyAlgorithm.RS256);
        var csr = new CsrInfo
        {
            CommonName = item.Domains[0]
        };
        var certChain = await order.Generate(csr, key, null, 1);
        return new IssuedCertResult(certChain.ToPem(), key.ToPem());
    }

    private async Task<AcmeContext> CreateAcmeContextAsync(string caDirUrl, string email, CancellationToken cancellationToken)
    {
        var dir = _configuration["Acme:AccountPath"];
        if (string.IsNullOrWhiteSpace(dir))
        {
            dir = Path.Combine(AppContext.BaseDirectory, "acme");
        }

        System.IO.Directory.CreateDirectory(dir);
        var keyFile = Path.Combine(dir, $"{Sanitize(email)}-{Sanitize(caDirUrl)}.pem");
        IKey key;
        if (File.Exists(keyFile))
        {
            var pem = await File.ReadAllTextAsync(keyFile, cancellationToken);
            key = KeyFactory.FromPem(pem);
        }
        else
        {
            key = KeyFactory.NewKey(KeyAlgorithm.ES256);
            await File.WriteAllTextAsync(keyFile, key.ToPem(), cancellationToken);
        }

        var ctx = new AcmeContext(new Uri(caDirUrl), key);
        await ctx.NewAccount(email, true);
        return ctx;
    }

    private static async Task WaitAuthorizationValidAsync(IAuthorizationContext auth, CancellationToken cancellationToken)
    {
        for (var i = 0; i < 60; i++)
        {
            var res = await auth.Resource();
            if (res.Status == AuthorizationStatus.Valid)
            {
                return;
            }
            if (res.Status == AuthorizationStatus.Invalid)
            {
                throw new InvalidOperationException(res.Challenges?.FirstOrDefault()?.Error?.Detail ?? "authorization invalid");
            }

            await Task.Delay(TimeSpan.FromSeconds(5), cancellationToken);
        }

        throw new TimeoutException("authorization timeout");
    }

    private async Task<bool> SendIssuedCertAsync(long certId, long taskId, IssuedCertResult issued, CancellationToken cancellationToken)
    {
        var baseUrl = _configuration["Api:BaseUrl"]?.TrimEnd('/') ?? "http://localhost:5000";
        var token = _configuration["Node:Token"] ?? _configuration["Agent:Token"] ?? string.Empty;
        var client = _httpClientFactory.CreateClient();
        if (!string.IsNullOrWhiteSpace(token))
        {
            client.DefaultRequestHeaders.Authorization = new AuthenticationHeaderValue("Bearer", token);
        }

        var request = new AgentIssuedCertRequest
        {
            CertId = certId,
            CertPem = issued.CertPem,
            KeyPem = issued.KeyPem,
            IssueTaskId = taskId
        };

        var response = await client.PostAsJsonAsync($"{baseUrl}/api/v1/agent/certs/issued", request, cancellationToken);
        if (!response.IsSuccessStatusCode)
        {
            return false;
        }

        var result = await response.Content.ReadFromJsonAsync<ApiResponse<object>>(cancellationToken: cancellationToken);
        return result != null && result.Code == ErrorCodes.Success;
    }

    private Task<bool> SendCertIssuedAsync(
        ClientWebSocket socket,
        long certId,
        long taskId,
        IssuedCertResult? issued,
        bool rateLimited,
        int rateCooldown,
        CancellationToken cancellationToken)
    {
        var payload = new Dictionary<string, object?>
        {
            ["kind"] = "cert_issued",
            ["cert_id"] = certId,
            ["issue_task_id"] = taskId
        };

        if (issued != null)
        {
            payload["cert"] = issued.CertPem;
            payload["key"] = issued.KeyPem;
        }

        if (rateLimited)
        {
            payload["rate_limited"] = true;
            if (rateCooldown > 0)
            {
                payload["rate_cooldown"] = rateCooldown;
            }
        }

        return SendAsync(socket, payload, cancellationToken);
    }

    private async Task SendTaskAckAsync(
        ClientWebSocket socket,
        TaskDispatchMessage message,
        string status,
        object? applied,
        string? ret,
        string? error,
        CancellationToken cancellationToken,
        TaskAckDiagnostics? diagnostics = null)
    {
        var payload = new Dictionary<string, object?>
        {
            ["kind"] = "task_ack",
            ["msg_id"] = message.MsgId,
            ["task_id"] = message.Task?.TaskId ?? 0,
            ["task_type"] = message.Task?.TaskType,
            ["status"] = status
        };

        if (applied != null)
        {
            payload["applied"] = applied;
        }
        if (!string.IsNullOrWhiteSpace(ret))
        {
            payload["ret"] = ret;
        }
        if (!string.IsNullOrWhiteSpace(error))
        {
            payload["error"] = error;
        }
        if (!string.IsNullOrWhiteSpace(diagnostics?.RetCode))
        {
            payload["ret_code"] = diagnostics.RetCode;
        }
        if (!string.IsNullOrWhiteSpace(diagnostics?.ErrorType))
        {
            payload["error_type"] = diagnostics.ErrorType;
        }
        if (diagnostics?.IsRetryable.HasValue == true)
        {
            payload["is_retryable"] = diagnostics.IsRetryable.Value;
        }
        if (diagnostics?.Attempt.HasValue == true)
        {
            payload["attempt"] = diagnostics.Attempt.Value;
        }
        if (diagnostics?.MaxAttempts.HasValue == true)
        {
            payload["max_attempts"] = diagnostics.MaxAttempts.Value;
        }
        if (diagnostics?.NextBackoffMs.HasValue == true)
        {
            payload["next_backoff_ms"] = diagnostics.NextBackoffMs.Value;
        }

        var taskId = message.Task?.TaskId ?? 0;
        var payloadJson = JsonSerializer.Serialize(payload, JsonOptions);
        var outboxId = _taskAckOutbox.Enqueue("task_ack", taskId > 0 ? taskId : null, payloadJson);
        var sent = await SendRawJsonAsync(socket, payloadJson, cancellationToken);
        if (sent)
        {
            _taskAckOutbox.MarkSent(outboxId);
        }
        else
        {
            _taskAckOutbox.MarkFailed(outboxId, "send_failed");
        }

        if (taskId > 0 && IsTerminalAckStatus(status))
        {
            _taskIdempotencyStore.SaveAck(taskId, status, applied, ret, error);
            _taskIdempotencyStore.MarkDone(taskId, ComputeTaskResultHash(status, ret, error));
        }
    }

    private Task SendTaskProgressAsync(
        ClientWebSocket socket,
        TaskDispatchMessage message,
        int percent,
        string? detail,
        CancellationToken cancellationToken)
    {
        var payload = new Dictionary<string, object?>
        {
            ["progress"] = percent
        };
        if (!string.IsNullOrWhiteSpace(detail))
        {
            payload["message"] = detail;
        }

        var ret = JsonSerializer.Serialize(payload, JsonOptions);
        return SendTaskAckAsync(socket, message, "progress", null, ret, null, cancellationToken);
    }

    private static bool IsTerminalAckStatus(string? status)
    {
        if (string.IsNullOrWhiteSpace(status))
        {
            return false;
        }

        return status.Trim().ToLowerInvariant() is "success" or "fail" or "ignored";
    }

    private static int ComputeRetryBackoffMs(int attempt)
    {
        var retryMinutes = attempt switch
        {
            <= 1 => 5,
            2 => 10,
            3 => 20,
            4 => 30,
            _ => 60
        };

        return retryMinutes * 60 * 1000;
    }

    private static string ComputeTaskResultHash(string? status, string? ret, string? error)
    {
        var input = $"{status}|{ret}|{error}";
        var hash = SHA256.HashData(Encoding.UTF8.GetBytes(input));
        return Convert.ToHexString(hash);
    }

    private static string ComputePayloadHash(string? payload)
    {
        var value = payload?.Trim() ?? string.Empty;
        var hash = SHA256.HashData(Encoding.UTF8.GetBytes(value));
        return Convert.ToHexString(hash);
    }

    private static string NormalizeCa(string? value)
    {
        if (string.IsNullOrWhiteSpace(value))
        {
            return "letsencrypt";
        }

        var normalized = value.Trim().ToLowerInvariant();
        return normalized switch
        {
            "letsencrypt" or "lets" or "let's encrypt" or "lets encrypt" => "letsencrypt",
            "zerossl" => "zerossl",
            "buypass" => "buypass",
            "google" => "google",
            _ => normalized
        };
    }

    private static string BuildCaDirUrl(string ca)
    {
        return ca switch
        {
            "letsencrypt" => "https://acme-v02.api.letsencrypt.org/directory",
            "zerossl" => "https://acme.zerossl.com/v2/DV90",
            "buypass" => "https://api.buypass.com/acme/directory",
            "google" => "https://dv.acme-v02.api.pki.goog/directory",
            _ => "https://acme-v02.api.letsencrypt.org/directory"
        };
    }

    private static string Sanitize(string input)
    {
        if (string.IsNullOrWhiteSpace(input))
        {
            return "default";
        }

        var chars = input.Select(ch => char.IsLetterOrDigit(ch) ? ch : '_').ToArray();
        return new string(chars);
    }

    private sealed record IssuedCertResult(string CertPem, string KeyPem);

    private sealed class TaskDispatchMessage
    {
        [JsonPropertyName("kind")]
        public string? Kind { get; set; }

        [JsonPropertyName("msg_id")]
        public string? MsgId { get; set; }

        [JsonPropertyName("task")]
        public TaskDispatchTask? Task { get; set; }
    }

    private sealed class TaskDispatchTask
    {
        [JsonPropertyName("task_id")]
        public long TaskId { get; set; }

        [JsonPropertyName("task_type")]
        public string? TaskType { get; set; }

        [JsonPropertyName("task_name")]
        public string? TaskName { get; set; }

        [JsonPropertyName("payload")]
        public string? Payload { get; set; }
    }

    private async Task HandleMessageAsync(ClientWebSocket socket, string json, CancellationToken cancellationToken)
    {
        if (string.IsNullOrWhiteSpace(json))
        {
            return;
        }

        try
        {
            using var doc = JsonDocument.Parse(json);
            if (!doc.RootElement.TryGetProperty("kind", out var kindElement))
            {
                return;
            }

            var kind = kindElement.GetString();
            if (string.IsNullOrWhiteSpace(kind))
            {
                return;
            }

            if (string.Equals(kind, "ack", StringComparison.OrdinalIgnoreCase))
            {
                return;
            }

            if (string.Equals(kind, "heartbeat_ack", StringComparison.OrdinalIgnoreCase))
            {
                await HandleHeartbeatAckAsync(socket, json, cancellationToken);
                return;
            }

            if (string.Equals(kind, "l2_nodes_response", StringComparison.OrdinalIgnoreCase))
            {
                HandleL2NodesResponse(json);
                return;
            }

            if (string.Equals(kind, "cache_config", StringComparison.OrdinalIgnoreCase))
            {
                if (doc.RootElement.TryGetProperty("data", out var dataElement))
                {
                    var config = dataElement.Deserialize<CacheSiteConfigDto>(JsonOptions);
                    if (config != null)
                    {
                        _cacheStore.UpsertSiteConfig(config);
                    }
                }
                return;
            }

            if (string.Equals(kind, "edge_config", StringComparison.OrdinalIgnoreCase))
            {
                if (doc.RootElement.TryGetProperty("data", out var dataElement))
                {
                    var result = await ApplyConfigPayloadAsync(dataElement.GetRawText(), false, cancellationToken);
                    if (!result.Success)
                    {
                        _logger.LogWarning(
                            "edge config push apply failed version={Version} old_version={OldVersion} error={Error}",
                            result.Version,
                            result.PreviousVersion,
                            result.Error ?? "unknown");
                    }
                }
                return;
            }

            if (string.Equals(kind, "task_dispatch", StringComparison.OrdinalIgnoreCase))
            {
                var message = JsonSerializer.Deserialize<TaskDispatchMessage>(json, JsonOptions);
                if (message?.Task == null)
                {
                    return;
                }

                await HandleTaskDispatchAsync(socket, message, cancellationToken);
            }
        }
        catch (Exception ex)
        {
            _logger.LogWarning(ex, "Failed to process agent message");
        }
    }

    private async Task HandleHeartbeatAckAsync(ClientWebSocket socket, string json, CancellationToken cancellationToken)
    {
        HeartbeatAckMessage? ack;
        try
        {
            ack = JsonSerializer.Deserialize<HeartbeatAckMessage>(json, JsonOptions);
        }
        catch
        {
            return;
        }

        var action = ack?.SyncAction?.Trim().ToLowerInvariant();
        if (string.IsNullOrWhiteSpace(action))
        {
            return;
        }

        var success = ApplyNodeSyncAction(action);
        await SendNodeSyncAsync(socket, action, success, cancellationToken);
    }

    private bool ApplyNodeSyncAction(string action)
    {
        switch (action)
        {
            case "enable":
                _nodeState.SetEnabled(true);
                return true;
            case "disable":
                _nodeState.SetEnabled(false);
                return true;
            default:
                return false;
        }
    }

    private async Task SendNodeSyncAsync(ClientWebSocket socket, string action, bool success, CancellationToken cancellationToken)
    {
        var payload = new
        {
            kind = "node_sync",
            action,
            success
        };

        var payloadJson = JsonSerializer.Serialize(payload, JsonOptions);
        var outboxId = _taskAckOutbox.Enqueue("node_sync", null, payloadJson);
        var sent = await SendRawJsonAsync(socket, payloadJson, cancellationToken);
        if (sent)
        {
            _taskAckOutbox.MarkSent(outboxId);
            return;
        }

        _taskAckOutbox.MarkFailed(outboxId, "send_failed");
    }

    private void RecordPendingNodeSync(string action, bool success, string error)
    {
        lock (_nodeSyncLock)
        {
            foreach (var item in _pendingNodeSyncs)
            {
                if (string.Equals(item.Action, action, StringComparison.OrdinalIgnoreCase) && item.Success == success)
                {
                    item.Attempts++;
                    item.LastError = error;
                    item.LastAt = DateTimeOffset.UtcNow;
                    return;
                }
            }

            _pendingNodeSyncs.Add(new NodeSyncAck
            {
                Action = action,
                Success = success,
                Attempts = 1,
                LastError = error,
                LastAt = DateTimeOffset.UtcNow
            });

            if (_pendingNodeSyncs.Count > 10)
            {
                _pendingNodeSyncs.RemoveAt(0);
            }
        }
    }

    private async Task RetryPendingNodeSyncAsync(ClientWebSocket socket, CancellationToken cancellationToken)
    {
        await RetryOutboxAsync(socket, cancellationToken);

        NodeSyncAck? pending = null;
        lock (_nodeSyncLock)
        {
            if (_pendingNodeSyncs.Count > 0)
            {
                pending = _pendingNodeSyncs[0];
            }
        }

        if (pending == null)
        {
            return;
        }

        var sent = await SendAsync(socket, new
        {
            kind = "node_sync",
            action = pending.Action,
            success = pending.Success
        }, cancellationToken);

        lock (_nodeSyncLock)
        {
            if (sent)
            {
                if (_pendingNodeSyncs.Count > 0)
                {
                    _pendingNodeSyncs.RemoveAt(0);
                }
            }
            else
            {
                pending.Attempts++;
                pending.LastError = "send_failed";
                pending.LastAt = DateTimeOffset.UtcNow;
            }
        }
    }

    private async Task RetryOutboxAsync(ClientWebSocket socket, CancellationToken cancellationToken)
    {
        var pending = _taskAckOutbox.ListPending(32);
        if (pending.Count == 0)
        {
            return;
        }

        foreach (var item in pending)
        {
            if (cancellationToken.IsCancellationRequested)
            {
                return;
            }

            var sent = await SendRawJsonAsync(socket, item.Payload, cancellationToken);
            if (sent)
            {
                _taskAckOutbox.MarkSent(item.Id);
                continue;
            }

            _taskAckOutbox.MarkFailed(item.Id, "send_failed");
            if (item.Attempts + 1 >= 20)
            {
                _logger.LogWarning(
                    "outbox retry exceeds threshold kind={Kind} id={Id} attempts={Attempts} last_error={LastError}",
                    item.Kind,
                    item.Id,
                    item.Attempts + 1,
                    item.LastError ?? string.Empty);
            }

            // connection likely unavailable now; retry remaining in next heartbeat
            return;
        }
    }

    private void HandleL2NodesResponse(string json)
    {
        L2NodesResponse? response;
        try
        {
            response = JsonSerializer.Deserialize<L2NodesResponse>(json, JsonOptions);
        }
        catch
        {
            return;
        }

        if (response == null || string.IsNullOrWhiteSpace(response.MsgId))
        {
            return;
        }

        if (_l2Waiters.TryRemove(response.MsgId, out var waiter))
        {
            waiter.TrySetResult(response);
        }
    }

    private async Task<IReadOnlyList<AgentL2NodeItem>> RequestL2NodesAsync(
        ClientWebSocket socket,
        TimeSpan timeout,
        CancellationToken cancellationToken)
    {
        var msgId = $"l2-{DateTimeOffset.UtcNow.ToUnixTimeMilliseconds()}-{Guid.NewGuid():N}";
        var tcs = new TaskCompletionSource<L2NodesResponse>(TaskCreationOptions.RunContinuationsAsynchronously);
        _l2Waiters[msgId] = tcs;

        var sent = await SendAsync(socket, new { kind = "l2_nodes_request", msg_id = msgId }, cancellationToken);
        if (!sent)
        {
            _l2Waiters.TryRemove(msgId, out _);
            return Array.Empty<AgentL2NodeItem>();
        }

        using var timeoutCts = new CancellationTokenSource(timeout);
        using var linked = CancellationTokenSource.CreateLinkedTokenSource(cancellationToken, timeoutCts.Token);

        try
        {
            var completed = await Task.WhenAny(tcs.Task, Task.Delay(Timeout.Infinite, linked.Token));
            if (completed == tcs.Task)
            {
                var response = await tcs.Task;
                return response.Nodes ?? new List<AgentL2NodeItem>();
            }
        }
        catch (OperationCanceledException)
        {
            // timeout or cancellation
        }
        finally
        {
            _l2Waiters.TryRemove(msgId, out _);
        }

        return Array.Empty<AgentL2NodeItem>();
    }

    private async Task CheckL2NodesAsync(ClientWebSocket socket, CancellationToken cancellationToken)
    {
        var nodes = await RequestL2NodesAsync(socket, TimeSpan.FromSeconds(5), cancellationToken);
        if (nodes.Count == 0)
        {
            var empty = new Dictionary<string, bool>();
            var changed = UpdateL2Snapshot(empty);
            if (changed)
            {
                WriteL2StatusSnapshot(empty);
            }
            return;
        }

        var onlineNow = new List<long>(nodes.Count);
        var snapshot = new Dictionary<string, bool>(StringComparer.OrdinalIgnoreCase);
        var seen = new HashSet<long>();
        foreach (var node in nodes)
        {
            if (node == null || node.Id <= 0)
            {
                continue;
            }

            seen.Add(node.Id);
            var alive = await IsL2AliveAsync(node, cancellationToken);
            bool isOnline;

            lock (_l2Lock)
            {
                if (!_l2States.TryGetValue(node.Id, out var state))
                {
                    state = new L2HealthState { Online = true };
                    _l2States[node.Id] = state;
                }

                if (alive)
                {
                    state.Fail = 0;
                    state.Success = Math.Min(state.Success + 1, 3);
                    if (!state.Online && state.Success >= 3)
                    {
                        state.Online = true;
                    }
                }
                else
                {
                    state.Success = 0;
                    state.Fail = Math.Min(state.Fail + 1, 3);
                    if (state.Online && state.Fail >= 3)
                    {
                        state.Online = false;
                    }
                }

                isOnline = state.Online;
            }

            snapshot[node.Id.ToString()] = isOnline;
            if (isOnline)
            {
                onlineNow.Add(node.Id);
            }
        }

        lock (_l2Lock)
        {
            var removed = _l2States.Keys.Where(id => !seen.Contains(id)).ToList();
            foreach (var id in removed)
            {
                _l2States.Remove(id);
            }
        }

        var snapshotChanged = UpdateL2Snapshot(snapshot);
        if (snapshotChanged)
        {
            WriteL2StatusSnapshot(snapshot);
        }

        if (onlineNow.Count > 0)
        {
            await SendAsync(socket, new { kind = "l2_heartbeat", nodes = onlineNow }, cancellationToken);
        }
    }

    private async Task<bool> IsL2AliveAsync(AgentL2NodeItem node, CancellationToken cancellationToken)
    {
        if (string.IsNullOrWhiteSpace(node.Ip))
        {
            return false;
        }

        var protocol = string.IsNullOrWhiteSpace(node.CheckProtocol) ? "tcp" : node.CheckProtocol.Trim().ToLowerInvariant();
        var port = node.CheckPort.GetValueOrDefault();
        if (port <= 0)
        {
            if (node.Port.HasValue && node.Port.Value > 0)
            {
                port = node.Port.Value;
            }
            else if (protocol == "https")
            {
                port = 443;
            }
            else
            {
                port = 80;
            }
        }

        var timeoutSeconds = node.CheckTimeout.GetValueOrDefault();
        if (timeoutSeconds <= 0)
        {
            timeoutSeconds = 5;
        }

        var timeout = TimeSpan.FromSeconds(timeoutSeconds);

        if (protocol is "http" or "https")
        {
            var path = string.IsNullOrWhiteSpace(node.CheckPath) ? "/" : node.CheckPath.Trim();
            if (!path.StartsWith("/", StringComparison.Ordinal))
            {
                path = "/" + path;
            }

            var target = $"{protocol}://{node.Ip}:{port}{path}";
            using var request = new HttpRequestMessage(HttpMethod.Get, target);
            if (!string.IsNullOrWhiteSpace(node.CheckHost))
            {
                request.Headers.Host = node.CheckHost.Trim();
            }

            using var cts = CancellationTokenSource.CreateLinkedTokenSource(cancellationToken);
            cts.CancelAfter(timeout);
            try
            {
                var client = BuildL2HttpClient(protocol == "https");
                using var response = await client.SendAsync(request, cts.Token);
                return (int)response.StatusCode >= 200 && (int)response.StatusCode < 400;
            }
            catch
            {
                return false;
            }
        }

        try
        {
            using var cts = CancellationTokenSource.CreateLinkedTokenSource(cancellationToken);
            cts.CancelAfter(timeout);
            using var tcp = new TcpClient();
            await tcp.ConnectAsync(node.Ip, port, cts.Token);
            return true;
        }
        catch
        {
            return false;
        }
    }

    private bool UpdateL2Snapshot(Dictionary<string, bool> next)
    {
        lock (_l2Lock)
        {
            if (_l2Snapshot.Count == next.Count)
            {
                var equal = true;
                foreach (var (key, value) in next)
                {
                    if (!_l2Snapshot.TryGetValue(key, out var existing) || existing != value)
                    {
                        equal = false;
                        break;
                    }
                }

                if (equal)
                {
                    return false;
                }
            }

            _l2Snapshot = new Dictionary<string, bool>(next, StringComparer.OrdinalIgnoreCase);
            return true;
        }
    }

    private void WriteL2StatusSnapshot(Dictionary<string, bool> snapshot)
    {
        try
        {
            Directory.CreateDirectory(_runtimePaths.ConfDir);
            var payload = new Dictionary<string, object?>
            {
                ["updated_at"] = DateTimeOffset.UtcNow.ToUnixTimeSeconds(),
                ["nodes"] = snapshot
            };
            var json = JsonSerializer.Serialize(payload, JsonOptions);
            File.WriteAllText(_runtimePaths.L2StatusPath, json);
        }
        catch
        {
            // ignore snapshot failures
        }
    }

    private HttpClient BuildL2HttpClient(bool https)
    {
        if (!https)
        {
            return _httpClientFactory.CreateClient();
        }

        return new HttpClient(new HttpClientHandler
        {
            ServerCertificateCustomValidationCallback = HttpClientHandler.DangerousAcceptAnyServerCertificateValidator
        })
        { Timeout = TimeSpan.FromSeconds(5) };
    }

    private async Task HandleTaskDispatchAsync(ClientWebSocket socket, TaskDispatchMessage message, CancellationToken cancellationToken)
    {
        var taskType = message.Task?.TaskType?.Trim();
        if (string.IsNullOrWhiteSpace(taskType))
        {
            await SendTaskAckAsync(socket, message, "ignored", null, null, null, cancellationToken);
            return;
        }

        var normalized = taskType.Trim();
        var lower = normalized.ToLowerInvariant();
        var taskId = message.Task?.TaskId ?? 0;
        if (taskId > 0)
        {
            var payloadHash = ComputePayloadHash(message.Task?.Payload);
            if (!_taskIdempotencyStore.TryBegin(taskId, normalized, payloadHash))
            {
                if (_taskIdempotencyStore.IsDone(taskId, out _))
                {
                    if (_taskIdempotencyStore.TryGetAck(taskId, out var ack) && ack != null)
                    {
                        await ReplayTaskAckAsync(socket, message, ack, cancellationToken);
                    }
                    else
                    {
                        await SendTaskAckAsync(socket, message, "success", null, null, null, cancellationToken);
                    }

                    return;
                }

                if (_taskIdempotencyStore.IsRunning(taskId))
                {
                    await SendTaskAckAsync(socket, message, "running", null, null, null, cancellationToken);
                    return;
                }

                await SendTaskAckAsync(socket, message, "ignored", null, null, null, cancellationToken);
                return;
            }
        }

        switch (lower)
        {
            case "issue_cert":
                await HandleIssueCertAsync(socket, message, cancellationToken);
                return;
            case "deploy_cert":
                await HandleDeployCertAsync(socket, message, cancellationToken);
                return;
            case "refresh_url":
                await HandleRefreshUrlAsync(socket, message, cancellationToken);
                return;
            case "refresh_dir":
                await HandleRefreshDirAsync(socket, message, cancellationToken);
                return;
            case "clear_cache":
                await HandleClearCacheAsync(socket, message, cancellationToken);
                return;
            case "preheat":
                await HandlePreheatAsync(socket, message, cancellationToken);
                return;
            case "config_sync":
                await HandleConfigSyncAsync(socket, message, cancellationToken);
                return;
            case "agent_upgrade":
                await HandleAgentUpgradeAsync(socket, message, cancellationToken);
                return;
            case "debug_switch":
            case "debug_log_switch":
                await HandleDebugSwitchAsync(socket, message, cancellationToken);
                return;
            case "manual_debug_log":
            case "debug_log_write":
                await HandleManualDebugLogAsync(socket, message, cancellationToken);
                return;
        }

        if (IsPackageSyncTask(normalized))
        {
            await HandlePackageSyncAsync(socket, message, cancellationToken);
            return;
        }

        await SendTaskAckAsync(socket, message, "ignored", null, null, null, cancellationToken);
    }

    private async Task ReplayTaskAckAsync(
        ClientWebSocket socket,
        TaskDispatchMessage message,
        TaskAckReplay replay,
        CancellationToken cancellationToken)
    {
        object? applied = null;
        if (!string.IsNullOrWhiteSpace(replay.AppliedJson))
        {
            try
            {
                using var doc = JsonDocument.Parse(replay.AppliedJson);
                applied = doc.RootElement.Clone();
            }
            catch
            {
                applied = replay.AppliedJson;
            }
        }

        await SendTaskAckAsync(
            socket,
            message,
            replay.Status,
            applied,
            replay.Ret,
            replay.Error,
            cancellationToken);
    }

    private async Task HandleRefreshUrlAsync(ClientWebSocket socket, TaskDispatchMessage message, CancellationToken cancellationToken)
    {
        var urls = SplitLines(message.Task?.Payload);
        var error = await PurgeUrlsAsync(urls);
        if (error != null)
        {
            await SendTaskAckAsync(socket, message, "fail", null, null, error, cancellationToken);
            return;
        }

        await SendTaskAckAsync(socket, message, "success", null, null, null, cancellationToken);
    }

    private async Task HandleDeployCertAsync(ClientWebSocket socket, TaskDispatchMessage message, CancellationToken cancellationToken)
    {
        var taskId = message.Task?.TaskId ?? 0;
        var attempt = taskId > 0
            ? _deployCertAttempts.AddOrUpdate(taskId, 1, static (_, current) => current + 1)
            : 1;
        const int maxAttempts = 3;

        if (string.IsNullOrWhiteSpace(message.Task?.Payload))
        {
            await SendTaskAckAsync(
                socket,
                message,
                "fail",
                null,
                null,
                "invalid payload",
                cancellationToken,
                new TaskAckDiagnostics
                {
                    RetCode = "INVALID_PAYLOAD",
                    ErrorType = "validation",
                    IsRetryable = false,
                    Attempt = attempt,
                    MaxAttempts = maxAttempts
                });
            if (taskId > 0)
            {
                _deployCertAttempts.TryRemove(taskId, out _);
            }
            return;
        }

        DeployCertTaskPayload? payload;
        try
        {
            payload = JsonSerializer.Deserialize<DeployCertTaskPayload>(message.Task.Payload, JsonOptions);
        }
        catch
        {
            payload = null;
        }

        if (payload == null || payload.CertId <= 0 ||
            string.IsNullOrWhiteSpace(payload.CertPem) ||
            string.IsNullOrWhiteSpace(payload.KeyPem) ||
            payload.Domains == null ||
            payload.Domains.Count == 0)
        {
            await SendTaskAckAsync(
                socket,
                message,
                "fail",
                null,
                null,
                "invalid payload",
                cancellationToken,
                new TaskAckDiagnostics
                {
                    RetCode = "INVALID_PAYLOAD",
                    ErrorType = "validation",
                    IsRetryable = false,
                    Attempt = attempt,
                    MaxAttempts = maxAttempts
                });
            if (taskId > 0)
            {
                _deployCertAttempts.TryRemove(taskId, out _);
            }
            return;
        }

        int applied;
        try
        {
            applied = ApplyDeployCertificateToRuntime(payload);
        }
        catch (Exception ex)
        {
            await SendTaskAckAsync(
                socket,
                message,
                "fail",
                null,
                null,
                ex.Message,
                cancellationToken,
                new TaskAckDiagnostics
                {
                    RetCode = "DEPLOY_RUNTIME_ERROR",
                    ErrorType = "runtime",
                    IsRetryable = true,
                    Attempt = attempt,
                    MaxAttempts = maxAttempts,
                    NextBackoffMs = attempt < maxAttempts ? ComputeRetryBackoffMs(attempt) : null
                });
            if (taskId > 0 && attempt >= maxAttempts)
            {
                _deployCertAttempts.TryRemove(taskId, out _);
            }
            return;
        }

        if (applied <= 0)
        {
            await SendTaskAckAsync(
                socket,
                message,
                "fail",
                null,
                null,
                "no matching domains on this node",
                cancellationToken,
                new TaskAckDiagnostics
                {
                    RetCode = "NO_MATCHING_DOMAINS",
                    ErrorType = "domain_mismatch",
                    IsRetryable = false,
                    Attempt = attempt,
                    MaxAttempts = maxAttempts
                });
            if (taskId > 0)
            {
                _deployCertAttempts.TryRemove(taskId, out _);
            }
            return;
        }

        var ret = JsonSerializer.Serialize(new
        {
            cert_id = payload.CertId,
            applied_domains = applied
        }, JsonOptions);
        await SendTaskAckAsync(
            socket,
            message,
            "success",
            null,
            ret,
            null,
            cancellationToken,
            new TaskAckDiagnostics
            {
                RetCode = "OK",
                Attempt = attempt,
                MaxAttempts = maxAttempts
            });
        if (taskId > 0)
        {
            _deployCertAttempts.TryRemove(taskId, out _);
        }
    }

    private async Task HandleRefreshDirAsync(ClientWebSocket socket, TaskDispatchMessage message, CancellationToken cancellationToken)
    {
        var urls = SplitLines(message.Task?.Payload);
        var error = await PurgeDirectoriesAsync(urls);
        if (error != null)
        {
            await SendTaskAckAsync(socket, message, "fail", null, null, error, cancellationToken);
            return;
        }

        await SendTaskAckAsync(socket, message, "success", null, null, null, cancellationToken);
    }

    private async Task HandleClearCacheAsync(ClientWebSocket socket, TaskDispatchMessage message, CancellationToken cancellationToken)
    {
        var error = ClearCache(message.Task?.Payload);
        if (error != null)
        {
            await SendTaskAckAsync(socket, message, "fail", null, null, error, cancellationToken);
            return;
        }

        await SendTaskAckAsync(socket, message, "success", null, null, null, cancellationToken);
    }

    private async Task HandlePreheatAsync(ClientWebSocket socket, TaskDispatchMessage message, CancellationToken cancellationToken)
    {
        var urls = SplitLines(message.Task?.Payload);
        var error = await PreheatUrlsAsync(urls, cancellationToken);
        if (error != null)
        {
            await SendTaskAckAsync(socket, message, "fail", null, null, error, cancellationToken);
            return;
        }

        await SendTaskAckAsync(socket, message, "success", null, null, null, cancellationToken);
    }

    private async Task HandleConfigSyncAsync(ClientWebSocket socket, TaskDispatchMessage message, CancellationToken cancellationToken)
    {
        var payload = message.Task?.Payload ?? string.Empty;
        try
        {
            var result = await ApplyConfigPayloadAsync(payload, true, cancellationToken);
            if (!result.Success)
            {
                await SendTaskAckAsync(
                    socket,
                    message,
                    "fail",
                    result.Applied,
                    result.Ret,
                    string.IsNullOrWhiteSpace(result.Error) ? "config apply failed" : result.Error,
                    cancellationToken);
                return;
            }

            await SendTaskAckAsync(socket, message, "success", result.Applied, result.Ret, null, cancellationToken);
        }
        catch (Exception ex)
        {
            await SendTaskAckAsync(socket, message, "fail", null, null, ex.Message, cancellationToken);
        }
    }

    private async Task HandlePackageSyncAsync(ClientWebSocket socket, TaskDispatchMessage message, CancellationToken cancellationToken)
    {
        if (string.IsNullOrWhiteSpace(message.Task?.Payload))
        {
            await SendTaskAckAsync(socket, message, "fail", null, null, "invalid payload", cancellationToken);
            return;
        }

        AgentPackagePayloadDto? payload;
        try
        {
            payload = JsonSerializer.Deserialize<AgentPackagePayloadDto>(message.Task.Payload, JsonOptions);
        }
        catch
        {
            payload = null;
        }

        if (payload == null)
        {
            await SendTaskAckAsync(socket, message, "fail", null, null, "invalid payload", cancellationToken);
            return;
        }

        var applied = await ApplyPackageSyncAsync(payload, cancellationToken);
        await SendTaskAckAsync(socket, message, "success", applied, null, null, cancellationToken);
    }

    private async Task HandleAgentUpgradeAsync(ClientWebSocket socket, TaskDispatchMessage message, CancellationToken cancellationToken)
    {
        var reporter = BuildProgressReporter(socket, message, cancellationToken);
        try
        {
            var result = await UpgradeAgentPackageAsync(message.Task?.Payload ?? string.Empty, reporter, cancellationToken);
            await SendTaskAckAsync(socket, message, "success", null, result, null, cancellationToken);
        }
        catch (Exception ex)
        {
            await SendTaskAckAsync(socket, message, "fail", null, null, ex.Message, cancellationToken);
        }
    }

    private async Task HandleDebugSwitchAsync(ClientWebSocket socket, TaskDispatchMessage message, CancellationToken cancellationToken)
    {
        var payload = message.Task?.Payload;
        var sessionOptions = ParseDebugSessionOptions(payload);
        var updates = ParseDebugSwitchUpdates(payload);
        var ttlSeconds = ParseDebugSwitchTtlSeconds(payload);
        if (updates.Count == 0 && sessionOptions == null && !ttlSeconds.HasValue)
        {
            await SendTaskAckAsync(socket, message, "fail", null, null, "invalid payload", cancellationToken);
            return;
        }

        var actor = $"task:{message.Task?.TaskId ?? 0}";
        var result = _debugSwitchStore.Apply(updates, actor, "ws task", ttlSeconds);
        if (sessionOptions != null)
        {
            var ttl = ttlSeconds.HasValue && ttlSeconds.Value > 0
                ? TimeSpan.FromSeconds(ttlSeconds.Value)
                : TimeSpan.Zero;
            _debugSessionService.Update(sessionOptions, ttl);
        }

        _debugAuditLogger.WriteSwitchUpdate(
            actor,
            "ws task",
            ttlSeconds,
            sessionOptions ?? DebugOptions.Disabled,
            result.Current);

        _manualDebugLogWriter.Write(
            "debug_switch",
            "debug switch updated",
            new
            {
                updated = result.Updated,
                current = result.Current,
                expires_at = result.ExpiresAt,
                session = sessionOptions == null ? null : new
                {
                    enabled = sessionOptions.Enabled,
                    modules = sessionOptions.Modules,
                    allow_header_token = sessionOptions.AllowHeaderToken,
                    allow_query_flag = sessionOptions.AllowQueryFlag,
                    sample_rate = sessionOptions.SampleRate,
                    max_events_per_sec = sessionOptions.MaxEventsPerSec
                }
            },
            actor);

        var ret = JsonSerializer.Serialize(new
        {
            applied = result.AppliedCount,
            updated = result.Updated,
            current = result.Current,
            expires_at = result.ExpiresAt
        }, JsonOptions);

        await SendTaskAckAsync(socket, message, "success", null, ret, null, cancellationToken);
    }

    private async Task HandleManualDebugLogAsync(ClientWebSocket socket, TaskDispatchMessage message, CancellationToken cancellationToken)
    {
        var payload = message.Task?.Payload;
        if (string.IsNullOrWhiteSpace(payload))
        {
            await SendTaskAckAsync(socket, message, "fail", null, null, "invalid payload", cancellationToken);
            return;
        }

        var actor = $"task:{message.Task?.TaskId ?? 0}";
        try
        {
            using var doc = JsonDocument.Parse(payload);
            var root = doc.RootElement;

            var category = root.TryGetProperty("category", out var categoryElement)
                ? categoryElement.GetString() ?? "manual"
                : "manual";

            var logMessage = root.TryGetProperty("message", out var messageElement)
                ? messageElement.GetString()
                : null;

            if (string.IsNullOrWhiteSpace(logMessage))
            {
                await SendTaskAckAsync(socket, message, "fail", null, null, "message is required", cancellationToken);
                return;
            }

            object? data = null;
            if (root.TryGetProperty("data", out var dataElement))
            {
                data = dataElement.Clone();
            }

            _manualDebugLogWriter.Write(category, logMessage, data, actor);
            await SendTaskAckAsync(socket, message, "success", null, "ok", null, cancellationToken);
        }
        catch (Exception ex)
        {
            await SendTaskAckAsync(socket, message, "fail", null, null, ex.Message, cancellationToken);
        }
    }

    private static Dictionary<string, bool> ParseDebugSwitchUpdates(string? payload)
    {
        var updates = new Dictionary<string, bool>(StringComparer.OrdinalIgnoreCase);
        if (string.IsNullOrWhiteSpace(payload))
        {
            return updates;
        }

        try
        {
            using var doc = JsonDocument.Parse(payload);
            var root = doc.RootElement;

            if (root.ValueKind != JsonValueKind.Object)
            {
                return updates;
            }

            if (root.TryGetProperty("switches", out var switchesElement) && switchesElement.ValueKind == JsonValueKind.Object)
            {
                foreach (var property in switchesElement.EnumerateObject())
                {
                    if (TryReadBool(property.Value, out var enabled))
                    {
                        updates[property.Name] = enabled;
                    }
                }

                return updates;
            }

            if (root.TryGetProperty("key", out var keyElement) &&
                root.TryGetProperty("enabled", out var enabledElement) &&
                TryReadBool(enabledElement, out var enabledValue))
            {
                var key = keyElement.GetString();
                if (!string.IsNullOrWhiteSpace(key))
                {
                    updates[key.Trim()] = enabledValue;
                }

                return updates;
            }

            foreach (var property in root.EnumerateObject())
            {
                if (TryReadBool(property.Value, out var enabled))
                {
                    updates[property.Name] = enabled;
                }
            }
        }
        catch
        {
            // ignore parse errors and return empty updates
        }

        return updates;
    }

    private static bool TryReadBool(JsonElement element, out bool value)
    {
        switch (element.ValueKind)
        {
            case JsonValueKind.True:
                value = true;
                return true;
            case JsonValueKind.False:
                value = false;
                return true;
            case JsonValueKind.Number:
                if (element.TryGetInt32(out var intValue))
                {
                    value = intValue != 0;
                    return true;
                }
                break;
            case JsonValueKind.String:
                var text = element.GetString();
                if (bool.TryParse(text, out var boolValue))
                {
                    value = boolValue;
                    return true;
                }

                if (int.TryParse(text, out var numberValue))
                {
                    value = numberValue != 0;
                    return true;
                }
                break;
        }

        value = false;
        return false;
    }

    private static int? ParseDebugSwitchTtlSeconds(string? payload)
    {
        if (string.IsNullOrWhiteSpace(payload))
        {
            return null;
        }

        try
        {
            using var doc = JsonDocument.Parse(payload);
            var root = doc.RootElement;
            if (root.ValueKind != JsonValueKind.Object)
            {
                return null;
            }

            if (!root.TryGetProperty("ttl_seconds", out var ttlElement))
            {
                return null;
            }

            if (ttlElement.ValueKind == JsonValueKind.Number && ttlElement.TryGetInt32(out var intValue))
            {
                return intValue;
            }

            if (ttlElement.ValueKind == JsonValueKind.String &&
                int.TryParse(ttlElement.GetString(), out var parsed))
            {
                return parsed;
            }
        }
        catch
        {
            // ignore parse errors
        }

        return null;
    }

    private static DebugOptions? ParseDebugSessionOptions(string? payload)
    {
        if (string.IsNullOrWhiteSpace(payload))
        {
            return null;
        }

        try
        {
            using var doc = JsonDocument.Parse(payload);
            var root = doc.RootElement;
            if (root.ValueKind != JsonValueKind.Object)
            {
                return null;
            }

            var options = new DebugOptions();
            var hasValue = false;

            ReadOptions(root, options, ref hasValue);
            if (root.TryGetProperty("request_debug", out var requestDebugElement) && requestDebugElement.ValueKind == JsonValueKind.Object)
            {
                ReadOptions(requestDebugElement, options, ref hasValue);
            }

            return hasValue ? options : null;
        }
        catch
        {
            return null;
        }
    }

    private static void ReadOptions(JsonElement source, DebugOptions options, ref bool hasValue)
    {
        if (source.TryGetProperty("debug_enabled", out var enabledElement) && TryReadBool(enabledElement, out var enabled))
        {
            options.Enabled = enabled;
            hasValue = true;
        }
        else if (source.TryGetProperty("enabled", out var enabled2) && TryReadBool(enabled2, out enabled))
        {
            options.Enabled = enabled;
            hasValue = true;
        }

        if (source.TryGetProperty("internal_ip_only", out var internalElement) && TryReadBool(internalElement, out var internalOnly))
        {
            options.InternalIpOnly = internalOnly;
            hasValue = true;
        }

        if (source.TryGetProperty("allow_header_token", out var allowHeaderElement) && TryReadBool(allowHeaderElement, out var allowHeader))
        {
            options.AllowHeaderToken = allowHeader;
            hasValue = true;
        }

        if (source.TryGetProperty("allow_query_flag", out var allowQueryElement) && TryReadBool(allowQueryElement, out var allowQuery))
        {
            options.AllowQueryFlag = allowQuery;
            hasValue = true;
        }

        if (source.TryGetProperty("sample_rate", out var sampleElement) && TryReadDouble(sampleElement, out var sample))
        {
            options.SampleRate = sample;
            hasValue = true;
        }

        if (source.TryGetProperty("max_events_per_sec", out var maxElement) && TryReadInt(maxElement, out var maxEvents))
        {
            options.MaxEventsPerSec = maxEvents;
            hasValue = true;
        }

        if (source.TryGetProperty("debug_token", out var tokenElement) && tokenElement.ValueKind == JsonValueKind.String)
        {
            options.Token = tokenElement.GetString();
            hasValue = true;
        }
        else if (source.TryGetProperty("token", out var tokenElement2) && tokenElement2.ValueKind == JsonValueKind.String)
        {
            options.Token = tokenElement2.GetString();
            hasValue = true;
        }

        if (source.TryGetProperty("modules", out var modulesElement) && modulesElement.ValueKind == JsonValueKind.Object)
        {
            foreach (var property in modulesElement.EnumerateObject())
            {
                if (TryReadBool(property.Value, out var moduleEnabled))
                {
                    options.Modules[property.Name] = moduleEnabled;
                    hasValue = true;
                }
            }
        }
    }

    private static bool TryReadInt(JsonElement element, out int value)
    {
        value = 0;
        if (element.ValueKind == JsonValueKind.Number && element.TryGetInt32(out var intValue))
        {
            value = intValue;
            return true;
        }

        if (element.ValueKind == JsonValueKind.String && int.TryParse(element.GetString(), out var parsed))
        {
            value = parsed;
            return true;
        }

        return false;
    }

    private static bool TryReadDouble(JsonElement element, out double value)
    {
        value = 0d;
        if (element.ValueKind == JsonValueKind.Number && element.TryGetDouble(out var doubleValue))
        {
            value = doubleValue;
            return true;
        }

        if (element.ValueKind == JsonValueKind.String && double.TryParse(element.GetString(), out var parsed))
        {
            value = parsed;
            return true;
        }

        return false;
    }

    private Func<int, string?, Task> BuildProgressReporter(ClientWebSocket socket, TaskDispatchMessage message, CancellationToken cancellationToken)
    {
        var last = -1;
        return async (percent, detail) =>
        {
            if (percent <= last)
            {
                return;
            }

            if (percent > 100)
            {
                percent = 100;
            }

            last = percent;
            await SendTaskProgressAsync(socket, message, percent, detail, cancellationToken);
        };
    }

    private async Task<ConfigApplyOutcome> ApplyConfigPayloadAsync(string payload, bool force, CancellationToken cancellationToken)
    {
        if (string.IsNullOrWhiteSpace(payload))
        {
            return ConfigApplyOutcome.Skipped(
                previousVersion: _configVersionTracker.ReadAppliedVersion(),
                version: 0,
                force: force,
                reason: "empty_payload");
        }

        EdgeConfigDto config;
        try
        {
            config = JsonSerializer.Deserialize<EdgeConfigDto>(payload, JsonOptions)
                ?? throw new InvalidOperationException("invalid edge config payload");
        }
        catch (Exception ex)
        {
            _logger.LogWarning(ex, "edge config parse failed");
            return ConfigApplyOutcome.Fail(
                previousVersion: _configVersionTracker.ReadAppliedVersion(),
                version: 0,
                force: force,
                error: "edge config parse failed");
        }

        var newVersion = config.Version;
        var previousVersion = _configVersionTracker.ReadAppliedVersion();
        _logger.LogInformation(
            "edge config apply evaluating old_version={OldVersion} new_version={NewVersion} force={Force}",
            previousVersion,
            newVersion,
            force);

        if (!_configVersionTracker.ShouldApply(newVersion, force))
        {
            _logger.LogInformation(
                "edge config apply skipped old_version={OldVersion} new_version={NewVersion} reason={Reason}",
                previousVersion,
                newVersion,
                "version_not_newer");
            return ConfigApplyOutcome.Skipped(
                previousVersion: previousVersion,
                version: newVersion,
                force: force,
                reason: "version_not_newer");
        }

        await WriteConfigWithBackupAsync(payload, cancellationToken);
        _edgeConfigStore.Update(config);
        _tlsCertificateStore.Reload(config);
        _tlsRuntimePolicyStore.Reload(config);
        await PersistDynamicConfigAsync(config, cancellationToken);

        var apply = _proxyRuntime.TryApply(config, force);
        if (!apply.Success)
        {
            _syncStateStore.MarkApplyError(newVersion, apply.Error, Guid.NewGuid().ToString("N"));
            _logger.LogError(
                "edge config apply failed version={Version} error={Error}",
                apply.Version,
                apply.Error ?? "unknown");
            return ConfigApplyOutcome.Fail(
                previousVersion: previousVersion,
                version: newVersion,
                force: force,
                error: apply.Error ?? "proxy apply failed");
        }

        var streamApply = _streamRuntime.Apply(config);
        var applied = BuildConfigAppliedSummary(previousVersion, newVersion, force, apply, streamApply);
        var ret = JsonSerializer.Serialize(applied, JsonOptions);
        if (!streamApply.Success)
        {
            var streamError = streamApply.Errors.Count > 0 ? streamApply.Errors[0] : "stream apply failed";
            _syncStateStore.MarkApplyError(newVersion, streamError, Guid.NewGuid().ToString("N"));
            _logger.LogWarning(
                "stream config apply failed version={Version} received={Received} planned={Planned} applied={Applied} skipped={Skipped} error={Error}",
                config.Version,
                streamApply.Received,
                streamApply.Planned,
                streamApply.Applied,
                streamApply.Skipped,
                streamError);
            return ConfigApplyOutcome.Fail(
                previousVersion: previousVersion,
                version: newVersion,
                force: force,
                error: streamError,
                ret: ret,
                applied: applied);
        }

        _configVersionTracker.MarkApplied(newVersion);

        _logger.LogInformation(
            "edge config apply {Status} version={Version} old_version={OldVersion} streams_received={Received} streams_planned={Planned} streams_applied={Applied} streams_skipped={Skipped}",
            apply.Status,
            apply.Version,
            previousVersion,
            streamApply.Received,
            streamApply.Planned,
            streamApply.Applied,
            streamApply.Skipped);
        return ConfigApplyOutcome.Ok(
            previousVersion: previousVersion,
            version: newVersion,
            force: force,
            ret: ret,
            applied: applied);
    }

    private static Dictionary<string, object?> BuildConfigAppliedSummary(
        long previousVersion,
        long newVersion,
        bool force,
        ProxyApplyResult proxyApply,
        StreamApplyResult streamApply)
    {
        return new Dictionary<string, object?>
        {
            ["old_version"] = previousVersion,
            ["new_version"] = newVersion,
            ["force"] = force,
            ["proxy"] = new Dictionary<string, object?>
            {
                ["success"] = proxyApply.Success,
                ["status"] = proxyApply.Status,
                ["error"] = proxyApply.Error
            },
            ["stream"] = new Dictionary<string, object?>
            {
                ["success"] = streamApply.Success,
                ["received"] = streamApply.Received,
                ["planned"] = streamApply.Planned,
                ["applied"] = streamApply.Applied,
                ["skipped"] = streamApply.Skipped,
                ["started"] = streamApply.Started,
                ["stopped"] = streamApply.Stopped,
                ["restarted"] = streamApply.Restarted,
                ["errors"] = streamApply.Errors,
                ["skip_reasons"] = streamApply.SkipReasons ?? Array.Empty<string>()
            }
        };
    }

    private async Task PersistDynamicConfigAsync(EdgeConfigDto config, CancellationToken cancellationToken)
    {
        if (config.Resources != null)
        {
            await WriteJsonAsync(_runtimePaths.ResourcesPath, config.Resources, cancellationToken);
        }
        else
        {
            TryDeleteFile(_runtimePaths.ResourcesPath);
        }

        if (config.ErrorPages != null && config.ErrorPages.Count > 0)
        {
            await WriteJsonAsync(_runtimePaths.ErrorPagesPath, config.ErrorPages, cancellationToken);
        }
        else
        {
            TryDeleteFile(_runtimePaths.ErrorPagesPath);
        }

        if (config.DefaultConfig != null)
        {
            await WriteJsonAsync(_runtimePaths.DefaultConfigPath, config.DefaultConfig, cancellationToken);
        }
        else
        {
            TryDeleteFile(_runtimePaths.DefaultConfigPath);
        }

        if (config.CcRules != null && config.CcRules.Count > 0)
        {
            await WriteJsonAsync(_runtimePaths.CcRulesPath, config.CcRules, cancellationToken);
        }
        else
        {
            TryDeleteFile(_runtimePaths.CcRulesPath);
        }

        if (config.CcMatchers != null && config.CcMatchers.Count > 0)
        {
            await WriteJsonAsync(_runtimePaths.CcMatchersPath, config.CcMatchers, cancellationToken);
        }
        else
        {
            TryDeleteFile(_runtimePaths.CcMatchersPath);
        }

        if (config.CcFilters != null && config.CcFilters.Count > 0)
        {
            await WriteJsonAsync(_runtimePaths.CcFiltersPath, config.CcFilters, cancellationToken);
        }
        else
        {
            TryDeleteFile(_runtimePaths.CcFiltersPath);
        }

        await PersistFallbackCertAsync(config, cancellationToken);
    }

    private async Task PersistFallbackCertAsync(EdgeConfigDto config, CancellationToken cancellationToken)
    {
        if (string.IsNullOrWhiteSpace(config.FallbackCertData) || string.IsNullOrWhiteSpace(config.FallbackKeyData))
        {
            return;
        }

        Directory.CreateDirectory(_runtimePaths.CertDir);
        await File.WriteAllTextAsync(Path.Combine(_runtimePaths.CertDir, "fallback.pem"), config.FallbackCertData.Trim(), cancellationToken);
        await File.WriteAllTextAsync(Path.Combine(_runtimePaths.CertDir, "fallback.key"), config.FallbackKeyData.Trim(), cancellationToken);
    }

    private static long ExtractConfigVersion(string payload)
    {
        try
        {
            using var doc = JsonDocument.Parse(payload);
            if (doc.RootElement.TryGetProperty("version", out var versionElement))
            {
                if (versionElement.ValueKind == JsonValueKind.Number && versionElement.TryGetInt64(out var parsed))
                {
                    return parsed;
                }
            }
        }
        catch
        {
            // ignore
        }

        return 0;
    }

    private long ReadConfigVersion()
    {
        if (!File.Exists(_runtimePaths.ConfigPath))
        {
            return 0;
        }

        try
        {
            var payload = File.ReadAllText(_runtimePaths.ConfigPath);
            return ExtractConfigVersion(payload);
        }
        catch
        {
            return 0;
        }
    }

    private async Task WriteConfigWithBackupAsync(string payload, CancellationToken cancellationToken)
    {
        Directory.CreateDirectory(_runtimePaths.ConfDir);

        if (File.Exists(_runtimePaths.ConfigPath))
        {
            try
            {
                File.Copy(_runtimePaths.ConfigPath, _runtimePaths.ConfigBackupPath, true);
            }
            catch
            {
                // ignore backup failure
            }
        }

        await WriteAtomicAsync(_runtimePaths.ConfigPath, payload, cancellationToken);
    }

    private async Task<List<PackageApplyResult>> ApplyPackageSyncAsync(
        AgentPackagePayloadDto payload,
        CancellationToken cancellationToken)
    {
        var applied = new List<PackageApplyResult>();
        if (payload.Packages == null || payload.Packages.Count == 0)
        {
            return applied;
        }

        Directory.CreateDirectory(_runtimePaths.PackagesDir);

        foreach (var pkg in payload.Packages)
        {
            if (pkg == null || pkg.PackageId <= 0 || pkg.Config == null)
            {
                continue;
            }

            var targetPath = Path.Combine(_runtimePaths.PackagesDir, $"{pkg.PackageId}.json");
            var currentVersion = ReadPackageVersion(targetPath);
            if (currentVersion >= pkg.Version)
            {
                applied.Add(new PackageApplyResult
                {
                    PackageId = pkg.PackageId,
                    Version = pkg.Version,
                    Status = "skipped"
                });
                continue;
            }

            var json = JsonSerializer.Serialize(pkg.Config, JsonOptions);
            await WriteAtomicAsync(targetPath, json, cancellationToken);
            _localPackages[pkg.PackageId] = pkg.Config;

            applied.Add(new PackageApplyResult
            {
                PackageId = pkg.PackageId,
                Version = pkg.Version,
                Status = "updated"
            });
        }

        var limiterResult = await _packageBandwidthLimiter.ApplyAsync(_localPackages.Values.ToList(), cancellationToken);
        if (!limiterResult.Applied)
        {
            _logger.LogWarning(
                "package bandwidth apply failed iface={Interface} limit={Limit}Mbps message={Message}",
                limiterResult.Interface,
                limiterResult.LimitMbps,
                limiterResult.Message);
        }
        else
        {
            _logger.LogInformation(
                "package bandwidth applied iface={Interface} limit={Limit}Mbps",
                limiterResult.Interface,
                limiterResult.LimitMbps);
        }

        return applied;
    }

    private static int ReadPackageVersion(string path)
    {
        if (!File.Exists(path))
        {
            return 0;
        }

        try
        {
            using var doc = JsonDocument.Parse(File.ReadAllText(path));
            if (doc.RootElement.TryGetProperty("version", out var versionElement))
            {
                if (versionElement.ValueKind == JsonValueKind.Number && versionElement.TryGetInt32(out var parsed))
                {
                    return parsed;
                }
                if (versionElement.ValueKind == JsonValueKind.String && int.TryParse(versionElement.GetString(), out parsed))
                {
                    return parsed;
                }
            }
        }
        catch
        {
            // ignore
        }

        return 0;
    }

    private static async Task WriteAtomicAsync(string path, string content, CancellationToken cancellationToken)
    {
        var dir = Path.GetDirectoryName(path);
        if (!string.IsNullOrWhiteSpace(dir))
        {
            Directory.CreateDirectory(dir);
        }

        var tmp = path + ".tmp";
        await File.WriteAllTextAsync(tmp, content, cancellationToken);
        File.Move(tmp, path, true);
    }

    private async Task WriteJsonAsync<T>(string path, T data, CancellationToken cancellationToken)
    {
        var json = JsonSerializer.Serialize(data, JsonOptions);
        await WriteAtomicAsync(path, json, cancellationToken);
    }

    private static void TryDeleteFile(string path)
    {
        try
        {
            if (File.Exists(path))
            {
                File.Delete(path);
            }
        }
        catch
        {
            // ignore
        }
    }

    private static bool IsPackageSyncTask(string taskType)
    {
        if (string.IsNullOrWhiteSpace(taskType))
        {
            return false;
        }

        var normalized = taskType.Trim();
        if (string.Equals(normalized, "package_sync", StringComparison.OrdinalIgnoreCase))
        {
            return true;
        }

        if (string.Equals(normalized, "Package sync", StringComparison.OrdinalIgnoreCase))
        {
            return true;
        }

        return string.Equals(normalized, "\u5957\u9910\u540c\u6b65", StringComparison.Ordinal);
    }

    private int ApplyDeployCertificateToRuntime(DeployCertTaskPayload payload)
    {
        var current = _edgeConfigStore.Current;
        if (current == null || current.Domains.Count == 0)
        {
            return 0;
        }

        var certPem = payload.CertPem?.Trim();
        var keyPem = payload.KeyPem?.Trim();
        if (string.IsNullOrWhiteSpace(certPem) || string.IsNullOrWhiteSpace(keyPem))
        {
            return 0;
        }

        var targetHosts = new HashSet<string>(StringComparer.OrdinalIgnoreCase);
        foreach (var raw in payload.Domains)
        {
            if (string.IsNullOrWhiteSpace(raw))
            {
                continue;
            }

            var parts = raw.Split([',', ';', '\n', '\r', '\t', ' '], StringSplitOptions.RemoveEmptyEntries);
            foreach (var part in parts)
            {
                var host = part.Trim().TrimEnd('.').ToLowerInvariant();
                if (!string.IsNullOrWhiteSpace(host))
                {
                    targetHosts.Add(host);
                }
            }
        }

        if (targetHosts.Count == 0)
        {
            return 0;
        }

        var cloned = JsonSerializer.Deserialize<EdgeConfigDto>(JsonSerializer.Serialize(current, JsonOptions), JsonOptions);
        if (cloned == null)
        {
            return 0;
        }

        var applied = 0;
        foreach (var domain in cloned.Domains)
        {
            if (!DomainMatches(domain.Name, targetHosts))
            {
                continue;
            }

            domain.SslCertData = certPem;
            domain.SslKeyData = keyPem;
            domain.SslCertPath = null;
            domain.SslKeyPath = null;
            applied++;
        }

        if (applied <= 0)
        {
            return 0;
        }

        _edgeConfigStore.Update(cloned);
        _tlsCertificateStore.Reload(cloned);
        return applied;
    }

    private static bool DomainMatches(string? domainName, IReadOnlySet<string> targetHosts)
    {
        if (string.IsNullOrWhiteSpace(domainName) || targetHosts.Count == 0)
        {
            return false;
        }

        var parts = domainName.Split([',', ';', '\n', '\r', '\t', ' '], StringSplitOptions.RemoveEmptyEntries);
        foreach (var part in parts)
        {
            var host = part.Trim().TrimEnd('.').ToLowerInvariant();
            if (string.IsNullOrWhiteSpace(host))
            {
                continue;
            }

            if (targetHosts.Contains(host))
            {
                return true;
            }
        }

        return false;
    }

    private static List<string> SplitLines(string? input)
    {
        if (string.IsNullOrWhiteSpace(input))
        {
            return new List<string>();
        }

        var list = new List<string>();
        using var reader = new StringReader(input);
        string? line;
        while ((line = reader.ReadLine()) != null)
        {
            line = line.Trim();
            if (line.Length > 0)
            {
                list.Add(line);
            }
        }

        return list;
    }

    private async Task<string?> PurgeUrlsAsync(IReadOnlyList<string> urls)
    {
        string? lastError = null;
        foreach (var raw in urls)
        {
            var error = await PurgeUrlAsync(raw);
            if (error != null)
            {
                lastError = error;
            }
        }

        return lastError;
    }

    private Task<string?> PurgeUrlAsync(string raw)
    {
        if (!Uri.TryCreate(raw?.Trim(), UriKind.Absolute, out var uri) || string.IsNullOrWhiteSpace(uri.Host))
        {
            return Task.FromResult<string?>($"invalid url: {raw}");
        }

        var key = BuildCacheKey(uri);
        if (string.IsNullOrWhiteSpace(key))
        {
            return Task.FromResult<string?>(null);
        }

        var root = ResolveCacheRoot();
        var dataPath = Path.Combine(root, key.Replace('/', Path.DirectorySeparatorChar));
        var metaPath = dataPath + ".meta";
        TryDeleteFile(dataPath);
        TryDeleteFile(metaPath);
        return Task.FromResult<string?>(null);
    }

    private string ResolveCacheRoot()
    {
        if (string.IsNullOrWhiteSpace(_cacheOptions.Root))
        {
            return _runtimePaths.CacheDir;
        }

        return _cacheOptions.Root;
    }

    private string? BuildCacheKey(Uri uri)
    {
        var context = new DefaultHttpContext();
        context.Request.Scheme = uri.Scheme;
        context.Request.Host = new HostString(uri.Host, uri.Port);
        context.Request.Path = uri.AbsolutePath;
        context.Request.QueryString = new QueryString(uri.Query);

        var decision = _cacheStore.Resolve(context);
        return CacheKeyBuilder.BuildRelativeKey(context, decision);
    }

    private string? ClearCache(string? payload)
    {
        var root = ResolveCacheRoot();
        if (!Directory.Exists(root))
        {
            return null;
        }

        try
        {
            var domains = ParseClearCacheDomains(payload);
            if (domains.Count > 0)
            {
                foreach (var domain in domains)
                {
                    if (string.IsNullOrWhiteSpace(domain))
                    {
                        continue;
                    }

                    DeleteCacheByPrefix(root, domain);
                }

                return null;
            }

            var siteIds = ParseClearCacheSiteIds(payload);
            if (siteIds.Count > 0)
            {
                var hosts = CachePurgePlanner.ResolveHostsForSiteIds(_edgeConfigStore.Current, siteIds);
                foreach (var host in hosts)
                {
                    if (string.IsNullOrWhiteSpace(host))
                    {
                        continue;
                    }

                    DeleteCacheByPrefix(root, host.Trim().ToLowerInvariant());
                }

                return null;
            }

            foreach (var entry in Directory.EnumerateFileSystemEntries(root))
            {
                try
                {
                    if (Directory.Exists(entry))
                    {
                        Directory.Delete(entry, true);
                    }
                    else
                    {
                        File.Delete(entry);
                    }
                }
                catch
                {
                    // ignore individual failures
                }
            }
        }
        catch (Exception ex)
        {
            return ex.Message;
        }

        return null;
    }

    private Task<string?> PurgeDirectoriesAsync(IReadOnlyList<string> urls)
    {
        if (urls == null || urls.Count == 0)
        {
            return Task.FromResult<string?>(null);
        }

        var root = ResolveCacheRoot();
        string? lastError = null;
        foreach (var raw in urls)
        {
            if (!Uri.TryCreate(raw?.Trim(), UriKind.Absolute, out var uri) || string.IsNullOrWhiteSpace(uri.Host))
            {
                lastError = $"invalid url: {raw}";
                continue;
            }

            var prefix = CachePurgePlanner.BuildDirectoryPrefix(uri);
            if (string.IsNullOrWhiteSpace(prefix))
            {
                continue;
            }

            try
            {
                DeleteCacheByPrefix(root, prefix);
            }
            catch (Exception ex)
            {
                lastError = ex.Message;
            }
        }

        return Task.FromResult(lastError);
    }

    private static void DeleteCacheByPrefix(string root, string prefix)
    {
        if (string.IsNullOrWhiteSpace(root) || string.IsNullOrWhiteSpace(prefix))
        {
            return;
        }

        var normalizedPrefix = prefix.Replace('/', Path.DirectorySeparatorChar).Trim(Path.DirectorySeparatorChar);
        if (normalizedPrefix.Length == 0)
        {
            return;
        }

        var targetPath = Path.Combine(root, normalizedPrefix);
        if (Directory.Exists(targetPath))
        {
            Directory.Delete(targetPath, true);
        }

        if (File.Exists(targetPath))
        {
            File.Delete(targetPath);
        }

        var metaPath = targetPath + ".meta";
        if (File.Exists(metaPath))
        {
            File.Delete(metaPath);
        }

        var parent = Path.GetDirectoryName(targetPath);
        var leaf = Path.GetFileName(targetPath);
        if (string.IsNullOrWhiteSpace(parent) || string.IsNullOrWhiteSpace(leaf) || !Directory.Exists(parent))
        {
            return;
        }

        foreach (var file in Directory.EnumerateFiles(parent, leaf + "__q=*"))
        {
            try
            {
                File.Delete(file);
            }
            catch
            {
                // ignore individual failures
            }

            try
            {
                var fileMeta = file + ".meta";
                if (File.Exists(fileMeta))
                {
                    File.Delete(fileMeta);
                }
            }
            catch
            {
                // ignore individual failures
            }
        }
    }

    private static HashSet<long> ParseClearCacheSiteIds(string? payload)
    {
        var result = new HashSet<long>();
        if (string.IsNullOrWhiteSpace(payload))
        {
            return result;
        }

        try
        {
            using var doc = JsonDocument.Parse(payload);
            var root = doc.RootElement;
            if (root.ValueKind != JsonValueKind.Object)
            {
                return result;
            }

            if (!root.TryGetProperty("site_ids", out var siteIdsElement))
            {
                return result;
            }

            if (siteIdsElement.ValueKind == JsonValueKind.Array)
            {
                foreach (var element in siteIdsElement.EnumerateArray())
                {
                    if (element.ValueKind == JsonValueKind.Number && element.TryGetInt64(out var id) && id > 0)
                    {
                        result.Add(id);
                    }
                    else if (element.ValueKind == JsonValueKind.String &&
                             long.TryParse(element.GetString(), out var parsed) &&
                             parsed > 0)
                    {
                        result.Add(parsed);
                    }
                }
            }
            else if (siteIdsElement.ValueKind == JsonValueKind.Number && siteIdsElement.TryGetInt64(out var single) && single > 0)
            {
                result.Add(single);
            }
            else if (siteIdsElement.ValueKind == JsonValueKind.String &&
                     long.TryParse(siteIdsElement.GetString(), out var parsedSingle) &&
                     parsedSingle > 0)
            {
                result.Add(parsedSingle);
            }
        }
        catch
        {
            // ignore parse failures
        }

        return result;
    }

    private static HashSet<string> ParseClearCacheDomains(string? payload)
    {
        var result = new HashSet<string>(StringComparer.OrdinalIgnoreCase);
        if (string.IsNullOrWhiteSpace(payload))
        {
            return result;
        }

        try
        {
            using var doc = JsonDocument.Parse(payload);
            var root = doc.RootElement;
            if (root.ValueKind != JsonValueKind.Object ||
                !root.TryGetProperty("domains", out var domainsElement))
            {
                return result;
            }

            if (domainsElement.ValueKind != JsonValueKind.Array)
            {
                return result;
            }

            foreach (var element in domainsElement.EnumerateArray())
            {
                if (element.ValueKind != JsonValueKind.String)
                {
                    continue;
                }

                var host = element.GetString()?.Trim().TrimEnd('.').ToLowerInvariant();
                if (!string.IsNullOrWhiteSpace(host))
                {
                    result.Add(host);
                }
            }
        }
        catch
        {
            // ignore parse failures
        }

        return result;
    }

    private async Task<string?> PreheatUrlsAsync(IReadOnlyList<string> urls, CancellationToken cancellationToken)
    {
        string? lastError = null;
        foreach (var raw in urls)
        {
            var error = await PreheatUrlAsync(raw, cancellationToken);
            if (error != null)
            {
                lastError = error;
            }
        }

        return lastError;
    }

    private async Task<string?> PreheatUrlAsync(string raw, CancellationToken cancellationToken)
    {
        if (!Uri.TryCreate(raw?.Trim(), UriKind.Absolute, out var uri) || string.IsNullOrWhiteSpace(uri.Host))
        {
            return $"invalid url: {raw}";
        }

        var scheme = string.IsNullOrWhiteSpace(uri.Scheme) ? "http" : uri.Scheme.ToLowerInvariant();
        var port = uri.Port;
        if (port <= 0)
        {
            port = scheme == "https" ? 443 : 80;
        }

        var localUrl = $"{scheme}://127.0.0.1:{port}{uri.PathAndQuery}";
        using var request = new HttpRequestMessage(HttpMethod.Get, localUrl);
        request.Headers.Host = uri.Host;

        try
        {
            var client = BuildPreheatClient(scheme == "https");
            using var response = await client.SendAsync(request, cancellationToken);
            return null;
        }
        catch (Exception ex)
        {
            return ex.Message;
        }
    }

    private static HttpClient BuildPreheatClient(bool https)
    {
        if (!https)
        {
            return new HttpClient { Timeout = TimeSpan.FromSeconds(15) };
        }

        return new HttpClient(new HttpClientHandler
        {
            ServerCertificateCustomValidationCallback = HttpClientHandler.DangerousAcceptAnyServerCertificateValidator
        })
        { Timeout = TimeSpan.FromSeconds(15) };
    }

    private async Task ShipAccessLogsAsync(ClientWebSocket socket, string nodeId, CancellationToken cancellationToken)
    {
        var logPath = Path.Combine(_runtimePaths.LogsDir, "access.json");
        var offsetPath = Path.Combine(_runtimePaths.LogsDir, "access.offset");
        await ShipLogFileAsync(socket, nodeId, logPath, offsetPath, "agent_logs_access", cancellationToken);
    }

    private async Task ShipStreamLogsAsync(ClientWebSocket socket, string nodeId, CancellationToken cancellationToken)
    {
        var logPath = Path.Combine(_runtimePaths.LogsDir, "stream_access.json");
        var offsetPath = Path.Combine(_runtimePaths.LogsDir, "stream_access.offset");
        await ShipLogFileAsync(socket, nodeId, logPath, offsetPath, "agent_logs_stream", cancellationToken);
    }

    private async Task ShipSecurityLogsAsync(ClientWebSocket socket, string nodeId, CancellationToken cancellationToken)
    {
        var logPath = Path.Combine(_runtimePaths.LogsDir, "security.json");
        var offsetPath = Path.Combine(_runtimePaths.LogsDir, "security.offset");
        await ShipEventLogFileAsync(socket, nodeId, logPath, offsetPath, "security", cancellationToken);
    }

    private async Task ShipManualDebugLogsAsync(ClientWebSocket socket, string nodeId, CancellationToken cancellationToken)
    {
        var logPath = Path.Combine(_runtimePaths.LogsDir, "manual_debug.jsonl");
        var offsetPath = Path.Combine(_runtimePaths.LogsDir, "manual_debug.offset");
        await ShipEventLogFileAsync(socket, nodeId, logPath, offsetPath, "manual_debug", cancellationToken);
    }

    private async Task ShipSystemLogsAsync(ClientWebSocket socket, string nodeId, CancellationToken cancellationToken)
    {
        var logPath = Path.Combine(_runtimePaths.LogsDir, "system.json");
        var offsetPath = Path.Combine(_runtimePaths.LogsDir, "system.offset");
        await ShipEventLogFileAsync(socket, nodeId, logPath, offsetPath, "system", cancellationToken);
    }

    private async Task ShipLogFileAsync(
        ClientWebSocket socket,
        string nodeId,
        string logPath,
        string offsetPath,
        string kind,
        CancellationToken cancellationToken)
    {
        if (!File.Exists(logPath))
        {
            return;
        }

        var info = new FileInfo(logPath);
        var offset = LoadOffset(offsetPath);
        if (offset > info.Length)
        {
            offset = 0;
        }

        var lines = new List<string>();
        try
        {
            using var stream = new FileStream(logPath, FileMode.Open, FileAccess.Read, FileShare.ReadWrite);
            stream.Seek(offset, SeekOrigin.Begin);
            using var reader = new StreamReader(stream);
            while (lines.Count < 200)
            {
                var line = await reader.ReadLineAsync();
                if (line == null)
                {
                    break;
                }
                line = line.Trim();
                if (line.Length > 0)
                {
                    lines.Add(line);
                }
            }

            if (lines.Count == 0)
            {
                return;
            }

            var newOffset = stream.Position;
            var sent = await SendAsync(socket, new
            {
                kind,
                node_id = nodeId,
                node_ip = string.Empty,
                lines
            }, cancellationToken);

            if (sent)
            {
                SaveOffset(offsetPath, newOffset);
            }
        }
        catch
        {
            // ignore ship errors
        }
    }

    private async Task ShipMetricsAsync(ClientWebSocket socket, string nodeId, CancellationToken cancellationToken)
    {
        try
        {
            var client = _httpClientFactory.CreateClient();
            using var cts = CancellationTokenSource.CreateLinkedTokenSource(cancellationToken);
            cts.CancelAfter(TimeSpan.FromSeconds(5));
            using var response = await client.GetAsync("http://127.0.0.1:9100/metrics", cts.Token);
            if (!response.IsSuccessStatusCode)
            {
                return;
            }

            var content = await response.Content.ReadAsStringAsync(cts.Token);
            if (string.IsNullOrWhiteSpace(content))
            {
                return;
            }

            await SendAsync(socket, new
            {
                kind = "agent_logs_metrics",
                node_id = nodeId,
                node_ip = string.Empty,
                content
            }, cancellationToken);
        }
        catch
        {
            // ignore metrics errors
        }
    }

    private async Task ShipEventLogFileAsync(
        ClientWebSocket socket,
        string nodeId,
        string logPath,
        string offsetPath,
        string eventType,
        CancellationToken cancellationToken)
    {
        if (!File.Exists(logPath))
        {
            return;
        }

        var info = new FileInfo(logPath);
        var offset = LoadOffset(offsetPath);
        if (offset > info.Length)
        {
            offset = 0;
        }

        var payloads = new List<string>();
        try
        {
            using var stream = new FileStream(logPath, FileMode.Open, FileAccess.Read, FileShare.ReadWrite);
            stream.Seek(offset, SeekOrigin.Begin);
            using var reader = new StreamReader(stream);
            while (payloads.Count < 200)
            {
                var line = await reader.ReadLineAsync();
                if (line == null)
                {
                    break;
                }

                line = line.Trim();
                if (line.Length > 0)
                {
                    payloads.Add(line);
                }
            }

            if (payloads.Count == 0)
            {
                return;
            }

            var newOffset = stream.Position;
            var sent = await SendAsync(socket, new
            {
                kind = "agent_logs_events",
                node_id = nodeId,
                node_ip = string.Empty,
                type = eventType,
                payloads
            }, cancellationToken);

            if (sent)
            {
                SaveOffset(offsetPath, newOffset);
            }
        }
        catch
        {
            // ignore ship errors
        }
    }

    private static long LoadOffset(string path)
    {
        try
        {
            if (!File.Exists(path))
            {
                return 0;
            }

            var text = File.ReadAllText(path).Trim();
            return long.TryParse(text, out var value) ? value : 0;
        }
        catch
        {
            return 0;
        }
    }

    private static void SaveOffset(string path, long offset)
    {
        try
        {
            File.WriteAllText(path, offset.ToString());
        }
        catch
        {
            // ignore
        }
    }

    private async Task<string> UpgradeAgentPackageAsync(
        string raw,
        Func<int, string?, Task> report,
        CancellationToken cancellationToken)
    {
        AgentUpgradePayload? payload;
        try
        {
            payload = JsonSerializer.Deserialize<AgentUpgradePayload>(raw, JsonOptions);
        }
        catch
        {
            payload = null;
        }

        if (payload == null || string.IsNullOrWhiteSpace(payload.Version))
        {
            throw new InvalidOperationException("version is required");
        }

        var version = payload.Version.Trim();
        var downloadUrl = string.IsNullOrWhiteSpace(payload.DownloadUrl)
            ? BuildAgentPackageUrl(version)
            : payload.DownloadUrl.Trim();
        if (string.IsNullOrWhiteSpace(downloadUrl))
        {
            throw new InvalidOperationException("download url is required");
        }

        await report(5, "prepare download");

        var tempDir = Directory.CreateTempSubdirectory("cdn-agent-upgrade-");
        try
        {
            var filename = string.IsNullOrWhiteSpace(payload.FileName)
                ? Path.GetFileName(downloadUrl)
                : payload.FileName.Trim();
            if (string.IsNullOrWhiteSpace(filename))
            {
                filename = "agent-package";
            }

            var packagePath = Path.Combine(tempDir.FullName, filename);
            await DownloadFileAsync(downloadUrl, packagePath, cancellationToken);
            await report(30, "downloaded");

            if (!string.IsNullOrWhiteSpace(payload.Sha256))
            {
                var sum = ComputeSha256(packagePath);
                if (!string.Equals(sum, payload.Sha256.Trim(), StringComparison.OrdinalIgnoreCase))
                {
                    throw new InvalidOperationException("sha256 mismatch");
                }
            }

            await report(40, "extracting");
            var extractDir = Path.Combine(tempDir.FullName, "extract");
            Directory.CreateDirectory(extractDir);
            await ExtractPackageAsync(packagePath, extractDir, cancellationToken);

            var edgeNodePath = FindEdgeNodeDirectory(extractDir);
            if (!string.IsNullOrWhiteSpace(edgeNodePath))
            {
                await report(55, "updating assets");
                ApplyEdgeNodeUpgrade(edgeNodePath, _runtimePaths.RuntimeRoot);
            }

            await report(100, "done");
            var result = new Dictionary<string, object?>
            {
                ["version"] = version,
                ["restart"] = false
            };
            return JsonSerializer.Serialize(result, JsonOptions);
        }
        finally
        {
            try
            {
                tempDir.Delete(true);
            }
            catch
            {
                // ignore cleanup failure
            }
        }
    }

    private string BuildAgentPackageUrl(string version)
    {
        var baseUrl = _configuration["Api:BaseUrl"]?.TrimEnd('/') ?? string.Empty;
        if (string.IsNullOrWhiteSpace(baseUrl))
        {
            return string.Empty;
        }

        return $"{baseUrl}/api/v1/agent/upgrade/package?version={Uri.EscapeDataString(version)}";
    }

    private async Task DownloadFileAsync(string downloadUrl, string dest, CancellationToken cancellationToken)
    {
        var client = _httpClientFactory.CreateClient();
        var token = _configuration["Node:Token"] ?? _configuration["Agent:Token"];
        using var request = new HttpRequestMessage(HttpMethod.Get, downloadUrl);
        if (!string.IsNullOrWhiteSpace(token))
        {
            request.Headers.Authorization = new AuthenticationHeaderValue("Bearer", token.Trim());
        }

        using var response = await client.SendAsync(request, cancellationToken);
        response.EnsureSuccessStatusCode();

        Directory.CreateDirectory(Path.GetDirectoryName(dest)!);
        var tmp = dest + ".tmp";
        await using (var output = File.Create(tmp))
        {
            await response.Content.CopyToAsync(output, cancellationToken);
        }
        File.Move(tmp, dest, true);
    }

    private static async Task ExtractPackageAsync(string path, string dest, CancellationToken cancellationToken)
    {
        var lower = path.ToLowerInvariant();
        if (lower.EndsWith(".zip", StringComparison.Ordinal))
        {
            ZipFile.ExtractToDirectory(path, dest, true);
            return;
        }

        if (lower.EndsWith(".tar.gz", StringComparison.Ordinal))
        {
            var tarPath = Path.Combine(dest, "package.tar");
            await using (var input = File.OpenRead(path))
            await using (var gzip = new GZipStream(input, CompressionMode.Decompress))
            await using (var output = File.Create(tarPath))
            {
                await gzip.CopyToAsync(output, cancellationToken);
            }

            TarFile.ExtractToDirectory(tarPath, dest, true);
            TryDeleteFile(tarPath);
            return;
        }

        throw new InvalidOperationException("unsupported package format");
    }

    private static string? FindEdgeNodeDirectory(string root)
    {
        try
        {
            foreach (var dir in Directory.EnumerateDirectories(root, "edge-node", SearchOption.AllDirectories))
            {
                return dir;
            }
        }
        catch
        {
            // ignore
        }

        return null;
    }

    private static void ApplyEdgeNodeUpgrade(string srcRoot, string destRoot)
    {
        if (string.IsNullOrWhiteSpace(srcRoot) || string.IsNullOrWhiteSpace(destRoot))
        {
            return;
        }

        foreach (var path in Directory.EnumerateFileSystemEntries(srcRoot, "*", SearchOption.AllDirectories))
        {
            var rel = Path.GetRelativePath(srcRoot, path).Replace('\\', '/');
            if (string.IsNullOrWhiteSpace(rel) || rel == ".")
            {
                continue;
            }

            if (ShouldSkipUpgradePath(rel))
            {
                if (Directory.Exists(path))
                {
                    continue;
                }
            }

            var targetPath = Path.Combine(destRoot, rel.Replace('/', Path.DirectorySeparatorChar));
            if (Directory.Exists(path))
            {
                Directory.CreateDirectory(targetPath);
                continue;
            }

            Directory.CreateDirectory(Path.GetDirectoryName(targetPath)!);
            File.Copy(path, targetPath, true);
        }
    }

    private static bool ShouldSkipUpgradePath(string rel)
    {
        rel = rel.TrimStart('/');
        if (rel.StartsWith("cert/", StringComparison.OrdinalIgnoreCase))
        {
            return true;
        }
        if (rel.StartsWith("logs/", StringComparison.OrdinalIgnoreCase))
        {
            return true;
        }
        if (rel.StartsWith("cache/", StringComparison.OrdinalIgnoreCase))
        {
            return true;
        }
        if (rel.StartsWith("packages/", StringComparison.OrdinalIgnoreCase))
        {
            return true;
        }
        if (rel.StartsWith("conf/dynamic/", StringComparison.OrdinalIgnoreCase))
        {
            return true;
        }
        if (string.Equals(rel, "conf/cdn_config.json", StringComparison.OrdinalIgnoreCase))
        {
            return true;
        }

        return false;
    }

    private static string ComputeSha256(string path)
    {
        try
        {
            using var stream = File.OpenRead(path);
            var hash = SHA256.HashData(stream);
            return Convert.ToHexString(hash);
        }
        catch
        {
            return string.Empty;
        }
    }

    private sealed class HeartbeatAckMessage
    {
        [JsonPropertyName("sync_action")]
        public string? SyncAction { get; set; }
    }

    private sealed class L2NodesResponse
    {
        [JsonPropertyName("msg_id")]
        public string? MsgId { get; set; }

        [JsonPropertyName("nodes")]
        public List<AgentL2NodeItem>? Nodes { get; set; }
    }

    private sealed class NodeSyncAck
    {
        public string Action { get; set; } = string.Empty;
        public bool Success { get; set; }
        public int Attempts { get; set; }
        public string LastError { get; set; } = string.Empty;
        public DateTimeOffset LastAt { get; set; }
    }

    private sealed class L2HealthState
    {
        public bool Online { get; set; }
        public int Success { get; set; }
        public int Fail { get; set; }
    }

    private sealed class ConfigApplyOutcome
    {
        public bool Success { get; init; }
        public string Status { get; init; } = "unknown";
        public long PreviousVersion { get; init; }
        public long Version { get; init; }
        public bool Force { get; init; }
        public string? Error { get; init; }
        public string? Ret { get; init; }
        public object? Applied { get; init; }

        public static ConfigApplyOutcome Ok(long previousVersion, long version, bool force, string ret, object applied)
        {
            return new ConfigApplyOutcome
            {
                Success = true,
                Status = "ok",
                PreviousVersion = previousVersion,
                Version = version,
                Force = force,
                Ret = ret,
                Applied = applied
            };
        }

        public static ConfigApplyOutcome Fail(
            long previousVersion,
            long version,
            bool force,
            string error,
            string? ret = null,
            object? applied = null)
        {
            return new ConfigApplyOutcome
            {
                Success = false,
                Status = "fail",
                PreviousVersion = previousVersion,
                Version = version,
                Force = force,
                Error = error,
                Ret = ret,
                Applied = applied
            };
        }

        public static ConfigApplyOutcome Skipped(long previousVersion, long version, bool force, string reason)
        {
            var applied = new Dictionary<string, object?>
            {
                ["old_version"] = previousVersion,
                ["new_version"] = version,
                ["force"] = force,
                ["reason"] = reason
            };
            return new ConfigApplyOutcome
            {
                Success = true,
                Status = "skipped",
                PreviousVersion = previousVersion,
                Version = version,
                Force = force,
                Ret = JsonSerializer.Serialize(applied, JsonOptions),
                Applied = applied
            };
        }
    }

    private sealed class TaskAckDiagnostics
    {
        public string? RetCode { get; init; }
        public string? ErrorType { get; init; }
        public bool? IsRetryable { get; init; }
        public int? Attempt { get; init; }
        public int? MaxAttempts { get; init; }
        public int? NextBackoffMs { get; init; }
    }

    private sealed class PackageApplyResult
    {
        [JsonPropertyName("package_id")]
        public long PackageId { get; set; }

        [JsonPropertyName("version")]
        public int Version { get; set; }

        [JsonPropertyName("status")]
        public string Status { get; set; } = "updated";
    }

    private sealed class AgentUpgradePayload
    {
        [JsonPropertyName("version")]
        public string? Version { get; set; }

        [JsonPropertyName("file_name")]
        public string? FileName { get; set; }

        [JsonPropertyName("sha256")]
        public string? Sha256 { get; set; }

        [JsonPropertyName("download_url")]
        public string? DownloadUrl { get; set; }
    }
}
