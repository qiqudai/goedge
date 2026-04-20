using Cnn.Common.Contracts.Admin;
using Cnn.Api.Services.Deletion;
using Cnn.Api.Services.Tasks.Workflow;
using Cnn.Common.Localization;
using Cnn.Api.Responses;
using Cnn.Api.Services.Admin;
using Cnn.Api.Services.Common;
using Microsoft.AspNetCore.Mvc;

namespace Cnn.Api.Controllers.Admin;

[ApiController]
[Route("api/v1/admin/certs")]
public sealed class CertsController : ControllerBase
{
    private readonly ICertService _service;
    private readonly IMessageLocalizer _localizer;
    private readonly IDeletionPreviewService _deletionPreviewService;
    private readonly IResourceDeleteRequestService _resourceDeleteRequestService;

    public CertsController(
        ICertService service,
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
    public async Task<IActionResult> ListAsync([FromQuery] CertListQuery query, CancellationToken cancellationToken)
    {
        var result = await _service.ListAsync(query ?? new CertListQuery(), null, true, cancellationToken);
        return ToResponse(result);
    }

    [HttpPost]
    public async Task<IActionResult> CreateAsync([FromBody] CertCreateRequest request, CancellationToken cancellationToken)
    {
        var result = await _service.CreateAsync(request, null, true, cancellationToken);
        return ToResponse(result);
    }

    [HttpPut("{id:long}")]
    public async Task<IActionResult> UpdateAsync(long id, [FromBody] CertUpdateRequest request, CancellationToken cancellationToken)
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
            DeleteRequestCommandFactory.Create(ResourceTypes.Certificate, id),
            cancellationToken);
    }

    [HttpGet("{id:long}/delete_preview")]
    public Task<IActionResult> DeletePreviewAsync(long id, CancellationToken cancellationToken)
    {
        return DeleteWorkflowResponseHelper.PreviewAsync(
            this,
            _localizer,
            _deletionPreviewService,
            ResourceTypes.Certificate,
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
            DeleteRequestCommandFactory.Create(ResourceTypes.Certificate, id),
            cancellationToken);
    }

    [HttpPost("batch")]
    public async Task<IActionResult> BatchCreateAsync([FromBody] CertBatchCreateRequest request, CancellationToken cancellationToken)
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

    [HttpPost("wildcard")]
    public async Task<IActionResult> WildcardAsync([FromBody] CertWildcardRequest request, CancellationToken cancellationToken)
    {
        var result = await _service.WildcardCreateAsync(request, null, true, cancellationToken);
        return ToResponse(result);
    }

    [HttpPost("batch_action")]
    public async Task<IActionResult> BatchActionAsync([FromBody] CertBatchActionRequest request, CancellationToken cancellationToken)
    {
        var result = await _service.BatchActionAsync(request, null, true, cancellationToken);
        return ToResponse(result);
    }

    [HttpPost("reissue")]
    public async Task<IActionResult> ReissueAsync([FromBody] CertReissueRequest request, CancellationToken cancellationToken)
    {
        var result = await _service.ReissueAsync(request, cancellationToken);
        return ToResponse(result);
    }

    [HttpGet("{id:long}/dns_challenge")]
    public async Task<IActionResult> GetDnsChallengeAsync(long id, CancellationToken cancellationToken)
    {
        var result = await _service.GetDnsChallengeAsync(id, null, true, cancellationToken);
        return ToResponse(result);
    }

    [HttpPost("{id:long}/verify_dns")]
    public async Task<IActionResult> VerifyDnsChallengeAsync(long id, CancellationToken cancellationToken)
    {
        var result = await _service.VerifyDnsChallengeAsync(id, null, true, cancellationToken);
        return ToResponse(result);
    }

    [HttpGet("{id:long}/download")]
    public async Task<IActionResult> DownloadAsync(long id, [FromQuery] string? domain, CancellationToken cancellationToken)
    {
        var result = await _service.DownloadAsync(id, null, true, domain, cancellationToken);
        if (!result.Success || result.Data == null)
        {
            return Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, result.ErrorCode, result.MessageKey));
        }

        return File(result.Data.Data, "application/zip", result.Data.FileName);
    }

    [HttpGet("default_settings")]
    public async Task<IActionResult> GetDefaultSettingsAsync([FromQuery(Name = "user_id")] long? userId, CancellationToken cancellationToken)
    {
        var result = await _service.GetDefaultSettingsAsync(userId, true, cancellationToken);
        return ToResponse(result);
    }

    [HttpPost("default_settings")]
    public async Task<IActionResult> UpdateDefaultSettingsAsync([FromBody] CertDefaultSettingsRequest request, CancellationToken cancellationToken)
    {
        var result = await _service.UpdateDefaultSettingsAsync(request, null, true, cancellationToken);
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
