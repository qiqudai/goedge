using Cnn.Domain.Entities;
using SqlSugar;
using TaskEntity = Cnn.Domain.Entities.Task;
using Task = System.Threading.Tasks.Task;
using StreamEntity = Cnn.Domain.Entities.Stream;

namespace Cnn.Api.Services.Users;

public sealed class UserPurgeExecutor : IUserPurgeExecutor
{
    private readonly ISqlSugarClient _db;

    public UserPurgeExecutor(ISqlSugarClient db)
    {
        _db = db;
    }

    public async Task ExecuteAsync(long userId, CancellationToken cancellationToken)
    {
        if (userId <= 0)
        {
            throw new ArgumentOutOfRangeException(nameof(userId));
        }

        var userExists = await _db.Queryable<User>()
            .Where(x => x.Id == userId)
            .AnyAsync();
        if (!userExists)
        {
            return;
        }

        var siteIds = await _db.Queryable<Site>()
            .Where(x => x.Uid == userId)
            .Select(x => x.Id)
            .ToListAsync();

        var streamIds = await _db.Queryable<StreamEntity>()
            .Where(x => x.Uid == userId)
            .Select(x => x.Id)
            .ToListAsync();

        var userPackageIds = await _db.Queryable<UserPackage>()
            .Where(x => x.Uid == userId)
            .Select(x => x.Id)
            .ToListAsync();

        await _db.Ado.UseTranAsync(async () =>
        {
            if (siteIds.Count > 0)
            {
                await _db.Deleteable<MergeSiteGroup>()
                    .Where(x => x.SiteId.HasValue && siteIds.Contains(x.SiteId.Value))
                    .ExecuteCommandAsync();

                await _db.Deleteable<Config>()
                    .Where(x => x.ScopeName == "site" && x.ScopeId.HasValue && siteIds.Contains(x.ScopeId.Value))
                    .ExecuteCommandAsync();

                await _db.Deleteable<Site>()
                    .Where(x => x.Uid == userId)
                    .ExecuteCommandAsync();
            }

            if (streamIds.Count > 0)
            {
                await _db.Deleteable<MergeStreamGroup>()
                    .Where(x => x.StreamId.HasValue && streamIds.Contains(x.StreamId.Value))
                    .ExecuteCommandAsync();

                await _db.Deleteable<StreamEntity>()
                    .Where(x => x.Uid == userId)
                    .ExecuteCommandAsync();
            }

            if (userPackageIds.Count > 0)
            {
                await _db.Deleteable<UserPackageUp>()
                    .Where(x => x.Uid == userId || (x.UserPackage.HasValue && userPackageIds.Contains(x.UserPackage.Value)))
                    .ExecuteCommandAsync();

                await _db.Deleteable<Config>()
                    .Where(x => x.ScopeName == "user_package" && x.ScopeId.HasValue && userPackageIds.Contains(x.ScopeId.Value))
                    .ExecuteCommandAsync();

                await _db.Deleteable<UserPackage>()
                    .Where(x => x.Uid == userId)
                    .ExecuteCommandAsync();
            }

            await _db.Deleteable<Config>()
                .Where(x => x.ScopeName == "user" && x.ScopeId == userId)
                .ExecuteCommandAsync();

            await _db.Deleteable<Cert>().Where(x => x.Uid == userId).ExecuteCommandAsync();
            await _db.Deleteable<CcRule>().Where(x => x.Uid == userId).ExecuteCommandAsync();
            await _db.Deleteable<Acl>().Where(x => x.Uid == userId).ExecuteCommandAsync();
            await _db.Deleteable<CcFilter>().Where(x => x.Uid == userId).ExecuteCommandAsync();
            await _db.Deleteable<CcMatch>().Where(x => x.Uid == userId).ExecuteCommandAsync();
            await _db.Deleteable<Dnsapi>().Where(x => x.Uid == userId).ExecuteCommandAsync();
            await _db.Deleteable<SiteGroup>().Where(x => x.Uid == userId).ExecuteCommandAsync();
            await _db.Deleteable<StreamGroup>().Where(x => x.Uid == userId).ExecuteCommandAsync();
            await _db.Deleteable<ApiKey>().Where(x => x.Uid == userId).ExecuteCommandAsync();
            await _db.Deleteable<Order>().Where(x => x.Uid == userId).ExecuteCommandAsync();
            await _db.Deleteable<MessageSub>().Where(x => x.Uid == userId).ExecuteCommandAsync();
            await _db.Deleteable<MessageSend>().Where(x => x.Uid == userId).ExecuteCommandAsync();
            await _db.Deleteable<MessageRead>().Where(x => x.Uid == userId).ExecuteCommandAsync();
            await _db.Deleteable<ResCount>().Where(x => x.Uid == userId).ExecuteCommandAsync();
            await _db.Deleteable<LoginLog>().Where(x => x.Uid == userId).ExecuteCommandAsync();
            await _db.Deleteable<OpLog>().Where(x => x.Uid == userId).ExecuteCommandAsync();

            await _db.Deleteable<TaskEntity>()
                .Where(x => x.Res != null && (x.Res.Contains($"\"owner_user_id\":{userId}") || x.Res.Contains($"\"operator_user_id\":{userId}")))
                .ExecuteCommandAsync();

            await _db.Deleteable<User>()
                .Where(x => x.Id == userId)
                .ExecuteCommandAsync();
        });
    }
}
