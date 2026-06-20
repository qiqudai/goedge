using Cnn.Common.Contracts.Admin;
using Cnn.Api.Helpers;
using Cnn.Common.Localization;
using Cnn.Api.Responses;
using Cnn.Api.Services.Admin;
using Microsoft.AspNetCore.Mvc;

namespace Cnn.Api.Controllers.Admin;

[ApiController]
[Route("api/v1/admin/logs/operation")]
public sealed class OperationLogsController : ControllerBase
{
    private readonly IOperationLogService _service;
    private readonly IMessageLocalizer _localizer;

    public OperationLogsController(IOperationLogService service, IMessageLocalizer localizer)
    {
        _service = service;
        _localizer = localizer;
    }

    [HttpGet]
    public async Task<IActionResult> ListAsync([FromQuery] OperationLogQuery query, CancellationToken cancellationToken)
    {
        var (start, end) = RequestTimeRange.Resolve(HttpContext.Request);
        var result = await _service.ListAdminAsync(query, start, end, cancellationToken);
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
