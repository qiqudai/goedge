using System.Text.Json;
using Cnn.Common.Contracts;
using Cnn.Common.Contracts.Admin;
using Cnn.Domain.Entities;
using SqlSugar;
using TaskEntity = Cnn.Domain.Entities.Task;
using Task = System.Threading.Tasks.Task;

namespace Cnn.Api.Services.Admin;

public sealed partial class CertService
{
    public async Task<ServiceResult<CertBatchProgressResult>> BatchProgressAsync(
        string batchId,
        long? userId,
        bool isAdmin,
        CancellationToken cancellationToken)
    {
        if (string.IsNullOrWhiteSpace(batchId))
        {
            return ServiceResult<CertBatchProgressResult>.Fail(ErrorCodes.MissingParam, "missing_param");
        }

        if (!int.TryParse(batchId, out var pid) || pid <= 0)
        {
            return ServiceResult<CertBatchProgressResult>.Fail(ErrorCodes.InvalidParam, "invalid_param");
        }

        if (!isAdmin && (userId ?? 0) <= 0)
        {
            return ServiceResult<CertBatchProgressResult>.Fail(ErrorCodes.InvalidParam, "user_id_required");
        }

        var tasks = await _db.Queryable<TaskEntity>()
            .Where(t => t.Type == "issue_cert" && t.Pid == pid)
            .ToListAsync();

        if (tasks.Count == 0)
        {
            return ServiceResult<CertBatchProgressResult>.Ok(new CertBatchProgressResult());
        }

        var parsed = tasks.Select(t => new ParsedCertTask(t, ExtractCertIdFromTask(t.Data))).ToList();
        if (!isAdmin)
        {
            var certIds = parsed.Where(x => x.CertId > 0).Select(x => x.CertId).Distinct().ToList();
            var owners = certIds.Count == 0
                ? new Dictionary<long, int>()
                : (await _db.Queryable<Cert>()
                    .Where(c => certIds.Contains(c.Id))
                    .Select(c => new { Id = (long)c.Id, Uid = c.Uid ?? 0 })
                    .ToListAsync())
                .ToDictionary(x => x.Id, x => x.Uid);

            var uid = (int)(userId ?? 0);
            parsed = parsed
                .Where(x => x.CertId > 0 && owners.TryGetValue(x.CertId, out var ownerId) && ownerId == uid)
                .ToList();
        }

        if (parsed.Count == 0)
        {
            return ServiceResult<CertBatchProgressResult>.Ok(new CertBatchProgressResult());
        }

        var failCertIds = parsed.Where(x => string.Equals(x.Task.State, "fail", StringComparison.OrdinalIgnoreCase) && x.CertId > 0)
            .Select(x => x.CertId)
            .Distinct()
            .ToList();

        var domainMap = failCertIds.Count == 0
            ? new Dictionary<long, string?>()
            : (await _db.Queryable<Cert>()
                .Where(c => failCertIds.Contains(c.Id))
                .Select(c => new { Id = (long)c.Id, c.Domain })
                .ToListAsync())
            .ToDictionary(x => x.Id, x => x.Domain);

        var success = 0;
        var fail = 0;
        var running = 0;
        var pending = 0;
        var failItems = new List<CertBatchFailItem>();

        foreach (var item in parsed)
        {
            var state = item.Task.State?.Trim().ToLowerInvariant();
            switch (state)
            {
                case "success":
                    success++;
                    break;
                case "fail":
                    fail++;
                    domainMap.TryGetValue(item.CertId, out var domain);
                    failItems.Add(new CertBatchFailItem
                    {
                        Domain = domain ?? string.Empty,
                        Reason = item.Task.Ret
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

        var total = parsed.Count;
        var done = success + fail;
        var percent = total <= 0 ? 0 : (int)Math.Round(done * 100d / total, MidpointRounding.AwayFromZero);
        if (percent > 100)
        {
            percent = 100;
        }

        return ServiceResult<CertBatchProgressResult>.Ok(new CertBatchProgressResult
        {
            Total = total,
            Success = success,
            Fail = fail,
            Running = running,
            Pending = pending,
            Done = done,
            Percent = percent,
            FailItems = failItems
        });
    }

    private static long ExtractCertIdFromTask(string? data)
    {
        if (string.IsNullOrWhiteSpace(data))
        {
            return 0;
        }

        try
        {
            using var doc = JsonDocument.Parse(data);
            if (!doc.RootElement.TryGetProperty("items", out var items) || items.ValueKind != JsonValueKind.Array)
            {
                return 0;
            }

            foreach (var item in items.EnumerateArray())
            {
                if (!item.TryGetProperty("cert_id", out var certId))
                {
                    continue;
                }

                if (certId.ValueKind == JsonValueKind.Number && certId.TryGetInt64(out var number))
                {
                    return number;
                }

                if (certId.ValueKind == JsonValueKind.String && long.TryParse(certId.GetString(), out var parsed))
                {
                    return parsed;
                }
            }
        }
        catch
        {
            return 0;
        }

        return 0;
    }

    private sealed record ParsedCertTask(TaskEntity Task, long CertId);
}
