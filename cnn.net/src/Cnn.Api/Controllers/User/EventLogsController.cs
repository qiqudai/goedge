using Cnn.Common.Contracts.Admin;
using Cnn.Api.Helpers;
using Cnn.Common.Localization;
using Cnn.Api.Responses;
using Cnn.Api.Services;
using Cnn.Api.Services.Admin;
using Microsoft.AspNetCore.Mvc;

namespace Cnn.Api.Controllers.User;

[ApiController]
[Route("api/v1/user/logs/events")]
public sealed class EventLogsController : ControllerBase
{
    private readonly IEventLogService _service;
    private readonly IAdminIdentityResolver _identityResolver;
    private readonly IMessageLocalizer _localizer;

    public EventLogsController(
        IEventLogService service,
        IAdminIdentityResolver identityResolver,
        IMessageLocalizer localizer)
    {
        _service = service;
        _identityResolver = identityResolver;
        _localizer = localizer;
    }

    [HttpGet]
    public async Task<IActionResult> ListAsync([FromQuery] EventLogQuery query, CancellationToken cancellationToken)
    {
        var identity = _identityResolver.Resolve(User);
        if (identity == null)
        {
            return Ok(ApiResponseFactory.Fail<EventLogListResult>(HttpContext, _localizer, ErrorCodes.AuthInvalid));
        }

        var (start, end) = RequestTimeRange.Resolve(HttpContext.Request);
        var result = await _service.ListAsync(query, start, end, identity.UserId, isAdmin: false, cancellationToken);
        return ToResponse(result);
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
