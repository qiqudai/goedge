using Cnn.Common.Contracts;
using Cnn.Common.Contracts.Admin;
using Cnn.Common.Localization;
using Cnn.Api.Responses;
using Cnn.Api.Services;
using Cnn.Api.Services.Admin;
using Cnn.Api.Services.Common;
using Microsoft.AspNetCore.Mvc;

namespace Cnn.Api.Controllers.User;

[ApiController]
[Route("api/v1/user/dns")]
public sealed class DnsController : ControllerBase
{
    private readonly IDnsProviderService _service;
    private readonly IMessageLocalizer _localizer;
    private readonly IAdminIdentityResolver _identityResolver;

    public DnsController(IDnsProviderService service, IMessageLocalizer localizer, IAdminIdentityResolver identityResolver)
    {
        _service = service;
        _localizer = localizer;
        _identityResolver = identityResolver;
    }

    [HttpGet("providers")]
    public async Task<IActionResult> ListProvidersAsync([FromQuery] DnsProviderListQuery query, CancellationToken cancellationToken)
    {
        var uid = ResolveUserId();
        if (uid <= 0)
        {
            return Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, ErrorCodes.InvalidParam, "user_id_required"));
        }

        query ??= new DnsProviderListQuery();
        query.UserId = uid;
        var result = await _service.ListProvidersAsync(query, cancellationToken);
        return ToResponse(result);
    }

    [HttpGet("providers/types")]
    public async Task<IActionResult> GetProviderTypesAsync(CancellationToken cancellationToken)
    {
        var result = await _service.GetProviderTypesAsync(cancellationToken);
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
