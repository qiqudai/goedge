using Cnn.Api.Services.Common;
using Cnn.Api.Services.Deletion;

namespace Cnn.Api.Services.Tasks.Workflow;

public interface IResourceDeleteRequestService
{
    Task<ServiceResult<DeleteRequestResult>> RequestDeleteAsync(RequestDeleteCommand command, CancellationToken cancellationToken);
}
