using Cnn.Common.Contracts.Admin;
using Cnn.Common.Localization;
using Cnn.Api.Responses;
using Cnn.Api.Services;
using Cnn.Api.Services.Admin;
using Cnn.Api.Services.Common;
using Microsoft.AspNetCore.Mvc;

namespace Cnn.Api.Controllers.Admin;

[ApiController]
[Route("api/v1/admin/dnsapi")]
public sealed class DnsApiController : ControllerBase
{
    private readonly IDnsApiService _service;
    private readonly IMessageLocalizer _localizer;
    private readonly IAdminIdentityResolver _identityResolver;

    public DnsApiController(IDnsApiService service, IMessageLocalizer localizer, IAdminIdentityResolver identityResolver)
    {
        _service = service;
        _localizer = localizer;
        _identityResolver = identityResolver;
    }

    [HttpGet]
    public async Task<IActionResult> ListAsync([FromQuery] DnsApiListQuery query, CancellationToken cancellationToken)
    {
        var result = await _service.ListAsync(query ?? new DnsApiListQuery(), null, false, cancellationToken);
        return ToResponse(result);
    }

    [HttpPost]
    public async Task<IActionResult> CreateAsync([FromBody] DnsApiCreateRequest request, CancellationToken cancellationToken)
    {
        var uid = ResolveUserId();
        var result = await _service.CreateAsync(request, uid > 0 ? uid : null, false, cancellationToken);
        return ToResponse(result);
    }

    [HttpPut("{id:long}")]
    public async Task<IActionResult> UpdateAsync(long id, [FromBody] DnsApiUpdateRequest request, CancellationToken cancellationToken)
    {
        var result = await _service.UpdateAsync(id, request, null, false, cancellationToken);
        return ToResponse(result);
    }

    [HttpDelete("{id:long}")]
    public async Task<IActionResult> DeleteAsync(long id, CancellationToken cancellationToken)
    {
        var result = await _service.DeleteAsync(id, null, false, cancellationToken);
        return ToResponse(result);
    }

    [HttpGet("types")]
    public async Task<IActionResult> TypesAsync(CancellationToken cancellationToken)
    {
        var result = await _service.GetTypesAsync(cancellationToken);
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

    private long ResolveUserId()
    {
        var identity = _identityResolver.Resolve(User);
        return identity?.UserId ?? 0;
    }
}


