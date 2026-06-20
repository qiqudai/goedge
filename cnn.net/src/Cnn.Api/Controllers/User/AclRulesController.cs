using Cnn.Common.Contracts;
using Cnn.Common.Contracts.Admin;
using Cnn.Common.Localization;
using Cnn.Api.Responses;
using Cnn.Api.Services;
using Cnn.Api.Services.Admin;
using Cnn.Api.Services.Common;
using Cnn.Api.Services.Deletion;
using Cnn.Api.Services.Tasks.Workflow;
using Microsoft.AspNetCore.Mvc;

namespace Cnn.Api.Controllers.User;

[ApiController]
[Route("api/v1/user/rules/acl")]
public sealed class AclRulesController : ControllerBase
{
    private readonly IAclService _service;
    private readonly IMessageLocalizer _localizer;
    private readonly IAdminIdentityResolver _identityResolver;
    private readonly IDeletionPreviewService _deletionPreviewService;
    private readonly IResourceDeleteRequestService _resourceDeleteRequestService;

    public AclRulesController(
        IAclService service,
        IMessageLocalizer localizer,
        IAdminIdentityResolver identityResolver,
        IDeletionPreviewService deletionPreviewService,
        IResourceDeleteRequestService resourceDeleteRequestService)
    {
        _service = service;
        _localizer = localizer;
        _identityResolver = identityResolver;
        _deletionPreviewService = deletionPreviewService;
        _resourceDeleteRequestService = resourceDeleteRequestService;
    }

    [HttpGet]
    public async Task<IActionResult> ListAsync([FromQuery] long? user_id, [FromQuery] string? name, [FromQuery] string? status, CancellationToken cancellationToken)
    {
        var uid = ResolveUserId(user_id);
        if (uid <= 0)
        {
            return Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, ErrorCodes.InvalidParam, "user_id_required"));
        }

        var query = new AclListQuery
        {
            UserId = uid,
            Name = name,
            Status = status
        };
        var result = await _service.ListAsync(query, cancellationToken);
        return ToResponse(result);
    }

    [HttpGet("{id:long}")]
    public async Task<IActionResult> GetAsync(long id, [FromQuery] long? user_id, CancellationToken cancellationToken)
    {
        var uid = ResolveUserId(user_id);
        if (uid <= 0)
        {
            return Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, ErrorCodes.InvalidParam, "user_id_required"));
        }

        var result = await _service.GetAsync(id, cancellationToken);
        if (result.Success && result.Data != null && result.Data.UserId != uid)
        {
            return Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, ErrorCodes.PermissionDenied));
        }

        return ToResponse(result);
    }

    [HttpPost]
    public async Task<IActionResult> CreateAsync([FromBody] AclUpsertRequest request, CancellationToken cancellationToken)
    {
        var uid = ResolveUserId(request.UserId);
        if (uid <= 0)
        {
            return Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, ErrorCodes.InvalidParam, "user_id_required"));
        }

        request.UserId = uid;
        var result = await _service.CreateAsync(request, cancellationToken);
        return ToResponse(result);
    }

    [HttpPut("{id:long}")]
    public async Task<IActionResult> UpdateAsync(long id, [FromBody] AclUpsertRequest request, CancellationToken cancellationToken)
    {
        var uid = ResolveUserId(request.UserId);
        if (uid <= 0)
        {
            return Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, ErrorCodes.InvalidParam, "user_id_required"));
        }

        var result = await _service.GetAsync(id, cancellationToken);
        if (!result.Success || result.Data == null)
        {
            return ToResponse(result);
        }

        if (result.Data.UserId != uid)
        {
            return Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, ErrorCodes.PermissionDenied));
        }

        request.UserId = uid;
        var update = await _service.UpdateAsync(id, request, cancellationToken);
        return ToResponse(update);
    }

    [HttpDelete("{id:long}")]
    public Task<IActionResult> DeleteAsync(long id, [FromQuery] long? user_id, CancellationToken cancellationToken)
    {
        var uid = ResolveUserId(user_id);
        if (uid <= 0)
        {
            return Task.FromResult<IActionResult>(Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, ErrorCodes.InvalidParam, "user_id_required")));
        }

        return RequestDeleteAuthorizedAsync(id, uid, cancellationToken);
    }

    [HttpGet("{id:long}/delete_preview")]
    public async Task<IActionResult> DeletePreviewAsync(long id, [FromQuery] long? user_id, CancellationToken cancellationToken)
    {
        var uid = ResolveUserId(user_id);
        if (uid <= 0)
        {
            return Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, ErrorCodes.InvalidParam, "user_id_required"));
        }

        var authResult = await EnsureAclOwnerAsync(id, uid, cancellationToken);
        if (authResult != null)
        {
            return authResult;
        }

        var result = await _deletionPreviewService.PreviewAsync(ResourceTypes.AclRule, id, cancellationToken);
        return Ok(ApiResponseFactory.Ok(HttpContext, _localizer, result));
    }

    [HttpPost("{id:long}/delete_request")]
    public async Task<IActionResult> RequestDeleteAsync(long id, [FromQuery] long? user_id, CancellationToken cancellationToken)
    {
        var uid = ResolveUserId(user_id);
        if (uid <= 0)
        {
            return Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, ErrorCodes.InvalidParam, "user_id_required"));
        }

        return await RequestDeleteAuthorizedAsync(id, uid, cancellationToken);
    }

    private IActionResult ToResponse<T>(ServiceResult<T> result)
    {
        if (result.Success)
        {
            return Ok(ApiResponseFactory.Ok(HttpContext, _localizer, result.Data));
        }

        return Ok(ApiResponseFactory.Fail<T>(HttpContext, _localizer, result.ErrorCode, result.MessageKey));
    }

    private long ResolveUserId(long? userId)
    {
        if (userId is > 0)
        {
            return userId.Value;
        }

        var identity = _identityResolver.Resolve(User);
        return identity?.UserId ?? 0;
    }

    private async Task<IActionResult?> EnsureAclOwnerAsync(long id, long uid, CancellationToken cancellationToken)
    {
        var result = await _service.GetAsync(id, cancellationToken);
        if (!result.Success || result.Data == null)
        {
            return ToResponse(result);
        }

        if (result.Data.UserId != uid)
        {
            return Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, ErrorCodes.PermissionDenied));
        }

        return null;
    }

    private async Task<IActionResult> RequestDeleteAuthorizedAsync(long id, long uid, CancellationToken cancellationToken)
    {
        var authResult = await EnsureAclOwnerAsync(id, uid, cancellationToken);
        if (authResult != null)
        {
            return authResult;
        }

        var result = await _resourceDeleteRequestService.RequestDeleteAsync(
            DeleteRequestCommandFactory.Create(ResourceTypes.AclRule, id, uid, uid),
            cancellationToken);
        if (result.Success)
        {
            return Ok(ApiResponseFactory.Ok(HttpContext, _localizer, result.Data));
        }

        return Ok(ApiResponseFactory.Fail<DeleteRequestResult>(
            HttpContext,
            _localizer,
            result.ErrorCode,
            result.MessageKey,
            data: result.Data));
    }
}
