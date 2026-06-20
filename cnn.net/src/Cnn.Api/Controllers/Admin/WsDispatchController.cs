using Cnn.Common.Contracts.Admin;
using Cnn.Common.Localization;
using Cnn.Api.Responses;
using Cnn.Api.Services.Agent;
using Cnn.Api.Services.Common;
using Microsoft.AspNetCore.Mvc;

namespace Cnn.Api.Controllers.Admin;

[ApiController]
[Route("api/v1/admin/ws")]
public sealed class WsDispatchController : ControllerBase
{
    private readonly IWsDispatchService _service;
    private readonly IMessageLocalizer _localizer;

    public WsDispatchController(IWsDispatchService service, IMessageLocalizer localizer)
    {
        _service = service;
        _localizer = localizer;
    }

    [HttpPost("dispatch")]
    public async Task<IActionResult> DispatchAsync([FromBody] WsDispatchRequest request, CancellationToken cancellationToken)
    {
        var result = await _service.DispatchAsync(request, cancellationToken);
        return ToResponse(result);
    }

    private IActionResult ToResponse<T>(ServiceResult<T> result)
    {
        if (result.Success)
        {
            return Ok(ApiResponseFactory.Ok(HttpContext, _localizer, result.Data));
        }

        return Ok(ApiResponseFactory.Fail(HttpContext, _localizer, result.ErrorCode, result.MessageKey, data: result.Data));
    }
}
