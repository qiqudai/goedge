using Cnn.Common.Contracts;
using Cnn.Common.Localization;
using Cnn.Api.Responses;
using Cnn.Api.Services.Common;
using Microsoft.AspNetCore.Mvc;

namespace Cnn.Api.Controllers;

[ApiController]
[Route("api/v1/pay")]
public sealed class PayController : ControllerBase
{
    private readonly IFinanceService _financeService;
    private readonly IMessageLocalizer _localizer;

    public PayController(IFinanceService financeService, IMessageLocalizer localizer)
    {
        _financeService = financeService;
        _localizer = localizer;
    }

    [HttpPost("shkeeper/callback")]
    public async Task<IActionResult> ShkeeperCallbackAsync([FromBody] ShkeeperCallbackPayload payload, CancellationToken cancellationToken)
    {
        var callbackKey = Request.Headers["X-Shkeeper-Api-Key"].ToString();
        var result = await _financeService.HandleShkeeperCallbackAsync(payload, callbackKey, cancellationToken);
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
