using Cnn.Common.Contracts.Admin;
using Cnn.Common.Localization;
using Cnn.Api.Responses;
using Cnn.Api.Services.Admin;
using Cnn.Api.Services.Common;
using Microsoft.AspNetCore.Mvc;

namespace Cnn.Api.Controllers.Admin;

[ApiController]
[Route("api/v1/admin/dns")]
public sealed class DnsController : ControllerBase
{
    private readonly IDnsProviderService _service;
    private readonly IMessageLocalizer _localizer;

    public DnsController(IDnsProviderService service, IMessageLocalizer localizer)
    {
        _service = service;
        _localizer = localizer;
    }

    [HttpGet("providers")]
    public async Task<IActionResult> ListProvidersAsync([FromQuery] DnsProviderListQuery query, CancellationToken cancellationToken)
    {
        var result = await _service.ListProvidersAsync(query, cancellationToken);
        return ToResponse(result);
    }

    [HttpGet("providers/types")]
    public async Task<IActionResult> GetProviderTypesAsync(CancellationToken cancellationToken)
    {
        var result = await _service.GetProviderTypesAsync(cancellationToken);
        return ToResponse(result);
    }

    [HttpPost("providers")]
    public async Task<IActionResult> CreateProviderAsync([FromBody] DnsProviderCreateRequest request, CancellationToken cancellationToken)
    {
        var result = await _service.CreateProviderAsync(request, cancellationToken);
        return ToResponse(result);
    }

    [HttpPut("providers/{id:long}")]
    public async Task<IActionResult> UpdateProviderAsync(long id, [FromBody] DnsProviderUpdateRequest request, CancellationToken cancellationToken)
    {
        var result = await _service.UpdateProviderAsync(id, request, cancellationToken);
        return ToResponse(result);
    }

    [HttpDelete("providers/{id:long}")]
    public async Task<IActionResult> DeleteProviderAsync(long id, CancellationToken cancellationToken)
    {
        var result = await _service.DeleteProviderAsync(id, cancellationToken);
        return ToResponse(result);
    }

    [HttpGet("test")]
    public async Task<IActionResult> TestAsync(CancellationToken cancellationToken)
    {
        var result = await _service.TestAsync(cancellationToken);
        return ToResponse(result);
    }

    [HttpPost("records/fix")]
    public async Task<IActionResult> FixRecordsAsync(CancellationToken cancellationToken)
    {
        var result = await _service.FixRecordsAsync(cancellationToken);
        return ToResponse(result);
    }

    [HttpPost("records/cleanup")]
    public async Task<IActionResult> CleanupRecordsAsync(CancellationToken cancellationToken)
    {
        var result = await _service.CleanupRecordsAsync(cancellationToken);
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
