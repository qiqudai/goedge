using Cnn.Common.Contracts;
using Cnn.Infrastructure.Db;
using Cnn.Api.Services;
using Cnn.Api.Services.Admin;
using Cnn.Api.Services.Agent;
using System.Net.WebSockets;
using System.Text;
using System.Text.Json;
using System.Text.Json.Serialization;
using SqlSugar;

namespace Cnn.Api.Services.Agent.Ws;

public interface IAgentWsSessionHandler
{
    Task HandleAsync(HttpContext context, WebSocket socket, CancellationToken cancellationToken);
}

public class AgentWsSessionHandler : IAgentWsSessionHandler
{
    private readonly JsonSerializerOptions jsonOptions;

    public AgentWsSessionHandler()
    {
        jsonOptions = new JsonSerializerOptions(JsonSerializerDefaults.Web)
        {
            PropertyNameCaseInsensitive = true,
            DefaultIgnoreCondition = JsonIgnoreCondition.WhenWritingNull
        };
    }

    public async Task HandleAsync(HttpContext context, WebSocket socket, CancellationToken cancellationToken)
    {
        var services = context.RequestServices;
        var nodeStatus = services.GetRequiredService<INodeStatusService>();
        var connections = services.GetRequiredService<IAgentConnectionManager>();
        var configuration = services.GetRequiredService<IConfiguration>();
        var db = services.GetRequiredService<ISqlSugarClient>();
        var edgeConfigService = services.GetRequiredService<IEdgeConfigService>();
        var nodeConfigService = services.GetRequiredService<INodeConfigService>();
        var agentNodeService = services.GetRequiredService<IAgentNodeService>();
        var logService = services.GetRequiredService<IAgentLogService>();
        var traceService = services.GetRequiredService<IAgentApiTraceService>();
        var certService = services.GetRequiredService<Cnn.Api.Services.Admin.ICertService>();
        var rateLimitService = services.GetRequiredService<INodeRateLimitService>();
        var ackService = services.GetRequiredService<IAgentTaskAckService>();
        var ackWaiter = services.GetRequiredService<IAgentAckWaiter>();
        var logger = services.GetRequiredService<ILogger<AgentWsSessionHandler>>();

        var connectionId = Guid.NewGuid().ToString("N");
        connections.Register(connectionId, socket);

        await SendWsAsync(socket, new { kind = AgentMessageKinds.Ack, connection_id = connectionId, heartbeat = 30 });
        _ = traceService.TraceAsync(new AgentApiTraceRecord
        {
            Direction = "out",
            Channel = "ws",
            Kind = AgentMessageKinds.Ack,
            NodeId = null,
            NodeIp = context.Connection.RemoteIpAddress?.ToString(),
            TraceId = context.TraceIdentifier
        }, context.RequestAborted);

        var buffer = new byte[8 * 1024];
        var handshakeDeadline = DateTimeOffset.UtcNow.AddSeconds(10);
        long boundNodeId = 0;
        string? boundNodeIdText = null;

        while (socket.State == WebSocketState.Open && boundNodeId <= 0)
        {
            var result = await socket.ReceiveAsync(buffer, CancellationToken.None);
            if (result.CloseStatus.HasValue) break;

            var json = Encoding.UTF8.GetString(buffer, 0, result.Count);
            if (string.IsNullOrWhiteSpace(json))
            {
                if (DateTimeOffset.UtcNow > handshakeDeadline) break;
                continue;
            }

            if (!TryReadKind(json, out var kind))
            {
                if (DateTimeOffset.UtcNow > handshakeDeadline) break;
                continue;
            }

            _ = traceService.TraceAsync(new AgentApiTraceRecord
            {
                Direction = "in",
                Channel = "ws",
                Kind = kind,
                NodeId = boundNodeIdText,
                NodeIp = context.Connection.RemoteIpAddress?.ToString(),
                TraceId = context.TraceIdentifier,
                Payload = json
            }, context.RequestAborted);

            if (IsHelloKind(kind))
            {
                AgentHelloMessage? hello = null;
                try
                {
                    hello = JsonSerializer.Deserialize<AgentHelloMessage>(json, jsonOptions);
                }
                catch
                {
                    await CloseWsAsync(socket, WebSocketCloseStatus.ProtocolError, "invalid hello");
                    connections.Remove(connectionId);
                    return;
                }

                var token = hello?.Token?.Trim();
                var nodeHint = hello?.NodeId?.Trim();
                var resolvedNodeId = await AuthenticateWsNodeAsync(token, nodeHint, configuration, db, context.RequestAborted);
                if (resolvedNodeId <= 0)
                {
                    await CloseWsAsync(socket, WebSocketCloseStatus.PolicyViolation, "auth failed");
                    connections.Remove(connectionId);
                    return;
                }

                boundNodeId = resolvedNodeId;
                boundNodeIdText = resolvedNodeId.ToString();
                connections.BindNode(connectionId, boundNodeIdText);
                nodeStatus.MarkOnline(boundNodeId);

                var version = hello?.AgentVersion?.Trim();
                if (string.IsNullOrWhiteSpace(version)) version = hello?.VersionFallback?.Trim();
                if (!string.IsNullOrWhiteSpace(version))
                {
                    await nodeConfigService.UpsertAsync(boundNodeId, "agent_version", version, context.RequestAborted);
                }

                var configResult = await edgeConfigService.GenerateAsync(boundNodeIdText, context.RequestAborted);
                if (configResult.Success && configResult.Data != null)
                {
                    logger.LogInformation(
                        "agent ws edge_config dispatch node_id={NodeId} version={Version} streams={Streams} domains={Domains} upstreams={Upstreams}",
                        boundNodeIdText,
                        configResult.Data.Version,
                        configResult.Data.Streams?.Count ?? 0,
                        configResult.Data.Domains?.Count ?? 0,
                        configResult.Data.Upstreams?.Count ?? 0);
                    await SendWsAsync(socket, new { kind = AgentMessageKinds.EdgeConfig, data = configResult.Data });
                    _ = traceService.TraceAsync(new AgentApiTraceRecord
                    {
                        Direction = "out",
                        Channel = "ws",
                        Kind = AgentMessageKinds.EdgeConfig,
                        NodeId = boundNodeIdText,
                        NodeIp = context.Connection.RemoteIpAddress?.ToString(),
                        TraceId = context.TraceIdentifier,
                        Payload = JsonSerializer.Serialize(configResult.Data, jsonOptions)
                    }, context.RequestAborted);
                }
                else
                {
                    logger.LogWarning(
                        "agent ws edge_config dispatch skipped node_id={NodeId} success={Success} error_code={ErrorCode} message={Message}",
                        boundNodeIdText,
                        configResult.Success,
                        configResult.ErrorCode,
                        configResult.MessageKey ?? string.Empty);
                }
                break;
            }

            if (DateTimeOffset.UtcNow > handshakeDeadline) break;
        }

        if (boundNodeId <= 0)
        {
            await CloseWsAsync(socket, WebSocketCloseStatus.PolicyViolation, "handshake timeout");
            connections.Remove(connectionId);
            return;
        }

        while (socket.State == WebSocketState.Open)
        {
            var result = await socket.ReceiveAsync(buffer, CancellationToken.None);
            if (result.CloseStatus.HasValue) break;

            var json = Encoding.UTF8.GetString(buffer, 0, result.Count);
            if (string.IsNullOrWhiteSpace(json)) continue;
            if (!TryReadKind(json, out var kind)) continue;

            if (!string.Equals(kind, AgentMessageKinds.Ping, StringComparison.OrdinalIgnoreCase) &&
                !string.Equals(kind, AgentMessageKinds.Pong, StringComparison.OrdinalIgnoreCase))
            {
                _ = traceService.TraceAsync(new AgentApiTraceRecord
                {
                    Direction = "in",
                    Channel = "ws",
                    Kind = kind,
                    NodeId = boundNodeIdText,
                    NodeIp = context.Connection.RemoteIpAddress?.ToString(),
                    TraceId = context.TraceIdentifier,
                    Payload = json
                }, context.RequestAborted);
            }

            if (string.Equals(kind, AgentMessageKinds.Ping, StringComparison.OrdinalIgnoreCase))
            {
                await SendWsAsync(socket, new { kind = AgentMessageKinds.Pong, ts = DateTimeOffset.UtcNow.ToUnixTimeSeconds() });
                nodeStatus.MarkOnline(boundNodeId);
                continue;
            }

            if (string.Equals(kind, AgentMessageKinds.TaskAck, StringComparison.OrdinalIgnoreCase))
            {
                try
                {
                    var ack = JsonSerializer.Deserialize<TaskAckMessage>(json, jsonOptions);
                    if (ack != null)
                    {
                        if (!ack.NodeId.HasValue || ack.NodeId.Value <= 0)
                        {
                            ack.NodeId = boundNodeId;
                        }
                        await ackService.HandleAsync(ack, context.RequestAborted);
                        ackWaiter.Notify(ack);
                    }
                }
                catch { }
                continue;
            }

            if (string.Equals(kind, AgentMessageKinds.Heartbeat, StringComparison.OrdinalIgnoreCase))
            {
                try
                {
                    var heartbeat = JsonSerializer.Deserialize<AgentHeartbeatRequest>(json, jsonOptions) ?? new AgentHeartbeatRequest();
                    var heartbeatResult = await agentNodeService.HeartbeatAsync(
                        heartbeat, boundNodeIdText, context.Connection.RemoteIpAddress?.ToString(), context.RequestAborted);

                    if (heartbeatResult.Success && heartbeatResult.Data != null)
                    {
                        await SendWsAsync(socket, new { kind = AgentMessageKinds.HeartbeatAck, sync_action = heartbeatResult.Data.SyncAction });
                        _ = traceService.TraceAsync(new AgentApiTraceRecord
                        {
                            Direction = "out",
                            Channel = "ws",
                            Kind = AgentMessageKinds.HeartbeatAck,
                            NodeId = boundNodeIdText,
                            NodeIp = context.Connection.RemoteIpAddress?.ToString(),
                            TraceId = context.TraceIdentifier,
                            Payload = JsonSerializer.Serialize(new { sync_action = heartbeatResult.Data.SyncAction }, jsonOptions)
                        }, context.RequestAborted);
                    }
                }
                catch { }
                continue;
            }

            if (string.Equals(kind, AgentMessageKinds.NodeSync, StringComparison.OrdinalIgnoreCase))
            {
                try
                {
                    var sync = JsonSerializer.Deserialize<AgentSyncRequest>(json, jsonOptions);
                    if (sync != null)
                    {
                        _ = await agentNodeService.SyncNodeStatusAsync(
                            sync, boundNodeIdText, context.Connection.RemoteIpAddress?.ToString(), context.RequestAborted);
                    }
                }
                catch { }
                continue;
            }

            if (string.Equals(kind, AgentMessageKinds.L2NodesRequest, StringComparison.OrdinalIgnoreCase))
            {
                try
                {
                    var req = JsonSerializer.Deserialize<L2NodesRequestMessage>(json, jsonOptions);
                    if (req != null && !string.IsNullOrWhiteSpace(req.MsgId))
                    {
                        var nodesResult = await agentNodeService.GetL2NodesAsync(boundNodeIdText, context.RequestAborted);
                        if (nodesResult.Success && nodesResult.Data != null)
                        {
                            await SendWsAsync(socket, new
                            {
                                kind = AgentMessageKinds.L2NodesResponse,
                                msg_id = req.MsgId,
                                nodes = nodesResult.Data.Nodes
                            });
                            _ = traceService.TraceAsync(new AgentApiTraceRecord
                            {
                                Direction = "out",
                                Channel = "ws",
                                Kind = AgentMessageKinds.L2NodesResponse,
                                NodeId = boundNodeIdText,
                                NodeIp = context.Connection.RemoteIpAddress?.ToString(),
                                TraceId = context.TraceIdentifier,
                                Payload = JsonSerializer.Serialize(new { msg_id = req.MsgId, count = nodesResult.Data.Nodes.Count }, jsonOptions)
                            }, context.RequestAborted);
                        }
                    }
                }
                catch { }
                continue;
            }

            if (string.Equals(kind, AgentMessageKinds.L2Heartbeat, StringComparison.OrdinalIgnoreCase))
            {
                try
                {
                    var l2Heartbeat = JsonSerializer.Deserialize<AgentL2HeartbeatRequest>(json, jsonOptions);
                    if (l2Heartbeat != null)
                    {
                        _ = await agentNodeService.ReportL2HeartbeatAsync(l2Heartbeat, context.RequestAborted);
                    }
                }
                catch { }
                continue;
            }

            if (string.Equals(kind, AgentMessageKinds.LogsAccess, StringComparison.OrdinalIgnoreCase))
            {
                try
                {
                    var req = JsonSerializer.Deserialize<AgentAccessLogRequest>(json, jsonOptions);
                    if (req != null)
                    {
                        var nodeId = NormalizeNodeId(req.NodeId, boundNodeIdText);
                        await logService.InsertAccessLogsAsync(nodeId, req.NodeIp, req.Lines, context.RequestAborted);
                    }
                }
                catch { }
                continue;
            }

            if (string.Equals(kind, AgentMessageKinds.LogsStream, StringComparison.OrdinalIgnoreCase))
            {
                try
                {
                    var req = JsonSerializer.Deserialize<AgentAccessLogRequest>(json, jsonOptions);
                    if (req != null)
                    {
                        var nodeId = NormalizeNodeId(req.NodeId, boundNodeIdText);
                        await logService.InsertStreamLogsAsync(nodeId, req.NodeIp, req.Lines, context.RequestAborted);
                    }
                }
                catch { }
                continue;
            }

            if (string.Equals(kind, AgentMessageKinds.LogsMetrics, StringComparison.OrdinalIgnoreCase))
            {
                try
                {
                    var req = JsonSerializer.Deserialize<AgentMetricLogRequest>(json, jsonOptions);
                    if (req != null)
                    {
                        var nodeId = NormalizeNodeId(req.NodeId, boundNodeIdText);
                        await logService.InsertMetricsAsync(nodeId, req.NodeIp, req.Content, context.RequestAborted);
                    }
                }
                catch { }
                continue;
            }

            if (string.Equals(kind, AgentMessageKinds.LogsEvents, StringComparison.OrdinalIgnoreCase))
            {
                try
                {
                    var req = JsonSerializer.Deserialize<AgentEventLogRequest>(json, jsonOptions);
                    if (req != null)
                    {
                        var nodeId = NormalizeNodeId(req.NodeId, boundNodeIdText);
                        await logService.InsertEventLogsAsync(nodeId, req.NodeIp, req.Type, req.Payloads, context.RequestAborted);
                    }
                }
                catch { }
                continue;
            }

            if (string.Equals(kind, AgentMessageKinds.CertIssued, StringComparison.OrdinalIgnoreCase))
            {
                try
                {
                    var req = JsonSerializer.Deserialize<AgentIssuedCertRequest>(json, jsonOptions);
                    if (req != null)
                    {
                        if (req.RateLimited)
                        {
                            var cooldown = req.RateCooldown > 0 ? TimeSpan.FromSeconds(req.RateCooldown) : TimeSpan.FromMinutes(10);
                            rateLimitService.MarkLimited(boundNodeId, cooldown);
                        }
                        _ = await certService.UpdateIssuedCertAsync(req, context.RequestAborted);
                    }
                }
                catch { }
            }
        }

        if (boundNodeId > 0)
        {
            nodeStatus.MarkOffline(boundNodeId);
        }
        connections.Remove(connectionId);
    }

    private static bool TryReadKind(string json, out string? kind)
    {
        kind = null;
        try
        {
            using var doc = JsonDocument.Parse(json);
            if (doc.RootElement.TryGetProperty("kind", out var kindEl))
            {
                kind = kindEl.GetString();
                return !string.IsNullOrWhiteSpace(kind);
            }
        }
        catch { }
        return false;
    }

    private static bool IsHelloKind(string? kind)
    {
        if (string.IsNullOrWhiteSpace(kind)) return false;
        return string.Equals(kind, AgentMessageKinds.AgentHello, StringComparison.OrdinalIgnoreCase)
            || string.Equals(kind, AgentMessageKinds.Hello, StringComparison.OrdinalIgnoreCase);
    }

    private static async Task<long> AuthenticateWsNodeAsync(
        string? token,
        string? nodeHint,
        IConfiguration configuration,
        ISqlSugarClient db,
        CancellationToken cancellationToken)
    {
        if (string.IsNullOrWhiteSpace(token)) return 0;

        var rawToken = token.Trim();
        var globalToken = configuration["Agent:Token"];
        if (string.IsNullOrWhiteSpace(globalToken)) globalToken = configuration["AgentToken"];

        if (!string.IsNullOrWhiteSpace(globalToken) &&
            string.Equals(rawToken, globalToken.Trim(), StringComparison.Ordinal))
        {
            return await ResolveNodeIdByHintAsync(nodeHint, db, cancellationToken);
        }

        var envToken = Environment.GetEnvironmentVariable("APP_AGENT_TOKEN");
        if (!string.IsNullOrWhiteSpace(envToken) &&
            string.Equals(rawToken, envToken.Trim(), StringComparison.Ordinal))
        {
            return await ResolveNodeIdByHintAsync(nodeHint, db, cancellationToken);
        }

        var node = await db.Queryable<Cnn.Domain.Entities.Node>()
            .Where(n => n.Token == rawToken)
            .Select(n => new Cnn.Domain.Entities.Node { Id = n.Id })
            .FirstAsync();

        return node?.Id ?? 0;
    }

    private static async Task<long> ResolveNodeIdByHintAsync(string? nodeHint, ISqlSugarClient db, CancellationToken cancellationToken)
    {
        if (string.IsNullOrWhiteSpace(nodeHint)) return 0;
        var trimmed = nodeHint.Trim();
        if (long.TryParse(trimmed, out var nodeId) && nodeId > 0) return nodeId;

        var node = await db.Queryable<Cnn.Domain.Entities.Node>()
            .Where(n => n.Name == trimmed || n.Host == trimmed || n.Ip == trimmed)
            .Select(n => new Cnn.Domain.Entities.Node { Id = n.Id })
            .FirstAsync();

        return node?.Id ?? 0;
    }

    private static string? NormalizeNodeId(string? nodeId, string? fallback)
    {
        if (!string.IsNullOrWhiteSpace(nodeId)) return nodeId.Trim();
        return string.IsNullOrWhiteSpace(fallback) ? null : fallback.Trim();
    }

    private static async Task CloseWsAsync(WebSocket socket, WebSocketCloseStatus status, string description)
    {
        try
        {
            if (socket.State == WebSocketState.Open || socket.State == WebSocketState.CloseReceived)
            {
                await socket.CloseAsync(status, description, CancellationToken.None);
            }
        }
        catch { }
    }

    private static Task SendWsAsync(WebSocket socket, object payload)
    {
        var json = JsonSerializer.Serialize(payload);
        var bytes = Encoding.UTF8.GetBytes(json);
        return socket.SendAsync(bytes, WebSocketMessageType.Text, true, CancellationToken.None);
    }
}

public sealed class AgentHelloMessage
{
    [JsonPropertyName("kind")]
    public string? Kind { get; set; }

    [JsonPropertyName("node_id")]
    public string? NodeId { get; set; }

    [JsonPropertyName("token")]
    public string? Token { get; set; }

    [JsonPropertyName("agent_version")]
    public string? AgentVersion { get; set; }

    [JsonPropertyName("version")]
    public string? VersionFallback { get; set; }

    [JsonPropertyName("capabilities")]
    public List<string>? Capabilities { get; set; }
}

public sealed class L2NodesRequestMessage
{
    [JsonPropertyName("kind")]
    public string? Kind { get; set; }

    [JsonPropertyName("msg_id")]
    public string? MsgId { get; set; }
}
