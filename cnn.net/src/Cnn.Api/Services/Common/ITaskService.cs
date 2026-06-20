using Cnn.Common.Contracts;

namespace Cnn.Api.Services.Common;

public interface ITaskService
{
    Task<ServiceResult<TaskListResult>> ListAdminAsync(TaskListQuery query, CancellationToken cancellationToken);
    Task<ServiceResult<TaskListResult>> ListUserAsync(TaskListQuery query, long userId, CancellationToken cancellationToken);
    Task<ServiceResult<TaskDetailDto>> GetAdminAsync(long id, CancellationToken cancellationToken);
    Task<ServiceResult<TaskDetailDto>> GetUserAsync(long id, long userId, CancellationToken cancellationToken);
    Task<ServiceResult<TaskUsagePayload>> GetUsageAsync(long userId, CancellationToken cancellationToken);
    Task<ServiceResult<bool>> CreateAsync(TaskCreateRequest request, long userId, bool adminMode, CancellationToken cancellationToken);
    Task<ServiceResult<bool>> ResubmitAsync(long id, long userId, bool adminMode, CancellationToken cancellationToken);
}
