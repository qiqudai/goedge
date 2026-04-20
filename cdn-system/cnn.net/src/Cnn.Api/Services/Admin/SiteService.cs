using System.Text.Json;
using System.Text.Json.Serialization;
using Cnn.Common.Contracts;
using Cnn.Common.Contracts.Admin;
using Cnn.Api.Services.Common;
using Cnn.Api.Services.Tasks.Workflow;
using Cnn.Domain.Entities;
using SqlSugar;

namespace Cnn.Api.Services.Admin;

public sealed partial class SiteService : ISiteService
{
    private const string DefaultCnameDomain = "cdn.node.com";

    private static readonly JsonSerializerOptions JsonOptions = new(JsonSerializerDefaults.Web)
    {
        PropertyNameCaseInsensitive = true,
        DefaultIgnoreCondition = JsonIgnoreCondition.WhenWritingNull
    };

    private readonly ISqlSugarClient _db;
    private readonly IConfigVersionService _configVersionService;
    private readonly IDnsSyncService _dnsSyncService;
    private readonly ISystemConfigService _systemConfigService;
    private readonly ICryptoService _cryptoService;
    private readonly IGlobalConfigService _globalConfigService;
    private readonly ICertService _certService;
    private readonly ISiteSettingsStore _siteSettingsStore;
    private readonly IResourceActionRequestService _resourceActionRequestService;

    public SiteService(
        ISqlSugarClient db,
        IConfigVersionService configVersionService,
        IDnsSyncService dnsSyncService,
        ISystemConfigService systemConfigService,
        ICryptoService cryptoService,
        IGlobalConfigService globalConfigService,
        ICertService certService,
        ISiteSettingsStore siteSettingsStore,
        IResourceActionRequestService resourceActionRequestService)
    {
        _db = db;
        _configVersionService = configVersionService;
        _dnsSyncService = dnsSyncService;
        _systemConfigService = systemConfigService;
        _cryptoService = cryptoService;
        _globalConfigService = globalConfigService;
        _certService = certService;
        _siteSettingsStore = siteSettingsStore;
        _resourceActionRequestService = resourceActionRequestService;
    }
}
