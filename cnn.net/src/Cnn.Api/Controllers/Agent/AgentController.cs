using Cnn.Common.Contracts;
using Cnn.Common.Contracts.Agent;
using Cnn.Common.Localization;
using Cnn.Api.Responses;
using Cnn.Api.Services.Agent;
using Cnn.Api.Services.Common;
using Microsoft.AspNetCore.Mvc;

namespace Cnn.Api.Controllers.Agent;

[ApiController]
[Route("api/v1/agent")]
public sealed class AgentController : ControllerBase
{
    private readonly IAgentNodeService _service;
    private readonly IMessageLocalizer _localizer;

    public AgentController(IAgentNodeService service, IMessageLocalizer localizer)
    {
        _service = service;
        _localizer = localizer;
    }

    [HttpPost("heartbeat")]
    public async Task<IActionResult> HeartbeatAsync([FromBody] AgentHeartbeatRequest request, CancellationToken cancellationToken)
    {
        var tokenNodeId = ResolveTokenNodeId();
        var clientIp = ResolveClientIp();
        var result = await _service.HeartbeatAsync(request, tokenNodeId, clientIp, cancellationToken);
        return ToResponse(result);
    }

    [HttpPost("node/sync")]
    public async Task<IActionResult> SyncNodeStatusAsync([FromBody] AgentSyncRequest request, CancellationToken cancellationToken)
    {
        var tokenNodeId = ResolveTokenNodeId();
        var clientIp = ResolveClientIp();
        var result = await _service.SyncNodeStatusAsync(request, tokenNodeId, clientIp, cancellationToken);
        return ToResponse(result);
    }

    [HttpGet("l2/nodes")]
    public async Task<IActionResult> GetL2NodesAsync([FromQuery(Name = "node_id")] string? nodeId, CancellationToken cancellationToken)
    {
        var tokenNodeId = ResolveTokenNodeId();
        var resolved = string.IsNullOrWhiteSpace(tokenNodeId) ? nodeId : tokenNodeId;
        var result = await _service.GetL2NodesAsync(resolved, cancellationToken);
        return ToResponse(result);
    }

    [HttpPost("l2/heartbeat")]
    public async Task<IActionResult> ReportL2HeartbeatAsync([FromBody] AgentL2HeartbeatRequest request, CancellationToken cancellationToken)
    {
        var result = await _service.ReportL2HeartbeatAsync(request, cancellationToken);
        return ToResponse(result);
    }

    private string? ResolveTokenNodeId()
    {
        if (HttpContext.Items.TryGetValue("node_id", out var value) && value != null)
        {
            return value.ToString();
        }

        return null;
    }

    private string? ResolveClientIp()
    {
        return HttpContext.Connection.RemoteIpAddress?.ToString();
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
