using Cnn.Common.Contracts.Admin;
using Cnn.Common.Localization;
using Cnn.Api.Responses;
using Cnn.Api.Services.Common;
using Microsoft.AspNetCore.Mvc;

namespace Cnn.Api.Controllers.Admin;

[ApiController]
[Route("api/v1/admin/config_items")]
public sealed class ConfigItemsController : ControllerBase
{
    private readonly IConfigItemService _service;
    private readonly IMessageLocalizer _localizer;

    public ConfigItemsController(IConfigItemService service, IMessageLocalizer localizer)
    {
        _service = service;
        _localizer = localizer;
    }

    [HttpGet]
    public async Task<IActionResult> ListAsync(
        [FromQuery(Name = "type")] string? type,
        [FromQuery(Name = "scope_name")] string? scopeName,
        [FromQuery(Name = "scope_id")] int? scopeId,
        CancellationToken cancellationToken)
    {
        var result = await _service.ListAsync(type, scopeName, scopeId, cancellationToken);
        return ToResponse(result);
    }

    [HttpPost]
    public async Task<IActionResult> UpsertAsync([FromBody] ConfigItemUpsertRequest request, CancellationToken cancellationToken)
    {
        var result = await _service.UpsertAsync(request, cancellationToken);
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
