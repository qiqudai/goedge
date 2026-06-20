using Cnn.Common.Contracts.Admin;
using Cnn.Common.Localization;
using Cnn.Api.Responses;
using Cnn.Api.Services.Admin;
using Cnn.Api.Services.Common;
using Cnn.Api.Services.Deletion;
using Cnn.Api.Services.Tasks.Workflow;
using Cnn.Api.Services.Users;
using Microsoft.AspNetCore.Mvc;

namespace Cnn.Api.Controllers.Admin;

[ApiController]
[Route("api/v1/admin/users")]
public sealed class UsersController : ControllerBase
{
    private readonly IUserService _service;
    private readonly IMessageLocalizer _localizer;
    private readonly IDeletionPreviewService _deletionPreviewService;
    private readonly IResourceDeleteRequestService _resourceDeleteRequestService;
    private readonly IUserPurgePlanner _userPurgePlanner;

    public UsersController(
        IUserService service,
        IMessageLocalizer localizer,
        IDeletionPreviewService deletionPreviewService,
        IResourceDeleteRequestService resourceDeleteRequestService,
        IUserPurgePlanner userPurgePlanner)
    {
        _service = service;
        _localizer = localizer;
        _deletionPreviewService = deletionPreviewService;
        _resourceDeleteRequestService = resourceDeleteRequestService;
        _userPurgePlanner = userPurgePlanner;
    }

    [HttpGet]
    public async Task<IActionResult> ListAsync([FromQuery] UserListQuery query, CancellationToken cancellationToken)
    {
        var result = await _service.ListAsync(query, cancellationToken);
        return ToResponse(result);
    }

    [HttpGet("{id:long}")]
    public async Task<IActionResult> GetAsync(long id, CancellationToken cancellationToken)
    {
        var result = await _service.GetAsync(id, cancellationToken);
        return ToResponse(result);
    }

    [HttpPost]
    public async Task<IActionResult> CreateAsync([FromBody] UserCreateRequest request, CancellationToken cancellationToken)
    {
        var result = await _service.CreateAsync(request, cancellationToken);
        return ToResponse(result);
    }

    [HttpPut("{id:long}/status")]
    public async Task<IActionResult> ToggleStatusAsync(long id, [FromBody] UserStatusUpdateRequest request, CancellationToken cancellationToken)
    {
        var result = await _service.ToggleStatusAsync(id, request, cancellationToken);
        return ToResponse(result);
    }

    [HttpPut("{id:long}")]
    public async Task<IActionResult> UpdateAsync(long id, [FromBody] UserUpdateRequest request, CancellationToken cancellationToken)
    {
        var result = await _service.UpdateAsync(id, request, cancellationToken);
        return ToResponse(result);
    }

    [HttpDelete("{id:long}")]
    public Task<IActionResult> DeleteAsync(long id, CancellationToken cancellationToken)
    {
        return DeleteWorkflowResponseHelper.RequestAsync(
            this,
            _localizer,
            _resourceDeleteRequestService,
            DeleteRequestCommandFactory.Create(ResourceTypes.UserAccount, id),
            cancellationToken);
    }

    [HttpGet("{id:long}/purge_preview")]
    public async Task<IActionResult> PurgePreviewAsync(long id, CancellationToken cancellationToken)
    {
        var result = await _userPurgePlanner.PlanAsync(id, cancellationToken);
        return Ok(ApiResponseFactory.Ok(HttpContext, _localizer, result));
    }

    [HttpGet("{id:long}/delete_preview")]
    public Task<IActionResult> DeletePreviewAsync(long id, CancellationToken cancellationToken)
    {
        return DeleteWorkflowResponseHelper.PreviewAsync(
            this,
            _localizer,
            _deletionPreviewService,
            ResourceTypes.UserAccount,
            id,
            cancellationToken);
    }

    [HttpPost("{id:long}/delete_request")]
    public Task<IActionResult> RequestDeleteAsync(long id, CancellationToken cancellationToken)
    {
        return DeleteWorkflowResponseHelper.RequestAsync(
            this,
            _localizer,
            _resourceDeleteRequestService,
            DeleteRequestCommandFactory.Create(ResourceTypes.UserAccount, id),
            cancellationToken);
    }

    [HttpPost("{id:long}/purge/reset")]
    public async Task<IActionResult> ResetPurgeAsync(long id, CancellationToken cancellationToken)
    {
        var result = await _service.ResetPurgeUsageAsync(id, cancellationToken);
        return ToResponse(result);
    }

    [HttpPost("{id:long}/impersonate")]
    public async Task<IActionResult> ImpersonateAsync(long id, CancellationToken cancellationToken)
    {
        var result = await _service.ImpersonateAsync(id, cancellationToken);
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
