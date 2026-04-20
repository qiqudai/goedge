using Cnn.Common.Contracts;
using Cnn.Common.Contracts.Agent;
using Cnn.Api.Services.Common;
using Cnn.Domain.Entities;
using SqlSugar;

namespace Cnn.Api.Services.Agent;

public interface IAgentNodeService
{
    Task<ServiceResult<AgentHeartbeatResponse>> HeartbeatAsync(
        AgentHeartbeatRequest request,
        string? tokenNodeId,
        string? clientIp,
        CancellationToken cancellationToken);

    Task<ServiceResult<AgentSyncResponse>> SyncNodeStatusAsync(
        AgentSyncRequest request,
        string? tokenNodeId,
        string? clientIp,
        CancellationToken cancellationToken);

    Task<ServiceResult<AgentL2NodesResult>> GetL2NodesAsync(
        string? nodeId,
        CancellationToken cancellationToken);

    Task<ServiceResult<AgentSyncResponse>> ReportL2HeartbeatAsync(
        AgentL2HeartbeatRequest request,
        CancellationToken cancellationToken);
}

public sealed class AgentNodeService : IAgentNodeService
{
    private readonly ISqlSugarClient _db;
    private readonly INodeStatusService _nodeStatus;
    private readonly INodeMonitorLogService _monitorLogService;

    public AgentNodeService(ISqlSugarClient db, INodeStatusService nodeStatus, INodeMonitorLogService monitorLogService)
    {
        _db = db;
        _nodeStatus = nodeStatus;
        _monitorLogService = monitorLogService;
    }

    public async Task<ServiceResult<AgentHeartbeatResponse>> HeartbeatAsync(
        AgentHeartbeatRequest request,
        string? tokenNodeId,
        string? clientIp,
        CancellationToken cancellationToken)
    {
        request ??= new AgentHeartbeatRequest();

        var nodeId = await ResolveHeartbeatNodeIdAsync(tokenNodeId, request.NodeId, clientIp, cancellationToken);
        var syncAction = string.Empty;

        if (nodeId > 0)
        {
            _nodeStatus.MarkOnline(nodeId, DateTime.Now);
            await _monitorLogService.WriteAsync(nodeId, "heartbeat", true, clientIp, cancellationToken);

            var configTask = await _db.Queryable<Node>()
                .Where(n => n.Id == nodeId)
                .Select(n => n.ConfigTask)
                .FirstAsync();

            var task = configTask?.Trim();
            if (string.Equals(task, "sync_enable", StringComparison.OrdinalIgnoreCase))
            {
                syncAction = "enable";
            }
            else if (string.Equals(task, "sync_disable", StringComparison.OrdinalIgnoreCase))
            {
                syncAction = "disable";
            }
        }

        var response = new AgentHeartbeatResponse
        {
            Status = "pong",
            SyncAction = string.IsNullOrWhiteSpace(syncAction) ? null : syncAction
        };

        return ServiceResult<AgentHeartbeatResponse>.Ok(response);
    }

    public async Task<ServiceResult<AgentSyncResponse>> SyncNodeStatusAsync(
        AgentSyncRequest request,
        string? tokenNodeId,
        string? clientIp,
        CancellationToken cancellationToken)
    {
        if (request == null)
        {
            return ServiceResult<AgentSyncResponse>.Fail(ErrorCodes.InvalidParam);
        }

        var nodeId = await ResolveSyncNodeIdAsync(tokenNodeId, request.NodeId, cancellationToken);
        if (nodeId <= 0)
        {
            return ServiceResult<AgentSyncResponse>.Fail(ErrorCodes.MissingParam);
        }

        var action = request.Action?.Trim().ToLowerInvariant();
        if (action is not ("enable" or "disable"))
        {
            return ServiceResult<AgentSyncResponse>.Fail(ErrorCodes.InvalidParam);
        }

        await _monitorLogService.WriteAsync(nodeId, "sync", request.Success, clientIp, cancellationToken);

        if (!request.Success)
        {
            return ServiceResult<AgentSyncResponse>.Ok(new AgentSyncResponse { Status = "ignored" });
        }

        await _db.Updateable<Node>()
            .SetColumns(n => new Node
            {
                ConfigTask = string.Empty,
                UpdateAt = DateTime.Now
            })
            .Where(n => n.Id == nodeId)
            .ExecuteCommandAsync();

        return ServiceResult<AgentSyncResponse>.Ok(new AgentSyncResponse { Status = "ok" });
    }

    public async Task<ServiceResult<AgentL2NodesResult>> GetL2NodesAsync(string? nodeId, CancellationToken cancellationToken)
    {
        var resolvedId = await ResolveAgentNodeIdAsync(nodeId, cancellationToken);
        if (resolvedId <= 0)
        {
            return ServiceResult<AgentL2NodesResult>.Fail(ErrorCodes.MissingParam);
        }

        var self = await _db.Queryable<Node>().Where(n => n.Id == resolvedId).FirstAsync();
        if (self == null)
        {
            return ServiceResult<AgentL2NodesResult>.Fail(ErrorCodes.NotFound);
        }

        if (self.Level != 1)
        {
            return ServiceResult<AgentL2NodesResult>.Ok(new AgentL2NodesResult());
        }

        var groupIds = await _db.Queryable<Line>()
            .Where(l => l.NodeId == resolvedId)
            .Select(l => l.NodeGroupId)
            .Distinct()
            .ToListAsync();

        var normalizedGroups = groupIds.Where(id => id.HasValue && id.Value > 0)
            .Select(id => id!.Value)
            .Distinct()
            .ToList();

        if (normalizedGroups.Count == 0)
        {
            return ServiceResult<AgentL2NodesResult>.Ok(new AgentL2NodesResult());
        }

        var l2NodeIds = await _db.Queryable<Line>()
            .Where(l => normalizedGroups.Contains(l.NodeGroupId ?? 0))
            .Where(l => l.NodeId != resolvedId)
            .Select(l => l.NodeId)
            .Distinct()
            .ToListAsync();

        var normalizedNodeIds = l2NodeIds.Where(id => id.HasValue && id.Value > 0)
            .Select(id => id!.Value)
            .Distinct()
            .ToList();

        if (normalizedNodeIds.Count == 0)
        {
            return ServiceResult<AgentL2NodesResult>.Ok(new AgentL2NodesResult());
        }

        var nodes = await _db.Queryable<Node>()
            .Where(n => normalizedNodeIds.Contains(n.Id) && n.Level == 2 && n.Enable == true)
            .Select(n => new Node
            {
                Id = n.Id,
                Ip = n.Ip,
                Port = n.Port,
                RegionId = n.RegionId,
                CheckProtocol = n.CheckProtocol,
                CheckPort = n.CheckPort,
                CheckHost = n.CheckHost,
                CheckPath = n.CheckPath,
                CheckTimeout = n.CheckTimeout
            })
            .ToListAsync();

        var metaService = new RegionMetaService(_db);
        var metaMap = await metaService.LoadAsync();

        var result = new AgentL2NodesResult();
        foreach (var node in nodes)
        {
            var checkPort = node.CheckPort.GetValueOrDefault();
            if (checkPort == 0)
            {
                checkPort = RegionMetaService.ResolveL2CheckPort(metaMap, node.RegionId);
            }

            var protocol = string.IsNullOrWhiteSpace(node.CheckProtocol) ? "tcp" : node.CheckProtocol!.Trim();

            result.Nodes.Add(new AgentL2NodeItem
            {
                Id = node.Id,
                Ip = node.Ip,
                Port = node.Port,
                CheckProtocol = protocol,
                CheckPort = checkPort,
                CheckHost = node.CheckHost,
                CheckPath = node.CheckPath,
                CheckTimeout = node.CheckTimeout
            });
        }

        return ServiceResult<AgentL2NodesResult>.Ok(result);
    }

    public async Task<ServiceResult<AgentSyncResponse>> ReportL2HeartbeatAsync(
        AgentL2HeartbeatRequest request,
        CancellationToken cancellationToken)
    {
        if (request == null)
        {
            return ServiceResult<AgentSyncResponse>.Fail(ErrorCodes.InvalidParam);
        }

        if (request.Nodes.Count == 0)
        {
            return ServiceResult<AgentSyncResponse>.Ok(new AgentSyncResponse { Status = "ok" });
        }

        var now = DateTime.Now;
        foreach (var id in request.Nodes)
        {
            if (id > 0)
            {
                _nodeStatus.MarkOnline(id, now);
            }
        }

        await _monitorLogService.WriteBatchAsync(request.Nodes, "l2_beat", true, null, cancellationToken);
        return ServiceResult<AgentSyncResponse>.Ok(new AgentSyncResponse { Status = "ok" });
    }

    private async Task<long> ResolveHeartbeatNodeIdAsync(
        string? tokenNodeId,
        string? payloadNodeId,
        string? clientIp,
        CancellationToken cancellationToken)
    {
        if (TryParseNodeId(tokenNodeId, out var resolved))
        {
            return resolved;
        }

        if (!string.IsNullOrWhiteSpace(payloadNodeId))
        {
            if (TryParseNodeId(payloadNodeId, out resolved))
            {
                return resolved;
            }

            var name = payloadNodeId.Trim();
            var node = await _db.Queryable<Node>()
                .Where(n => n.Name == name || n.Host == name)
                .Select(n => new Node { Id = n.Id })
                .FirstAsync();
            if (node != null && node.Id > 0)
            {
                return node.Id;
            }
        }

        if (!string.IsNullOrWhiteSpace(clientIp))
        {
            var node = await _db.Queryable<Node>()
                .Where(n => n.Ip == clientIp)
                .Select(n => new Node { Id = n.Id })
                .FirstAsync();
            if (node != null && node.Id > 0)
            {
                return node.Id;
            }
        }

        return 0;
    }

    private async Task<long> ResolveSyncNodeIdAsync(
        string? tokenNodeId,
        string? payloadNodeId,
        CancellationToken cancellationToken)
    {
        if (TryParseNodeId(tokenNodeId, out var resolved))
        {
            return resolved;
        }

        if (string.IsNullOrWhiteSpace(payloadNodeId))
        {
            return 0;
        }

        if (TryParseNodeId(payloadNodeId, out resolved))
        {
            return resolved;
        }

        var name = payloadNodeId.Trim();
        var node = await _db.Queryable<Node>()
            .Where(n => n.Name == name && n.Pid == 0)
            .Select(n => new Node { Id = n.Id })
            .FirstAsync();
        return node?.Id ?? 0;
    }

    private async Task<long> ResolveAgentNodeIdAsync(string? nodeId, CancellationToken cancellationToken)
    {
        if (TryParseNodeId(nodeId, out var resolved))
        {
            return resolved;
        }

        if (string.IsNullOrWhiteSpace(nodeId))
        {
            return 0;
        }

        var name = nodeId.Trim();
        var node = await _db.Queryable<Node>()
            .Where(n => n.Name == name && n.Pid == 0)
            .Select(n => new Node { Id = n.Id })
            .FirstAsync();
        return node?.Id ?? 0;
    }

    private static bool TryParseNodeId(string? raw, out long nodeId)
    {
        nodeId = 0;
        if (string.IsNullOrWhiteSpace(raw))
        {
            return false;
        }

        return long.TryParse(raw.Trim(), out nodeId) && nodeId > 0;
    }
}
