using System.Text.Json;
using Cnn.Common.Contracts;
using Cnn.Common.Contracts.Admin;
using Cnn.Common.Localization;
using Cnn.Api.Responses;
using Cnn.Api.Services.Admin;
using Cnn.Api.Services.Common;
using Cnn.Api.Services;
using Microsoft.AspNetCore.Mvc;

namespace Cnn.Api.Controllers.Admin;

[ApiController]
[Route("api/v1/admin/user_plans")]
public sealed class UserPlansController : ControllerBase
{
    private readonly IPlanService _service;
    private readonly IUserPackageService _userPackageService;
    private readonly IMessageLocalizer _localizer;

    public UserPlansController(IPlanService service, IUserPackageService userPackageService, IMessageLocalizer localizer)
    {
        _service = service;
        _userPackageService = userPackageService;
        _localizer = localizer;
    }

    [HttpGet]
    public async Task<IActionResult> ListAsync(CancellationToken cancellationToken)
    {
        var result = await _service.ListUserPlansAsync(cancellationToken);
        return ToResponse(result);
    }

    [HttpPost("assign")]
    public async Task<IActionResult> AssignAsync([FromBody] AssignUserPlanRequest request, CancellationToken cancellationToken)
    {
        var result = await _service.AssignUserPlanAsync(request, cancellationToken);
        return ToResponse(result);
    }

    [HttpPut("{id:long}")]
    public async Task<IActionResult> UpdateAsync(long id, [FromBody] JsonElement payload, CancellationToken cancellationToken)
    {
        var result = await _service.UpdateUserPlanAsync(id, payload, cancellationToken);
        return ToResponse(result);
    }

    [HttpPost("{id:long}/switch")]
    public async Task<IActionResult> SwitchAsync(long id, [FromBody] SwitchUserPackageRequest request, CancellationToken cancellationToken)
    {
        var result = await _userPackageService.SwitchAsync(id, request, null, false, cancellationToken);
        return ToResponse(result);
    }

    [HttpDelete]
    public async Task<IActionResult> DeleteAsync([FromBody] DeleteUserPlansRequest request, CancellationToken cancellationToken)
    {
        var result = await _service.DeleteUserPlansAsync(request, cancellationToken);
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
