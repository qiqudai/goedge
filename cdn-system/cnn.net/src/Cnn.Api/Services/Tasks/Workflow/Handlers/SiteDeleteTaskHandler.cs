using System.Text.Json;
using System.Text.Json.Serialization;
using Cnn.Api.Services.Common;
using Cnn.Domain.Entities;
using SqlSugar;
using Task = System.Threading.Tasks.Task;

namespace Cnn.Api.Services.Tasks.Workflow.Handlers;

public sealed class SiteDeleteTaskHandler : ITaskHandler
{
    private readonly ISqlSugarClient _db;
    private readonly IDnsSyncService _dnsSyncService;
    private readonly IConfigVersionService _configVersionService;

    public SiteDeleteTaskHandler(
        ISqlSugarClient db,
        IDnsSyncService dnsSyncService,
        IConfigVersionService configVersionService)
    {
        _db = db;
        _dnsSyncService = dnsSyncService;
        _configVersionService = configVersionService;
    }

    public string TaskType => AsyncTaskTypes.SiteDelete;

    public async Task HandleAsync(long taskId, string payloadJson, CancellationToken cancellationToken)
    {
        var payload = ParsePayload(payloadJson);
        if (payload.ResourceId <= 0)
        {
            throw new InvalidOperationException("site delete payload is missing resource_id");
        }

        var siteId = (int)payload.ResourceId;
        var site = await _db.Queryable<Site>()
            .Where(x => x.Id == siteId)
            .FirstAsync();
        if (site == null)
        {
            return;
        }

        if (site.Enable == true)
        {
            throw new InvalidOperationException("site must be disabled before delete");
        }

        await _dnsSyncService.SyncUserDnsRecordsAsync(site, null);

        await _db.Ado.UseTranAsync(async () =>
        {
            await _db.Deleteable<MergeSiteGroup>()
                .Where(x => x.SiteId == siteId)
                .ExecuteCommandAsync();

            await _db.Deleteable<Config>()
                .Where(x => x.ScopeName == "site" && x.ScopeId == siteId)
                .ExecuteCommandAsync();

            await _db.Deleteable<SiteConfCache>()
                .Where(x => x.SiteId == siteId)
                .ExecuteCommandAsync();

            await _db.Deleteable<Site>()
                .Where(x => x.Id == siteId)
                .ExecuteCommandAsync();
        });

        await _configVersionService.BumpAsync("site", new[] { payload.ResourceId }, cancellationToken);
    }

    private static DeletePayload ParsePayload(string payloadJson)
    {
        if (string.IsNullOrWhiteSpace(payloadJson))
        {
            return new DeletePayload();
        }

        try
        {
            return JsonSerializer.Deserialize<DeletePayload>(payloadJson, new JsonSerializerOptions(JsonSerializerDefaults.Web))
                   ?? new DeletePayload();
        }
        catch
        {
            return new DeletePayload();
        }
    }

    private sealed class DeletePayload
    {
        [JsonPropertyName("resource_id")]
        public long ResourceId { get; init; }
    }
}
