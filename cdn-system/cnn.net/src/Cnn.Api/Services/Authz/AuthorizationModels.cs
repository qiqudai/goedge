namespace Cnn.Api.Services.Authz;

public sealed record AccessContext(
    long? UserId,
    string Role,
    string? NodeId,
    string? TraceId,
    IReadOnlyDictionary<string, string> Claims) : IAccessContext;

public sealed record PermissionRule(string Permission, string MinRole, string? ResourceScope = null);

public sealed record AuthorizationDecision(bool Allowed, string Reason)
{
    public static AuthorizationDecision Allow() => new(true, "ok");
    public static AuthorizationDecision Deny(string reason) => new(false, reason);
}

public static class PermissionScopes
{
    public const string None = "none";
    public const string NodeSelf = "node:self";
    public const string UserSelf = "user:self";
    public const string TenantSelf = "tenant:self";
    public const string PluginExecute = "plugin:execute";
}

public static class RoleLevels
{
    private static readonly Dictionary<string, int> Levels = new(StringComparer.OrdinalIgnoreCase)
    {
        ["plugin"] = 10,
        ["agent"] = 20,
        ["user"] = 30,
        ["operator"] = 40,
        ["admin"] = 50,
        ["root"] = 60
    };

    public static bool Meets(string actualRole, string minRole)
    {
        var actual = ResolveLevel(actualRole);
        var minimum = ResolveLevel(minRole);
        return actual >= minimum;
    }

    private static int ResolveLevel(string? role)
    {
        if (string.IsNullOrWhiteSpace(role))
        {
            return 0;
        }

        return Levels.TryGetValue(role.Trim(), out var value) ? value : 0;
    }
}
