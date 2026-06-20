using System.Security.Claims;
using System.Text.Json;
using Cnn.Common.Contracts;
using Cnn.Common.Localization;
using Cnn.Api.Responses;
using Cnn.Api.Services.Admin;
using Cnn.Api.Services.Auth;
using Cnn.Api.Services.Authz;
using Cnn.Api.Services.Common;
using Cnn.Domain.Entities;
using Microsoft.Extensions.Configuration;
using SqlSugar;
using Task = System.Threading.Tasks.Task;

namespace Cnn.Api.Middleware;

public sealed class ApiAuthMiddleware
{
    private readonly RequestDelegate _next;

    public ApiAuthMiddleware(RequestDelegate next)
    {
        _next = next;
    }

    public async Task InvokeAsync(
        HttpContext context,
        IMessageLocalizer localizer,
        IAuthTokenService tokenService,
        IOperationLogService operationLogService,
        IApiPermissionResolver permissionResolver,
        IAuthorizationService authorizationService,
        ISystemConfigService systemConfigService,
        IConfiguration configuration,
        ISqlSugarClient db)
    {
        var path = context.Request.Path.Value ?? string.Empty;
        if (IsPublicPath(path))
        {
            await _next(context);
            return;
        }

        var requiredPermission = permissionResolver.Resolve(path, context.Request.Method);

        if (path.StartsWith("/api/v1/agent", StringComparison.OrdinalIgnoreCase))
        {
            if (!await AuthenticateAgentAsync(context, localizer, configuration, db))
            {
                return;
            }

            if (!string.IsNullOrWhiteSpace(requiredPermission))
            {
                var agentResourceId = AccessResourceResolver.ResolveAgentNodeResourceId(context);

                var agentAccess = new AccessContext(
                    UserId: null,
                    Role: "agent",
                    NodeId: agentResourceId,
                    TraceId: context.TraceIdentifier,
                    Claims: new Dictionary<string, string>(StringComparer.OrdinalIgnoreCase)
                    {
                        ["role"] = "agent",
                        ["node_id"] = agentResourceId ?? string.Empty
                    });

                var agentAuthz = authorizationService.Check(agentAccess, requiredPermission, agentResourceId);
                if (!agentAuthz.Allowed)
                {
                    await WriteAuthFailAsync(context, localizer, ErrorCodes.PermissionDenied);
                    return;
                }
            }

            await _next(context);
            return;
        }

        if (requiredPermission == null)
        {
            await _next(context);
            return;
        }

        var token = ResolveBearerToken(context);
        if (string.IsNullOrWhiteSpace(token))
        {
            await WriteAuthFailAsync(context, localizer, ErrorCodes.AuthInvalid);
            return;
        }

        var validation = tokenService.Validate(token);
        if (!validation.Success)
        {
            await WriteAuthFailAsync(context, localizer, validation.Expired ? ErrorCodes.AuthExpired : ErrorCodes.AuthInvalid);
            return;
        }

        var role = validation.Role ?? string.Empty;
        var uid = validation.UserId > 0 ? (long?)validation.UserId : null;
        var access = new AccessContext(
            uid,
            role,
            context.Items["node_id"]?.ToString(),
            context.TraceIdentifier,
            BuildClaimsMap(validation.Principal));

        var resourceId = AccessResourceResolver.ResolveGeneralResourceId(context, path);
        var authz = authorizationService.Check(access, requiredPermission, resourceId);
        if (!authz.Allowed)
        {
            await WriteAuthFailAsync(context, localizer, ErrorCodes.PermissionDenied);
            return;
        }

        context.User = validation.Principal ?? new ClaimsPrincipal();

        var systemConfig = await systemConfigService.LoadSystemConfigAsync(context.RequestAborted);
        var ttl = ResolveLoginSessionTtl(systemConfig);
        if (ttl > TimeSpan.Zero)
        {
            var refreshed = tokenService.GenerateToken(validation.UserId, role, ttl);
            context.Response.Headers["X-Auth-Token"] = refreshed;
        }

        var shouldWriteOperationLog = ShouldWriteOperationLog(path, context.Request.Method, role, systemConfig);
        if (!shouldWriteOperationLog)
        {
            await _next(context);
            return;
        }

        try
        {
            await _next(context);
        }
        finally
        {
            await TryWriteOperationLogAsync(
                context,
                operationLogService,
                validation.UserId,
                role,
                path,
                systemConfig,
                context.RequestAborted);
        }
    }

    private static bool IsPublicPath(string path)
    {
        if (string.IsNullOrWhiteSpace(path))
        {
            return true;
        }

        if (path.StartsWith("/.well-known", StringComparison.OrdinalIgnoreCase))
        {
            return true;
        }

        if (path.StartsWith("/ws/", StringComparison.OrdinalIgnoreCase))
        {
            return true;
        }

        return path.Equals("/api/health", StringComparison.OrdinalIgnoreCase)
            || path.Equals("/health", StringComparison.OrdinalIgnoreCase)
            || path.Equals("/api/v1/login", StringComparison.OrdinalIgnoreCase)
            || path.Equals("/api/v1/admin/login", StringComparison.OrdinalIgnoreCase)
            || path.Equals("/api/v1/user/login", StringComparison.OrdinalIgnoreCase)
            || path.Equals("/api/v1/login/captcha", StringComparison.OrdinalIgnoreCase)
            || path.Equals("/api/v1/admin/login/captcha", StringComparison.OrdinalIgnoreCase)
            || path.Equals("/api/v1/user/login/captcha", StringComparison.OrdinalIgnoreCase)
            || path.Equals("/api/v1/register", StringComparison.OrdinalIgnoreCase)
            || path.Equals("/api/v1/user/register", StringComparison.OrdinalIgnoreCase)
            || path.Equals("/api/v1/system_info", StringComparison.OrdinalIgnoreCase)
            || path.Equals("/api/v1/pay/shkeeper/callback", StringComparison.OrdinalIgnoreCase);
    }

    private static string? ResolveBearerToken(HttpContext context)
    {
        var auth = context.Request.Headers.Authorization.ToString();
        if (string.IsNullOrWhiteSpace(auth))
        {
            return null;
        }

        var parts = auth.Split(' ', 2, StringSplitOptions.RemoveEmptyEntries);
        if (parts.Length != 2 || !string.Equals(parts[0], "Bearer", StringComparison.OrdinalIgnoreCase))
        {
            return null;
        }

        return parts[1].Trim();
    }

    private static async Task<bool> AuthenticateAgentAsync(
        HttpContext context,
        IMessageLocalizer localizer,
        IConfiguration configuration,
        ISqlSugarClient db)
    {
        var token = ResolveBearerToken(context);
        if (string.IsNullOrWhiteSpace(token))
        {
            await WriteAuthFailAsync(context, localizer, ErrorCodes.AgentAuthFailed);
            return false;
        }

        var globalToken = configuration["Agent:Token"];
        if (string.IsNullOrWhiteSpace(globalToken))
        {
            globalToken = configuration["AgentToken"];
        }

        if (!string.IsNullOrWhiteSpace(globalToken))
        {
            if (!string.Equals(token, globalToken.Trim(), StringComparison.Ordinal))
            {
                await WriteAuthFailAsync(context, localizer, ErrorCodes.AgentAuthFailed);
                return false;
            }

            context.Items["agent_token"] = token;
            return true;
        }

        var envToken = Environment.GetEnvironmentVariable("APP_AGENT_TOKEN");
        if (!string.IsNullOrWhiteSpace(envToken))
        {
            if (!string.Equals(token, envToken.Trim(), StringComparison.Ordinal))
            {
                await WriteAuthFailAsync(context, localizer, ErrorCodes.AgentAuthFailed);
                return false;
            }

            context.Items["agent_token"] = token;
            return true;
        }

        var node = await db.Queryable<Node>().Where(n => n.Token == token).FirstAsync();
        if (node == null)
        {
            await WriteAuthFailAsync(context, localizer, ErrorCodes.AgentAuthFailed);
            return false;
        }

        context.Items["agent_token"] = token;
        context.Items["node_id"] = node.Id;
        return true;
    }

    private static Task WriteAuthFailAsync(HttpContext context, IMessageLocalizer localizer, int code)
    {
        context.Response.StatusCode = StatusCodes.Status200OK;
        var payload = ApiResponseFactory.Fail<object>(context, localizer, code);
        return context.Response.WriteAsJsonAsync(payload);
    }

    private static TimeSpan ResolveLoginSessionTtl(IReadOnlyDictionary<string, string> cfg)
    {
        if (!cfg.TryGetValue("login_session_valid_time", out var raw))
        {
            return TimeSpan.FromHours(24);
        }

        raw = raw?.Trim();
        if (string.IsNullOrWhiteSpace(raw) || !int.TryParse(raw, out var seconds) || seconds <= 0)
        {
            return TimeSpan.FromHours(24);
        }

        return TimeSpan.FromSeconds(seconds);
    }

    private static bool ShouldWriteOperationLog(
        string path,
        string method,
        string role,
        IReadOnlyDictionary<string, string> cfg)
    {
        if (string.IsNullOrWhiteSpace(path) || string.IsNullOrWhiteSpace(method) || string.IsNullOrWhiteSpace(role))
        {
            return false;
        }

        if (method.Equals("GET", StringComparison.OrdinalIgnoreCase) ||
            method.Equals("HEAD", StringComparison.OrdinalIgnoreCase) ||
            method.Equals("OPTIONS", StringComparison.OrdinalIgnoreCase))
        {
            return false;
        }

        if (!path.StartsWith("/api/v1/admin", StringComparison.OrdinalIgnoreCase) &&
            !path.StartsWith("/api/v1/user", StringComparison.OrdinalIgnoreCase))
        {
            return false;
        }

        if (!role.Equals("admin", StringComparison.OrdinalIgnoreCase) &&
            !role.Equals("user", StringComparison.OrdinalIgnoreCase))
        {
            return false;
        }

        return ReadBool(cfg, DebugSwitchKeys.OperationLogEnabled, true);
    }

    private static async Task TryWriteOperationLogAsync(
        HttpContext context,
        IOperationLogService operationLogService,
        long userId,
        string role,
        string path,
        IReadOnlyDictionary<string, string> cfg,
        CancellationToken cancellationToken)
    {
        try
        {
            var request = context.Request;
            var response = context.Response;

            await operationLogService.WriteAsync(new OperationLogWriteRequest
            {
                UserId = (userId > 0 && userId <= int.MaxValue) ? (int)userId : null,
                Type = role,
                Action = $"{request.Method} {path}",
                Ip = ResolveClientIp(context, cfg),
                Process = $"status={response.StatusCode}",
                Content = JsonSerializer.Serialize(new
                {
                    path,
                    method = request.Method,
                    query = request.QueryString.HasValue ? request.QueryString.Value : string.Empty,
                    status = response.StatusCode,
                    content_size = request.ContentLength ?? 0,
                    trace_id = context.TraceIdentifier
                })
            }, cancellationToken);
        }
        catch
        {
            // Middleware should never fail the request because of operation logs.
        }
    }

    private static string ResolveClientIp(HttpContext context, IReadOnlyDictionary<string, string> cfg)
    {
        if (cfg.TryGetValue("master_client_ip_header", out var header) && !string.IsNullOrWhiteSpace(header))
        {
            var raw = context.Request.Headers[header.Trim()].ToString();
            if (!string.IsNullOrWhiteSpace(raw))
            {
                var index = raw.IndexOf(',');
                if (index >= 0)
                {
                    raw = raw[..index];
                }

                raw = raw.Trim();
                if (!string.IsNullOrWhiteSpace(raw))
                {
                    return raw;
                }
            }
        }

        var xff = context.Request.Headers["X-Forwarded-For"].ToString();
        if (!string.IsNullOrWhiteSpace(xff))
        {
            var index = xff.IndexOf(',');
            if (index >= 0)
            {
                xff = xff[..index];
            }

            xff = xff.Trim();
            if (!string.IsNullOrWhiteSpace(xff))
            {
                return xff;
            }
        }

        return context.Connection.RemoteIpAddress?.ToString() ?? string.Empty;
    }

    private static bool ReadBool(IReadOnlyDictionary<string, string> map, string key, bool defaultValue)
    {
        if (map.TryGetValue(key, out var value))
        {
            var normalized = value?.Trim().ToLowerInvariant();
            if (normalized is "1" or "true" or "yes" or "on")
            {
                return true;
            }

            if (normalized is "0" or "false" or "no" or "off")
            {
                return false;
            }
        }

        var alias = key.Replace('-', '_');
        if (!string.Equals(alias, key, StringComparison.Ordinal) && map.TryGetValue(alias, out value))
        {
            var normalized = value?.Trim().ToLowerInvariant();
            if (normalized is "1" or "true" or "yes" or "on")
            {
                return true;
            }

            if (normalized is "0" or "false" or "no" or "off")
            {
                return false;
            }
        }

        return defaultValue;
    }

    private static IReadOnlyDictionary<string, string> BuildClaimsMap(ClaimsPrincipal? principal)
    {
        if (principal == null)
        {
            return new Dictionary<string, string>(StringComparer.OrdinalIgnoreCase);
        }

        var map = new Dictionary<string, string>(StringComparer.OrdinalIgnoreCase);
        foreach (var claim in principal.Claims)
        {
            if (!map.ContainsKey(claim.Type))
            {
                map[claim.Type] = claim.Value;
            }
        }

        return map;
    }

}
