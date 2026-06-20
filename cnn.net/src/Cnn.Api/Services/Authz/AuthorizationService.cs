using Cnn.Common.Contracts;

namespace Cnn.Api.Services.Authz;

public sealed class AuthorizationService : IAuthorizationService
{
    private readonly IPermissionCatalog _catalog;
    private readonly IAuthorizationAuditSink _auditSink;

    public AuthorizationService(IPermissionCatalog catalog, IAuthorizationAuditSink auditSink)
    {
        _catalog = catalog;
        _auditSink = auditSink;
    }

    public AuthorizationDecision Check(IAccessContext context, string permission, string? resourceId = null)
    {
        var started = DateTime.UtcNow;
        AuthorizationDecision result;

        if (string.IsNullOrWhiteSpace(permission))
        {
            result = AuthorizationDecision.Allow();
            WriteAudit(context, permission, resourceId, result, started);
            return result;
        }

        if (!_catalog.TryGet(permission, out var rule))
        {
            result = AuthorizationDecision.Deny("permission_not_defined");
            WriteAudit(context, permission, resourceId, result, started);
            return result;
        }

        if (!RoleLevels.Meets(context.Role, rule.MinRole))
        {
            result = AuthorizationDecision.Deny("role_not_enough");
            WriteAudit(context, permission, resourceId, result, started);
            return result;
        }

        if (IsOperatorForbidden(context.Role, permission))
        {
            result = AuthorizationDecision.Deny("operator_scope_forbidden");
            WriteAudit(context, permission, resourceId, result, started);
            return result;
        }

        if (IsPluginForbidden(context.Role, permission))
        {
            result = AuthorizationDecision.Deny("plugin_scope_forbidden");
            WriteAudit(context, permission, resourceId, result, started);
            return result;
        }

        if (!CheckResourceScope(rule.ResourceScope, context, resourceId, permission, out var scopeReason))
        {
            result = AuthorizationDecision.Deny(scopeReason);
            WriteAudit(context, permission, resourceId, result, started);
            return result;
        }

        result = AuthorizationDecision.Allow();
        WriteAudit(context, permission, resourceId, result, started);
        return result;
    }

    public void Demand(IAccessContext context, string permission, string? resourceId = null)
    {
        var decision = Check(context, permission, resourceId);
        if (!decision.Allowed)
        {
            throw new AuthorizationException(decision.Reason);
        }
    }

    private static bool IsOperatorForbidden(string role, string permission)
    {
        if (!string.Equals(role, "operator", StringComparison.OrdinalIgnoreCase))
        {
            return false;
        }

        return permission.StartsWith("finance:", StringComparison.OrdinalIgnoreCase)
            || permission.Contains(":dangerous", StringComparison.OrdinalIgnoreCase);
    }

    private static bool IsPluginForbidden(string role, string permission)
    {
        if (!string.Equals(role, "plugin", StringComparison.OrdinalIgnoreCase))
        {
            return false;
        }

        return !permission.StartsWith("plugin:execute", StringComparison.OrdinalIgnoreCase);
    }

    private static bool CheckResourceScope(
        string? resourceScope,
        IAccessContext context,
        string? resourceId,
        string permission,
        out string reason)
    {
        reason = "resource_scope_mismatch";

        var scope = NormalizeScope(resourceScope);
        if (scope == PermissionScopes.None)
        {
            if (string.Equals(context.Role, "agent", StringComparison.OrdinalIgnoreCase)
                && !string.IsNullOrWhiteSpace(resourceId)
                && !string.IsNullOrWhiteSpace(context.NodeId)
                && !string.Equals(resourceId, context.NodeId, StringComparison.OrdinalIgnoreCase))
            {
                return false;
            }

            return true;
        }

        if (scope == PermissionScopes.PluginExecute)
        {
            return permission.StartsWith("plugin:execute", StringComparison.OrdinalIgnoreCase);
        }

        if (string.IsNullOrWhiteSpace(resourceId))
        {
            return true;
        }

        if (scope == PermissionScopes.NodeSelf)
        {
            if (string.IsNullOrWhiteSpace(context.NodeId))
            {
                return false;
            }

            return string.Equals(resourceId, context.NodeId, StringComparison.OrdinalIgnoreCase);
        }

        if (scope == PermissionScopes.UserSelf)
        {
            var uid = context.UserId?.ToString();
            if (string.IsNullOrWhiteSpace(uid))
            {
                uid = GetClaim(context, "uid", "user_id", "sub", "nameidentifier");
            }

            if (string.IsNullOrWhiteSpace(uid))
            {
                return false;
            }

            return string.Equals(resourceId, uid, StringComparison.OrdinalIgnoreCase);
        }

        if (scope == PermissionScopes.TenantSelf)
        {
            var tenant = GetClaim(context, "tenant_id", "tenant", "tenantid");
            if (string.IsNullOrWhiteSpace(tenant))
            {
                return false;
            }

            return string.Equals(resourceId, tenant, StringComparison.OrdinalIgnoreCase);
        }

        return true;
    }

    private static string NormalizeScope(string? scope)
    {
        if (string.IsNullOrWhiteSpace(scope))
        {
            return PermissionScopes.None;
        }

        var normalized = scope.Trim().ToLowerInvariant();
        return normalized switch
        {
            "node:self" => PermissionScopes.NodeSelf,
            "user:self" => PermissionScopes.UserSelf,
            "tenant:self" => PermissionScopes.TenantSelf,
            "plugin:execute" => PermissionScopes.PluginExecute,
            _ => PermissionScopes.None
        };
    }

    private static string? GetClaim(IAccessContext context, params string[] keys)
    {
        foreach (var key in keys)
        {
            if (context.Claims.TryGetValue(key, out var value) && !string.IsNullOrWhiteSpace(value))
            {
                return value.Trim();
            }
        }

        return null;
    }

    private void WriteAudit(IAccessContext context, string permission, string? resourceId, AuthorizationDecision decision, DateTime startedUtc)
    {
        var duration = DateTime.UtcNow - startedUtc;
        var micros = (long)(duration.TotalMilliseconds * 1000);

        _auditSink.Write(new AuthorizationAuditRecord(
            Timestamp: DateTimeOffset.UtcNow,
            Role: context.Role,
            UserId: context.UserId,
            Permission: permission,
            ResourceId: resourceId,
            Allowed: decision.Allowed,
            Reason: decision.Reason,
            TraceId: context.TraceId,
            NodeId: context.NodeId,
            DurationMicroseconds: micros));
    }
}

public sealed class AuthorizationException : Exception
{
    public AuthorizationException(string reason)
        : base(reason)
    {
    }

    public int ErrorCode => ErrorCodes.PermissionDenied;
}
