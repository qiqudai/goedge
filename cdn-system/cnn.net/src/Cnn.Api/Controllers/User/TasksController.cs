using Cnn.Common.Contracts;
using Cnn.Common.Localization;
using Cnn.Api.Responses;
using Cnn.Api.Services;
using Cnn.Api.Services.Common;
using Microsoft.AspNetCore.Mvc;

namespace Cnn.Api.Controllers.User;

[ApiController]
[Route("api/v1/user/tasks")]
public sealed class TasksController : BaseApiController
{
    private readonly ITaskService _service;

    public TasksController(ITaskService service, IMessageLocalizer localizer, IAdminIdentityResolver identityResolver) : base(identityResolver, localizer)
    {
        _service = service;
    }

    [HttpGet]
    public async Task<IActionResult> ListAsync([FromQuery] TaskListQuery query, CancellationToken cancellationToken)
    {
        var identity = _identityResolver.Resolve(User);
        if (identity == null)
        {
            return Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, ErrorCodes.AuthInvalid));
        }

        var result = await _service.ListUserAsync(query, identity.UserId, cancellationToken);
        return ToListResponse(result);
    }

    [HttpGet("usage")]
    public async Task<IActionResult> UsageAsync(CancellationToken cancellationToken)
    {
        var identity = _identityResolver.Resolve(User);
        if (identity == null)
        {
            return Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, ErrorCodes.AuthInvalid));
        }

        var result = await _service.GetUsageAsync(identity.UserId, cancellationToken);
        return ToResponse(result);
    }

    [HttpPost]
    public async Task<IActionResult> CreateAsync([FromBody] TaskCreateRequest request, CancellationToken cancellationToken)
    {
        var identity = _identityResolver.Resolve(User);
        if (identity == null)
        {
            return Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, ErrorCodes.AuthInvalid));
        }

        var result = await _service.CreateAsync(request, identity.UserId, false, cancellationToken);
        return ToResponse(result);
    }

    [HttpGet("{id:long}")]
    public async Task<IActionResult> GetAsync(long id, CancellationToken cancellationToken)
    {
        var identity = _identityResolver.Resolve(User);
        if (identity == null)
        {
            return Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, ErrorCodes.AuthInvalid));
        }

        var result = await _service.GetUserAsync(id, identity.UserId, cancellationToken);
        return ToResponse(result);
    }

    [HttpPost("{id:long}/resubmit")]
    public async Task<IActionResult> ResubmitAsync(long id, CancellationToken cancellationToken)
    {
        var identity = _identityResolver.Resolve(User);
        if (identity == null)
        {
            return Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, ErrorCodes.AuthInvalid));
        }

        var result = await _service.ResubmitAsync(id, identity.UserId, false, cancellationToken);
        return ToResponse(result);
    }

    private IActionResult ToListResponse(ServiceResult<TaskListResult> result)
    {
        if (!result.Success)
        {
            return Ok(ApiResponseFactory.Fail<TaskListResult>(HttpContext, _localizer, result.ErrorCode, result.MessageKey));
        }

        var response = ApiResponseFactory.Ok(HttpContext, _localizer, result.Data);
        return Ok(new
        {
            code = response.Code,
            message = response.Message,
            data = response.Data,
            trace_id = response.TraceId,
            list = result.Data?.List,
            total = result.Data?.Total,
            page = result.Data?.Page
        });
    }

}
