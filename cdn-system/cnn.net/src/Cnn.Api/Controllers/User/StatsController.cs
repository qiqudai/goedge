using Cnn.Common.Contracts;
using Cnn.Common.Localization;
using Cnn.Api.Responses;
using Cnn.Api.Services;
using Cnn.Api.Services.Common;
using Cnn.Api.Services.Stats;
using Microsoft.AspNetCore.Mvc;

namespace Cnn.Api.Controllers.User;

[ApiController]
[Route("api/v1/user/stats")]
public sealed class StatsController : BaseApiController
{
    private readonly IStatsService _service;

    public StatsController(IStatsService service, IAdminIdentityResolver identityResolver, IMessageLocalizer localizer) : base(identityResolver, localizer)
    {
        _service = service;
    }

    [HttpGet("ranking")]
    public async Task<IActionResult> RankingAsync([FromQuery] string? type, [FromQuery] string? keyword, CancellationToken cancellationToken)
    {
        type = string.IsNullOrWhiteSpace(type) ? "domain" : type.Trim().ToLowerInvariant();
        var range = ResolveStatsRangeFromRequest();
        var userId = ResolveUserId();

        if (type == "latency")
        {
            var latency = await _service.GetLatencyRankingAsync(keyword, range, AccessScope.User(userId), cancellationToken);
            return ToResponse(latency);
        }

        var result = await _service.GetRankingAsync(type, keyword, range, AccessScope.User(userId), cancellationToken);
        return ToResponse(result);
    }

    [HttpGet("basic")]
    public async Task<IActionResult> BasicAsync(CancellationToken cancellationToken)
    {
        var range = ResolveStatsRangeFromRequest();
        var result = await _service.GetBasicAsync(range, AccessScope.User(ResolveUserId()), cancellationToken);
        return ToResponse(result);
    }

    [HttpGet("quality")]
    public async Task<IActionResult> QualityAsync(CancellationToken cancellationToken)
    {
        var range = ResolveStatsRangeFromRequest();
        var result = await _service.GetQualityAsync(range, AccessScope.User(ResolveUserId()), cancellationToken);
        return ToResponse(result);
    }

    [HttpGet("origin")]
    public async Task<IActionResult> OriginAsync(CancellationToken cancellationToken)
    {
        var range = ResolveStatsRangeFromRequest();
        var result = await _service.GetOriginAsync(range, AccessScope.User(ResolveUserId()), cancellationToken);
        return ToResponse(result);
    }

}
