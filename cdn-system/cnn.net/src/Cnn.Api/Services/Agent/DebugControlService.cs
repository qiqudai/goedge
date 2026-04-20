using System.Text.Json;
using Cnn.Common.Contracts;
using Cnn.Common.Contracts.Admin;
using Cnn.Api.Services.Common;

namespace Cnn.Api.Services.Agent;

public interface IDebugControlService
{
    Task<ServiceResult<WsDispatchResponse>> UpdateSwitchesAsync(DebugSwitchDispatchRequest request, CancellationToken cancellationToken);
    Task<ServiceResult<WsDispatchResponse>> WriteManualLogAsync(ManualDebugLogDispatchRequest request, CancellationToken cancellationToken);
    Task<ServiceResult<ServerDebugSwitchesDto>> GetServerSwitchesAsync(CancellationToken cancellationToken);
    Task<ServiceResult<bool>> UpdateServerSwitchesAsync(ServerDebugSwitchesUpdateRequest request, CancellationToken cancellationToken);
}

public sealed class DebugControlService : IDebugControlService
{
    private readonly IWsDispatchService _dispatchService;
    private readonly IConfigItemService _configItemService;
    private readonly ISystemConfigService _systemConfigService;
    private static readonly JsonSerializerOptions JsonOptions = new(JsonSerializerDefaults.Web);

    public DebugControlService(
        IWsDispatchService dispatchService,
        IConfigItemService configItemService,
        ISystemConfigService systemConfigService)
    {
        _dispatchService = dispatchService;
        _configItemService = configItemService;
        _systemConfigService = systemConfigService;
    }

    public Task<ServiceResult<WsDispatchResponse>> UpdateSwitchesAsync(DebugSwitchDispatchRequest request, CancellationToken cancellationToken)
    {
        if (request == null || request.NodeId.GetValueOrDefault() <= 0)
        {
            return Task.FromResult(ServiceResult<WsDispatchResponse>.Fail(ErrorCodes.InvalidParam));
        }

        var hasSwitchUpdates = request.Switches != null && request.Switches.Count > 0;
        var hasSessionUpdates =
            request.DebugEnabled.HasValue ||
            request.InternalIpOnly.HasValue ||
            request.AllowHeaderToken.HasValue ||
            request.AllowQueryFlag.HasValue ||
            !string.IsNullOrWhiteSpace(request.DebugToken) ||
            request.SampleRate.HasValue ||
            request.MaxEventsPerSec.HasValue ||
            (request.Modules != null && request.Modules.Count > 0);

        if (!hasSwitchUpdates && !hasSessionUpdates)
        {
            return Task.FromResult(ServiceResult<WsDispatchResponse>.Fail(ErrorCodes.InvalidParam));
        }

        var waitSeconds = request.WaitSeconds.GetValueOrDefault();
        if (waitSeconds <= 0)
        {
            waitSeconds = 8;
        }

        var payload = JsonSerializer.Serialize(new
        {
            switches = request.Switches,
            ttl_seconds = request.TtlSeconds,
            debug_enabled = request.DebugEnabled,
            internal_ip_only = request.InternalIpOnly,
            debug_token = request.DebugToken,
            allow_header_token = request.AllowHeaderToken,
            allow_query_flag = request.AllowQueryFlag,
            modules = request.Modules,
            sample_rate = request.SampleRate,
            max_events_per_sec = request.MaxEventsPerSec,
            reason = request.Reason
        }, JsonOptions);

        return _dispatchService.DispatchAsync(new WsDispatchRequest
        {
            NodeId = request.NodeId,
            TaskType = AgentTaskTypes.DebugSwitch,
            Payload = payload,
            WaitSeconds = waitSeconds
        }, cancellationToken);
    }

    public Task<ServiceResult<WsDispatchResponse>> WriteManualLogAsync(ManualDebugLogDispatchRequest request, CancellationToken cancellationToken)
    {
        if (request == null || request.NodeId.GetValueOrDefault() <= 0 || string.IsNullOrWhiteSpace(request.Message))
        {
            return Task.FromResult(ServiceResult<WsDispatchResponse>.Fail(ErrorCodes.InvalidParam));
        }

        var waitSeconds = request.WaitSeconds.GetValueOrDefault();
        if (waitSeconds <= 0)
        {
            waitSeconds = 8;
        }

        var payload = JsonSerializer.Serialize(new
        {
            category = string.IsNullOrWhiteSpace(request.Category) ? "manual" : request.Category.Trim(),
            message = request.Message.Trim(),
            data = request.Data
        }, JsonOptions);

        return _dispatchService.DispatchAsync(new WsDispatchRequest
        {
            NodeId = request.NodeId,
            TaskType = AgentTaskTypes.ManualDebugLog,
            Payload = payload,
            WaitSeconds = waitSeconds
        }, cancellationToken);
    }

    public async Task<ServiceResult<ServerDebugSwitchesDto>> GetServerSwitchesAsync(CancellationToken cancellationToken)
    {
        var cfg = await _systemConfigService.LoadSystemConfigAsync(cancellationToken);
        var dto = new ServerDebugSwitchesDto
        {
            OperationLogEnabled = ReadBool(cfg, DebugSwitchKeys.OperationLogEnabled, true),
            AgentApiTraceEnabled = ReadBool(cfg, DebugSwitchKeys.AgentApiTraceEnabled, false),
            AgentApiTracePayloadEnabled = ReadBool(cfg, DebugSwitchKeys.AgentApiTracePayloadEnabled, false),
            AgentApiTraceMaxPayload = ReadInt(cfg, DebugSwitchKeys.AgentApiTraceMaxPayload, 2048, 256, 65536),
            AgentApiTraceMaxEventsPerSec = ReadInt(cfg, DebugSwitchKeys.AgentApiTraceMaxEventsPerSec, 0, 0, 50000)
        };

        return ServiceResult<ServerDebugSwitchesDto>.Ok(dto);
    }

    public async Task<ServiceResult<bool>> UpdateServerSwitchesAsync(ServerDebugSwitchesUpdateRequest request, CancellationToken cancellationToken)
    {
        request ??= new ServerDebugSwitchesUpdateRequest();
        var items = new List<ConfigItemPayloadDto>();

        if (request.OperationLogEnabled.HasValue)
        {
            items.Add(new ConfigItemPayloadDto
            {
                Name = DebugSwitchKeys.OperationLogEnabled,
                Value = request.OperationLogEnabled.Value ? "1" : "0",
                Enable = true
            });
        }

        if (request.AgentApiTraceEnabled.HasValue)
        {
            items.Add(new ConfigItemPayloadDto
            {
                Name = DebugSwitchKeys.AgentApiTraceEnabled,
                Value = request.AgentApiTraceEnabled.Value ? "1" : "0",
                Enable = true
            });
        }

        if (request.AgentApiTracePayloadEnabled.HasValue)
        {
            items.Add(new ConfigItemPayloadDto
            {
                Name = DebugSwitchKeys.AgentApiTracePayloadEnabled,
                Value = request.AgentApiTracePayloadEnabled.Value ? "1" : "0",
                Enable = true
            });
        }

        if (request.AgentApiTraceMaxPayload.HasValue)
        {
            items.Add(new ConfigItemPayloadDto
            {
                Name = DebugSwitchKeys.AgentApiTraceMaxPayload,
                Value = Math.Clamp(request.AgentApiTraceMaxPayload.Value, 256, 65536).ToString(),
                Enable = true
            });
        }

        if (request.AgentApiTraceMaxEventsPerSec.HasValue)
        {
            items.Add(new ConfigItemPayloadDto
            {
                Name = DebugSwitchKeys.AgentApiTraceMaxEventsPerSec,
                Value = Math.Clamp(request.AgentApiTraceMaxEventsPerSec.Value, 0, 50000).ToString(),
                Enable = true
            });
        }

        if (items.Count == 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.MissingParam);
        }

        var upsert = await _configItemService.UpsertAsync(new ConfigItemUpsertRequest
        {
            Type = "system",
            ScopeName = "global",
            ScopeId = 0,
            Items = items
        }, cancellationToken);

        if (!upsert.Success)
        {
            return ServiceResult<bool>.Fail(upsert.ErrorCode, upsert.MessageKey);
        }

        return ServiceResult<bool>.Ok(true);
    }

    private static bool ReadBool(IReadOnlyDictionary<string, string> map, string key, bool defaultValue)
    {
        if (map.TryGetValue(key, out var value))
        {
            var normalized = value?.Trim().ToLowerInvariant();
            if (normalized is "1" or "true" or "yes" or "on")
            {
                return true;
            }
            if (normalized is "0" or "false" or "no" or "off")
            {
                return false;
            }
        }

        var alias = key.Replace('-', '_');
        if (!string.Equals(alias, key, StringComparison.Ordinal) && map.TryGetValue(alias, out value))
        {
            var normalized = value?.Trim().ToLowerInvariant();
            if (normalized is "1" or "true" or "yes" or "on")
            {
                return true;
            }
            if (normalized is "0" or "false" or "no" or "off")
            {
                return false;
            }
        }

        return defaultValue;
    }

    private static int ReadInt(IReadOnlyDictionary<string, string> map, string key, int defaultValue, int min, int max)
    {
        if (map.TryGetValue(key, out var value) && int.TryParse(value?.Trim(), out var parsed))
        {
            return Math.Clamp(parsed, min, max);
        }

        var alias = key.Replace('-', '_');
        if (!string.Equals(alias, key, StringComparison.Ordinal) &&
            map.TryGetValue(alias, out value) &&
            int.TryParse(value?.Trim(), out parsed))
        {
            return Math.Clamp(parsed, min, max);
        }

        return Math.Clamp(defaultValue, min, max);
    }
}
