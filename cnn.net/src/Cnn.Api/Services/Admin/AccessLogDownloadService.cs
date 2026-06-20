using System.Text.Json;
using Cnn.Common.Contracts;
using Cnn.Common.Contracts.Admin;
using Cnn.Domain.Entities;
using SqlSugar;

namespace Cnn.Api.Services.Admin;

public interface IAccessLogDownloadService
{
    Task<ServiceResult<AccessLogDownloadApplyResult>> ApplyAsync(
        AccessLogDownloadApplyRequest request,
        long requesterUserId,
        bool isAdmin,
        string? requesterIp,
        CancellationToken cancellationToken);

    Task<ServiceResult<bool>> CompleteAsync(
        long id,
        AccessLogDownloadCompleteRequest request,
        long requesterUserId,
        bool isAdmin,
        string? requesterIp,
        CancellationToken cancellationToken);

    Task<ServiceResult<AccessLogDownloadListResult>> ListAsync(
        AccessLogDownloadQuery query,
        long requesterUserId,
        bool isAdmin,
        CancellationToken cancellationToken);
}

public sealed class AccessLogDownloadService : IAccessLogDownloadService
{
    private const int DefaultPageSize = 20;
    private const int MaxPageSize = 200;

    private static readonly JsonSerializerOptions JsonOptions = new(JsonSerializerDefaults.Web);

    private readonly ISqlSugarClient _db;
    private readonly IGlobalConfigService _globalConfigService;
    private readonly IOperationLogService _operationLogService;

    public AccessLogDownloadService(
        ISqlSugarClient db,
        IGlobalConfigService globalConfigService,
        IOperationLogService operationLogService)
    {
        _db = db;
        _globalConfigService = globalConfigService;
        _operationLogService = operationLogService;
    }

    public async Task<ServiceResult<AccessLogDownloadApplyResult>> ApplyAsync(
        AccessLogDownloadApplyRequest request,
        long requesterUserId,
        bool isAdmin,
        string? requesterIp,
        CancellationToken cancellationToken)
    {
        request ??= new AccessLogDownloadApplyRequest();
        var limitCheck = await CheckDailyLimitAsync(requesterUserId, isAdmin, cancellationToken);
        if (!limitCheck.Success)
        {
            return ServiceResult<AccessLogDownloadApplyResult>.Fail(limitCheck.ErrorCode, limitCheck.MessageKey);
        }

        var now = DateTime.Now;
        var fileName = NormalizeFileName(request.FileName, now);

        var payload = JsonSerializer.Serialize(new
        {
            query = request.Query ?? new AccessLogQuery(),
            start_time = request.StartTime,
            end_time = request.EndTime
        }, JsonOptions);

        var entity = new AccessLogDownload
        {
            UserId = requesterUserId > 0 ? requesterUserId : null,
            IsAdmin = isAdmin,
            Scope = isAdmin ? "admin" : "user",
            State = "pending",
            QueryJson = payload,
            FileName = fileName,
            Rows = 0,
            CreateAt = now
        };

        var inserted = await _db.Insertable(entity).ExecuteReturnEntityAsync();
        if (inserted == null || inserted.Id <= 0)
        {
            return ServiceResult<AccessLogDownloadApplyResult>.Fail(ErrorCodes.DbError, "db_save_error");
        }

        await _operationLogService.WriteAsync(new OperationLogWriteRequest
        {
            UserId = NormalizeUserId(requesterUserId),
            Type = isAdmin ? "admin" : "user",
            Action = "logs.access.download.apply",
            Content = JsonSerializer.Serialize(new
            {
                download_id = inserted.Id,
                scope = entity.Scope,
                file_name = fileName,
                start_time = request.StartTime,
                end_time = request.EndTime
            }, JsonOptions),
            Ip = requesterIp,
            Process = "ok"
        }, cancellationToken);

        return ServiceResult<AccessLogDownloadApplyResult>.Ok(new AccessLogDownloadApplyResult(inserted.Id, "pending"));
    }

    public async Task<ServiceResult<bool>> CompleteAsync(
        long id,
        AccessLogDownloadCompleteRequest request,
        long requesterUserId,
        bool isAdmin,
        string? requesterIp,
        CancellationToken cancellationToken)
    {
        if (id <= 0 || request == null)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam);
        }

        var item = await _db.Queryable<AccessLogDownload>().Where(r => r.Id == id).FirstAsync();
        if (item == null)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.NotFound, "not_found");
        }

        if (!isAdmin && item.UserId.GetValueOrDefault() != requesterUserId)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.PermissionDenied);
        }

        var rows = request.Rows.GetValueOrDefault();
        if (rows < 0)
        {
            rows = 0;
        }

        var now = DateTime.Now;
        var state = request.Success ? "done" : "failed";
        var updateRows = await _db.Ado.ExecuteCommandAsync(
            "UPDATE access_log_download SET state=@state, rows=@rows, error=@error, finish_at=@finish_at WHERE id=@id",
            new[]
            {
                new SugarParameter("@state", state),
                new SugarParameter("@rows", rows),
                new SugarParameter("@error", request.Success ? string.Empty : (request.Error ?? string.Empty)),
                new SugarParameter("@finish_at", now),
                new SugarParameter("@id", id)
            });

        if (updateRows <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.DbError, "db_save_error");
        }

        await _operationLogService.WriteAsync(new OperationLogWriteRequest
        {
            UserId = NormalizeUserId(requesterUserId),
            Type = isAdmin ? "admin" : "user",
            Action = "logs.access.download.complete",
            Content = JsonSerializer.Serialize(new
            {
                download_id = id,
                state,
                rows,
                error = request.Success ? null : (request.Error ?? string.Empty)
            }, JsonOptions),
            Ip = requesterIp,
            Process = request.Success ? "ok" : "failed"
        }, cancellationToken);

        return ServiceResult<bool>.Ok(true);
    }

    public async Task<ServiceResult<AccessLogDownloadListResult>> ListAsync(
        AccessLogDownloadQuery query,
        long requesterUserId,
        bool isAdmin,
        CancellationToken cancellationToken)
    {
        query ??= new AccessLogDownloadQuery();
        var (page, pageSize) = ResolvePaging(query);
        var keyword = query.Keyword?.Trim();
        var state = query.State?.Trim();

        var q = _db.Queryable<AccessLogDownload>();
        if (!isAdmin)
        {
            q = q.Where(r => r.UserId == requesterUserId);
        }

        if (!string.IsNullOrWhiteSpace(keyword))
        {
            var lower = keyword.ToLowerInvariant();
            if (long.TryParse(lower, out var id) && id > 0)
            {
                q = q.Where(r => r.Id == id || SqlFunc.ToLower(r.FileName)!.Contains(lower));
            }
            else
            {
                q = q.Where(r => SqlFunc.ToLower(r.FileName)!.Contains(lower));
            }
        }

        if (!string.IsNullOrWhiteSpace(state))
        {
            var normalized = state.ToLowerInvariant();
            q = q.Where(r => SqlFunc.ToLower(r.State) == normalized);
        }

        var total = await q.CountAsync();
        var rows = await q.OrderBy(r => r.Id, OrderByType.Desc)
            .ToPageListAsync(page, pageSize);

        var list = rows.Select(item => new AccessLogDownloadItem
        {
            Id = item.Id,
            FileName = item.FileName,
            State = item.State,
            Rows = item.Rows.GetValueOrDefault(),
            Error = item.Error,
            RequesterUserId = item.UserId.GetValueOrDefault(),
            Scope = item.Scope,
            CreatedAt = FormatTime(item.CreateAt),
            FinishedAt = FormatTime(item.FinishAt)
        }).ToList();

        return ServiceResult<AccessLogDownloadListResult>.Ok(new AccessLogDownloadListResult(list, total));
    }

    private static (int Page, int PageSize) ResolvePaging(AccessLogDownloadQuery query)
    {
        var page = query.Page.GetValueOrDefault() < 1 ? 1 : query.Page!.Value;
        var pageSize = query.PageSize.GetValueOrDefault() < 1 ? DefaultPageSize : query.PageSize!.Value;
        if (pageSize > MaxPageSize)
        {
            pageSize = MaxPageSize;
        }

        return (page, pageSize);
    }

    private static string NormalizeFileName(string? value, DateTime now)
    {
        var raw = value?.Trim();
        if (string.IsNullOrWhiteSpace(raw))
        {
            return $"access-logs-{now:yyyyMMddHHmmss}.csv";
        }

        if (!raw.EndsWith(".csv", StringComparison.OrdinalIgnoreCase))
        {
            raw += ".csv";
        }

        return raw;
    }

    private static string? FormatTime(DateTime? value)
    {
        if (!value.HasValue)
        {
            return null;
        }

        return value.Value.ToString("yyyy-MM-dd HH:mm:ss");
    }

    private async Task<ServiceResult<bool>> CheckDailyLimitAsync(long requesterUserId, bool isAdmin, CancellationToken cancellationToken)
    {
        if (isAdmin || requesterUserId <= 0)
        {
            return ServiceResult<bool>.Ok(true);
        }

        var configResult = await _globalConfigService.GetAsync(cancellationToken);
        if (!configResult.Success || configResult.Data == null)
        {
            return ServiceResult<bool>.Fail(
                configResult.ErrorCode == ErrorCodes.Success ? ErrorCodes.ConfigError : configResult.ErrorCode,
                string.IsNullOrWhiteSpace(configResult.MessageKey) ? "config_error" : configResult.MessageKey);
        }

        var limit = configResult.Data.Resources?.Website?.DailyLogDownloadLimit ?? 0;
        if (limit <= 0)
        {
            return ServiceResult<bool>.Ok(true);
        }

        var start = DateTime.Today;
        var end = start.AddDays(1);
        var used = await _db.Queryable<AccessLogDownload>()
            .Where(r => r.UserId == requesterUserId && r.CreateAt >= start && r.CreateAt < end)
            .CountAsync();

        if (used >= limit)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.QuotaExceeded, "quota_exceeded");
        }

        return ServiceResult<bool>.Ok(true);
    }

    private static int? NormalizeUserId(long userId)
    {
        if (userId <= 0 || userId > int.MaxValue)
        {
            return null;
        }

        return (int)userId;
    }
}
