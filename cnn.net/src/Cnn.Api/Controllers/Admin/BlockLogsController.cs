using Cnn.Common.Contracts.Admin;
using Cnn.Api.Helpers;
using Cnn.Common.Localization;
using Cnn.Api.Responses;
using Cnn.Api.Services.Admin;
using Cnn.Api.Services.Stats;
using Microsoft.AspNetCore.Mvc;

namespace Cnn.Api.Controllers.Admin;

[ApiController]
[Route("api/v1/admin/logs/block")]
public sealed class BlockLogsController : ControllerBase
{
    private readonly IBlockLogService _service;
    private readonly IMessageLocalizer _localizer;

    public BlockLogsController(IBlockLogService service, IMessageLocalizer localizer)
    {
        _service = service;
        _localizer = localizer;
    }

    [HttpGet("current")]
    public async Task<IActionResult> CurrentAsync([FromQuery] BlockLogQuery query, CancellationToken cancellationToken)
    {
        var range = ResolveRange(query);
        var result = await _service.ListCurrentAsync(query, range.Start, range.End, null, true, cancellationToken);
        return ToResponse(result);
    }

    [HttpGet("stats")]
    public async Task<IActionResult> StatsAsync([FromQuery] BlockLogQuery query, CancellationToken cancellationToken)
    {
        var range = ResolveRange(query);
        var result = await _service.ListStatsAsync(query, range.Start, range.End, null, true, cancellationToken);
        return ToResponse(result);
    }

    [HttpGet("history")]
    public async Task<IActionResult> HistoryAsync([FromQuery] BlockLogQuery query, CancellationToken cancellationToken)
    {
        var range = ResolveHistoryRange();
        var result = await _service.ListHistoryAsync(query, range.Start, range.End, null, true, cancellationToken);
        return ToResponse(result);
    }

    [HttpPost("unblock_ip")]
    public async Task<IActionResult> UnblockIpAsync([FromBody] UnblockIpRequest request, CancellationToken cancellationToken)
    {
        var result = await _service.UnblockIpAsync(request.Ip ?? string.Empty, null, true, cancellationToken);
        return ToResponse(result);
    }

    [HttpPost("unblock_batch")]
    public async Task<IActionResult> UnblockBatchAsync([FromBody] UnblockBatchRequest request, CancellationToken cancellationToken)
    {
        var ips = request.Items?
            .Select(i => i.Ip ?? string.Empty)
            .ToList() ?? new List<string>();
        var result = await _service.UnblockBatchAsync(ips, null, true, cancellationToken);
        return ToResponse(result);
    }

    [HttpPost("unblock_site")]
    public async Task<IActionResult> UnblockSiteAsync([FromBody] UnblockSiteRequest request, CancellationToken cancellationToken)
    {
        var result = await _service.UnblockSiteAsync(request.SiteIds ?? Array.Empty<long>(), null, true, cancellationToken);
        return ToResponse(result);
    }

    private static StatsRange ResolveRange(BlockLogQuery query)
    {
        var key = string.IsNullOrWhiteSpace(query.TimeRange) ? query.Range : query.TimeRange;
        if (string.IsNullOrWhiteSpace(key))
        {
            key = "7d";
        }

        return StatsRangeResolver.Resolve(key, null, null, DateTime.Now);
    }

    private StatsRange ResolveHistoryRange()
    {
        var (start, end) = RequestTimeRange.Resolve(HttpContext.Request);
        if (start.HasValue && end.HasValue && end.Value >= start.Value)
        {
            return new StatsRange(start.Value, end.Value, TimeSpan.FromDays(1), "MM-dd");
        }

        return StatsRangeResolver.Resolve("7d", null, null, DateTime.Now);
    }

    private IActionResult ToResponse<T>(ServiceResult<T> result)
    {
        if (result.Success)
        {
            return Ok(ApiResponseFactory.Ok(HttpContext, _localizer, result.Data));
        }

        return Ok(ApiResponseFactory.Fail<T>(HttpContext, _localizer, result.ErrorCode, result.MessageKey));
    }

    public sealed class UnblockIpRequest
    {
        public string? Ip { get; set; }
    }

    public sealed class UnblockBatchRequest
    {
        public IReadOnlyList<UnblockBatchItem>? Items { get; set; }
    }

    public sealed class UnblockBatchItem
    {
        public string? Ip { get; set; }
    }

    public sealed class UnblockSiteRequest
    {
        public IReadOnlyList<long>? SiteIds { get; set; }
    }
}
