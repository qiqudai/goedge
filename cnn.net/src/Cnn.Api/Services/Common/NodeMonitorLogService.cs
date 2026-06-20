using Cnn.Domain.Entities;
using SqlSugar;
using Task = System.Threading.Tasks.Task;

namespace Cnn.Api.Services.Common;

public interface INodeMonitorLogService
{
    Task WriteAsync(long nodeId, string? type, bool success, string? ip, CancellationToken cancellationToken);
    Task WriteBatchAsync(IReadOnlyList<long> nodeIds, string? type, bool success, IReadOnlyDictionary<long, string?>? ipMap, CancellationToken cancellationToken);
}

public sealed class NodeMonitorLogService : INodeMonitorLogService
{
    private readonly ISqlSugarClient _db;

    public NodeMonitorLogService(ISqlSugarClient db)
    {
        _db = db;
    }

    public Task WriteAsync(long nodeId, string? type, bool success, string? ip, CancellationToken cancellationToken)
    {
        if (nodeId <= 0)
        {
            return Task.CompletedTask;
        }

        var map = string.IsNullOrWhiteSpace(ip)
            ? null
            : new Dictionary<long, string?> { [nodeId] = ip };
        return WriteBatchAsync(new[] { nodeId }, type, success, map, cancellationToken);
    }

    public async Task WriteBatchAsync(
        IReadOnlyList<long> nodeIds,
        string? type,
        bool success,
        IReadOnlyDictionary<long, string?>? ipMap,
        CancellationToken cancellationToken)
    {
        if (nodeIds == null || nodeIds.Count == 0)
        {
            return;
        }

        var logType = string.IsNullOrWhiteSpace(type) ? "heartbeat" : type.Trim();
        var now = DateTime.Now;
        var eventId = new DateTimeOffset(now).ToUnixTimeSeconds().ToString();
        var successValue = success ? "1" : "0";

        var logs = new List<NodeMonitorLog>(nodeIds.Count);
        foreach (var nodeId in nodeIds)
        {
            if (nodeId <= 0)
            {
                continue;
            }

            string? ip = null;
            ipMap?.TryGetValue(nodeId, out ip);
            logs.Add(new NodeMonitorLog
            {
                CreateAt = now,
                Type = logType,
                EventId = eventId,
                Ip = ip?.Trim(),
                Success = successValue,
                NodeId = (int)nodeId
            });
        }

        if (logs.Count == 0)
        {
            return;
        }

        await _db.Insertable(logs).ExecuteCommandAsync();
    }
}
