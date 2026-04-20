using Cnn.Api.Services.Common;

namespace Cnn.Api.Services.Stats;

/// <summary>
/// 负责将访问范围解析为具体的统计过滤规则（HostFilter）。
/// </summary>
public interface IHostFilterResolver
{
    Task<HostFilter> ResolveAsync(AccessScope scope, CancellationToken cancellationToken);
}
