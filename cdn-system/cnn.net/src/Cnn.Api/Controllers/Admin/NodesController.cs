using Cnn.Common.Contracts.Admin;
using Cnn.Common.Contracts;
using Cnn.Common.Localization;
using Cnn.Api.Responses;
using Cnn.Api.Services.Admin;
using Cnn.Api.Services.Common;
using Cnn.Api.Services.Deletion;
using Cnn.Api.Services.Tasks.Workflow;
using Microsoft.AspNetCore.Mvc;

namespace Cnn.Api.Controllers.Admin;

[ApiController]
[Route("api/v1/admin/nodes")]
public sealed class NodesController : ControllerBase
{
    private readonly INodeService _service;
    private readonly IMessageLocalizer _localizer;
    private readonly IDeletionPreviewService _deletionPreviewService;
    private readonly IResourceDeleteRequestService _resourceDeleteRequestService;
    private readonly IResourceActionRequestService _resourceActionRequestService;

    public NodesController(
        INodeService service,
        IMessageLocalizer localizer,
        IDeletionPreviewService deletionPreviewService,
        IResourceDeleteRequestService resourceDeleteRequestService,
        IResourceActionRequestService resourceActionRequestService)
    {
        _service = service;
        _localizer = localizer;
        _deletionPreviewService = deletionPreviewService;
        _resourceDeleteRequestService = resourceDeleteRequestService;
        _resourceActionRequestService = resourceActionRequestService;
    }

    [HttpGet]
    public async Task<IActionResult> ListAsync([FromQuery] NodeListQuery query, CancellationToken cancellationToken)
    {
        var result = await _service.ListAsync(query, cancellationToken);
        return ToResponse(result);
    }

    [HttpGet("{id:long}/monitor_logs")]
    public async Task<IActionResult> MonitorLogsAsync(long id, [FromQuery] NodeMonitorLogQuery query, CancellationToken cancellationToken)
    {
        var result = await _service.ListMonitorLogsAsync(id, query, cancellationToken);
        return ToResponse(result);
    }

    [HttpPost]
    public async Task<IActionResult> CreateAsync([FromBody] NodeCreateRequest request, CancellationToken cancellationToken)
    {
        var result = await _service.CreateAsync(request, cancellationToken);
        return ToResponse(result);
    }

    [HttpPut("{id:long}")]
    public async Task<IActionResult> UpdateAsync(long id, [FromBody] NodeUpdateRequest request, CancellationToken cancellationToken)
    {
        var result = await _service.UpdateAsync(id, request, cancellationToken);
        return ToResponse(result);
    }

    [HttpPut("{id:long}/status")]
    public async Task<IActionResult> UpdateStatusAsync(long id, [FromBody] NodeStatusRequest request, CancellationToken cancellationToken)
    {
        if (request?.Enable == null)
        {
            return Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, Cnn.Common.Contracts.ErrorCodes.InvalidParam, "invalid_param"));
        }

        var result = await _resourceActionRequestService.RequestAsync(
            NodeActionCommandFactory.CreateStatusChange(id, request.Enable.Value),
            cancellationToken);

        if (result.Success)
        {
            return Ok(ApiResponseFactory.Ok(HttpContext, _localizer, result.Data));
        }

        return Ok(ApiResponseFactory.Fail<TaskRequestResult>(HttpContext, _localizer, result.ErrorCode, result.MessageKey));
    }

    [HttpPut("{id:long}/anti_blocking")]
    public async Task<IActionResult> UpdateAntiBlockingAsync(long id, [FromBody] NodeAntiBlockingRequest request, CancellationToken cancellationToken)
    {
        var result = await _service.UpdateAntiBlockingAsync(id, request, cancellationToken);
        return ToResponse(result);
    }

    [HttpDelete("{id:long}")]
    public Task<IActionResult> DeleteAsync(long id, CancellationToken cancellationToken)
    {
        return DeleteWorkflowResponseHelper.RequestAsync(
            this,
            _localizer,
            _resourceDeleteRequestService,
            DeleteRequestCommandFactory.Create(ResourceTypes.Node, id),
            cancellationToken);
    }

    [HttpGet("{id:long}/delete_preview")]
    public Task<IActionResult> DeletePreviewAsync(long id, CancellationToken cancellationToken)
    {
        return DeleteWorkflowResponseHelper.PreviewAsync(
            this,
            _localizer,
            _deletionPreviewService,
            ResourceTypes.Node,
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
            DeleteRequestCommandFactory.Create(ResourceTypes.Node, id),
            cancellationToken);
    }

    [HttpPost("batch")]
    public async Task<IActionResult> BatchAsync([FromBody] NodeBatchRequest request, CancellationToken cancellationToken)
    {
        var result = await _service.BatchAsync(request, cancellationToken);
        return ToResponse(result);
    }

    [HttpPost("batch_action")]
    public Task<IActionResult> BatchActionAsync([FromBody] NodeBatchRequest request, CancellationToken cancellationToken)
    {
        return BatchAsync(request, cancellationToken);
    }

    [HttpPost("{id:long}/install")]
    public async Task<IActionResult> InstallAsync(long id, CancellationToken cancellationToken)
    {
        var result = await _service.InstallAsync(id, cancellationToken);
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
