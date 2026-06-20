using Cnn.Common.Contracts.Admin;
using Cnn.Api.Services.Common;

namespace Cnn.Api.Services.Admin;

public interface IUserGroupService
{
    Task<ServiceResult<UserGroupListResult>> ListAsync(CancellationToken cancellationToken);
    Task<ServiceResult<UserGroupDto>> CreateAsync(UserGroupUpsertRequest request, CancellationToken cancellationToken);
    Task<ServiceResult<bool>> DeleteAsync(long id, CancellationToken cancellationToken);
}

