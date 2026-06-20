using Cnn.Common.Contracts.Admin;

namespace Cnn.Api.Services.Admin;

/// <summary>
/// 管理站点 settings（site_settings 表）和 site_type meta（site_meta 表）的读写。
/// 不负责业务校验、Normalizer、模板默认值等逻辑 —— 只封装存储协议。
/// </summary>
public interface ISiteSettingsStore
{
    // ── Settings ──────────────────────────────────────────────────────────

    /// <summary>加载单个站点的 settings 字典，若不存在返回空字典。</summary>
    Task<Dictionary<string, object?>> LoadSettingsAsync(long siteId, CancellationToken cancellationToken = default);

    /// <summary>批量加载多个站点的 settings 字典，key = siteId。</summary>
    Task<Dictionary<long, Dictionary<string, object?>>> LoadSettingsMapAsync(
        IReadOnlyList<long> siteIds,
        CancellationToken cancellationToken = default);

    /// <summary>保存（upsert）站点 settings 字典。</summary>
    Task SaveSettingsAsync(long siteId, Dictionary<string, object?> settings, CancellationToken cancellationToken = default);

    // ── Site type meta ────────────────────────────────────────────────────

    /// <summary>批量加载多个站点的 site_type，key = siteId。</summary>
    Task<Dictionary<long, string>> LoadSiteTypeMapAsync(
        IReadOnlyList<long> siteIds,
        CancellationToken cancellationToken = default);

    /// <summary>保存（upsert）站点 site_type。</summary>
    Task SaveSiteTypeAsync(long siteId, string siteType, CancellationToken cancellationToken = default);
}
