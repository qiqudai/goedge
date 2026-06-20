using Cnn.Domain.Entities;
using SqlSugar;
using StreamEntity = Cnn.Domain.Entities.Stream;

namespace Cnn.Api.Services.Deletion;

public sealed class StreamDeletionGuard : IDeletionGuard
{
    private readonly ISqlSugarClient _db;

    public StreamDeletionGuard(ISqlSugarClient db)
    {
        _db = db;
    }

    public string ResourceType => ResourceTypes.StreamApp;

    public async Task<DeleteGuardResult> CheckAsync(long resourceId, CancellationToken cancellationToken)
    {
        if (resourceId <= 0)
        {
            return DeleteGuardResult.Deny("INVALID_RESOURCE_ID", "四层转发 ID 无效");
        }

        var exists = await _db.Queryable<StreamEntity>()
            .Where(x => x.Id == resourceId)
            .AnyAsync();

        return exists
            ? DeleteGuardResult.Allow()
            : DeleteGuardResult.Deny("STREAM_NOT_FOUND", "四层转发不存在，无法删除。");
    }
}
