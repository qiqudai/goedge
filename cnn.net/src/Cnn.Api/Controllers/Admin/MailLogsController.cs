using Cnn.Common.Contracts.Admin;
using Cnn.Api.Helpers;
using Cnn.Common.Localization;
using Cnn.Api.Responses;
using Cnn.Api.Services.Admin;
using Microsoft.AspNetCore.Mvc;

namespace Cnn.Api.Controllers.Admin;

[ApiController]
[Route("api/v1/admin/logs/mail")]
public sealed class MailLogsController : ControllerBase
{
    private readonly IMailLogService _service;
    private readonly IMessageLocalizer _localizer;

    public MailLogsController(IMailLogService service, IMessageLocalizer localizer)
    {
        _service = service;
        _localizer = localizer;
    }

    [HttpGet]
    public async Task<IActionResult> ListAsync([FromQuery] MailLogQuery query, CancellationToken cancellationToken)
    {
        var (start, end) = RequestTimeRange.Resolve(HttpContext.Request);
        var result = await _service.ListAsync(query, start, end, cancellationToken);
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
