using Cnn.Common.Contracts.Admin;
using Cnn.Common.Localization;
using Cnn.Api.Responses;
using Cnn.Api.Services.Admin;
using Cnn.Api.Services.Common;
using Microsoft.AspNetCore.Mvc;

namespace Cnn.Api.Controllers.Admin;

[ApiController]
[Route("api/v1/admin/forward_defaults")]
public sealed class ForwardDefaultsController : ControllerBase
{
    private readonly IForwardDefaultService _service;
    private readonly IMessageLocalizer _localizer;

    public ForwardDefaultsController(IForwardDefaultService service, IMessageLocalizer localizer)
    {
        _service = service;
        _localizer = localizer;
    }

    [HttpGet]
    public async Task<IActionResult> ListAsync([FromQuery(Name = "user_id")] long? userId, CancellationToken cancellationToken)
    {
        var result = await _service.ListAsync(userId, true, cancellationToken);
        return ToResponse(result);
    }

    [HttpPost]
    public async Task<IActionResult> CreateAsync([FromBody] ForwardDefaultCreateRequest request, CancellationToken cancellationToken)
    {
        var result = await _service.CreateAsync(request, request.UserId > 0 ? request.UserId : null, true, cancellationToken);
        return ToResponse(result);
    }

    [HttpDelete]
    public async Task<IActionResult> DeleteAsync([FromBody] ForwardDefaultDeleteRequest request, CancellationToken cancellationToken)
    {
        var result = await _service.DeleteAsync(request, request.UserId > 0 ? request.UserId : null, true, cancellationToken);
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
