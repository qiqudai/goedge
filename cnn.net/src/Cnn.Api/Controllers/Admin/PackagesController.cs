using Cnn.Common.Contracts.Admin;
using Cnn.Common.Localization;
using Cnn.Api.Responses;
using Cnn.Api.Services.Admin;
using Cnn.Api.Services.Common;
using Microsoft.AspNetCore.Mvc;

namespace Cnn.Api.Controllers.Admin;

[ApiController]
[Route("api/v1/admin/packages")]
public sealed class PackagesController : ControllerBase
{
    private readonly IAgentPackageService _service;
    private readonly IMessageLocalizer _localizer;

    public PackagesController(IAgentPackageService service, IMessageLocalizer localizer)
    {
        _service = service;
        _localizer = localizer;
    }

    [HttpGet]
    public async Task<IActionResult> ListAsync(CancellationToken cancellationToken)
    {
        var result = await _service.ListAsync(cancellationToken);
        return ToResponse(result);
    }

    [HttpPost]
    public async Task<IActionResult> UploadAsync([FromForm(Name = "version")] string? version, [FromForm(Name = "file")] IFormFile? file, CancellationToken cancellationToken)
    {
        var result = await _service.UploadAsync(version, file, cancellationToken);
        return ToResponse(result);
    }

    [HttpPost("grayscale")]
    public async Task<IActionResult> GrayScaleAsync([FromBody] AgentPackageGrayRequest request, CancellationToken cancellationToken)
    {
        var result = await _service.UpdateGrayAsync(request, cancellationToken);
        return ToResponse(result);
    }

    [HttpPost("stable")]
    public async Task<IActionResult> StableAsync([FromBody] AgentPackageStableRequest request, CancellationToken cancellationToken)
    {
        var result = await _service.SetStableAsync(request, cancellationToken);
        return ToResponse(result);
    }

    [HttpGet("nodes")]
    public async Task<IActionResult> ListNodesAsync([FromQuery(Name = "version")] string? version, CancellationToken cancellationToken)
    {
        var result = await _service.ListNodesAsync(version, cancellationToken);
        return ToResponse(result);
    }

    [HttpPost("upgrade")]
    public async Task<IActionResult> UpgradeAsync([FromBody] AgentPackageUpgradeRequest request, CancellationToken cancellationToken)
    {
        var apiBaseUrl = ResolveApiBaseUrl(Request);
        var result = await _service.UpgradeAsync(request, apiBaseUrl, cancellationToken);
        return ToResponse(result);
    }

    [HttpGet("upgrade/status")]
    public async Task<IActionResult> UpgradeStatusAsync([FromQuery(Name = "task_id")] long taskId, CancellationToken cancellationToken)
    {
        var result = await _service.UpgradeStatusAsync(taskId, cancellationToken);
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

    private static string? ResolveApiBaseUrl(HttpRequest request)
    {
        if (request == null)
        {
            return null;
        }

        var host = request.Host.HasValue ? request.Host.Value : string.Empty;
        if (string.IsNullOrWhiteSpace(host))
        {
            return null;
        }

        var scheme = string.IsNullOrWhiteSpace(request.Scheme) ? "http" : request.Scheme;
        var basePath = request.PathBase.HasValue ? request.PathBase.Value : string.Empty;
        return $"{scheme}://{host}{basePath}";
    }
}
