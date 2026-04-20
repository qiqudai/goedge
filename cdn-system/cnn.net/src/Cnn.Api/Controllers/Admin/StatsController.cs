using Cnn.Common.Contracts;
using Cnn.Common.Localization;
using Cnn.Api.Responses;
using Cnn.Api.Services.Stats;
using Cnn.Api.Services.Common;
using Microsoft.AspNetCore.Mvc;

namespace Cnn.Api.Controllers.Admin;

[ApiController]
[Route("api/v1/admin/stats")]
public sealed class StatsController : BaseApiController
{
    private readonly IStatsService _service;

    public StatsController(IStatsService service, Cnn.Api.Services.IAdminIdentityResolver identityResolver, IMessageLocalizer localizer) : base(identityResolver, localizer)
    {
        _service = service;
    }

    [HttpGet("ranking")]
    public async Task<IActionResult> RankingAsync([FromQuery] string? type, [FromQuery] string? keyword, CancellationToken cancellationToken)
    {
        type = string.IsNullOrWhiteSpace(type) ? "domain" : type.Trim().ToLowerInvariant();
        var range = ResolveStatsRangeFromRequest();

        if (type == "latency")
        {
            var latency = await _service.GetLatencyRankingAsync(keyword, range, AccessScope.Admin(), cancellationToken);
            return ToResponse(latency);
        }

        var result = await _service.GetRankingAsync(type, keyword, range, AccessScope.Admin(), cancellationToken);
        return ToResponse(result);
    }

    [HttpGet("basic")]
    public async Task<IActionResult> BasicAsync(CancellationToken cancellationToken)
    {
        var range = ResolveStatsRangeFromRequest();
        var result = await _service.GetBasicAsync(range, AccessScope.Admin(), cancellationToken);
        return ToResponse(result);
    }

    [HttpGet("quality")]
    public async Task<IActionResult> QualityAsync(CancellationToken cancellationToken)
    {
        var range = ResolveStatsRangeFromRequest();
        var result = await _service.GetQualityAsync(range, AccessScope.Admin(), cancellationToken);
        return ToResponse(result);
    }

    [HttpGet("origin")]
    public async Task<IActionResult> OriginAsync(CancellationToken cancellationToken)
    {
        var range = ResolveStatsRangeFromRequest();
        var result = await _service.GetOriginAsync(range, AccessScope.Admin(), cancellationToken);
        return ToResponse(result);
    }

    [HttpGet("node_traffic")]
    public async Task<IActionResult> NodeTrafficAsync([FromQuery] string? window, CancellationToken cancellationToken)
    {
        var result = await _service.GetNodeTrafficAsync(window, cancellationToken);
        return ToResponse(result);
    }

    [HttpGet("node_ranking")]
    public async Task<IActionResult> NodeRankingAsync([FromQuery] string? metric, [FromQuery] string? window, CancellationToken cancellationToken)
    {
        var result = await _service.GetNodeRankingAsync(metric, window, cancellationToken);
        return ToResponse(result);
    }

    [HttpGet("node_metrics")]
    public async Task<IActionResult> NodeMetricsAsync([FromQuery] string? metric, [FromQuery] string? window, [FromQuery(Name = "start_time")] string? startTime, [FromQuery(Name = "end_time")] string? endTime, CancellationToken cancellationToken)
    {
        var result = await _service.GetNodeMetricsAsync(metric, window, startTime, endTime, cancellationToken);
        return ToResponse(result);
    }

}
