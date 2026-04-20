namespace Cnn.Api.Services.Authz;

public sealed class ApiPermissionResolver : IApiPermissionResolver
{
    private static readonly RouteRule[] Rules =
    [
        new("/api/v1/admin/debug", null, "task:dispatch"),
        new("/api/v1/admin/finance", "GET", "finance:read"),
        new("/api/v1/admin/finance", null, "finance:write"),
        new("/api/v1/admin/system/config", null, "global:dangerous:write"),
        new("/api/v1/admin/nodes", "GET", "node:config:read"),
        new("/api/v1/admin/nodes", null, "node:config:write"),
        new("/api/v1/admin/logs", "GET", "log:read:security"),
        new("/api/v1/user/logs", "GET", "log:read:user"),

        new("/api/v1/agent/config", "GET", "agent:config:read"),
        new("/api/v1/agent/tasks", "GET", "agent:task:read"),
        new("/api/v1/agent/tasks", null, "agent:task:write"),
        new("/api/v1/agent/logs", null, "agent:log:write"),
        new("/api/v1/agent/upgrade", "GET", "agent:upgrade:read"),
        new("/api/v1/agent/certs", null, "agent:cert:write"),
        new("/api/v1/agent/acme/tokens", null, "agent:acme:write"),
        new("/api/v1/agent/heartbeat", null, "agent:heartbeat:write"),
        new("/api/v1/agent/node/sync", null, "agent:heartbeat:write"),
        new("/api/v1/agent/l2", null, "agent:heartbeat:write"),

        new("/api/v1/admin", null, "api:admin"),
        new("/api/v1/user", null, "api:user"),
        new("/api/v1/acls", null, "api:acls")
    ];

    public string? Resolve(string path, string? method = null)
    {
        if (string.IsNullOrWhiteSpace(path))
        {
            return null;
        }

        foreach (var route in Rules)
        {
            if (!path.StartsWith(route.Prefix, StringComparison.OrdinalIgnoreCase))
            {
                continue;
            }

            if (route.Method == null || string.Equals(route.Method, method, StringComparison.OrdinalIgnoreCase))
            {
                return route.Permission;
            }
        }

        return null;
    }

    private sealed record RouteRule(string Prefix, string? Method, string Permission);
}
