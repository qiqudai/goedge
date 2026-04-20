using Cnn.Common.Contracts.Admin;
using Cnn.Common.Localization;
using Cnn.Api.Responses;
using Cnn.Api.Services.Admin;
using Cnn.Api.Services.Common;
using Microsoft.AspNetCore.Mvc;

namespace Cnn.Api.Controllers.Admin;

[ApiController]
[Route("api/v1/admin/site_defaults")]
public sealed class SiteDefaultsController : ControllerBase
{
    private readonly ISiteDefaultService _service;
    private readonly IMessageLocalizer _localizer;

    public SiteDefaultsController(ISiteDefaultService service, IMessageLocalizer localizer)
    {
        _service = service;
        _localizer = localizer;
    }

    [HttpGet]
    public async Task<IActionResult> ListAsync(
        [FromQuery(Name = "scope_name")] string? scopeName = null,
        [FromQuery(Name = "scope_id")] long? scopeId = null,
        [FromQuery(Name = "user_id")] long? userId = null,
        CancellationToken cancellationToken = default)
    {
        var query = new SiteDefaultListQuery
        {
            ScopeName = scopeName,
            ScopeId = scopeId,
            UserId = userId
        };

        var result = await _service.ListAsync(query, null, true, cancellationToken);
        return ToResponse(result);
    }

    [HttpPost]
    public async Task<IActionResult> CreateAsync([FromBody] SiteDefaultCreateRequest request, CancellationToken cancellationToken)
    {
        var result = await _service.CreateAsync(request, null, true, cancellationToken);
        return ToResponse(result);
    }

    [HttpPut("{name}")]
    public async Task<IActionResult> UpdateAsync(string name, [FromBody] SiteDefaultUpdateRequest request, CancellationToken cancellationToken)
    {
        var result = await _service.UpdateAsync(name, request, request.UserId, true, cancellationToken);
        return ToResponse(result);
    }

    [HttpDelete("{name}")]
    public async Task<IActionResult> DeleteAsync(
        string name,
        [FromQuery(Name = "scope_name")] string? scopeName = null,
        [FromQuery(Name = "scope_id")] long? scopeId = null,
        [FromQuery(Name = "user_id")] long? userId = null,
        CancellationToken cancellationToken = default)
    {
        var result = await _service.DeleteAsync(name, scopeName, scopeId, userId, true, cancellationToken);
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
