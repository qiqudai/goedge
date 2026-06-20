using Cnn.Common.Contracts.Admin;
using Cnn.Api.Helpers;
using Cnn.Common.Localization;
using Cnn.Api.Responses;
using Cnn.Api.Services.Admin;
using Microsoft.AspNetCore.Mvc;

namespace Cnn.Api.Controllers.Admin;

[ApiController]
[Route("api/v1/admin/logs/events")]
public sealed class EventLogsController : ControllerBase
{
    private readonly IEventLogService _service;
    private readonly IMessageLocalizer _localizer;

    public EventLogsController(IEventLogService service, IMessageLocalizer localizer)
    {
        _service = service;
        _localizer = localizer;
    }

    [HttpGet]
    public async Task<IActionResult> ListAsync([FromQuery] EventLogQuery query, CancellationToken cancellationToken)
    {
        var (start, end) = RequestTimeRange.Resolve(HttpContext.Request);
        var result = await _service.ListAsync(query, start, end, userId: null, isAdmin: true, cancellationToken);
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
