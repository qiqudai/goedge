using Cnn.Common.Contracts;
using Cnn.Common.Localization;
using Cnn.Api.Responses;
using Cnn.Api.Services;
using Cnn.Api.Services.Common;
using Microsoft.AspNetCore.Mvc;

namespace Cnn.Api.Controllers.User;

[ApiController]
[Route("api/v1/user/domains")]
public sealed class DomainsController : ControllerBase
{
    private readonly IDomainService _service;
    private readonly IAdminIdentityResolver _identityResolver;
    private readonly IMessageLocalizer _localizer;

    public DomainsController(IDomainService service, IAdminIdentityResolver identityResolver, IMessageLocalizer localizer)
    {
        _service = service;
        _identityResolver = identityResolver;
        _localizer = localizer;
    }

    [HttpGet]
    public async Task<IActionResult> ListAsync([FromQuery] DomainListQuery query, CancellationToken cancellationToken)
    {
        var userId = ResolveUserId();
        if (userId <= 0)
        {
            return Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, ErrorCodes.AuthInvalid));
        }

        var result = await _service.ListUserAsync(query, userId, cancellationToken);
        return ToResponse(result);
    }

    [HttpPost]
    public async Task<IActionResult> CreateAsync([FromBody] CreateDomainRequest request, CancellationToken cancellationToken)
    {
        var userId = ResolveUserId();
        if (userId <= 0)
        {
            return Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, ErrorCodes.AuthInvalid));
        }

        var result = await _service.CreateAsync(userId, request, cancellationToken);
        return ToResponse(result);
    }

    [HttpGet("{id:long}/config")]
    public async Task<IActionResult> GetConfigAsync(long id, CancellationToken cancellationToken)
    {
        var userId = ResolveUserId();
        if (userId <= 0)
        {
            return Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, ErrorCodes.AuthInvalid));
        }

        var result = await _service.GetConfigAsync(userId, id, cancellationToken);
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
