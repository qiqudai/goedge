using Cnn.Common.Localization;
using Cnn.Api.Responses;
using Cnn.Api.Services.Admin;
using Cnn.Api.Services.Common;
using Microsoft.AspNetCore.Mvc;

namespace Cnn.Api.Controllers.Admin;

[ApiController]
[Route("api/v1/admin/forward")]
public sealed class ForwardMonitorController : ControllerBase
{
    private readonly IForwardMonitorService _service;
    private readonly IMessageLocalizer _localizer;

    public ForwardMonitorController(IForwardMonitorService service, IMessageLocalizer localizer)
    {
        _service = service;
        _localizer = localizer;
    }

    [HttpGet("traffic")]
    public async Task<IActionResult> TrafficAsync(
        [FromQuery(Name = "range")] string? range,
        [FromQuery(Name = "keyword")] string? keyword,
        CancellationToken cancellationToken)
    {
        var result = await _service.GetTrafficAsync(range, keyword, null, true, cancellationToken);
        return ToResponse(result);
    }

    [HttpGet("ranking")]
    public async Task<IActionResult> RankingAsync(
        [FromQuery(Name = "range")] string? range,
        CancellationToken cancellationToken)
    {
        var result = await _service.GetRankingAsync(range, null, true, cancellationToken);
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
