using Cnn.Common.Contracts.Admin;
using Cnn.Common.Localization;
using Cnn.Api.Responses;
using Cnn.Api.Services.Agent;
using Cnn.Api.Services.Common;
using Microsoft.AspNetCore.Mvc;

namespace Cnn.Api.Controllers.Admin;

[ApiController]
[Route("api/v1/admin/debug")]
public sealed class DebugController : ControllerBase
{
    private readonly IDebugControlService _service;
    private readonly IMessageLocalizer _localizer;

    public DebugController(IDebugControlService service, IMessageLocalizer localizer)
    {
        _service = service;
        _localizer = localizer;
    }

    [HttpPost("switches")]
    public async Task<IActionResult> SwitchesAsync([FromBody] DebugSwitchDispatchRequest request, CancellationToken cancellationToken)
    {
        var result = await _service.UpdateSwitchesAsync(request, cancellationToken);
        return ToResponse(result);
    }

    [HttpPost("manual_logs")]
    public async Task<IActionResult> ManualLogsAsync([FromBody] ManualDebugLogDispatchRequest request, CancellationToken cancellationToken)
    {
        var result = await _service.WriteManualLogAsync(request, cancellationToken);
        return ToResponse(result);
    }

    [HttpGet("server_switches")]
    public async Task<IActionResult> ServerSwitchesAsync(CancellationToken cancellationToken)
    {
        var result = await _service.GetServerSwitchesAsync(cancellationToken);
        return ToResponse(result);
    }

    [HttpPost("server_switches")]
    public async Task<IActionResult> UpdateServerSwitchesAsync([FromBody] ServerDebugSwitchesUpdateRequest request, CancellationToken cancellationToken)
    {
        var result = await _service.UpdateServerSwitchesAsync(request, cancellationToken);
        return ToResponse(result);
    }

    private IActionResult ToResponse<T>(ServiceResult<T> result)
    {
        if (result.Success)
        {
            return Ok(ApiResponseFactory.Ok(HttpContext, _localizer, result.Data));
        }

        return Ok(ApiResponseFactory.Fail(HttpContext, _localizer, result.ErrorCode, result.MessageKey, data: result.Data));
    }
}
