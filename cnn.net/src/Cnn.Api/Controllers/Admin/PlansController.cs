using System.Text.Json;
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
[Route("api/v1/admin/plans")]
public sealed class PlansController : ControllerBase
{
    private readonly IPlanService _service;
    private readonly IMessageLocalizer _localizer;
    private readonly IDeletionPreviewService _deletionPreviewService;
    private readonly IResourceDeleteRequestService _resourceDeleteRequestService;

    public PlansController(
        IPlanService service,
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
    public async Task<IActionResult> ListAsync(CancellationToken cancellationToken)
    {
        var result = await _service.ListAsync(cancellationToken);
        return ToResponse(result);
    }

    [HttpGet("{id:long}")]
    public async Task<IActionResult> GetAsync(long id, CancellationToken cancellationToken)
    {
        var result = await _service.GetAsync(id, cancellationToken);
        return ToResponse(result);
    }

    [HttpPost]
    public async Task<IActionResult> CreateAsync([FromBody] JsonElement payload, CancellationToken cancellationToken)
    {
        var result = await _service.CreateAsync(payload, cancellationToken);
        return ToResponse(result);
    }

    [HttpPut("{id:long}")]
    public async Task<IActionResult> UpdateAsync(long id, [FromBody] JsonElement payload, CancellationToken cancellationToken)
    {
        var result = await _service.UpdateAsync(id, payload, cancellationToken);
        return ToResponse(result);
    }

    [HttpDelete("{id:long}")]
    public Task<IActionResult> DeleteAsync(long id, CancellationToken cancellationToken)
    {
        return DeleteWorkflowResponseHelper.RequestAsync(
            this,
            _localizer,
            _resourceDeleteRequestService,
            DeleteRequestCommandFactory.Create(ResourceTypes.ProductPlan, id),
            cancellationToken);
    }

    [HttpGet("{id:long}/delete_preview")]
    public Task<IActionResult> DeletePreviewAsync(long id, CancellationToken cancellationToken)
    {
        return DeleteWorkflowResponseHelper.PreviewAsync(
            this,
            _localizer,
            _deletionPreviewService,
            ResourceTypes.ProductPlan,
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
            DeleteRequestCommandFactory.Create(ResourceTypes.ProductPlan, id),
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
