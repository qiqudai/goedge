using Cnn.Common.Localization;
using Cnn.Api.Responses;
using Cnn.Api.Services.Common;
using Microsoft.AspNetCore.Mvc;

namespace Cnn.Api.Controllers;

[ApiController]
[Route("api/v1")]
public sealed class SystemInfoController : ControllerBase
{
    private readonly ISystemInfoService _service;
    private readonly IMessageLocalizer _localizer;

    public SystemInfoController(ISystemInfoService service, IMessageLocalizer localizer)
    {
        _service = service;
        _localizer = localizer;
    }

    [HttpGet("system_info")]
    [HttpGet("admin/system_info")]
    public async Task<IActionResult> GetAsync(CancellationToken cancellationToken)
    {
        var payload = await _service.GetAsync(cancellationToken);
        return Ok(ApiResponseFactory.Ok(HttpContext, _localizer, payload));
    }
}
