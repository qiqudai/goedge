using Cnn.Common.Contracts;
using Cnn.Api.Services.Stats;
using Task = System.Threading.Tasks.Task;

namespace Cnn.Api.Services.Common;

public interface IUsageService
{
    Task<ServiceResult<UsageResultDto>> GetUsageAsync(string? range, AccessScope scope, CancellationToken cancellationToken);
}

public sealed class UsageService : IUsageService
{
    private readonly IStatsService _statsService;

    public UsageService(IStatsService statsService)
    {
        _statsService = statsService;
    }

    public async Task<ServiceResult<UsageResultDto>> GetUsageAsync(string? range, AccessScope scope, CancellationToken cancellationToken)
    {
        if (!scope.HasUserId)
        {
            return ServiceResult<UsageResultDto>.Fail(ErrorCodes.InvalidParam, "user_id_required");
        }

        return await _statsService.GetUsageAsync(range, scope, cancellationToken);
    }
}
