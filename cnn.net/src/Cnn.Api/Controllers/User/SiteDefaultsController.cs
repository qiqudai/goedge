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
[Route("api/v1/user/site_defaults")]
public sealed class SiteDefaultsController : ControllerBase
{
    private readonly ISiteDefaultService _service;
    private readonly IMessageLocalizer _localizer;
    private readonly IAdminIdentityResolver _identityResolver;

    public SiteDefaultsController(
        ISiteDefaultService service,
        IMessageLocalizer localizer,
        IAdminIdentityResolver identityResolver)
    {
        _service = service;
        _localizer = localizer;
        _identityResolver = identityResolver;
    }

    [HttpGet]
    public async Task<IActionResult> ListAsync(
        [FromQuery(Name = "scope_name")] string? scopeName = null,
        [FromQuery(Name = "scope_id")] long? scopeId = null,
        [FromQuery(Name = "user_id")] long? userId = null,
        CancellationToken cancellationToken = default)
    {
        var uid = ResolveUserId(userId);
        if (uid <= 0)
        {
            return Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, ErrorCodes.InvalidParam, "user_id_required"));
        }

        var query = new SiteDefaultListQuery
        {
            ScopeName = scopeName,
            ScopeId = scopeId,
            UserId = uid
        };

        var result = await _service.ListAsync(query, uid, false, cancellationToken);
        return ToResponse(result);
    }

    [HttpPost]
    public async Task<IActionResult> CreateAsync([FromBody] SiteDefaultCreateRequest request, CancellationToken cancellationToken)
    {
        var uid = ResolveUserId(request.UserId);
        if (uid <= 0)
        {
            return Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, ErrorCodes.InvalidParam, "user_id_required"));
        }

        request.UserId = uid;
        var result = await _service.CreateAsync(request, uid, false, cancellationToken);
        return ToResponse(result);
    }

    [HttpPut("{name}")]
    public async Task<IActionResult> UpdateAsync(string name, [FromBody] SiteDefaultUpdateRequest request, CancellationToken cancellationToken)
    {
        var uid = ResolveUserId(request.UserId);
        if (uid <= 0)
        {
            return Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, ErrorCodes.InvalidParam, "user_id_required"));
        }

        request.UserId = uid;
        var result = await _service.UpdateAsync(name, request, uid, false, cancellationToken);
        return ToResponse(result);
    }

    [HttpDelete("{name}")]
    public async Task<IActionResult> DeleteAsync(
        string name,
        [FromQuery(Name = "scope_name")] string? scopeName = null,
        [FromQuery(Name = "scope_id")] long? scopeId = null,
        CancellationToken cancellationToken = default)
    {
        var uid = ResolveUserId(null);
        if (uid <= 0)
        {
            return Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, ErrorCodes.InvalidParam, "user_id_required"));
        }

        var result = await _service.DeleteAsync(name, scopeName, scopeId, uid, false, cancellationToken);
        return ToResponse(result);
    }

    private long ResolveUserId(long? userId)
    {
        if (userId is > 0)
        {
            return userId.Value;
        }

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
