using System.Text.Json;
using Cnn.Api.Services.Common;
using Cnn.Api.Services.Deletion;
using Cnn.Common.Contracts;

namespace Cnn.Api.Services.Tasks.Workflow;

public sealed class ResourceDeleteRequestService : IResourceDeleteRequestService
{
    private readonly IDeletionPreviewService _deletionPreviewService;
    private readonly ITaskCommandFactory _taskCommandFactory;

    public ResourceDeleteRequestService(
        IDeletionPreviewService deletionPreviewService,
        ITaskCommandFactory taskCommandFactory)
    {
        _deletionPreviewService = deletionPreviewService;
        _taskCommandFactory = taskCommandFactory;
    }

    public async Task<ServiceResult<DeleteRequestResult>> RequestDeleteAsync(RequestDeleteCommand command, CancellationToken cancellationToken)
    {
        if (string.IsNullOrWhiteSpace(command.ResourceType) || command.ResourceId <= 0)
        {
            return ServiceResult<DeleteRequestResult>.Fail(ErrorCodes.InvalidParam, "resource_delete_invalid");
        }

        var preview = await _deletionPreviewService.PreviewAsync(command.ResourceType, command.ResourceId, cancellationToken);
        if (!preview.CanDelete)
        {
            return ServiceResult<DeleteRequestResult>.FailWithData(
                ErrorCodes.InUse,
                new DeleteRequestResult
                {
                    Queued = false,
                    ErrorCode = preview.ErrorCode,
                    Message = preview.Message,
                    References = preview.References
                },
                preview.ErrorCode ?? "resource_in_use");
        }

        var taskType = ResolveDeleteTaskType(command.ResourceType);
        if (taskType == null)
        {
            return ServiceResult<DeleteRequestResult>.Fail(ErrorCodes.InvalidParam, "resource_delete_unsupported");
        }

        var payload = JsonSerializer.Serialize(new
        {
            resource_type = command.ResourceType,
            resource_id = command.ResourceId,
            owner_user_id = command.OwnerUserId,
            operator_user_id = command.OperatorUserId,
            reason = command.RequestedReason
        });

        var task = await _taskCommandFactory.CreateAsync(new CreateTaskCommand
        {
            TaskType = taskType,
            OwnerUserId = command.OwnerUserId,
            OperatorUserId = command.OperatorUserId,
            ResourceType = command.ResourceType,
            ResourceId = command.ResourceId,
            DedupeKey = command.DedupeKey ?? $"{taskType}:{command.ResourceId}",
            PayloadJson = payload
        }, cancellationToken);

        return ServiceResult<DeleteRequestResult>.Ok(new DeleteRequestResult
        {
            Queued = true,
            Task = task,
            Message = "delete task queued"
        });
    }

    private static string? ResolveDeleteTaskType(string resourceType)
    {
        return resourceType.ToLowerInvariant() switch
        {
            ResourceTypes.Node => AsyncTaskTypes.NodeDelete,
            ResourceTypes.Certificate => AsyncTaskTypes.CertificateDelete,
            ResourceTypes.Subscription => AsyncTaskTypes.SubscriptionDelete,
            ResourceTypes.LineGroup => AsyncTaskTypes.LineGroupDelete,
            ResourceTypes.SecurityRule => AsyncTaskTypes.SecurityRuleDelete,
            ResourceTypes.CcRuleGroup => AsyncTaskTypes.SecurityRuleDelete,
            ResourceTypes.CcMatcher => AsyncTaskTypes.SecurityRuleDelete,
            ResourceTypes.CcFilter => AsyncTaskTypes.SecurityRuleDelete,
            ResourceTypes.AclRule => AsyncTaskTypes.AclRuleDelete,
            ResourceTypes.ProductPlan => AsyncTaskTypes.ProductPlanDelete,
            ResourceTypes.SiteGroup => AsyncTaskTypes.SiteGroupDelete,
            ResourceTypes.Site => AsyncTaskTypes.SiteDelete,
            ResourceTypes.StreamApp => AsyncTaskTypes.StreamDelete,
            ResourceTypes.StreamGroup => AsyncTaskTypes.StreamGroupDelete,
            ResourceTypes.UserAccount => AsyncTaskTypes.UserPurge,
            _ => null
        };
    }
}
