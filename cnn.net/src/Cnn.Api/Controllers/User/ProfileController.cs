using Cnn.Common.Contracts;
using Cnn.Common.Localization;
using Cnn.Api.Responses;
using Cnn.Api.Services;
using Cnn.Api.Services.Common;
using Microsoft.AspNetCore.Mvc;

namespace Cnn.Api.Controllers.User;

[ApiController]
[Route("api/v1/user")]
public sealed class ProfileController : ControllerBase
{
    private readonly IUserProfileService _service;
    private readonly IAdminIdentityResolver _identityResolver;
    private readonly IMessageLocalizer _localizer;

    public ProfileController(IUserProfileService service, IAdminIdentityResolver identityResolver, IMessageLocalizer localizer)
    {
        _service = service;
        _identityResolver = identityResolver;
        _localizer = localizer;
    }

    [HttpGet("profile")]
    public async Task<IActionResult> GetAsync(CancellationToken cancellationToken)
    {
        var userId = ResolveUserId();
        if (userId <= 0)
        {
            return Ok(ApiResponseFactory.Fail<UserProfileDto>(HttpContext, _localizer, ErrorCodes.AuthInvalid));
        }

        var result = await _service.GetAsync(userId, cancellationToken);
        return ToResponse(result);
    }

    [HttpPut("profile")]
    public async Task<IActionResult> UpdateAsync([FromBody] UpdateProfileRequest request, CancellationToken cancellationToken)
    {
        var userId = ResolveUserId();
        if (userId <= 0)
        {
            return Ok(ApiResponseFactory.Fail<bool>(HttpContext, _localizer, ErrorCodes.AuthInvalid));
        }

        var result = await _service.UpdateAsync(userId, request, cancellationToken);
        return ToResponse(result);
    }

    [HttpPut("password")]
    public async Task<IActionResult> UpdatePasswordAsync([FromBody] UpdatePasswordRequest request, CancellationToken cancellationToken)
    {
        var userId = ResolveUserId();
        if (userId <= 0)
        {
            return Ok(ApiResponseFactory.Fail<bool>(HttpContext, _localizer, ErrorCodes.AuthInvalid));
        }

        var result = await _service.UpdatePasswordAsync(userId, request, cancellationToken);
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
