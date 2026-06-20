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
[Route("api/v1/admin/forward_groups")]
public sealed class ForwardGroupsController : ControllerBase
{
    private readonly IForwardGroupService _service;
    private readonly IMessageLocalizer _localizer;
    private readonly IDeletionPreviewService _deletionPreviewService;
    private readonly IResourceDeleteRequestService _resourceDeleteRequestService;

    public ForwardGroupsController(
        IForwardGroupService service,
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
        [FromQuery(Name = "user_id")] long? userId,
        CancellationToken cancellationToken)
    {
        var result = await _service.ListAsync(keyword, userId, true, cancellationToken);
        return ToResponse(result);
    }

    [HttpPost]
    public async Task<IActionResult> CreateAsync([FromBody] ForwardGroupUpsertRequest request, CancellationToken cancellationToken)
    {
        var result = await _service.CreateAsync(request, null, true, cancellationToken);
        return ToResponse(result);
    }

    [HttpPut]
    public async Task<IActionResult> UpdateAsync([FromBody] ForwardGroupUpsertRequest request, CancellationToken cancellationToken)
    {
        var result = await _service.UpdateAsync(request, null, true, cancellationToken);
        return ToResponse(result);
    }

    [HttpDelete]
    public async Task<IActionResult> DeleteAsync([FromBody] ForwardGroupDeleteRequest request, CancellationToken cancellationToken)
    {
        var result = await _service.DeleteAsync(request.Id, null, true, cancellationToken);
        return ToResponse(result);
    }

    [HttpGet("{id:long}/delete_preview")]
    public Task<IActionResult> DeletePreviewAsync(long id, CancellationToken cancellationToken)
    {
        return DeleteWorkflowResponseHelper.PreviewAsync(
            this,
            _localizer,
            _deletionPreviewService,
            ResourceTypes.StreamGroup,
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
            DeleteRequestCommandFactory.Create(ResourceTypes.StreamGroup, id),
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
