using Cnn.Common.Contracts;
using Cnn.Common.Localization;
using Cnn.Api.Responses;
using Cnn.Api.Services.Agent;
using Cnn.Api.Services.Common;
using Microsoft.AspNetCore.Mvc;

namespace Cnn.Api.Controllers.Agent;

[ApiController]
[Route("api/v1/agent/tasks")]
public sealed class TasksController : ControllerBase
{
    private readonly IAgentTaskService _service;
    private readonly IMessageLocalizer _localizer;

    public TasksController(IAgentTaskService service, IMessageLocalizer localizer)
    {
        _service = service;
        _localizer = localizer;
    }

    [HttpGet]
    public async Task<IActionResult> ListAsync([FromQuery(Name = "node_id")] string? nodeId, CancellationToken cancellationToken)
    {
        var resolvedNodeId = ResolveNodeId(nodeId);
        var result = await _service.ListAsync(resolvedNodeId, cancellationToken);
        return ToResponse(result);
    }

    [HttpPost("{id:long}/finish")]
    public async Task<IActionResult> FinishAsync(
        long id,
        [FromBody] AgentTaskFinishRequest request,
        [FromQuery(Name = "node_id")] string? nodeId,
        CancellationToken cancellationToken)
    {
        var resolvedNodeId = ResolveNodeId(nodeId);
        var result = await _service.FinishAsync(id, resolvedNodeId, request, cancellationToken);
        return ToResponse(result);
    }

    private string? ResolveNodeId(string? queryNodeId)
    {
        if (HttpContext.Items.TryGetValue("node_id", out var value) && value != null)
        {
            return value.ToString();
        }

        var nodeId = queryNodeId?.Trim();
        return string.IsNullOrWhiteSpace(nodeId) ? null : nodeId;
    }

    private IActionResult ToResponse<T>(ServiceResult<T> result)
    {
        if (result.Success)
        {
            return Ok(ApiResponseFactory.Ok(HttpContext, _localizer, result.Data));
        }

        return Ok(ApiResponseFactory.Fail<T>(HttpContext, _localizer, result.ErrorCode, result.MessageKey));
    }
}
