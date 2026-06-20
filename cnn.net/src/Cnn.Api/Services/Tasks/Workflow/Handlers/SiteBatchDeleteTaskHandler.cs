using System.Text.Json;
using System.Text.Json.Serialization;
using Cnn.Api.Services.Common;
using Cnn.Domain.Entities;
using SqlSugar;
using Task = System.Threading.Tasks.Task;

namespace Cnn.Api.Services.Tasks.Workflow.Handlers;

public sealed class SiteBatchDeleteTaskHandler : ITaskHandler
{
    private readonly ISqlSugarClient _db;
    private readonly IDnsSyncService _dnsSyncService;
    private readonly IConfigVersionService _configVersionService;

    public SiteBatchDeleteTaskHandler(
        ISqlSugarClient db,
        IDnsSyncService dnsSyncService,
        IConfigVersionService configVersionService)
    {
        _db = db;
        _dnsSyncService = dnsSyncService;
        _configVersionService = configVersionService;
    }

    public string TaskType => AsyncTaskTypes.SiteBatchDelete;

    public async Task HandleAsync(long taskId, string payloadJson, CancellationToken cancellationToken)
    {
        var payload = ParsePayload(payloadJson);
        var siteIds = NormalizeIds(payload.ResourceIds);
        if (siteIds.Count == 0)
        {
            throw new InvalidOperationException("site batch delete payload is missing resource_ids");
        }

        var sites = await _db.Queryable<Site>()
            .Where(x => siteIds.Contains(x.Id))
            .ToListAsync();

        foreach (var site in sites)
        {
            if (site.Enable == true)
            {
                throw new InvalidOperationException($"site {site.Id} must be disabled before delete");
            }
        }

        foreach (var site in sites)
        {
            await _dnsSyncService.SyncUserDnsRecordsAsync(site, null);
        }

        await _db.Ado.UseTranAsync(async () =>
        {
            await _db.Deleteable<MergeSiteGroup>()
                .Where(x => x.SiteId.HasValue && siteIds.Contains(x.SiteId.Value))
                .ExecuteCommandAsync();

            await _db.Deleteable<Config>()
                .Where(x => x.ScopeName == "site" && x.ScopeId.HasValue && siteIds.Contains(x.ScopeId.Value))
                .ExecuteCommandAsync();

            await _db.Deleteable<SiteConfCache>()
                .Where(x => x.SiteId.HasValue && siteIds.Contains(x.SiteId.Value))
                .ExecuteCommandAsync();

            await _db.Deleteable<Site>()
                .Where(x => siteIds.Contains(x.Id))
                .ExecuteCommandAsync();
        });

        await _configVersionService.BumpAsync("site", siteIds, cancellationToken);
    }

    private static BatchDeletePayload ParsePayload(string payloadJson)
    {
        if (string.IsNullOrWhiteSpace(payloadJson))
        {
            return new BatchDeletePayload();
        }

        try
        {
            return JsonSerializer.Deserialize<BatchDeletePayload>(payloadJson, new JsonSerializerOptions(JsonSerializerDefaults.Web))
                   ?? new BatchDeletePayload();
        }
        catch
        {
            return new BatchDeletePayload();
        }
    }

    private static List<long> NormalizeIds(IReadOnlyList<long>? ids)
    {
        return ids?
            .Where(id => id > 0)
            .Distinct()
            .OrderBy(id => id)
            .ToList()
               ?? new List<long>();
    }

    private sealed class BatchDeletePayload
    {
        [JsonPropertyName("resource_ids")]
        public IReadOnlyList<long>? ResourceIds { get; init; }
    }
}
