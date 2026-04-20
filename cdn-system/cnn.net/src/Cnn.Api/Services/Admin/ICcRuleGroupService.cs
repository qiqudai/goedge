using Cnn.Common.Contracts.Admin;
using Cnn.Api.Services.Common;

namespace Cnn.Api.Services.Admin;

public interface ICcRuleGroupService
{
    Task<ServiceResult<CcListResult<CcRuleGroupListItem>>> ListAsync(CcListQuery query, long? userId, bool userScope, CancellationToken cancellationToken);

    Task<ServiceResult<CcRuleGroupDetailDto>> GetAsync(long id, CancellationToken cancellationToken);

    Task<ServiceResult<bool>> CreateAsync(CcRuleGroupUpsertRequest request, long? userId, bool isUserRequest, CancellationToken cancellationToken);

    Task<ServiceResult<bool>> UpdateAsync(long id, CcRuleGroupUpsertRequest request, long? userId, bool isUserRequest, CancellationToken cancellationToken);

    Task<ServiceResult<bool>> DeleteAsync(long id, long? userId, bool isUserRequest, CancellationToken cancellationToken);
}
