using Cnn.Common.Contracts;
using Cnn.Common.Localization;
using Cnn.Api.Responses;
using Cnn.Api.Services.Auth;
using Cnn.Api.Services.Common;
using Microsoft.AspNetCore.Mvc;

namespace Cnn.Api.Controllers;

[ApiController]
[Route("api/v1")]
public sealed class AuthController : ControllerBase
{
    private readonly IAuthService _authService;
    private readonly IMessageLocalizer _localizer;
    private readonly ISystemConfigService _systemConfigService;

    public AuthController(IAuthService authService, IMessageLocalizer localizer, ISystemConfigService systemConfigService)
    {
        _authService = authService;
        _localizer = localizer;
        _systemConfigService = systemConfigService;
    }

    [HttpPost("login")]
    [HttpPost("admin/login")]
    [HttpPost("user/login")]
    public async Task<IActionResult> LoginAsync([FromBody] LoginRequest request, CancellationToken cancellationToken)
    {
        var clientIp = await ResolveClientIpAsync(cancellationToken);
        var host = ResolveRequestHost();
        var result = await _authService.LoginAsync(request, clientIp, host, cancellationToken);
        return ToResponse(result);
    }

    [HttpPost("login/captcha")]
    [HttpPost("admin/login/captcha")]
    [HttpPost("user/login/captcha")]
    public async Task<IActionResult> SendCaptchaAsync([FromBody] LoginCaptchaRequest request, CancellationToken cancellationToken)
    {
        var clientIp = await ResolveClientIpAsync(cancellationToken);
        var result = await _authService.SendLoginCaptchaAsync(request, clientIp, cancellationToken);
        return ToResponse(result);
    }

    [HttpPost("register")]
    [HttpPost("user/register")]
    public async Task<IActionResult> RegisterAsync([FromBody] RegisterRequest request, CancellationToken cancellationToken)
    {
        var result = await _authService.RegisterAsync(request, cancellationToken);
        return ToResponse(result);
    }

    private IActionResult ToResponse(ServiceResult<object> result)
    {
        if (result.Success)
        {
            return Ok(ApiResponseFactory.Ok(HttpContext, _localizer, result.Data));
        }

        return Ok(ApiResponseFactory.Fail(HttpContext, _localizer, result.ErrorCode, result.MessageKey, null, result.Data));
    }

    private async Task<string> ResolveClientIpAsync(CancellationToken cancellationToken)
    {
        var cfg = await _systemConfigService.LoadSystemConfigAsync(cancellationToken);
        if (cfg.TryGetValue("master_client_ip_header", out var header) && !string.IsNullOrWhiteSpace(header))
        {
            var raw = Request.Headers[header.Trim()].ToString();
            if (!string.IsNullOrWhiteSpace(raw))
            {
                var idx = raw.IndexOf(',');
                if (idx >= 0)
                {
                    raw = raw[..idx];
                }
                raw = raw.Trim();
                if (!string.IsNullOrWhiteSpace(raw))
                {
                    return raw;
                }
            }
        }

        return HttpContext.Connection.RemoteIpAddress?.ToString() ?? string.Empty;
    }

    private string ResolveRequestHost()
    {
        var host = Request.Headers["X-Forwarded-Host"].ToString();
        if (string.IsNullOrWhiteSpace(host))
        {
            host = Request.Host.Value ?? string.Empty;
        }

        if (string.IsNullOrWhiteSpace(host))
        {
            return string.Empty;
        }

        if (host.Contains(','))
        {
            host = host.Split(',', 2)[0];
        }

        return host.Trim();
    }
}
