using Cnn.Common.Contracts;
using Cnn.Common.Localization;
using Cnn.Api.Responses;
using Cnn.Api.Services.Admin;
using Cnn.Api.Services.Agent;
using Microsoft.AspNetCore.Mvc;

namespace Cnn.Api.Controllers.Agent;

[ApiController]
[Route("api/v1/agent/upgrade")]
public sealed class UpgradeController : ControllerBase
{
    private readonly IAgentPackageService _service;
    private readonly IAgentUpgradeService _upgradeService;
    private readonly IMessageLocalizer _localizer;

    public UpgradeController(IAgentPackageService service, IAgentUpgradeService upgradeService, IMessageLocalizer localizer)
    {
        _service = service;
        _upgradeService = upgradeService;
        _localizer = localizer;
    }

    [HttpGet]
    public async Task<IActionResult> GetAsync([FromQuery(Name = "node_id")] string? nodeId, CancellationToken cancellationToken)
    {
        var resolvedNodeId = ResolveNodeId(nodeId);
        if (resolvedNodeId <= 0)
        {
            return Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, ErrorCodes.MissingParam));
        }

        var result = await _upgradeService.GetUpgradeInfoAsync(resolvedNodeId, cancellationToken);
        if (result.Success)
        {
            return Ok(ApiResponseFactory.Ok(HttpContext, _localizer, result.Data));
        }

        return Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, result.ErrorCode, result.MessageKey));
    }

    [HttpGet("package")]
    public async Task<IActionResult> DownloadAsync([FromQuery] string? version, CancellationToken cancellationToken)
    {
        var result = await _service.ResolveDownloadAsync(version, cancellationToken);
        if (!result.Success || result.Data == null)
        {
            return Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, result.ErrorCode, result.MessageKey));
        }

        if (string.IsNullOrWhiteSpace(result.Data.FilePath) || string.IsNullOrWhiteSpace(result.Data.FileName))
        {
            return Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, ErrorCodes.NotFound, "file_not_found"));
        }

        return PhysicalFile(result.Data.FilePath, "application/octet-stream", result.Data.FileName);
    }

    private long ResolveNodeId(string? queryNodeId)
    {
        if (HttpContext.Items.TryGetValue("node_id", out var value) && value != null)
        {
            if (long.TryParse(value.ToString(), out var nodeId) && nodeId > 0)
            {
                return nodeId;
            }
        }

        var raw = queryNodeId?.Trim();
        if (string.IsNullOrWhiteSpace(raw))
        {
            return 0;
        }

        return long.TryParse(raw, out var parsed) ? parsed : 0;
    }
}
