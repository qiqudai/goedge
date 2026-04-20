using Cnn.Api.Services;
using Cnn.Api.Services.Deletion;
using Cnn.Api.Services.Tasks.Workflow;
using Microsoft.JSInterop;

namespace Cnn.Api.Shared;

public sealed class DeleteWorkflowResult
{
    public bool Success { get; init; }
    public bool Cancelled { get; init; }
    public string Message { get; init; } = string.Empty;
    public long TaskId { get; init; }
}

public static class DeleteWorkflowHelper
{
    public static async Task<DeleteWorkflowResult> PreviewAsync(
        ApiClient api,
        string resourcePath,
        ApiScope scope,
        string previewFailedMessage = "无法获取删除预览")
    {
        var response = await api.GetAsync<DeleteGuardResult>($"{resourcePath}/delete_preview", scope);
        if (response?.Data == null)
        {
            return Fail(response?.Message ?? previewFailedMessage);
        }

        if (!response.Data.CanDelete)
        {
            return Fail(BuildBlockedMessage(response.Data.Message, response.Data.References));
        }

        return Ok();
    }

    public static async Task<DeleteWorkflowResult> RequestAsync(
        ApiClient api,
        string resourcePath,
        ApiScope scope,
        string successMessage,
        string requestFailedMessage = "删除请求提交失败")
    {
        var response = await api.PostAsync<DeleteRequestResult>($"{resourcePath}/delete_request", null, scope);
        if (response == null)
        {
            return Fail(requestFailedMessage);
        }

        if (response.Code != ErrorCodes.Success || response.Data?.Queued != true)
        {
            return Fail(BuildBlockedMessage(response.Data?.Message ?? response.Message, response.Data?.References));
        }

        var taskId = response.Data.Task?.TaskId ?? 0;
        var message = taskId > 0 ? $"{successMessage}：{taskId}" : successMessage;
        return Ok(message, taskId);
    }

    public static async Task<DeleteWorkflowResult> ConfirmAndRequestAsync(
        ApiClient api,
        IJSRuntime js,
        string resourcePath,
        ApiScope scope,
        string confirmMessage,
        string successMessage,
        string previewFailedMessage = "无法获取删除预览",
        string requestFailedMessage = "删除请求提交失败")
    {
        var previewResult = await PreviewAsync(api, resourcePath, scope, previewFailedMessage);
        if (!previewResult.Success)
        {
            return previewResult;
        }

        var confirmed = await js.InvokeAsync<bool>("confirm", confirmMessage);
        if (!confirmed)
        {
            return new DeleteWorkflowResult
            {
                Cancelled = true
            };
        }

        return await RequestAsync(api, resourcePath, scope, successMessage, requestFailedMessage);
    }

    public static string BuildBlockedMessage(string? message, IReadOnlyList<DeleteReferenceItem>? references)
    {
        var lines = new List<string>();
        if (!string.IsNullOrWhiteSpace(message))
        {
            lines.Add(message.Trim());
        }

        foreach (var item in references?.Take(5) ?? Array.Empty<DeleteReferenceItem>())
        {
            lines.Add($"- {item.DisplayName} [{item.Relation}]");
        }

        if ((references?.Count ?? 0) > 5)
        {
            lines.Add($"- 还有 {references!.Count - 5} 项依赖未展示");
        }

        return lines.Count == 0 ? "当前资源暂时无法删除" : string.Join("\n", lines);
    }

    private static DeleteWorkflowResult Ok(string message = "", long taskId = 0)
    {
        return new DeleteWorkflowResult
        {
            Success = true,
            Message = message,
            TaskId = taskId
        };
    }

    private static DeleteWorkflowResult Fail(string message)
    {
        return new DeleteWorkflowResult
        {
            Message = message
        };
    }
}
