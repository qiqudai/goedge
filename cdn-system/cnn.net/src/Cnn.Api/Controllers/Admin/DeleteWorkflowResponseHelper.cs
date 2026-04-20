using Cnn.Api.Responses;
using Cnn.Api.Services.Deletion;
using Cnn.Api.Services.Tasks.Workflow;
using Cnn.Common.Localization;
using Microsoft.AspNetCore.Mvc;

namespace Cnn.Api.Controllers.Admin;

internal static class DeleteWorkflowResponseHelper
{
    public static async Task<IActionResult> PreviewAsync(
        ControllerBase controller,
        IMessageLocalizer localizer,
        IDeletionPreviewService deletionPreviewService,
        string resourceType,
        long resourceId,
        CancellationToken cancellationToken)
    {
        var result = await deletionPreviewService.PreviewAsync(resourceType, resourceId, cancellationToken);
        return controller.Ok(ApiResponseFactory.Ok(controller.HttpContext, localizer, result));
    }

    public static async Task<IActionResult> RequestAsync(
        ControllerBase controller,
        IMessageLocalizer localizer,
        IResourceDeleteRequestService resourceDeleteRequestService,
        RequestDeleteCommand command,
        CancellationToken cancellationToken)
    {
        var result = await resourceDeleteRequestService.RequestDeleteAsync(command, cancellationToken);
        if (result.Success)
        {
            return controller.Ok(ApiResponseFactory.Ok(controller.HttpContext, localizer, result.Data));
        }

        return controller.Ok(ApiResponseFactory.Fail<DeleteRequestResult>(
            controller.HttpContext,
            localizer,
            result.ErrorCode,
            result.MessageKey,
            data: result.Data));
    }
}
