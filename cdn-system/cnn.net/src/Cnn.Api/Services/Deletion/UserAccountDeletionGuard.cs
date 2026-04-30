using System.Linq;
using Cnn.Domain.Entities;
using SqlSugar;
using StreamEntity = Cnn.Domain.Entities.Stream;

namespace Cnn.Api.Services.Deletion;

public sealed class UserAccountDeletionGuard : IDeletionGuard
{
    private readonly ISqlSugarClient _db;

    public UserAccountDeletionGuard(ISqlSugarClient db)
    {
        _db = db;
    }

    public string ResourceType => ResourceTypes.UserAccount;

    public async Task<DeleteGuardResult> CheckAsync(long resourceId, CancellationToken cancellationToken)
    {
        if (resourceId <= 0)
        {
            return DeleteGuardResult.Deny("INVALID_RESOURCE_ID", "用户 ID 无效");
        }

        if (resourceId == 1)
        {
            return DeleteGuardResult.Deny("BUILTIN_ADMIN_PROTECTED", "Built-in admin (ID=1) cannot be deleted");
        }

        var exists = await _db.Queryable<User>()
            .Where(x => x.Id == resourceId)
            .AnyAsync();

        if (!exists)
        {
            return DeleteGuardResult.Deny("USER_NOT_FOUND", "用户不存在，无法删除。");
        }

        var refs = new List<DeleteReferenceItem>();
        var uid = (int)resourceId;

        void AppendRef(string type, long count, string relation)
        {
            if (count <= 0)
            {
                return;
            }

            refs.Add(new DeleteReferenceItem
            {
                ResourceType = type,
                ResourceId = count,
                DisplayName = $"{type}:{count}",
                Relation = relation
            });
        }

        AppendRef(ResourceTypes.Certificate, await _db.Queryable<Cert>().Where(x => x.Uid == uid).CountAsync(), "cert.uid");
        AppendRef(ResourceTypes.Subscription, await _db.Queryable<UserPackage>().Where(x => x.Uid == uid).CountAsync(), "user_package.uid");
        AppendRef(ResourceTypes.Site, await _db.Queryable<Site>().Where(x => x.Uid == uid).CountAsync(), "site.uid");
        AppendRef(ResourceTypes.StreamApp, await _db.Queryable<StreamEntity>().Where(x => x.Uid == uid).CountAsync(), "stream.uid");
        AppendRef("dnsapi", await _db.Queryable<Dnsapi>().Where(x => x.Uid == uid).CountAsync(), "dnsapi.uid");
        AppendRef(ResourceTypes.AclRule, await _db.Queryable<Acl>().Where(x => x.Uid == uid).CountAsync(), "acl.uid");
        AppendRef(ResourceTypes.CcRuleGroup, await _db.Queryable<CcRule>().Where(x => x.Uid == uid).CountAsync(), "cc_rule.uid");
        AppendRef(ResourceTypes.CcMatcher, await _db.Queryable<CcMatch>().Where(x => x.Uid == uid).CountAsync(), "cc_match.uid");
        AppendRef(ResourceTypes.CcFilter, await _db.Queryable<CcFilter>().Where(x => x.Uid == uid).CountAsync(), "cc_filter.uid");
        AppendRef("api_key", await _db.Queryable<ApiKey>().Where(x => x.Uid == uid).CountAsync(), "api_key.uid");

        if (refs.Count > 0)
        {
            var summary = string.Join(", ", refs.Select(x => x.DisplayName));
            return DeleteGuardResult.Deny("USER_HAS_DEPENDENCIES", $"User has related resources, delete blocked: {summary}", refs);
        }

        return DeleteGuardResult.Allow();
    }
}
