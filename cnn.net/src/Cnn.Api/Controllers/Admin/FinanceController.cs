using Cnn.Common.Contracts;
using Cnn.Common.Localization;
using Cnn.Api.Responses;
using Cnn.Api.Services;
using Cnn.Api.Services.Common;
using Microsoft.AspNetCore.Mvc;

namespace Cnn.Api.Controllers.Admin;

[ApiController]
[Route("api/v1/admin")]
public sealed class FinanceController : ControllerBase
{
    private readonly IFinanceService _service;
    private readonly IAdminIdentityResolver _identityResolver;
    private readonly IMessageLocalizer _localizer;

    public FinanceController(IFinanceService service, IAdminIdentityResolver identityResolver, IMessageLocalizer localizer)
    {
        _service = service;
        _identityResolver = identityResolver;
        _localizer = localizer;
    }

    [HttpGet("orders")]
    public async Task<IActionResult> ListOrdersAsync([FromQuery] OrderListQuery query, CancellationToken cancellationToken)
    {
        var result = await _service.ListAdminAsync(query, cancellationToken);
        return ToResponse(result);
    }

    [HttpGet("balance_logs")]
    public async Task<IActionResult> ListBalanceLogsAsync([FromQuery] BalanceLogListQuery query, CancellationToken cancellationToken)
    {
        var result = await _service.ListAdminBalanceLogsAsync(query, cancellationToken);
        return ToResponse(result);
    }

    [HttpPost("recharge")]
    public async Task<IActionResult> RechargeAsync([FromBody] AdminRechargeRequest request, CancellationToken cancellationToken)
    {
        var result = await _service.AdminRechargeAsync(request, cancellationToken);
        return ToResponse(result);
    }

    [HttpPost("balance/adjust")]
    public async Task<IActionResult> AdjustBalanceAsync([FromBody] AdminAdjustBalanceRequest request, CancellationToken cancellationToken)
    {
        var operatorId = ResolveUserId();
        var result = await _service.AdminAdjustBalanceAsync(request, operatorId, cancellationToken);
        return ToResponse(result);
    }

    [HttpPost("orders/{id:long}/mark_paid")]
    public async Task<IActionResult> MarkPaidAsync(long id, [FromBody] AdminMarkOrderPaidRequest request, CancellationToken cancellationToken)
    {
        var operatorId = ResolveUserId();
        var result = await _service.MarkOrderPaidAsync(id, request, operatorId, cancellationToken);
        return ToResponse(result);
    }

    private long ResolveUserId()
    {
        var identity = _identityResolver.Resolve(User);
        return identity?.UserId ?? 0;
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
