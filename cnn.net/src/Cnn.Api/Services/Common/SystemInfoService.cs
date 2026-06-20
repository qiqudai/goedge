using System.Text.Json;
using Task = System.Threading.Tasks.Task;
using Cnn.Common.Contracts;

namespace Cnn.Api.Services.Common;

public interface ISystemInfoService
{
    Task<SystemInfoDto> GetAsync(CancellationToken cancellationToken);
}

public sealed class SystemInfoService : ISystemInfoService
{
    private static readonly JsonSerializerOptions JsonOptions = new(JsonSerializerDefaults.Web)
    {
        PropertyNameCaseInsensitive = true
    };

    private readonly ISystemConfigService _systemConfigService;

    public SystemInfoService(ISystemConfigService systemConfigService)
    {
        _systemConfigService = systemConfigService;
    }

    public async Task<SystemInfoDto> GetAsync(CancellationToken cancellationToken)
    {
        var payload = new SystemInfoDto();
        var cfg = await _systemConfigService.LoadSystemConfigAsync(cancellationToken);
        if (cfg.Count == 0)
        {
            return payload;
        }

        if (cfg.TryGetValue("system_info", out var raw) && !string.IsNullOrWhiteSpace(raw))
        {
            try
            {
                var parsed = JsonSerializer.Deserialize<SystemInfoDto>(raw, JsonOptions);
                if (parsed != null)
                {
                    payload = parsed;
                }
            }
            catch
            {
                payload = new SystemInfoDto();
            }
        }

        payload.EnableEmailLogin = _systemConfigService.ParseBoolFlag(Get(cfg, "allow-enable-email-captcha-login"));
        payload.EnableSmsLogin = _systemConfigService.ParseBoolFlag(Get(cfg, "allow-enable-sms-captcha-login"));
        payload.AllowRegister = _systemConfigService.ParseBoolFlag(Get(cfg, "allow_register"));

        return payload;
    }

    private static string? Get(Dictionary<string, string> map, string key)
    {
        return map.TryGetValue(key, out var value) ? value : null;
    }
}
