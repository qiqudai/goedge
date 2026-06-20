using Cnn.Common.Contracts;
using Cnn.Common.Contracts.Agent;
using Cnn.Common.Localization;
using Cnn.Api.Responses;
using Cnn.Api.Services.Agent;
using Microsoft.AspNetCore.Mvc;

namespace Cnn.Api.Controllers.Agent;

[ApiController]
[Route("api/v1/agent/logs")]
public sealed class LogsController : ControllerBase
{
    private readonly IAgentLogService _service;
    private readonly IAgentApiTraceService _traceService;
    private readonly IMessageLocalizer _localizer;

    public LogsController(IAgentLogService service, IAgentApiTraceService traceService, IMessageLocalizer localizer)
    {
        _service = service;
        _traceService = traceService;
        _localizer = localizer;
    }

    [HttpPost("access")]
    public async Task<IActionResult> AccessAsync([FromBody] AgentAccessLogRequest request, CancellationToken cancellationToken)
    {
        if (request == null)
        {
            return Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, ErrorCodes.InvalidParam));
        }

        var nodeId = ResolveNodeId(request.NodeId);
        var nodeIp = ResolveNodeIp(request.NodeIp);
        var inserted = await _service.InsertAccessLogsAsync(nodeId, nodeIp, request.Lines, cancellationToken);
        _ = _traceService.TraceAsync(new AgentApiTraceRecord
        {
            Direction = "in",
            Channel = "http",
            Kind = "agent_logs_access",
            NodeId = nodeId,
            NodeIp = nodeIp,
            Path = HttpContext.Request.Path.Value,
            Method = HttpContext.Request.Method,
            StatusCode = StatusCodes.Status200OK,
            TraceId = HttpContext.TraceIdentifier,
            Payload = $"lines={request.Lines.Count},inserted={inserted}"
        }, cancellationToken);
        return Ok(ApiResponseFactory.Ok(HttpContext, _localizer, new { status = "ok" }));
    }

    [HttpPost("metrics")]
    public async Task<IActionResult> MetricsAsync([FromBody] AgentMetricLogRequest request, CancellationToken cancellationToken)
    {
        if (request == null)
        {
            return Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, ErrorCodes.InvalidParam));
        }

        var nodeId = ResolveNodeId(request.NodeId);
        var nodeIp = ResolveNodeIp(request.NodeIp);
        var inserted = await _service.InsertMetricsAsync(nodeId, nodeIp, request.Content, cancellationToken);
        _ = _traceService.TraceAsync(new AgentApiTraceRecord
        {
            Direction = "in",
            Channel = "http",
            Kind = "agent_logs_metrics",
            NodeId = nodeId,
            NodeIp = nodeIp,
            Path = HttpContext.Request.Path.Value,
            Method = HttpContext.Request.Method,
            StatusCode = StatusCodes.Status200OK,
            TraceId = HttpContext.TraceIdentifier,
            Payload = $"inserted={inserted}"
        }, cancellationToken);
        return Ok(ApiResponseFactory.Ok(HttpContext, _localizer, new { status = "ok" }));
    }

    [HttpPost("events")]
    public async Task<IActionResult> EventsAsync([FromBody] AgentEventLogRequest request, CancellationToken cancellationToken)
    {
        if (request == null)
        {
            return Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, ErrorCodes.InvalidParam));
        }

        var nodeId = ResolveNodeId(request.NodeId);
        var nodeIp = ResolveNodeIp(request.NodeIp);
        var inserted = await _service.InsertEventLogsAsync(nodeId, nodeIp, request.Type, request.Payloads, cancellationToken);
        _ = _traceService.TraceAsync(new AgentApiTraceRecord
        {
            Direction = "in",
            Channel = "http",
            Kind = "agent_logs_events",
            NodeId = nodeId,
            NodeIp = nodeIp,
            Path = HttpContext.Request.Path.Value,
            Method = HttpContext.Request.Method,
            StatusCode = StatusCodes.Status200OK,
            TraceId = HttpContext.TraceIdentifier,
            Payload = $"type={request.Type},payloads={request.Payloads.Count},inserted={inserted}"
        }, cancellationToken);
        return Ok(ApiResponseFactory.Ok(HttpContext, _localizer, new { status = "ok" }));
    }

    private string? ResolveNodeId(string? nodeId)
    {
        if (!string.IsNullOrWhiteSpace(nodeId))
        {
            return nodeId.Trim();
        }

        if (HttpContext.Items.TryGetValue("node_id", out var value) && value != null)
        {
            return value.ToString();
        }

        return null;
    }

    private string? ResolveNodeIp(string? nodeIp)
    {
        if (!string.IsNullOrWhiteSpace(nodeIp))
        {
            return nodeIp.Trim();
        }

        return HttpContext.Connection.RemoteIpAddress?.ToString();
    }
}
