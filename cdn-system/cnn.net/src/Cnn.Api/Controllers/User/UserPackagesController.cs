using Cnn.Common.Contracts;
using Cnn.Common.Localization;
using Cnn.Api.Responses;
using Cnn.Api.Services;
using Cnn.Api.Services.Common;
using Microsoft.AspNetCore.Mvc;

namespace Cnn.Api.Controllers.User;

[ApiController]
[Route("api/v1/user/user_packages")]
public sealed class UserPackagesController : ControllerBase
{
    private readonly IUserPackageService _service;
    private readonly IAdminIdentityResolver _identityResolver;
    private readonly IMessageLocalizer _localizer;

    public UserPackagesController(
        IUserPackageService service,
        IAdminIdentityResolver identityResolver,
        IMessageLocalizer localizer)
    {
        _service = service;
        _identityResolver = identityResolver;
        _localizer = localizer;
    }

    [HttpGet]
    public async Task<IActionResult> ListAsync([FromQuery] UserPackageListQuery query, CancellationToken cancellationToken)
    {
        var userId = ResolveUserId();
        var result = await _service.ListAsync(query, userId, true, cancellationToken);
        return ToResponse(result);
    }

    [HttpPut("{id:long}")]
    public async Task<IActionResult> UpdateAsync(long id, [FromBody] UserPackageUpdateRequest request, CancellationToken cancellationToken)
    {
        var userId = ResolveUserId();
        var result = await _service.UpdateAsync(id, request, userId, true, cancellationToken);
        return ToResponse(result);
    }

    [HttpPost("{id:long}/renew")]
    public async Task<IActionResult> RenewAsync(long id, [FromBody] RenewUserPackageRequest request, CancellationToken cancellationToken)
    {
        var userId = ResolveUserId();
        var result = await _service.RenewAsync(id, request, userId, true, cancellationToken);
        return ToResponse(result);
    }

    [HttpPost("{id:long}/switch")]
    public async Task<IActionResult> SwitchAsync(long id, [FromBody] SwitchUserPackageRequest request, CancellationToken cancellationToken)
    {
        var userId = ResolveUserId();
        var result = await _service.SwitchAsync(id, request, userId, true, cancellationToken);
        return ToResponse(result);
    }

    private long? ResolveUserId()
    {
        var identity = _identityResolver.Resolve(User);
        return identity?.UserId;
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
