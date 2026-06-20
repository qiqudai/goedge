using Cnn.Common.Contracts.Admin;
using Cnn.Api.Services.Common;

namespace Cnn.Api.Services.Admin;

public interface IAnnouncementService
{
    Task<ServiceResult<AnnouncementListResult>> ListAsync(AnnouncementListQuery query, CancellationToken cancellationToken);
    Task<ServiceResult<bool>> CreateAsync(AnnouncementUpsertRequest request, CancellationToken cancellationToken);
    Task<ServiceResult<bool>> UpdateAsync(long id, AnnouncementUpsertRequest request, CancellationToken cancellationToken);
    Task<ServiceResult<bool>> DeleteAsync(long id, CancellationToken cancellationToken);
}
