using Cnn.Common.Contracts;
using Cnn.Common.Localization;
using Cnn.Api.Responses;
using Cnn.Api.Services;
using Cnn.Api.Services.Common;
using Microsoft.AspNetCore.Mvc;

namespace Cnn.Api.Controllers.User;

[ApiController]
[Route("api/v1/user")]
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
    public async Task<IActionResult> ListOrdersAsync([FromQuery] UserOrderListQuery query, CancellationToken cancellationToken)
    {
        var userId = ResolveUserId();
        if (userId <= 0)
        {
            return Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, ErrorCodes.AuthInvalid));
        }

        var language = LanguageResolver.Resolve(HttpContext, _localizer.DefaultLanguage);
        var result = await _service.ListUserAsync(query, userId, language, cancellationToken);
        return ToResponse(result);
    }

    [HttpPost("recharge")]
    public async Task<IActionResult> RechargeAsync([FromBody] UserRechargeRequest request, CancellationToken cancellationToken)
    {
        var userId = ResolveUserId();
        if (userId <= 0)
        {
            return Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, ErrorCodes.AuthInvalid));
        }

        var callbackBaseUrl = ResolveCallbackBaseUrl();
        var result = await _service.UserRechargeAsync(userId, request, callbackBaseUrl, cancellationToken);
        return ToResponse(result);
    }

    [HttpGet("balance_logs")]
    public async Task<IActionResult> BalanceLogsAsync([FromQuery] BalanceLogListQuery query, CancellationToken cancellationToken)
    {
        var userId = ResolveUserId();
        if (userId <= 0)
        {
            return Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, ErrorCodes.AuthInvalid));
        }

        var result = await _service.ListUserBalanceLogsAsync(query, userId, cancellationToken);
        return ToResponse(result);
    }

    [HttpPost("orders/package/open")]
    public async Task<IActionResult> OpenPackageOrderAsync([FromBody] UserPackageOpenOrderRequest request, CancellationToken cancellationToken)
    {
        var userId = ResolveUserId();
        if (userId <= 0)
        {
            return Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, ErrorCodes.AuthInvalid));
        }

        var callbackBaseUrl = ResolveCallbackBaseUrl();
        var result = await _service.CreateUserPackageOpenOrderAsync(userId, request, callbackBaseUrl, cancellationToken);
        return ToResponse(result);
    }

    [HttpPost("orders/package/renew")]
    public async Task<IActionResult> RenewPackageOrderAsync([FromBody] UserPackageRenewOrderRequest request, CancellationToken cancellationToken)
    {
        var userId = ResolveUserId();
        if (userId <= 0)
        {
            return Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, ErrorCodes.AuthInvalid));
        }

        var callbackBaseUrl = ResolveCallbackBaseUrl();
        var result = await _service.CreateUserPackageRenewOrderAsync(userId, request, callbackBaseUrl, cancellationToken);
        return ToResponse(result);
    }

    private long ResolveUserId()
    {
        var identity = _identityResolver.Resolve(User);
        return identity?.UserId ?? 0;
    }

    private string ResolveCallbackBaseUrl()
    {
        var proto = Request.Headers["X-Forwarded-Proto"].ToString();
        if (string.IsNullOrWhiteSpace(proto))
        {
            proto = Request.Scheme;
        }
        else
        {
            proto = proto.Split(',', StringSplitOptions.RemoveEmptyEntries | StringSplitOptions.TrimEntries).FirstOrDefault() ?? Request.Scheme;
        }

        var host = Request.Headers["X-Forwarded-Host"].ToString();
        if (string.IsNullOrWhiteSpace(host))
        {
            host = Request.Host.Value;
        }

        if (string.IsNullOrWhiteSpace(host))
        {
            return string.Empty;
        }

        return $"{proto}://{host}";
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
