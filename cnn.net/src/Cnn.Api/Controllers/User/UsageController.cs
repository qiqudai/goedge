using Cnn.Common.Localization;
using Cnn.Api.Responses;
using Cnn.Api.Services;
using Cnn.Api.Services.Common;
using Microsoft.AspNetCore.Mvc;

namespace Cnn.Api.Controllers.User;

[ApiController]
[Route("api/v1/user/usage")]
public sealed class UsageController : ControllerBase
{
    private readonly IUsageService _service;
    private readonly IAdminIdentityResolver _identityResolver;
    private readonly IMessageLocalizer _localizer;

    public UsageController(
        IUsageService service,
        IAdminIdentityResolver identityResolver,
        IMessageLocalizer localizer)
    {
        _service = service;
        _identityResolver = identityResolver;
        _localizer = localizer;
    }

    [HttpGet]
    public async Task<IActionResult> ListAsync([FromQuery(Name = "range")] string? range, CancellationToken cancellationToken)
    {
        var userId = ResolveUserId();
        var scope = userId.HasValue ? AccessScope.User(userId.Value) : AccessScope.Admin();
        var result = await _service.GetUsageAsync(range, scope, cancellationToken);
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
