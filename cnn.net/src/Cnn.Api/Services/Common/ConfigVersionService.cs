using System.Text.Json;
using Cnn.Api.Services.Tasks.Workflow;
using Task = System.Threading.Tasks.Task;
using Cnn.Domain.Entities;
using SqlSugar;
using TaskEntity = Cnn.Domain.Entities.Task;

namespace Cnn.Api.Services.Common;

public sealed class ConfigVersionService : IConfigVersionService
{
    private const string ConfigVersionKey = "edge_config_version";

    private readonly ISqlSugarClient _db;

    public ConfigVersionService(ISqlSugarClient db)
    {
        _db = db;
    }

    public async Task<long> BumpAsync(string resource, IReadOnlyList<long> ids, CancellationToken cancellationToken)
    {
        var now = DateTime.Now;
        var version = 1L;
        var existing = await _db.Queryable<Config>()
            .Where(c => c.Name == ConfigVersionKey && c.Type == "system")
            .FirstAsync();

        if (existing == null)
        {
            var created = new Config
            {
                Name = ConfigVersionKey,
                Type = "system",
                ScopeId = 0,
                ScopeName = "global",
                Value = "1",
                Enable = true,
                CreateAt = now,
                UpdateAt = now
            };
            await _db.Insertable(created).ExecuteCommandAsync();
        }
        else
        {
            if (long.TryParse(existing.Value, out var current))
            {
                version = current + 1;
            }
            existing.Value = version.ToString();
            existing.UpdateAt = now;
            await _db.Updateable<Config>()
                .SetColumns(c => new Config
                {
                    Value = existing.Value,
                    UpdateAt = existing.UpdateAt
                })
                .Where(c => c.Name == ConfigVersionKey && c.Type == "system")
                .ExecuteCommandAsync();
        }

        await CreateConfigSyncTaskAsync(resource, ids, version, now);
        return version;
    }

    private async Task CreateConfigSyncTaskAsync(string resource, IReadOnlyList<long> ids, long version, DateTime now)
    {
        var change = new ConfigChange
        {
            Version = version,
            Resource = resource,
            Ids = ids?.ToArray() ?? Array.Empty<long>(),
            Timestamp = now
        };
        var data = JsonSerializer.Serialize(change);

        var task = new TaskEntity
        {
            Type = AsyncTaskTypes.ConfigSync,
            State = "waiting",
            Enable = true,
            Data = data,
            CreateAt = now,
            StartAt = now,
            EndAt = now,
            RetryAt = now
        };
        await _db.Insertable(task).ExecuteCommandAsync();
    }

    private sealed class ConfigChange
    {
        public long Version { get; set; }

        public string Resource { get; set; } = string.Empty;

        public long[] Ids { get; set; } = Array.Empty<long>();

        public DateTime Timestamp { get; set; }
    }
}
