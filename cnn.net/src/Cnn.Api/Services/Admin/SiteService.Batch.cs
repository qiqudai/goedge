using System.Text.Json;
using Cnn.Api.Services.Common;
using Cnn.Common.Contracts;
using Cnn.Common.Contracts.Admin;
using Cnn.Domain.Entities;
using SqlSugar;
using TaskEntity = Cnn.Domain.Entities.Task;

namespace Cnn.Api.Services.Admin;

public sealed partial class SiteService
{
    public async Task<ServiceResult<SiteBatchCreateResult>> BatchCreateAsync(
        SiteBatchCreateRequest request,
        long? userId,
        bool isAdmin,
        CancellationToken cancellationToken)
    {
        request ??= new SiteBatchCreateRequest();

        var targetUserId = ResolveUserId(request.UserId, userId, isAdmin);
        if (targetUserId <= 0)
        {
            return ServiceResult<SiteBatchCreateResult>.Fail(ErrorCodes.InvalidParam, "user_id_required");
        }

        var userPackageId = request.UserPackageId;
        if (userPackageId <= 0)
        {
            userPackageId = await ResolveDefaultUserPackageIdAsync(targetUserId);
        }
        if (userPackageId <= 0)
        {
            return ServiceResult<SiteBatchCreateResult>.Fail(ErrorCodes.NotFound, "package_not_found");
        }

        if (!isAdmin && !await EnsureUserPackageOwnershipAsync(targetUserId, userPackageId))
        {
            return ServiceResult<SiteBatchCreateResult>.Fail(ErrorCodes.PermissionDenied);
        }

        if (string.IsNullOrWhiteSpace(request.Data))
        {
            return ServiceResult<SiteBatchCreateResult>.Fail(ErrorCodes.MissingParam, "missing_param");
        }

        var nodeGroupId = await ResolveNodeGroupFromPackageAsync(userPackageId, request.NodeGroupId);
        var batchId = Guid.NewGuid().ToString("N");

        var lines = SplitLines(request.Data);
        var items = new List<BatchSiteItem>(lines.Count);
        var allDomains = new List<string>();

        foreach (var line in lines)
        {
            var parsed = ParseBatchLine(line);
            if (!parsed.Ok)
            {
                if (request.IgnoreError)
                {
                    continue;
                }
                return ServiceResult<SiteBatchCreateResult>.Fail(ErrorCodes.InvalidParam, "invalid_param");
            }

            items.Add(parsed.Value!);
            allDomains.AddRange(parsed.Value!.Domains);
        }

        var limitOk = await CheckDomainLimitAsync(targetUserId, userPackageId, allDomains, null);
        if (!limitOk.Success)
        {
            return ServiceResult<SiteBatchCreateResult>.Fail(limitOk.ErrorCode, limitOk.MessageKey);
        }

        var now = DateTime.Now;
        var tasks = new List<SiteBatchTaskItem>();
        var created = 0;

        foreach (var item in items)
        {
            foreach (var domain in item.Domains)
            {
                var payload = new SiteCreatePayload
                {
                    UserId = targetUserId,
                    UserPackageId = userPackageId,
                    DnsProviderId = request.DnsProviderId,
                    NodeGroupId = nodeGroupId,
                    GroupId = request.GroupId,
                    Domain = domain,
                    Backends = item.Backends,
                    BatchId = batchId
                };

                var task = new TaskEntity
                {
                    Type = "site_create",
                    Name = "Create Site " + domain,
                    Data = JsonSerializer.Serialize(payload, JsonOptions),
                    Res = JsonSerializer.Serialize(new { user_id = targetUserId, batch_id = batchId }, JsonOptions),
                    State = "waiting",
                    Enable = true,
                    CreateAt = now
                };

                var id = await _db.Insertable(task).ExecuteReturnIdentityAsync();
                if (id <= 0)
                {
                    continue;
                }

                task.Id = id;
                created++;
                tasks.Add(new SiteBatchTaskItem
                {
                    Id = task.Id,
                    Type = task.Type,
                    Name = task.Name,
                    State = task.State,
                    Ret = task.Ret
                });
            }
        }

        return ServiceResult<SiteBatchCreateResult>.Ok(new SiteBatchCreateResult(batchId, created, tasks));
    }

    public async Task<ServiceResult<SiteBatchProgressResult>> BatchProgressAsync(
        string batchId,
        long? userId,
        bool isAdmin,
        CancellationToken cancellationToken)
    {
        if (string.IsNullOrWhiteSpace(batchId))
        {
            return ServiceResult<SiteBatchProgressResult>.Fail(ErrorCodes.MissingParam, "missing_param");
        }

        var query = _db.Queryable<TaskEntity>()
            .Where(t => t.Type == "site_create")
            .Where(t => SqlFunc.Contains(t.Data, batchId) || SqlFunc.Contains(t.Res, batchId));

        if (!isAdmin && userId is > 0)
        {
            var uid = userId.Value;
            query = query.Where(t => SqlFunc.Contains(t.Res, "\"user_id\":" + uid));
        }

        var tasks = await query.ToListAsync();
        var total = tasks.Count;
        var success = 0;
        var fail = 0;
        var running = 0;
        var pending = 0;
        var failItems = new List<SiteBatchFailItem>();

        foreach (var task in tasks)
        {
            switch (task.State)
            {
                case "success":
                    success++;
                    break;
                case "fail":
                case "failed":
                case "failure":
                    fail++;
                    var domain = ExtractDomainFromTask(task.Data);
                    failItems.Add(new SiteBatchFailItem
                    {
                        Domain = domain,
                        Reason = task.Ret
                    });
                    break;
                case "running":
                case "retrying":
                    running++;
                    break;
                default:
                    pending++;
                    break;
            }
        }

        var result = new SiteBatchProgressResult
        {
            Total = total,
            Success = success,
            Fail = fail,
            Running = running,
            Pending = pending,
            Done = success + fail,
            Percent = total == 0 ? 0 : (int)Math.Round((success + fail) * 100d / total),
            FailItems = failItems
        };

        return ServiceResult<SiteBatchProgressResult>.Ok(result);
    }

    private static string ExtractDomainFromTask(string? data)
    {
        if (string.IsNullOrWhiteSpace(data))
        {
            return string.Empty;
        }
        try
        {
            var payload = JsonSerializer.Deserialize<SiteCreatePayload>(data, JsonOptions);
            return payload?.Domain ?? string.Empty;
        }
        catch
        {
            return string.Empty;
        }
    }

    private sealed class BatchSiteItem
    {
        public List<string> Domains { get; set; } = new();
        public List<string> Backends { get; set; } = new();
    }

    private sealed class BatchParseResult
    {
        public bool Ok { get; init; }
        public BatchSiteItem? Value { get; init; }

        public static BatchParseResult Fail() => new() { Ok = false };
        public static BatchParseResult Success(BatchSiteItem item) => new() { Ok = true, Value = item };
    }

    private static BatchParseResult ParseBatchLine(string line)
    {
        var item = new BatchSiteItem();
        var segments = line.Split('|', StringSplitOptions.RemoveEmptyEntries);
        foreach (var seg in segments)
        {
            var trimmed = seg.Trim();
            if (string.IsNullOrWhiteSpace(trimmed))
            {
                continue;
            }
            var kv = trimmed.Split('=', 2);
            if (kv.Length != 2)
            {
                return BatchParseResult.Fail();
            }
            var key = kv[0].Trim();
            var val = kv[1].Trim();
            switch (key)
            {
                case "domain":
                    item.Domains = SplitByComma(val);
                    break;
                case "ip":
                    item.Backends = SplitByComma(val);
                    break;
            }
        }
        if (item.Domains.Count == 0)
        {
            return BatchParseResult.Fail();
        }
        if (item.Domains.Any(domain => !DomainHelper.IsValidDomain(domain)))
        {
            return BatchParseResult.Fail();
        }
        return BatchParseResult.Success(item);
    }

    private sealed class SiteCreatePayload
    {
        public long UserId { get; init; }
        public long UserPackageId { get; init; }
        public long DnsProviderId { get; init; }
        public long NodeGroupId { get; init; }
        public long GroupId { get; init; }
        public string Domain { get; init; } = string.Empty;
        public List<string> Backends { get; init; } = new();
        public string BatchId { get; init; } = string.Empty;
    }
}
