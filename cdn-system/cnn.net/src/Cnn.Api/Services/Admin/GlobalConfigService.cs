using System.Text.Json;
using System.Text.Json.Serialization;
using Task = System.Threading.Tasks.Task;
using Cnn.Common.Contracts;
using Cnn.Common.Contracts.Admin;
using Cnn.Api.Services.Common;
using Cnn.Domain.Entities;
using SqlSugar;

namespace Cnn.Api.Services.Admin;

public sealed class GlobalConfigService : IGlobalConfigService
{
    private const string ConfigKey = "global_config";
    private const string ConfigType = "system";
    private const string ErrorPageName = "error-page";
    private const string ErrorPageType = "error_page";
    private const string GlobalScopeName = "global";

    private static readonly JsonSerializerOptions JsonOptions = new(JsonSerializerDefaults.Web)
    {
        PropertyNameCaseInsensitive = true,
        DefaultIgnoreCondition = JsonIgnoreCondition.WhenWritingNull
    };

    private readonly ISqlSugarClient _db;
    private readonly IConfigVersionService _configVersionService;

    public GlobalConfigService(ISqlSugarClient db, IConfigVersionService configVersionService)
    {
        _db = db;
        _configVersionService = configVersionService;
    }

    public async Task<ServiceResult<GlobalConfigDto>> GetAsync(CancellationToken cancellationToken)
    {
        var record = await _db.Queryable<Config>()
            .Where(c => c.Name == ConfigKey && c.Type == ConfigType)
            .FirstAsync();

        GlobalConfigDto? config = null;
        if (record == null || string.IsNullOrWhiteSpace(record.Value))
        {
            config = await BuildDefaultAsync();
            await TryCreateAsync(config);
        }
        else
        {
            try
            {
                config = JsonSerializer.Deserialize<GlobalConfigDto>(record.Value, JsonOptions);
            }
            catch
            {
                config = null;
            }

            if (config == null)
            {
                config = await BuildDefaultAsync();
            }
        }

        if (config.ErrorPages == null || config.ErrorPages.Count == 0)
        {
            config.ErrorPages = (await BuildDefaultAsync()).ErrorPages;
        }

        return ServiceResult<GlobalConfigDto>.Ok(config);
    }

    public async Task<ServiceResult<bool>> UpdateAsync(GlobalConfigDto config, CancellationToken cancellationToken)
    {
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
                ScopeId = 0,
                ScopeName = GlobalScopeName,
                Value = payload,
                Enable = true,
                CreateAt = now,
                UpdateAt = now
            };
            await _db.Insertable(record).ExecuteCommandAsync();
        }
        else
        {
            await _db.Updateable<Config>()
                .SetColumns(c => new Config { Value = payload, UpdateAt = now })
                .Where(c => c.Name == ConfigKey && c.Type == ConfigType)
                .ExecuteCommandAsync();
        }

        await _configVersionService.BumpAsync("global_config", Array.Empty<long>(), cancellationToken);
        return ServiceResult<bool>.Ok(true);
    }

    private async Task TryCreateAsync(GlobalConfigDto config)
    {
        var now = DateTime.Now;
        var payload = JsonSerializer.Serialize(config, JsonOptions);
        var record = new Config
        {
            Name = ConfigKey,
            Type = ConfigType,
            ScopeId = 0,
            ScopeName = GlobalScopeName,
            Value = payload,
            Enable = true,
            CreateAt = now,
            UpdateAt = now
        };
        try
        {
            await _db.Insertable(record).ExecuteCommandAsync();
        }
        catch
        {
            // ignore
        }
    }

    private async Task<GlobalConfigDto> BuildDefaultAsync()
    {
        var errorPages = await BuildDefaultErrorPagesAsync();

        return new GlobalConfigDto
        {
            Waf = new WafConfigDto
            {
                Enable = true,
                DefaultBlockAction = "disconnect",
                AutoIpSetEnable = true,
                AutoIpSetThreshold = 200,
                BlockPageRateLimitEnable = true,
                BlockPageRateLimit = 200,
                BlockPageTrafficFree = false,
                BlacklistTimeout = 3600,
                TempWhitelistTimeout = 21600,
                TempWhitelistLimitTotal = 400,
                TempWhitelistLimitUrl = 50,
                PreventTlsHandshake = true,
                BlockUnboundDomain = true,
                DisablePing = false,
                DefaultPageProtection = "auto",
                DefaultPageProtectionThreshold = 100,
                SecretKey = "KPS1CC6oGp",
                NodeLogCleanStrategy = "none",
                CcRuleAutoSwitch = false,
                AntiCcImageSource = "system",
                AntiCcImageCustomUrl = string.Empty,
                AntiCcType = "slide",
                AntiCcDebug = false,
                WellKnownProtectionThreshold = 600,
                ResourceProtectionEnable = false,
                ResourceProtectionThreshold = 50,
                ResourceProtectionBlockTimeout = 3600,
                ResourceProtectionRules = new List<ResourceRuleDto>
                {
                    new(120, 20)
                }
            },
            Nginx = new NginxConfigDto
            {
                WorkerProcesses = "auto",
                WorkerConnections = 51200,
                WorkerRlimitNofile = 51200,
                WorkerShutdownTimeout = "60s",
                LogDirectory = "/usr/local/openresty/nginx/logs/",
                KeepaliveTimeout = 60,
                Gzip = true
            },
            DefaultConfig = new DefaultSiteConfigDto
            {
                Website = new SiteTemplateDto
                {
                    CacheEnable = true,
                    CacheTtl = 86400,
                    Gzip = true,
                    WafEnable = true,
                    SslCiphers = "ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384:ECDHE-ECDSA-CHACHA20-POLY1305:ECDHE-RSA-CHACHA20-POLY1305:DHE-RSA-AES128-GCM-SHA256:DHE-RSA-AES256-GCM-SHA384:DHE-RSA-CHACHA20-POLY1305:ECDHE-ECDSA-AES128-SHA256:ECDHE-RSA-AES128-SHA256:ECDHE-ECDSA-AES128-SHA:ECDHE-RSA-AES128-SHA:ECDHE-ECDSA-AES256-SHA384:ECDHE-RSA-AES256-SHA384:ECDHE-ECDSA-AES256-SHA:ECDHE-RSA-AES256-SHA:DHE-RSA-AES128-SHA256:DHE-RSA-AES256-SHA256:AES128-GCM-SHA256:AES256-GCM-SHA384:AES128-SHA256:AES256-SHA256:AES128-SHA:AES256-SHA:DES-CBC3-SHA"
                },
                Api = new SiteTemplateDto
                {
                    CacheEnable = false,
                    CacheTtl = 0,
                    Gzip = true,
                    WafEnable = true
                },
                Download = new SiteTemplateDto
                {
                    CacheEnable = false,
                    CacheTtl = 0,
                    Gzip = false,
                    WafEnable = true
                }
            },
            Resources = new GlobalResourceConfigDto
            {
                Website = new WebsiteResourceConfigDto
                {
                    MinLimit = 1000,
                    MaxLimitMultiplier = 200,
                    MaxBlacklistIps = 50,
                    MaxWhitelistIps = 50,
                    DailyUrlPurgeLimit = 2000,
                    DailyDirPurgeLimit = 500,
                    DailyPreloadLimit = 2000,
                    DailyUnlockIpLimit = 1000,
                    UnlockIpBatchLimit = 50,
                    MaxCcRulesPerGroup = 5,
                    MaxAclRules = 5,
                    DailyLogDownloadLimit = 10,
                    LogStorageDir = "/data/download-temp/",
                    LogStorageHours = 12,
                    MaxDomainsPerSite = 100,
                    DefaultListen80 = true
                },
                Forward = new ForwardResourceConfigDto
                {
                    DisabledPorts = "80 443",
                    MinLimit = 1000,
                    MaxLimitMultiplier = 200,
                    MaxAclRules = 10
                },
                Public = new PublicResourceConfigDto
                {
                    DisabledCustomPorts = "22",
                    AllowedCustomPorts = "1-65535"
                }
            },
            ErrorPages = errorPages
        };
    }

    private async Task<Dictionary<string, string>> BuildDefaultErrorPagesAsync()
    {
        var source = await LoadErrorPageSourceAsync();
        return new Dictionary<string, string>
        {
            ["400"] = ResolveErrorPage(source, "p400", "<html><body><h1>400 Bad Request</h1><p>Our systems have detected unusual traffic.</p></body></html>"),
            ["403"] = ResolveErrorPage(source, "p403", "<html><body><h1>403 Forbidden</h1><p>Access Denied.</p></body></html>"),
            ["502"] = ResolveErrorPage(source, "p502", "<html><body><h1>502 Bad Gateway</h1><p>The server is busy.</p></body></html>"),
            ["504"] = ResolveErrorPage(source, "p504", "<html><body><h1>504 Gateway Timeout</h1><p>The origin server did not respond.</p></body></html>"),
            ["traffic_limit"] = ResolveErrorPage(source, "p513", "<h1>Traffic Limit Exceeded</h1>"),
            ["site_locked"] = ResolveErrorPage(source, "p514", "<h1>Site Locked</h1>"),
            ["domain_invalid"] = ResolveErrorPage(source, "host_not_found", "<h1>Domain Not Configured</h1>"),
            ["conn_limit"] = ResolveErrorPage(source, "p515", "<h1>Connection Limit Exceeded</h1>"),
            ["timeout"] = ResolveErrorPage(source, "p512", "<h1>Package Expired</h1>"),
            ["ip"] = ResolveErrorPage(source, "access_ip_not_allow", "<h1>IP Forbidden</h1>")
        };
    }

    private async Task<Dictionary<string, string>> LoadErrorPageSourceAsync()
    {
        var record = await _db.Queryable<Config>()
            .Where(c => c.Name == ErrorPageName && c.Type == ErrorPageType && c.ScopeName == GlobalScopeName && c.ScopeId == 0)
            .FirstAsync();

        if (record == null || string.IsNullOrWhiteSpace(record.Value))
        {
            return new Dictionary<string, string>();
        }

        try
        {
            var parsed = JsonSerializer.Deserialize<Dictionary<string, string>>(record.Value, JsonOptions);
            return parsed ?? new Dictionary<string, string>();
        }
        catch
        {
            return new Dictionary<string, string>();
        }
    }

    private static string ResolveErrorPage(Dictionary<string, string> map, string key, string fallback)
    {
        if (map.TryGetValue(key, out var value) && !string.IsNullOrWhiteSpace(value))
        {
            return value;
        }

        return fallback;
    }
}
