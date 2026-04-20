using Cnn.Api.Services.Common;
using Cnn.Common.Contracts;

namespace Cnn.Api.Services.Tasks.Workflow;

public sealed class ResourceActionRequestService : IResourceActionRequestService
{
    private readonly ITaskCommandFactory _taskCommandFactory;

    public ResourceActionRequestService(ITaskCommandFactory taskCommandFactory)
    {
        _taskCommandFactory = taskCommandFactory;
    }

    public async Task<ServiceResult<TaskRequestResult>> RequestAsync(RequestActionCommand command, CancellationToken cancellationToken)
    {
        if (string.IsNullOrWhiteSpace(command.TaskType))
        {
            return ServiceResult<TaskRequestResult>.Fail(ErrorCodes.InvalidParam, "resource_action_invalid");
        }

        var task = await _taskCommandFactory.CreateAsync(new CreateTaskCommand
        {
            TaskType = command.TaskType,
            OwnerUserId = command.OwnerUserId,
            OperatorUserId = command.OperatorUserId,
            ResourceType = command.ResourceType,
            ResourceId = command.ResourceId,
            DedupeKey = command.DedupeKey,
            PayloadJson = string.IsNullOrWhiteSpace(command.PayloadJson) ? "{}" : command.PayloadJson
        }, cancellationToken);

        return ServiceResult<TaskRequestResult>.Ok(task);
    }
}
