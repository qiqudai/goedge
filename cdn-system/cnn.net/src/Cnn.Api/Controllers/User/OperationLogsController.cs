using Cnn.Common.Contracts.Admin;
using Cnn.Api.Helpers;
using Cnn.Common.Localization;
using Cnn.Api.Responses;
using Cnn.Api.Services.Admin;
using Cnn.Api.Services;
using Microsoft.AspNetCore.Mvc;

namespace Cnn.Api.Controllers.User;

[ApiController]
[Route("api/v1/user/logs/operation")]
public sealed class OperationLogsController : ControllerBase
{
    private readonly IOperationLogService _service;
    private readonly IAdminIdentityResolver _identityResolver;
    private readonly IMessageLocalizer _localizer;

    public OperationLogsController(
        IOperationLogService service,
        IAdminIdentityResolver identityResolver,
        IMessageLocalizer localizer)
    {
        _service = service;
        _identityResolver = identityResolver;
        _localizer = localizer;
    }

    [HttpGet]
    public async Task<IActionResult> ListAsync([FromQuery] OperationLogQuery query, CancellationToken cancellationToken)
    {
        var identity = _identityResolver.Resolve(User);
        if (identity == null)
        {
            return Ok(ApiResponseFactory.Fail<OperationLogListResult>(HttpContext, _localizer, ErrorCodes.AuthInvalid));
        }

        var (start, end) = RequestTimeRange.Resolve(HttpContext.Request);
        var result = await _service.ListUserAsync(identity.UserId, query, start, end, cancellationToken);
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
