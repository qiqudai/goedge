using Cnn.Common.Contracts.Admin;
using Cnn.Common.Localization;
using Cnn.Api.Responses;
using Cnn.Api.Services;
using Cnn.Api.Services.Admin;
using Cnn.Api.Services.Common;
using Cnn.Api.Services.Deletion;
using Cnn.Api.Services.Tasks.Workflow;
using Microsoft.AspNetCore.Mvc;

namespace Cnn.Api.Controllers.Admin;

[ApiController]
[Route("api/v1/admin/site_groups")]
public sealed class SiteGroupsController : ControllerBase
{
    private readonly ISiteGroupService _service;
    private readonly IMessageLocalizer _localizer;
    private readonly IAdminIdentityResolver _identityResolver;
    private readonly IDeletionPreviewService _deletionPreviewService;
    private readonly IResourceDeleteRequestService _resourceDeleteRequestService;

    public SiteGroupsController(
        ISiteGroupService service,
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
        [FromQuery(Name = "page")] int page = 1,
        [FromQuery(Name = "pageSize")] int pageSize = 10,
        [FromQuery(Name = "keyword")] string? keyword = null,
        [FromQuery(Name = "user_id")] long? userId = null,
        CancellationToken cancellationToken = default)
    {
        var query = new SiteGroupListQuery
        {
            Page = page,
            PageSize = pageSize,
            Keyword = keyword,
            UserId = userId
        };

        var result = await _service.ListAsync(query, null, true, cancellationToken);
        return ToResponse(result);
    }

    [HttpPost]
    public async Task<IActionResult> CreateAsync([FromBody] SiteGroupUpsertRequest request, CancellationToken cancellationToken)
    {
        var userId = ResolveUserId();
        var result = await _service.CreateAsync(request, userId > 0 ? userId : null, true, cancellationToken);
        return ToResponse(result);
    }

    [HttpPut("{id:long}")]
    public async Task<IActionResult> UpdateAsync(long id, [FromBody] SiteGroupUpsertRequest request, CancellationToken cancellationToken)
    {
        var result = await _service.UpdateAsync(id, request, null, true, cancellationToken);
        return ToResponse(result);
    }

    [HttpDelete("{id:long}")]
    public Task<IActionResult> DeleteAsync(long id, CancellationToken cancellationToken)
    {
        return DeleteWorkflowResponseHelper.RequestAsync(
            this,
            _localizer,
            _resourceDeleteRequestService,
            DeleteRequestCommandFactory.Create(ResourceTypes.SiteGroup, id),
            cancellationToken);
    }

    [HttpGet("{id:long}/delete_preview")]
    public Task<IActionResult> DeletePreviewAsync(long id, CancellationToken cancellationToken)
    {
        return DeleteWorkflowResponseHelper.PreviewAsync(
            this,
            _localizer,
            _deletionPreviewService,
            ResourceTypes.SiteGroup,
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
            DeleteRequestCommandFactory.Create(ResourceTypes.SiteGroup, id),
            cancellationToken);
    }

    private long ResolveUserId()
    {
        var identity = _identityResolver.Resolve(User);
        return identity?.UserId ?? 0;
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
