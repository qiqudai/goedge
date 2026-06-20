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
[Route("api/v1/user/forward_groups")]
public sealed class ForwardGroupsController : ControllerBase
{
    private readonly IForwardGroupService _service;
    private readonly IMessageLocalizer _localizer;
    private readonly IAdminIdentityResolver _identityResolver;
    private readonly IDeletionPreviewService _deletionPreviewService;
    private readonly IResourceDeleteRequestService _resourceDeleteRequestService;

    public ForwardGroupsController(
        IForwardGroupService service,
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
    public async Task<IActionResult> ListAsync(
        [FromQuery(Name = "keyword")] string? keyword,
        [FromQuery(Name = "user_id")] long? userId,
        CancellationToken cancellationToken)
    {
        var uid = ResolveUserId(userId);
        if (uid <= 0)
        {
            return Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, ErrorCodes.InvalidParam, "user_id_required"));
        }

        var result = await _service.ListAsync(keyword, uid, false, cancellationToken);
        return ToResponse(result);
    }

    [HttpPost]
    public async Task<IActionResult> CreateAsync([FromBody] ForwardGroupUpsertRequest request, CancellationToken cancellationToken)
    {
        var uid = ResolveUserId(request.UserId);
        if (uid <= 0)
        {
            return Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, ErrorCodes.InvalidParam, "user_id_required"));
        }

        request.UserId = uid;
        var result = await _service.CreateAsync(request, uid, false, cancellationToken);
        return ToResponse(result);
    }

    [HttpPut]
    public async Task<IActionResult> UpdateAsync([FromBody] ForwardGroupUpsertRequest request, CancellationToken cancellationToken)
    {
        var uid = ResolveUserId(request.UserId);
        if (uid <= 0)
        {
            return Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, ErrorCodes.InvalidParam, "user_id_required"));
        }

        request.UserId = uid;
        var result = await _service.UpdateAsync(request, uid, false, cancellationToken);
        return ToResponse(result);
    }

    [HttpDelete]
    public async Task<IActionResult> DeleteAsync([FromBody] ForwardGroupDeleteRequest request, CancellationToken cancellationToken)
    {
        var uid = ResolveUserId(null);
        if (uid <= 0)
        {
            return Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, ErrorCodes.InvalidParam, "user_id_required"));
        }

        var result = await _service.DeleteAsync(request.Id, uid, false, cancellationToken);
        return ToResponse(result);
    }

    [HttpGet("{id:long}/delete_preview")]
    public async Task<IActionResult> DeletePreviewAsync(long id, CancellationToken cancellationToken)
    {
        var uid = ResolveUserId(null);
        if (uid <= 0)
        {
            return Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, ErrorCodes.InvalidParam, "user_id_required"));
        }

        var authResult = await EnsureGroupOwnerAsync(id, uid, cancellationToken);
        if (authResult != null)
        {
            return authResult;
        }

        var result = await _deletionPreviewService.PreviewAsync(ResourceTypes.StreamGroup, id, cancellationToken);
        return Ok(ApiResponseFactory.Ok(HttpContext, _localizer, result));
    }

    [HttpPost("{id:long}/delete_request")]
    public async Task<IActionResult> RequestDeleteAsync(long id, CancellationToken cancellationToken)
    {
        var uid = ResolveUserId(null);
        if (uid <= 0)
        {
            return Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, ErrorCodes.InvalidParam, "user_id_required"));
        }

        var authResult = await EnsureGroupOwnerAsync(id, uid, cancellationToken);
        if (authResult != null)
        {
            return authResult;
        }

        var result = await _resourceDeleteRequestService.RequestDeleteAsync(
            DeleteRequestCommandFactory.Create(ResourceTypes.StreamGroup, id, uid, uid),
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

    private long ResolveUserId(long? userId)
    {
        if (userId is > 0)
        {
            return userId.Value;
        }

        var identity = _identityResolver.Resolve(User);
        return identity?.UserId ?? 0;
    }

    private async Task<IActionResult?> EnsureGroupOwnerAsync(long id, long userId, CancellationToken cancellationToken)
    {
        var list = await _service.ListAsync(null, userId, false, cancellationToken);
        if (!list.Success || list.Data == null)
        {
            return ToResponse(list);
        }

        var exists = list.Data.List.Any(item => item.Id == id);
        if (!exists)
        {
            return Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, ErrorCodes.PermissionDenied));
        }

        return null;
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
