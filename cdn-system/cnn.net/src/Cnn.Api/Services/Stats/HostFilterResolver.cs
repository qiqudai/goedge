using Cnn.Api.Services.Common;

namespace Cnn.Api.Services.Stats;

public sealed class HostFilterResolver : IHostFilterResolver
{
    private readonly ISiteHostIndexService _hostIndexService;

    public HostFilterResolver(ISiteHostIndexService hostIndexService)
    {
        _hostIndexService = hostIndexService;
    }

    public async Task<HostFilter> ResolveAsync(AccessScope scope, CancellationToken cancellationToken)
    {
        // 1. 如果是全局管理员（isAdmin 且没有指定 userId），返回空过滤（即访问所有数据）
        if (scope.IsGlobalAdmin)
        {
            return new HostFilter();
        }

        // 2. 如果指定了 UserId（无论是管理员代理还是普通用户自己），则需要解析其名下的站点 Host 列表
        if (scope.HasUserId)
        {
            var index = await _hostIndexService.LoadAsync(scope.UserId, cancellationToken);
            return index.Filter;
        }

        // 3. 默认返回一个空的 Filter。
        // 注意：StatsService 逻辑中如果 filter.Empty 且 !isAdmin 则会直接返回空统计结果。
        return new HostFilter();
    }
}
