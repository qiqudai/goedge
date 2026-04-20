using Cnn.Common.Contracts;
using Cnn.Common.Localization;
using Cnn.Api.Responses;
using Cnn.Api.Services.Agent;
using Cnn.Api.Services.Common;
using Microsoft.AspNetCore.Mvc;

namespace Cnn.Api.Controllers.Agent;

[ApiController]
[Route("api/v1/agent/config")]
public sealed class AgentConfigController : ControllerBase
{
    private readonly IEdgeConfigService _service;
    private readonly IAgentApiTraceService _traceService;
    private readonly IMessageLocalizer _localizer;

    public AgentConfigController(IEdgeConfigService service, IAgentApiTraceService traceService, IMessageLocalizer localizer)
    {
        _service = service;
        _traceService = traceService;
        _localizer = localizer;
    }

    [HttpGet]
    public async Task<IActionResult> GetAsync(
        [FromQuery(Name = "node_id")] string? nodeId,
        [FromQuery(Name = "version")] string? version,
        CancellationToken cancellationToken)
    {
        if (string.IsNullOrWhiteSpace(nodeId))
        {
            return Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, ErrorCodes.MissingParam));
        }

        var result = await _service.GenerateAsync(nodeId, cancellationToken);
        if (result.Success)
        {
            if (!string.IsNullOrWhiteSpace(version) &&
                long.TryParse(version.Trim(), out var clientVersion) &&
                result.Data != null &&
                clientVersion == result.Data.Version)
            {
                _ = _traceService.TraceAsync(new AgentApiTraceRecord
                {
                    Direction = "out",
                    Channel = "http",
                    Kind = "agent_config_not_modified",
                    NodeId = nodeId,
                    NodeIp = HttpContext.Connection.RemoteIpAddress?.ToString(),
                    Path = HttpContext.Request.Path.Value,
                    Method = HttpContext.Request.Method,
                    StatusCode = StatusCodes.Status304NotModified,
                    TraceId = HttpContext.TraceIdentifier
                }, cancellationToken);
                return StatusCode(StatusCodes.Status304NotModified);
            }

            _ = _traceService.TraceAsync(new AgentApiTraceRecord
            {
                Direction = "out",
                Channel = "http",
                Kind = "agent_config_response",
                NodeId = nodeId,
                NodeIp = HttpContext.Connection.RemoteIpAddress?.ToString(),
                Path = HttpContext.Request.Path.Value,
                Method = HttpContext.Request.Method,
                StatusCode = StatusCodes.Status200OK,
                TraceId = HttpContext.TraceIdentifier,
                Payload = result.Data == null ? string.Empty : $"version={result.Data.Version}"
            }, cancellationToken);
            return Ok(ApiResponseFactory.Ok(HttpContext, _localizer, result.Data));
        }

        _ = _traceService.TraceAsync(new AgentApiTraceRecord
        {
            Direction = "out",
            Channel = "http",
            Kind = "agent_config_error",
            NodeId = nodeId,
            NodeIp = HttpContext.Connection.RemoteIpAddress?.ToString(),
            Path = HttpContext.Request.Path.Value,
            Method = HttpContext.Request.Method,
            StatusCode = StatusCodes.Status200OK,
            TraceId = HttpContext.TraceIdentifier,
            Payload = result.MessageKey
        }, cancellationToken);
        return Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, result.ErrorCode, result.MessageKey));
    }
}
