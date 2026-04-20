using Cnn.Api.Services.Common;

namespace Cnn.Api.Services.Tasks.Workflow;

public interface IResourceActionRequestService
{
    Task<ServiceResult<TaskRequestResult>> RequestAsync(RequestActionCommand command, CancellationToken cancellationToken);
}
