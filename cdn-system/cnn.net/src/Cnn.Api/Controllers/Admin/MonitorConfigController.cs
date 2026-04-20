using System.Text;
using System.Text.Json;
using Cnn.Common.Contracts;
using Cnn.Common.Contracts.Admin;
using Cnn.Common.Localization;
using Cnn.Api.Responses;
using Cnn.Api.Services.Admin;
using Cnn.Api.Services.Common;
using Microsoft.AspNetCore.Mvc;

namespace Cnn.Api.Controllers.Admin;

[ApiController]
[Route("api/v1/admin/monitor_config")]
public sealed class MonitorConfigController : ControllerBase
{
    private static readonly JsonSerializerOptions JsonOptions = new(JsonSerializerDefaults.Web)
    {
        PropertyNameCaseInsensitive = true
    };

    private readonly IMonitorConfigService _service;
    private readonly IMessageLocalizer _localizer;

    public MonitorConfigController(IMonitorConfigService service, IMessageLocalizer localizer)
    {
        _service = service;
        _localizer = localizer;
    }

    [HttpGet]
    public async Task<IActionResult> GetAsync(CancellationToken cancellationToken)
    {
        var result = await _service.GetAsync(cancellationToken);
        return ToResponse(result);
    }

    [HttpPost]
    public async Task<IActionResult> UpdateAsync(CancellationToken cancellationToken)
    {
        string raw;
        using (var reader = new StreamReader(Request.Body, Encoding.UTF8))
        {
            raw = await reader.ReadToEndAsync();
        }

        if (string.IsNullOrWhiteSpace(raw))
        {
            return Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, ErrorCodes.InvalidParam, "invalid_json"));
        }

        NodeMonitorConfigDto? config;
        try
        {
            config = JsonSerializer.Deserialize<NodeMonitorConfigDto>(raw, JsonOptions);
        }
        catch
        {
            return Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, ErrorCodes.InvalidParam, "invalid_json"));
        }

        if (config == null)
        {
            return Ok(ApiResponseFactory.Fail<object>(HttpContext, _localizer, ErrorCodes.InvalidParam, "invalid_json"));
        }

        var result = await _service.UpdateAsync(config, cancellationToken);
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
