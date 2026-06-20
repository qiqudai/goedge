using Cnn.Domain.Entities;
using SqlSugar;
using Task = System.Threading.Tasks.Task;

namespace Cnn.Api.Services.Common;

public interface INodeConfigService
{
    Task UpsertAsync(long nodeId, string name, string value, CancellationToken cancellationToken);
    Task<Dictionary<long, string>> GetMapAsync(string name, CancellationToken cancellationToken);
    Task<string?> GetValueAsync(long nodeId, string name, CancellationToken cancellationToken);
}

public sealed class NodeConfigService : INodeConfigService
{
    private const string ConfigType = "node_config";
    private const string ScopeName = "node";

    private readonly ISqlSugarClient _db;

    public NodeConfigService(ISqlSugarClient db)
    {
        _db = db;
    }

    public async Task UpsertAsync(long nodeId, string name, string value, CancellationToken cancellationToken)
    {
        if (nodeId <= 0 || string.IsNullOrWhiteSpace(name))
        {
            return;
        }

        var item = await _db.Queryable<Config>()
            .Where(c => c.Type == ConfigType && c.ScopeName == ScopeName && c.ScopeId == nodeId && c.Name == name)
            .FirstAsync();

        var now = DateTime.Now;
        if (item != null)
        {
            await _db.Updateable<Config>()
                .SetColumns(c => new Config
                {
                    Value = value,
                    Enable = true,
                    UpdateAt = now
                })
                .Where(c => c.Type == ConfigType && c.ScopeName == ScopeName && c.ScopeId == nodeId && c.Name == name)
                .ExecuteCommandAsync();
            return;
        }

        var defaultTaskId = _db.CurrentConnectionConfig?.DbType == DbType.Sqlite
            ? 0L
            : (long?)null;

        await _db.Insertable(new Config
        {
            Name = name,
            Value = value,
            Type = ConfigType,
            ScopeName = ScopeName,
            ScopeId = (int)nodeId,
            Enable = true,
            // MySQL schema enforces FK on task_id, while SQLite test schema expects a non-null default.
            TaskId = defaultTaskId,
            CreateAt = now,
            UpdateAt = now
        }).ExecuteCommandAsync();
    }

    public async Task<Dictionary<long, string>> GetMapAsync(string name, CancellationToken cancellationToken)
    {
        var result = new Dictionary<long, string>();
        if (string.IsNullOrWhiteSpace(name))
        {
            return result;
        }

        var items = await _db.Queryable<Config>()
            .Where(c => c.Type == ConfigType && c.ScopeName == ScopeName && c.Name == name)
            .ToListAsync();

        foreach (var item in items)
        {
            if (item.ScopeId.HasValue)
            {
                result[item.ScopeId.Value] = item.Value ?? string.Empty;
            }
        }

        return result;
    }

    public async Task<string?> GetValueAsync(long nodeId, string name, CancellationToken cancellationToken)
    {
        if (nodeId <= 0 || string.IsNullOrWhiteSpace(name))
        {
            return null;
        }

        var item = await _db.Queryable<Config>()
            .Where(c => c.Type == ConfigType && c.ScopeName == ScopeName && c.ScopeId == nodeId && c.Name == name)
            .FirstAsync();

        return item?.Value;
    }
}
