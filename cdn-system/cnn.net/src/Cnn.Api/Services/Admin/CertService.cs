using System.Text.Json;
using System.Text.Json.Serialization;
using Cnn.Common.Contracts.Admin;
using Cnn.Common.Contracts.Agent;
using Cnn.Api.Services.Common;
using Cnn.Api.Services.Tasks.Workflow;
using Microsoft.Extensions.Configuration;
using SqlSugar;
using TaskEntity = Cnn.Domain.Entities.Task;
using Task = System.Threading.Tasks.Task;

namespace Cnn.Api.Services.Admin;

public interface ICertService
{
    Task<ServiceResult<CertListResult>> ListAsync(CertListQuery query, long? userId, bool isAdmin, CancellationToken cancellationToken);
    Task<ServiceResult<CertItemDto>> CreateAsync(CertCreateRequest request, long? userId, bool isAdmin, CancellationToken cancellationToken);
    Task<ServiceResult<bool>> UpdateAsync(long id, CertUpdateRequest request, long? userId, bool isAdmin, CancellationToken cancellationToken);
    Task<ServiceResult<bool>> DeleteAsync(long id, long? userId, bool isAdmin, CancellationToken cancellationToken);
    Task<ServiceResult<CertBatchCreateResult>> BatchCreateAsync(CertBatchCreateRequest request, long? userId, bool isAdmin, CancellationToken cancellationToken);
    Task<ServiceResult<CertBatchProgressResult>> BatchProgressAsync(string batchId, long? userId, bool isAdmin, CancellationToken cancellationToken);
    Task<ServiceResult<CertWildcardResult>> WildcardCreateAsync(CertWildcardRequest request, long? userId, bool isAdmin, CancellationToken cancellationToken);
    Task<ServiceResult<CertBatchActionResult>> BatchActionAsync(CertBatchActionRequest request, long? userId, bool isAdmin, CancellationToken cancellationToken);
    Task<ServiceResult<bool>> ReissueAsync(CertReissueRequest request, CancellationToken cancellationToken);
    Task<ServiceResult<DnsChallengeInfoDto?>> GetDnsChallengeAsync(long id, long? userId, bool isAdmin, CancellationToken cancellationToken);
    Task<ServiceResult<bool>> VerifyDnsChallengeAsync(long id, long? userId, bool isAdmin, CancellationToken cancellationToken);
    Task<ServiceResult<CertDownloadPayload>> DownloadAsync(long id, long? userId, bool isAdmin, string? domain, CancellationToken cancellationToken);
    Task<ServiceResult<CertDefaultSettingsDto>> GetDefaultSettingsAsync(long? userId, bool isAdmin, CancellationToken cancellationToken);
    Task<ServiceResult<bool>> UpdateDefaultSettingsAsync(CertDefaultSettingsRequest request, long? userId, bool isAdmin, CancellationToken cancellationToken);
    Task<ServiceResult<bool>> UpdateIssuedCertAsync(AgentIssuedCertRequest request, CancellationToken cancellationToken);
}

public sealed partial class CertService : ICertService
{
    private const string CertDefaultSettingsKey = "cert_default_settings";
    private const string CertDefaultSettingsType = "system";
    private const string CertDefaultScope = "global";
    private const string CertDefaultUserScope = "user";
    private const string SiteSettingsType = "site_settings";
    private const string SiteSettingsScope = "site";

    private static readonly JsonSerializerOptions JsonOptions = new(JsonSerializerDefaults.Web)
    {
        PropertyNameCaseInsensitive = true,
        DefaultIgnoreCondition = JsonIgnoreCondition.WhenWritingNull
    };

    private readonly ISqlSugarClient _db;
    private readonly ICryptoService _cryptoService;
    private readonly IConfigVersionService _configVersionService;
    private readonly IConfiguration _configuration;
    private readonly IResourceActionRequestService _resourceActionRequestService;

    public CertService(
        ISqlSugarClient db,
        ICryptoService cryptoService,
        IConfigVersionService configVersionService,
        IConfiguration configuration,
        IResourceActionRequestService resourceActionRequestService)
    {
        _db = db;
        _cryptoService = cryptoService;
        _configVersionService = configVersionService;
        _configuration = configuration;
        _resourceActionRequestService = resourceActionRequestService;
    }

    private sealed record TaskInfo(string? State, string? Ret, DateTime? RetryAt, int? ErrTimes);
}

public sealed record CertDownloadPayload(byte[] Data, string FileName);
