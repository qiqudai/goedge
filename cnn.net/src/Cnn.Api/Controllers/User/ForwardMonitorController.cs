using Cnn.Common.Contracts;
using Cnn.Common.Localization;
using Cnn.Api.Responses;
using Cnn.Api.Services;
using Cnn.Api.Services.Admin;
using Cnn.Api.Services.Common;
using Microsoft.AspNetCore.Mvc;

namespace Cnn.Api.Controllers.User;

[ApiController]
[Route("api/v1/user/forward")]
public sealed class ForwardMonitorController : ControllerBase
{
    private readonly IForwardMonitorService _service;
    private readonly IMessageLocalizer _localizer;
    private readonly IAdminIdentityResolver _identityResolver;

    public ForwardMonitorController(IForwardMonitorService service, IMessageLocalizer localizer, IAdminIdentityResolver identityResolver)
    {
        _service = service;
        _localizer = localizer;
        _identityResolver = identityResolver;
    }

    [HttpGet("traffic")]
    public async Task<IActionResult> TrafficAsync(
        [FromQuery(Name = "range")] string? range,
        [FromQuery(Name = "keyword")] string? keyword,
        CancellationToken cancellationToken)
    {
        var userId = ResolveUserId();
        var result = await _service.GetTrafficAsync(range, keyword, userId, false, cancellationToken);
        return ToResponse(result);
    }

    [HttpGet("ranking")]
    public async Task<IActionResult> RankingAsync(
        [FromQuery(Name = "range")] string? range,
        CancellationToken cancellationToken)
    {
        var userId = ResolveUserId();
        var result = await _service.GetRankingAsync(range, userId, false, cancellationToken);
        return ToResponse(result);
    }

    private long? ResolveUserId()
    {
        var identity = _identityResolver.Resolve(User);
        return identity?.UserId;
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
