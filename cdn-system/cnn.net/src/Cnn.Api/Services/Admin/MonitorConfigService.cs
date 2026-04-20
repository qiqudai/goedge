using System.Text.Json;
using Task = System.Threading.Tasks.Task;
using Cnn.Common.Contracts;
using Cnn.Common.Contracts.Admin;
using Cnn.Api.Services.Common;
using Cnn.Domain.Entities;
using SqlSugar;

namespace Cnn.Api.Services.Admin;

public sealed class MonitorConfigService : IMonitorConfigService
{
    private const string ConfigKey = "node_monitor_config";
    private const string ConfigType = "system";

    private static readonly JsonSerializerOptions JsonOptions = new(JsonSerializerDefaults.Web)
    {
        PropertyNameCaseInsensitive = true
    };

    private readonly ISqlSugarClient _db;

    public MonitorConfigService(ISqlSugarClient db)
    {
        _db = db;
    }

    public async Task<ServiceResult<NodeMonitorConfigDto>> GetAsync(CancellationToken cancellationToken)
    {
        try
        {
            var record = await _db.Queryable<Config>()
                .Where(c => c.Name == ConfigKey && c.Type == ConfigType)
                .FirstAsync();

            if (record == null || string.IsNullOrWhiteSpace(record.Value))
            {
                return ServiceResult<NodeMonitorConfigDto>.Ok(BuildDefault());
            }

            NodeMonitorConfigDto? config;
            try
            {
                config = JsonSerializer.Deserialize<NodeMonitorConfigDto>(record.Value, JsonOptions);
            }
            catch
            {
                config = null;
            }

            if (config == null)
            {
                config = new NodeMonitorConfigDto();
            }

            Normalize(config);
            return ServiceResult<NodeMonitorConfigDto>.Ok(config);
        }
        catch
        {
            return ServiceResult<NodeMonitorConfigDto>.Ok(BuildDefault());
        }
    }

    public async Task<ServiceResult<bool>> UpdateAsync(NodeMonitorConfigDto config, CancellationToken cancellationToken)
    {
        Normalize(config);
        var payload = JsonSerializer.Serialize(config, JsonOptions);
        var now = DateTime.Now;

        var existing = await _db.Queryable<Config>()
            .Where(c => c.Name == ConfigKey && c.Type == ConfigType)
            .FirstAsync();

        if (existing == null)
        {
            var record = new Config
            {
                Name = ConfigKey,
                Type = ConfigType,
                Value = payload,
                Enable = true,
                ScopeId = 0,
                ScopeName = string.Empty,
                TaskId = null,
                CreateAt = now,
                UpdateAt = now
            };

            var created = await _db.Insertable(record).ExecuteCommandAsync();
            if (created <= 0)
            {
                return ServiceResult<bool>.Fail(ErrorCodes.DbError, "db_create_error");
            }

            return ServiceResult<bool>.Ok(true);
        }

        var updated = await _db.Updateable<Config>()
            .SetColumns(c => new Config { Value = payload, UpdateAt = now })
            .Where(c => c.Name == ConfigKey && c.Type == ConfigType)
            .ExecuteCommandAsync();

        if (updated <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.DbError, "db_save_error");
        }

        return ServiceResult<bool>.Ok(true);
    }

    private static NodeMonitorConfigDto BuildDefault()
    {
        return new NodeMonitorConfigDto
        {
            NotificationPeriod = "8-22",
            NotifyMethod = "email sms",
            NotifyMsgType = "node_ip_dns bandwidth monitor backup_ip backup_default_line backup_group",
            Email = string.Empty,
            Phone = string.Empty,
            BwExceedTimes = 2,
            AutoSwitchEnable = false,
            AutoSwitchThreshold = 90,
            AutoSwitchDuration = 30,
            AutoSwitchRecover = 300,
            AutoSwitchMinWeight = 1,
            MonitorApi = string.Empty,
            Interval = 30,
            FailedTimes = 3,
            FailedRate = "50"
        };
    }

    private static void Normalize(NodeMonitorConfigDto? config)
    {
        if (config == null)
        {
            return;
        }

        if (config.AutoSwitchThreshold <= 0 || config.AutoSwitchThreshold > 100)
        {
            config.AutoSwitchThreshold = 90;
        }

        if (config.AutoSwitchDuration <= 0)
        {
            config.AutoSwitchDuration = 30;
        }

        if (config.AutoSwitchRecover < 300)
        {
            config.AutoSwitchRecover = 300;
        }

        if (config.AutoSwitchMinWeight <= 0)
        {
            config.AutoSwitchMinWeight = 1;
        }
    }
}
