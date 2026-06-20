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
[Route("api/v1/admin/node-groups")]
public sealed class NodeGroupsController : ControllerBase
{
    private readonly INodeGroupService _service;
    private readonly IMessageLocalizer _localizer;
    private readonly IDeletionPreviewService _deletionPreviewService;
    private readonly IResourceDeleteRequestService _resourceDeleteRequestService;

    public NodeGroupsController(
        INodeGroupService service,
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
    public async Task<IActionResult> ListAsync([FromQuery] NodeGroupListQuery query, CancellationToken cancellationToken)
    {
        var result = await _service.ListAsync(query, cancellationToken);
        return ToResponse(result);
    }

    [HttpPost]
    public async Task<IActionResult> CreateAsync([FromBody] NodeGroupUpsertRequest request, CancellationToken cancellationToken)
    {
        var result = await _service.CreateAsync(request, cancellationToken);
        return ToResponse(result);
    }

    [HttpPut("{id:long}")]
    public async Task<IActionResult> UpdateAsync(long id, [FromBody] NodeGroupUpsertRequest request, CancellationToken cancellationToken)
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
            DeleteRequestCommandFactory.Create(ResourceTypes.LineGroup, id),
            cancellationToken);
    }

    [HttpGet("{id:long}/delete_preview")]
    public Task<IActionResult> DeletePreviewAsync(long id, CancellationToken cancellationToken)
    {
        return DeleteWorkflowResponseHelper.PreviewAsync(
            this,
            _localizer,
            _deletionPreviewService,
            ResourceTypes.LineGroup,
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
            DeleteRequestCommandFactory.Create(ResourceTypes.LineGroup, id),
            cancellationToken);
    }

    [HttpGet("{id:long}/resolution")]
    public async Task<IActionResult> GetResolutionAsync(long id, [FromQuery] NodeGroupResolutionQuery query, CancellationToken cancellationToken)
    {
        var result = await _service.GetResolutionAsync(id, query, cancellationToken);
        return ToResponse(result);
    }

    [HttpPost("{id:long}/resolution/assign")]
    public async Task<IActionResult> AssignResolutionAsync(long id, [FromBody] NodeGroupAssignRequest request, CancellationToken cancellationToken)
    {
        var result = await _service.AssignResolutionAsync(id, request, cancellationToken);
        return ToResponse(result);
    }

    [HttpPost("{id:long}/resolution/action")]
    public async Task<IActionResult> ResolutionActionAsync(long id, [FromBody] NodeGroupActionRequest request, CancellationToken cancellationToken)
    {
        var result = await _service.ResolutionActionAsync(id, request, cancellationToken);
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
