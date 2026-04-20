using Cnn.Common.Contracts.Admin;
using Cnn.Common.Localization;
using Cnn.Api.Responses;
using Cnn.Api.Services.Admin;
using Cnn.Api.Services.Common;
using Cnn.Api.Services.Deletion;
using Cnn.Api.Services.Tasks.Workflow;
using Microsoft.AspNetCore.Mvc;

namespace Cnn.Api.Controllers.Admin;

[ApiController]
[Route("api/v1/admin/sites")]
public sealed class SitesController : ControllerBase
{
    private readonly ISiteService _service;
    private readonly IMessageLocalizer _localizer;
    private readonly IDeletionPreviewService _deletionPreviewService;
    private readonly IResourceDeleteRequestService _resourceDeleteRequestService;

    public SitesController(
        ISiteService service,
        IMessageLocalizer localizer,
        IDeletionPreviewService deletionPreviewService,
        IResourceDeleteRequestService resourceDeleteRequestService)
    {
        _service = service;
        _localizer = localizer;
        _deletionPreviewService = deletionPreviewService;
        _resourceDeleteRequestService = resourceDeleteRequestService;
    }

    [HttpGet]
    public async Task<IActionResult> ListAsync(
        [FromQuery(Name = "keyword")] string? keyword,
        [FromQuery(Name = "search_field")] string? searchField,
        [FromQuery(Name = "user_id")] long? userId,
        [FromQuery(Name = "user_package_id")] long? userPackageId,
        [FromQuery(Name = "group_id")] string? groupId,
        [FromQuery(Name = "node_group_id")] long? nodeGroupId,
        [FromQuery(Name = "status")] string? status,
        [FromQuery(Name = "https")] string? https,
        [FromQuery(Name = "page")] int page = 1,
        [FromQuery(Name = "pageSize")] int pageSize = 10,
        [FromQuery(Name = "size")] int? size = null,
        CancellationToken cancellationToken = default)
    {
        var query = new SiteListQuery
        {
            Keyword = keyword,
            SearchField = searchField,
            UserId = userId,
            UserPackageId = userPackageId,
            GroupId = groupId,
            NodeGroupId = nodeGroupId,
            Status = status,
            Https = https,
            Page = page,
            PageSize = pageSize,
            Size = size
        };

        var result = await _service.ListAsync(query, null, true, cancellationToken);
        return ToResponse(result);
    }

    [HttpGet("export")]
    public async Task<IActionResult> ExportAsync(
        [FromQuery(Name = "keyword")] string? keyword,
        [FromQuery(Name = "search_field")] string? searchField,
        [FromQuery(Name = "user_id")] long? userId,
        [FromQuery(Name = "user_package_id")] long? userPackageId,
        [FromQuery(Name = "group_id")] string? groupId,
        [FromQuery(Name = "node_group_id")] long? nodeGroupId,
        [FromQuery(Name = "status")] string? status,
        [FromQuery(Name = "https")] string? https,
        [FromQuery(Name = "page")] int page = 1,
        [FromQuery(Name = "pageSize")] int pageSize = 10,
        [FromQuery(Name = "size")] int? size = null,
        CancellationToken cancellationToken = default)
    {
        var query = new SiteListQuery
        {
            Keyword = keyword,
            SearchField = searchField,
            UserId = userId,
            UserPackageId = userPackageId,
            GroupId = groupId,
            NodeGroupId = nodeGroupId,
            Status = status,
            Https = https,
            Page = page,
            PageSize = pageSize,
            Size = size
        };

        var result = await _service.ExportAsync(query, null, true, cancellationToken);
        return ToResponse(result);
    }

    [HttpGet("resolve")]
    public async Task<IActionResult> ResolveAsync([FromQuery(Name = "domain")] string? domain, CancellationToken cancellationToken)
    {
        var result = await _service.ResolveAsync(domain ?? string.Empty, cancellationToken);
        return ToResponse(result);
    }

    [HttpGet("{id:long}")]
    public async Task<IActionResult> GetAsync(long id, CancellationToken cancellationToken)
    {
        var result = await _service.GetAsync(id, null, true, cancellationToken);
        return ToResponse(result);
    }

    [HttpPost]
    public async Task<IActionResult> CreateAsync([FromBody] SiteCreateRequest request, CancellationToken cancellationToken)
    {
        var result = await _service.CreateAsync(request, null, true, cancellationToken);
        return ToResponse(result);
    }

    [HttpPut("{id:long}")]
    public async Task<IActionResult> UpdateAsync(long id, [FromBody] SiteUpdateRequest request, CancellationToken cancellationToken)
    {
        var result = await _service.UpdateAsync(id, request, null, true, cancellationToken);
        return ToResponse(result);
    }

    [HttpPost("batch")]
    public async Task<IActionResult> BatchCreateAsync([FromBody] SiteBatchCreateRequest request, CancellationToken cancellationToken)
    {
        var result = await _service.BatchCreateAsync(request, null, true, cancellationToken);
        return ToResponse(result);
    }

    [HttpGet("batch/{id}/progress")]
    public async Task<IActionResult> BatchProgressAsync(string id, CancellationToken cancellationToken)
    {
        var result = await _service.BatchProgressAsync(id, null, true, cancellationToken);
        return ToResponse(result);
    }

    [HttpPost("batch_update")]
    public async Task<IActionResult> BatchUpdateAsync([FromBody] SiteBatchUpdateRequest request, CancellationToken cancellationToken)
    {
        var result = await _service.BatchUpdateAsync(request, null, true, cancellationToken);
        return ToResponse(result);
    }

    [HttpPost("batch_action")]
    public async Task<IActionResult> BatchActionAsync([FromBody] SiteBatchActionRequest request, CancellationToken cancellationToken)
    {
        var result = await _service.BatchActionAsync(request, null, true, cancellationToken);
        return ToResponse(result);
    }

    [HttpPost("apply_cert")]
    public async Task<IActionResult> ApplyCertAsync([FromBody] SiteApplyCertRequest request, CancellationToken cancellationToken)
    {
        var result = await _service.ApplyCertAsync(request, null, true, cancellationToken);
        return ToResponse(result);
    }

    [HttpDelete("{id:long}")]
    public Task<IActionResult> DeleteAsync(long id, CancellationToken cancellationToken)
    {
        return DeleteWorkflowResponseHelper.RequestAsync(
            this,
            _localizer,
            _resourceDeleteRequestService,
            DeleteRequestCommandFactory.Create(ResourceTypes.Site, id),
            cancellationToken);
    }

    [HttpGet("{id:long}/delete_preview")]
    public Task<IActionResult> DeletePreviewAsync(long id, CancellationToken cancellationToken)
    {
        return DeleteWorkflowResponseHelper.PreviewAsync(
            this,
            _localizer,
            _deletionPreviewService,
            ResourceTypes.Site,
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
            DeleteRequestCommandFactory.Create(ResourceTypes.Site, id),
            cancellationToken);
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
