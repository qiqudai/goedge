using Cnn.Common.Contracts.Admin;
using Cnn.Api.Services.Common;

namespace Cnn.Api.Services.Admin;

public interface IForwardMonitorService
{
    Task<ServiceResult<ForwardTrafficResult>> GetTrafficAsync(string? range, string? keyword, long? userId, bool isAdmin, CancellationToken cancellationToken);
    Task<ServiceResult<ForwardRankingResult>> GetRankingAsync(string? range, long? userId, bool isAdmin, CancellationToken cancellationToken);
}
