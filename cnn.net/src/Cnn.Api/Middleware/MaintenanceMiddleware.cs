using System.Text.Json;
using Microsoft.AspNetCore.Http;
using Cnn.Api.Services.Common;

namespace Cnn.Api.Middleware;

public sealed class MaintenanceMiddleware
{
    private static readonly JsonSerializerOptions JsonOptions = new(JsonSerializerDefaults.Web)
    {
        PropertyNameCaseInsensitive = true
    };

    private readonly RequestDelegate _next;

    public MaintenanceMiddleware(RequestDelegate next)
    {
        _next = next;
    }

    public async Task InvokeAsync(HttpContext context, ISystemConfigService systemConfigService)
    {
        if (!IsUserApiPath(context.Request.Path))
        {
            await _next(context);
            return;
        }

        if (context.User?.Identity?.IsAuthenticated != true)
        {
            await _next(context);
            return;
        }

        Dictionary<string, string> config;
        try
        {
            config = await systemConfigService.LoadSystemConfigAsync(context.RequestAborted);
        }
        catch
        {
            await _next(context);
            return;
        }

        if (!config.TryGetValue("maintain", out var raw) || string.IsNullOrWhiteSpace(raw))
        {
            await _next(context);
            return;
        }

        if (!TryParseMaintenance(raw, out var message))
        {
            await _next(context);
            return;
        }

        if (string.IsNullOrWhiteSpace(context.TraceIdentifier))
        {
            context.TraceIdentifier = Guid.NewGuid().ToString("N");
        }

        if (string.IsNullOrWhiteSpace(message))
        {
            message = "系统维护中，请稍后再试";
        }

        context.Response.StatusCode = StatusCodes.Status503ServiceUnavailable;
        context.Response.ContentType = "application/json";
        await context.Response.WriteAsJsonAsync(new
        {
            code = StatusCodes.Status503ServiceUnavailable,
            message,
            maintenance = true,
            data = new { message },
            trace_id = context.TraceIdentifier
        });
    }

    private static bool IsUserApiPath(PathString path)
    {
        if (!path.HasValue)
        {
            return false;
        }

        return path.Value!.StartsWith("/api/v1/user", StringComparison.OrdinalIgnoreCase);
    }

    private static bool TryParseMaintenance(string raw, out string message)
    {
        message = string.Empty;
        try
        {
            var payload = JsonSerializer.Deserialize<MaintenancePayload>(raw, JsonOptions);
            if (payload == null || payload.Enable != 1)
            {
                return false;
            }

            message = payload.Msg?.Trim() ?? string.Empty;
            return true;
        }
        catch
        {
            return false;
        }
    }

    private sealed class MaintenancePayload
    {
        public int Enable { get; set; }
        public string? Msg { get; set; }
    }
}
