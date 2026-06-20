using Cnn.Common.Contracts;
using Cnn.Common.Localization;
using Cnn.Api.Responses;
using Cnn.Api.Services;
using Cnn.Api.Services.Common;
using Cnn.Api.Services.Stats;
using Microsoft.AspNetCore.Mvc;

namespace Cnn.Api.Controllers;

public abstract class BaseApiController : ControllerBase
{
    protected readonly IAdminIdentityResolver _identityResolver;
    protected readonly IMessageLocalizer _localizer;

    protected BaseApiController(IAdminIdentityResolver identityResolver, IMessageLocalizer localizer)
    {
        _identityResolver = identityResolver;
        _localizer = localizer;
    }

    protected long? ResolveUserIdNullable()
    {
        var identity = _identityResolver.Resolve(User);
        return identity?.UserId;
    }

    protected long ResolveUserId()
    {
        return ResolveUserIdNullable() ?? 0;
    }

    protected IActionResult ToResponse<T>(ServiceResult<T> result)
    {
        if (result.Success)
        {
            return Ok(ApiResponseFactory.Ok(HttpContext, _localizer, result.Data));
        }

        return Ok(ApiResponseFactory.Fail<T>(HttpContext, _localizer, result.ErrorCode, result.MessageKey));
    }

    protected StatsRange ResolveStatsRangeFromRequest()
    {
        var query = HttpContext.Request.Query;
        var rangeKey = query["time_range"].ToString();
        if (string.IsNullOrWhiteSpace(rangeKey))
        {
            rangeKey = query["range"].ToString();
        }

        var (startRaw, endRaw) = ResolveCustomRangeParams();
        return StatsRangeResolver.Resolve(rangeKey, startRaw, endRaw, DateTime.Now);
    }

    protected (string? Start, string? End) ResolveCustomRangeParams()
    {
        var query = HttpContext.Request.Query;
        var start = query["start_time"].ToString();
        var end = query["end_time"].ToString();

        if (string.IsNullOrWhiteSpace(start) || string.IsNullOrWhiteSpace(end))
        {
            if (query.TryGetValue("timeRange[]", out var values) && values.Count >= 2)
            {
                start = values[0];
                end = values[1];
            }
        }

        if (string.IsNullOrWhiteSpace(start) || string.IsNullOrWhiteSpace(end))
        {
            if (query.TryGetValue("timeRange", out var values) && values.Count >= 2)
            {
                start = values[0];
                end = values[1];
            }
        }

        return (start, end);
    }
}
