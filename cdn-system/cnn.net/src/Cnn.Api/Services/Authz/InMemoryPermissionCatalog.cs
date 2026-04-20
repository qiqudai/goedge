namespace Cnn.Api.Services.Authz;

public sealed class InMemoryPermissionCatalog : IPermissionCatalog
{
    private static readonly PermissionRule[] DefaultRules =
    [
        new("api:admin", "admin"),
        new("api:user", "user"),
        new("api:acls", "user"),
        new("task:dispatch", "operator"),
        new("node:config:read", "operator"),
        new("node:config:write", "operator"),
        new("log:read:security", "operator"),
        new("log:read:user", "user"),
        new("finance:read", "admin"),
        new("finance:write", "admin"),
        new("global:dangerous:write", "admin"),
        new("agent:heartbeat:write", "agent", PermissionScopes.NodeSelf),
        new("agent:config:read", "agent", PermissionScopes.NodeSelf),
        new("agent:task:read", "agent", PermissionScopes.NodeSelf),
        new("agent:task:write", "agent", PermissionScopes.NodeSelf),
        new("agent:log:write", "agent", PermissionScopes.NodeSelf),
        new("agent:upgrade:read", "agent", PermissionScopes.NodeSelf),
        new("agent:cert:write", "agent", PermissionScopes.NodeSelf),
        new("agent:acme:write", "agent", PermissionScopes.NodeSelf),
        new("plugin:execute", "plugin", PermissionScopes.PluginExecute)
    ];

    private PermissionCatalogSnapshot _snapshot;

    public InMemoryPermissionCatalog()
    {
        _snapshot = PermissionCatalogSnapshot.Create(DefaultRules, version: 1);
    }

    public bool TryGet(string permission, out PermissionRule rule)
    {
        var snapshot = Volatile.Read(ref _snapshot);
        return snapshot.Index.TryGetValue(permission, out rule!);
    }

    public IReadOnlyCollection<PermissionRule> ListAll()
    {
        var snapshot = Volatile.Read(ref _snapshot);
        return snapshot.Rules;
    }

    public void ReplaceRules(IEnumerable<PermissionRule> rules)
    {
        var next = PermissionCatalogSnapshot.Create(rules, Volatile.Read(ref _snapshot).Version + 1);
        Volatile.Write(ref _snapshot, next);
    }

    private sealed class PermissionCatalogSnapshot
    {
        private PermissionCatalogSnapshot(
            long version,
            IReadOnlyDictionary<string, PermissionRule> index,
            IReadOnlyCollection<PermissionRule> rules)
        {
            Version = version;
            Index = index;
            Rules = rules;
        }

        public long Version { get; }
        public IReadOnlyDictionary<string, PermissionRule> Index { get; }
        public IReadOnlyCollection<PermissionRule> Rules { get; }

        public static PermissionCatalogSnapshot Create(IEnumerable<PermissionRule> rules, long version)
        {
            var list = (rules ?? Array.Empty<PermissionRule>())
                .Where(static r => !string.IsNullOrWhiteSpace(r.Permission) && !string.IsNullOrWhiteSpace(r.MinRole))
                .Select(static r => new PermissionRule(
                    Permission: r.Permission.Trim(),
                    MinRole: r.MinRole.Trim(),
                    ResourceScope: string.IsNullOrWhiteSpace(r.ResourceScope) ? null : r.ResourceScope.Trim()))
                .ToArray();

            var index = list.ToDictionary(static r => r.Permission, StringComparer.OrdinalIgnoreCase);
            return new PermissionCatalogSnapshot(version, index, list);
        }
    }
}
