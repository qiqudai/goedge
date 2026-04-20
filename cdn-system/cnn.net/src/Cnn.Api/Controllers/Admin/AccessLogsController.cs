using Cnn.Common.Contracts.Admin;
using Cnn.Api.Helpers;
using Cnn.Common.Localization;
using Cnn.Api.Responses;
using Cnn.Api.Services;
using Cnn.Api.Services.Admin;
using Microsoft.AspNetCore.Mvc;

namespace Cnn.Api.Controllers.Admin;

[ApiController]
[Route("api/v1/admin/logs/access")]
public sealed class AccessLogsController : BaseApiController
{
    private readonly IAccessLogService _service;
    private readonly IAccessLogDownloadService _downloadService;

    public AccessLogsController(
        IAccessLogService service,
        IAccessLogDownloadService downloadService,
        IAdminIdentityResolver identityResolver,
        IMessageLocalizer localizer) : base(identityResolver, localizer)
    {
        _service = service;
        _downloadService = downloadService;
    }

    [HttpGet]
    public async Task<IActionResult> ListAsync([FromQuery] AccessLogQuery query, CancellationToken cancellationToken)
    {
        var (start, end) = RequestTimeRange.Resolve(HttpContext.Request);
        var result = await _service.ListAsync(query, start, end, null, true, cancellationToken);
        return ToResponse(result);
    }

    [HttpPost("downloads")]
    public async Task<IActionResult> ApplyDownloadAsync([FromBody] AccessLogDownloadApplyRequest request, CancellationToken cancellationToken)
    {
        var identity = _identityResolver.Resolve(User);
        var requesterUserId = identity?.UserId ?? 0;
        var result = await _downloadService.ApplyAsync(request, requesterUserId, true, ResolveRequesterIp(), cancellationToken);
        return ToResponse(result);
    }

    [HttpPut("downloads/{id:long}")]
    public async Task<IActionResult> CompleteDownloadAsync(long id, [FromBody] AccessLogDownloadCompleteRequest request, CancellationToken cancellationToken)
    {
        var identity = _identityResolver.Resolve(User);
        var requesterUserId = identity?.UserId ?? 0;
        var result = await _downloadService.CompleteAsync(id, request, requesterUserId, true, ResolveRequesterIp(), cancellationToken);
        return ToResponse(result);
    }

    [HttpGet("downloads")]
    public async Task<IActionResult> ListDownloadsAsync([FromQuery] AccessLogDownloadQuery query, CancellationToken cancellationToken)
    {
        var identity = _identityResolver.Resolve(User);
        var requesterUserId = identity?.UserId ?? 0;
        var result = await _downloadService.ListAsync(query, requesterUserId, true, cancellationToken);
        return ToResponse(result);
    }

    private string ResolveRequesterIp()
    {
        var header = HttpContext.Request.Headers["X-Forwarded-For"].ToString();
        if (!string.IsNullOrWhiteSpace(header))
        {
            var first = header.Split(',', 2)[0].Trim();
            if (!string.IsNullOrWhiteSpace(first))
            {
                return first;
            }
        }

        return HttpContext.Connection.RemoteIpAddress?.ToString() ?? string.Empty;
    }
}
