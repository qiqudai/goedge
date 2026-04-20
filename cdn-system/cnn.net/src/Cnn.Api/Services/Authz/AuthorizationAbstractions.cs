namespace Cnn.Api.Services.Authz;

public interface IAccessContext
{
    long? UserId { get; }
    string Role { get; }
    string? NodeId { get; }
    string? TraceId { get; }
    IReadOnlyDictionary<string, string> Claims { get; }
}

public interface IPermissionCatalog
{
    bool TryGet(string permission, out PermissionRule rule);
    IReadOnlyCollection<PermissionRule> ListAll();
}

public interface IAuthorizationService
{
    AuthorizationDecision Check(IAccessContext context, string permission, string? resourceId = null);
    void Demand(IAccessContext context, string permission, string? resourceId = null);
}

public interface IApiPermissionResolver
{
    string? Resolve(string path, string? method = null);
}
