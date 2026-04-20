using Cnn.Common.Contracts;
using Cnn.Common.Contracts.Admin;
using Cnn.Domain.Entities;
using SqlSugar;
using TaskEntity = Cnn.Domain.Entities.Task;
using Task = System.Threading.Tasks.Task;

namespace Cnn.Api.Services.Admin;

public sealed partial class CertService
{
    public async Task<ServiceResult<CertListResult>> ListAsync(
        CertListQuery query,
        long? userId,
        bool isAdmin,
        CancellationToken cancellationToken)
    {
        query ??= new CertListQuery();
        var q = _db.Queryable<Cert>();

        if (!isAdmin)
        {
            var uid = userId ?? 0;
            if (uid <= 0)
            {
                return ServiceResult<CertListResult>.Fail(ErrorCodes.InvalidParam, "user_id_required");
            }
            q = q.Where(c => c.Uid == (int)uid);
        }
        else if (query.UserId is > 0)
        {
            q = q.Where(c => c.Uid == (int)query.UserId.Value);
        }

        var keyword = query.Keyword?.Trim();
        if (!string.IsNullOrWhiteSpace(keyword))
        {
            var field = query.SearchField?.Trim().ToLowerInvariant();
            if (field == "name")
            {
                q = q.Where(c => SqlFunc.Contains(c.Name, keyword));
            }
            else if (field == "domain")
            {
                q = q.Where(c => SqlFunc.Contains(c.Domain, keyword));
            }
            else if (field == "type")
            {
                q = q.Where(c => SqlFunc.Contains(c.Type, keyword));
            }
            else
            {
                q = q.Where(c => SqlFunc.Contains(c.Name, keyword) || SqlFunc.Contains(c.Domain, keyword) || SqlFunc.Contains(c.Type, keyword));
            }
        }

        var page = query.Page < 1 ? 1 : query.Page;
        var pageSize = query.PageSize < 1 ? 10 : query.PageSize;

        var total = await q.CountAsync();
        var list = await q.OrderBy(c => c.Id, OrderByType.Desc)
            .Skip((page - 1) * pageSize)
            .Take(pageSize)
            .ToListAsync();

        if (list.Count == 0)
        {
            return ServiceResult<CertListResult>.Ok(new CertListResult(Array.Empty<CertItemDto>(), total));
        }

        var userIds = list.Select(c => c.Uid ?? 0).Where(id => id > 0).Distinct().ToList();
        var userMap = new Dictionary<long, string>();
        if (userIds.Count > 0)
        {
            var users = await _db.Queryable<User>()
                .Where(u => userIds.Contains(u.Id))
                .Select(u => new { u.Id, u.Name })
                .ToListAsync();
            foreach (var user in users)
            {
                userMap[user.Id] = user.Name ?? string.Empty;
            }
        }

        var taskIds = list.Select(c => c.IssueTaskId ?? 0).Where(id => id > 0).Distinct().ToList();
        var taskMap = new Dictionary<long, TaskInfo>();
        if (taskIds.Count > 0)
        {
            var tasks = await _db.Queryable<TaskEntity>()
                .Where(t => taskIds.Contains(t.Id))
                .Select(t => new { t.Id, t.State, t.Ret, t.RetryAt, t.ErrTimes })
                .ToListAsync();
            foreach (var task in tasks)
            {
                taskMap[task.Id] = new TaskInfo(task.State, task.Ret, task.RetryAt, task.ErrTimes);
            }
        }

        var items = new List<CertItemDto>(list.Count);
        foreach (var cert in list)
        {
            var taskInfo = cert.IssueTaskId is > 0 && taskMap.TryGetValue(cert.IssueTaskId.Value, out var info)
                ? info
                : null;

            var state = ResolveCertState(taskInfo?.State, cert.Type);
            if (string.IsNullOrWhiteSpace(state))
            {
                state = cert.State ?? string.Empty;
            }

            var exposeCert = ShouldExposeCertData(cert.Type, state);
            var certPem = exposeCert ? cert.CertPem : string.Empty;
            var keyPem = exposeCert ? DecryptKey(cert.Key) : string.Empty;
            var ret = !string.IsNullOrWhiteSpace(taskInfo?.Ret) ? taskInfo?.Ret : cert.Ret;

            var dto = new CertItemDto
            {
                Id = cert.Id,
                UserId = cert.Uid ?? 0,
                UserName = cert.Uid is > 0 && userMap.TryGetValue(cert.Uid.Value, out var name) ? name : null,
                Name = cert.Name,
                Description = cert.Des,
                Type = cert.Type,
                Domain = cert.Domain,
                DnsApi = cert.Dnsapi ?? 0,
                CertPem = certPem,
                KeyPem = keyPem,
                StartTime = cert.StartTime,
                ExpireTime = cert.ExpireTime,
                AutoRenew = cert.AutoRenew ?? false,
                CreateAt = cert.CreateAt,
                UpdateAt = cert.UpdateAt,
                Enable = cert.Enable ?? false,
                TaskId = cert.TaskId,
                State = state,
                Ret = ret,
                Version = cert.Version,
                IssueTaskRet = taskInfo?.Ret,
                IssueTaskState = taskInfo?.State,
                IssueTaskRetryAt = taskInfo?.RetryAt,
                IssueTaskErrTimes = taskInfo?.ErrTimes
            };
            items.Add(dto);
        }

        return ServiceResult<CertListResult>.Ok(new CertListResult(items, total));
    }
}


