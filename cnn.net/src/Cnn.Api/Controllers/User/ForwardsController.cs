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
using SqlSugar;

namespace Cnn.Api.Controllers.User;

[ApiController]
[Route("api/v1/user/forwards")]
public sealed class ForwardsController : ControllerBase
{
    private readonly IForwardService _service;
    private readonly IMessageLocalizer _localizer;
    private readonly IAdminIdentityResolver _identityResolver;
    private readonly ISqlSugarClient _db;
    private readonly IDeletionPreviewService _deletionPreviewService;
    private readonly IResourceDeleteRequestService _resourceDeleteRequestService;

    public ForwardsController(
        IForwardService service,
        IMessageLocalizer localizer,
        IAdminIdentityResolver identityResolver,
        ISqlSugarClient db,
        IDeletionPreviewService deletionPreviewService,
        IResourceDeleteRequestService resourceDeleteRequestService)
    {
        _service = service;
        _localizer = localizer;
        _identityResolver = identityResolver;
        _db = db;
        _deletionPreviewService = deletionPreviewService;
        _resourceDeleteRequestService = resourceDeleteRequestService;
    }

    [HttpGet]
    public async Task<IActionResult> ListAsync(
        [FromQuery(Name = "user_id")] long? userId,
        [FromQuery(Name = "keyword")] string? keyword,
        [FromQuery(Name = "search_field")] string? searchField,
        [FromQuery(Name = "user_package_id")] long? userPackageId,
        [FromQuery(Name = "group_id")] long? groupId,
        [FromQuery(Name = "status")] string? status,
        [FromQuery(Name = "page")] int page = 1,
        [FromQuery(Name = "pageSize")] int pageSize = 10,
        CancellationToken cancellationToken = default)
    {
        var uid = ResolveUserId(userId);
        if (uid <= 0)
        {
            return Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, ErrorCodes.InvalidParam, "user_id_required"));
        }

        var query = new ForwardListQuery
        {
            Keyword = keyword,
            SearchField = searchField,
            UserId = uid,
            UserPackageId = userPackageId,
            GroupId = groupId,
            Status = status,
            Page = page,
            PageSize = pageSize
        };

        var result = await _service.ListAsync(query, uid, false, cancellationToken);
        return ToResponse(result);
    }

    [HttpPost]
    public async Task<IActionResult> CreateAsync([FromBody] ForwardCreateRequest request, CancellationToken cancellationToken)
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

    [HttpPut("{id:long}")]
    public async Task<IActionResult> UpdateAsync(long id, [FromBody] ForwardUpdateRequest request, CancellationToken cancellationToken)
    {
        var uid = ResolveUserId(request.UserId);
        if (uid <= 0)
        {
            return Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, ErrorCodes.InvalidParam, "user_id_required"));
        }

        request.UserId = uid;
        var result = await _service.UpdateAsync(id, request, uid, false, cancellationToken);
        return ToResponse(result);
    }

    [HttpPost("batch")]
    public async Task<IActionResult> BatchCreateAsync([FromBody] ForwardBatchCreateRequest request, CancellationToken cancellationToken)
    {
        var uid = ResolveUserId(request.UserId);
        if (uid <= 0)
        {
            return Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, ErrorCodes.InvalidParam, "user_id_required"));
        }

        request.UserId = uid;
        var result = await _service.BatchCreateAsync(request, uid, false, cancellationToken);
        return ToResponse(result);
    }

    [HttpPost("batch_update")]
    public async Task<IActionResult> BatchUpdateAsync([FromBody] ForwardBatchUpdateRequest request, CancellationToken cancellationToken)
    {
        var uid = ResolveUserId(null);
        if (uid <= 0)
        {
            return Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, ErrorCodes.InvalidParam, "user_id_required"));
        }

        var result = await _service.BatchUpdateAsync(request, uid, false, cancellationToken);
        return ToResponse(result);
    }

    [HttpPost("batch_action")]
    public async Task<IActionResult> BatchActionAsync([FromBody] ForwardBatchActionRequest request, CancellationToken cancellationToken)
    {
        var uid = ResolveUserId(null);
        if (uid <= 0)
        {
            return Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, ErrorCodes.InvalidParam, "user_id_required"));
        }

        var result = await _service.BatchActionAsync(request, uid, false, cancellationToken);
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

        var authResult = await EnsureForwardOwnerAsync(id, uid);
        if (authResult != null)
        {
            return authResult;
        }

        var result = await _deletionPreviewService.PreviewAsync(ResourceTypes.StreamApp, id, cancellationToken);
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

        var authResult = await EnsureForwardOwnerAsync(id, uid);
        if (authResult != null)
        {
            return authResult;
        }

        var result = await _resourceDeleteRequestService.RequestDeleteAsync(
            DeleteRequestCommandFactory.Create(ResourceTypes.StreamApp, id, uid, uid),
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

    private async Task<IActionResult?> EnsureForwardOwnerAsync(long id, long userId)
    {
        if (id <= 0)
        {
            return Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, ErrorCodes.InvalidParam));
        }

        var ownerId = await _db.Queryable<Cnn.Domain.Entities.Stream>()
            .Where(x => x.Id == id)
            .Select(x => x.Uid)
            .FirstAsync();

        if (!ownerId.HasValue)
        {
            return Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, ErrorCodes.NotFound, "forward_not_found"));
        }

        if (ownerId.Value != userId)
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
