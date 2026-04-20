using Cnn.Domain.Entities;
using SqlSugar;
using TaskEntity = Cnn.Domain.Entities.Task;
using StreamEntity = Cnn.Domain.Entities.Stream;

namespace Cnn.Api.Services.Users;

public sealed class UserPurgePlanner : IUserPurgePlanner
{
    private readonly ISqlSugarClient _db;

    public UserPurgePlanner(ISqlSugarClient db)
    {
        _db = db;
    }

    public async Task<UserPurgePlan> PlanAsync(long userId, CancellationToken cancellationToken)
    {
        if (userId <= 0)
        {
            return new UserPurgePlan();
        }

        var user = await _db.Queryable<User>()
            .Where(x => x.Id == userId)
            .FirstAsync();

        var summary = new UserPurgeSummary
        {
            UserId = userId,
            Username = user?.Name?.Trim(),
            SiteCount = await _db.Queryable<Site>().Where(x => x.Uid == userId).CountAsync(),
            StreamCount = await _db.Queryable<StreamEntity>().Where(x => x.Uid == userId).CountAsync(),
            CertificateCount = await _db.Queryable<Cert>().Where(x => x.Uid == userId).CountAsync(),
            RuleCount = await _db.Queryable<CcRule>().Where(x => x.Uid == userId).CountAsync(),
            SiteGroupCount = await _db.Queryable<SiteGroup>().Where(x => x.Uid == userId).CountAsync(),
            SubscriptionCount = await _db.Queryable<UserPackage>().Where(x => x.Uid == userId).CountAsync(),
            DefaultConfigCount = await _db.Queryable<Config>()
                .Where(x => x.ScopeName == "user" && x.ScopeId == userId)
                .CountAsync(),
            TaskCount = await _db.Queryable<TaskEntity>()
                .Where(x => x.Res != null && x.Res.Contains($"\"owner_user_id\":{userId}"))
                .CountAsync()
        };

        return new UserPurgePlan
        {
            Summary = summary,
            Steps = new[]
            {
                "删除用户下的站点与站点分组关系",
                "删除用户下的四层转发与转发分组关系",
                "删除用户下的已售套餐及其配置",
                "删除用户下的证书、CC 规则、ACL、DNS API、分组与配置",
                "删除用户相关任务、消息与最终用户记录"
            }
        };
    }
}
