using Cnn.Common.Contracts;
using Cnn.Common.Localization;
using Cnn.Domain.Entities;
using SqlSugar;

namespace Cnn.Api.Services.Common;

public sealed class MessageService : IMessageService
{
    private static readonly string[] DefaultMessageTypes =
    {
        "package-expire",
        "traffic-exceed",
        "connection-exceed",
        "bandwidth-exceed",
        "cc-switch",
        "cert-expire",
        "refresh_url",
        "refresh_dir",
        "preheat"
    };

    private const int DefaultPageSize = 20;
    private const int MaxPageSize = 200;

    private readonly ISqlSugarClient _db;
    private readonly IMessageLocalizer _localizer;

    public MessageService(ISqlSugarClient db, IMessageLocalizer localizer)
    {
        _db = db;
        _localizer = localizer;
    }

    public async Task<ServiceResult<MessageListResult>> ListAdminAsync(
        MessageListQuery query,
        string language,
        CancellationToken cancellationToken)
    {
        query ??= new MessageListQuery();
        var (page, pageSize) = ResolvePaging(query);
        var msgType = query.Type?.Trim();
        var keyword = query.Keyword?.Trim();

        var q = _db.Queryable<Message>();
        if (!string.IsNullOrWhiteSpace(msgType))
        {
            q = q.Where(m => m.Type == msgType);
        }

        if (!string.IsNullOrWhiteSpace(keyword))
        {
            if (long.TryParse(keyword, out var siteId))
            {
                q = q.Where(m => SqlFunc.Contains(m.Title, keyword!) || SqlFunc.Contains(m.Content, keyword!) || m.SiteId == siteId);
            }
            else
            {
                q = q.Where(m => SqlFunc.Contains(m.Title, keyword!) || SqlFunc.Contains(m.Content, keyword!));
            }
        }

        var total = await q.CountAsync();
        var list = await q.OrderBy(m => m.Id, OrderByType.Desc)
            .Skip((page - 1) * pageSize)
            .Take(pageSize)
            .ToListAsync();

        var items = list.Select(item => BuildMessageItem(item, language, false)).ToList();
        return ServiceResult<MessageListResult>.Ok(new MessageListResult(items, total));
    }

    public async Task<ServiceResult<MessageListResult>> ListUserAsync(
        MessageListQuery query,
        long? userId,
        string language,
        CancellationToken cancellationToken)
    {
        var uid = ResolveUserId(userId);
        if (uid == 0)
        {
            return ServiceResult<MessageListResult>.Fail(ErrorCodes.InvalidParam, "user_id_required");
        }

        query ??= new MessageListQuery();
        var (page, pageSize) = ResolvePaging(query);
        var msgType = query.Type?.Trim();
        var keyword = query.Keyword?.Trim();

        var q = _db.Queryable<Message>().Where(m => m.Receive == uid);
        if (!string.IsNullOrWhiteSpace(msgType))
        {
            q = q.Where(m => m.Type == msgType);
        }

        if (!string.IsNullOrWhiteSpace(keyword))
        {
            if (long.TryParse(keyword, out var siteId))
            {
                q = q.Where(m => SqlFunc.Contains(m.Title, keyword!) || SqlFunc.Contains(m.Content, keyword!) || m.SiteId == siteId);
            }
            else
            {
                q = q.Where(m => SqlFunc.Contains(m.Title, keyword!) || SqlFunc.Contains(m.Content, keyword!));
            }
        }

        var total = await q.CountAsync();
        var list = await q.OrderBy(m => m.Id, OrderByType.Desc)
            .Skip((page - 1) * pageSize)
            .Take(pageSize)
            .ToListAsync();

        var readSet = await LoadReadSetAsync(uid, list.Select(m => m.Id).ToList());
        var items = list.Select(item => BuildMessageItem(item, language, readSet.Contains(item.Id))).ToList();
        return ServiceResult<MessageListResult>.Ok(new MessageListResult(items, total));
    }

    public async Task<ServiceResult<MessageUnreadResult>> GetUnreadAsync(
        long? userId,
        string language,
        CancellationToken cancellationToken)
    {
        var uid = ResolveUserId(userId);
        if (uid == 0)
        {
            return ServiceResult<MessageUnreadResult>.Fail(ErrorCodes.InvalidParam, "user_id_required");
        }

        var baseQuery = BuildUnreadQuery(uid);
        var total = await baseQuery.CountAsync();
        MessageItemDto? latest = null;

        if (total > 0)
        {
            var latestMsg = await baseQuery
                .OrderBy((m, r) => m.Id, OrderByType.Desc)
                .Select((m, r) => new Message
                {
                    Id = m.Id,
                    Type = m.Type,
                    Title = m.Title,
                    Content = m.Content,
                    PhoneContent = m.PhoneContent,
                    SiteId = m.SiteId,
                    CreateAt = m.CreateAt
                })
                .FirstAsync();

            if (latestMsg != null && latestMsg.Id > 0)
            {
                latest = BuildMessageItem(latestMsg, language, false);
            }
        }

        return ServiceResult<MessageUnreadResult>.Ok(new MessageUnreadResult(total, latest));
    }

    public async Task<ServiceResult<bool>> MarkReadAsync(long? userId, long messageId, CancellationToken cancellationToken)
    {
        var uid = ResolveUserId(userId);
        if (uid == 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam, "user_id_required");
        }

        if (messageId <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam);
        }

        var message = await _db.Queryable<Message>()
            .Where(m => m.Id == messageId)
            .Select(m => new { m.Id, m.Receive })
            .FirstAsync();

        if (message == null || message.Receive != uid)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.PermissionDenied);
        }

        var exists = await _db.Queryable<MessageRead>()
            .Where(r => r.Uid == uid && r.MsgId == messageId)
            .AnyAsync();

        if (exists)
        {
            return ServiceResult<bool>.Ok(true);
        }

        var record = new MessageRead
        {
            Uid = uid,
            MsgId = messageId,
            CreateAt = DateTime.Now
        };
        await _db.Insertable(record).ExecuteCommandAsync();
        return ServiceResult<bool>.Ok(true);
    }

    public async Task<ServiceResult<MessageSubListResult>> ListSubscriptionsAsync(
        long? userId,
        string language,
        CancellationToken cancellationToken)
    {
        var uid = ResolveUserId(userId);
        if (uid == 0)
        {
            return ServiceResult<MessageSubListResult>.Fail(ErrorCodes.InvalidParam, "user_id_required");
        }

        var subs = await _db.Queryable<MessageSub>().Where(s => s.Uid == uid).ToListAsync();
        var list = new List<MessageSubItemDto>();

        if (subs.Count == 0)
        {
            foreach (var t in DefaultMessageTypes)
            {
                list.Add(new MessageSubItemDto
                {
                    MsgType = t,
                    Name = ResolveTypeLabel(t, language),
                    Phone = true,
                    Email = true
                });
            }
        }
        else
        {
            foreach (var item in subs)
            {
                var msgType = item.MsgType ?? string.Empty;
                list.Add(new MessageSubItemDto
                {
                    MsgType = msgType,
                    Name = ResolveTypeLabel(msgType, language),
                    Phone = item.Phone ?? false,
                    Email = item.Email ?? false
                });
            }
        }

        return ServiceResult<MessageSubListResult>.Ok(new MessageSubListResult(list));
    }

    public async Task<ServiceResult<bool>> UpdateSubscriptionsAsync(
        long? userId,
        MessageSubUpdateRequest request,
        CancellationToken cancellationToken)
    {
        var uid = ResolveUserId(userId);
        if (uid == 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam, "user_id_required");
        }

        if (request == null)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam);
        }

        var items = request.List ?? new List<MessageSubUpdateItem>();
        var tran = await _db.Ado.UseTranAsync(async () =>
        {
            await _db.Deleteable<MessageSub>().Where(s => s.Uid == uid).ExecuteCommandAsync();

            foreach (var item in items)
            {
                var msgType = item.MsgType?.Trim();
                if (string.IsNullOrWhiteSpace(msgType))
                {
                    continue;
                }

                var record = new MessageSub
                {
                    Uid = uid,
                    MsgType = msgType,
                    Phone = item.Phone ?? false,
                    Email = item.Email ?? false
                };
                await _db.Insertable(record).ExecuteCommandAsync();
            }
        });

        if (!tran.IsSuccess)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.DbError, "db_save_error");
        }

        return ServiceResult<bool>.Ok(true);
    }

    private static (int Page, int PageSize) ResolvePaging(MessageListQuery query)
    {
        var page = query.Page.GetValueOrDefault() < 1 ? 1 : query.Page!.Value;
        var pageSize = query.PageSize.GetValueOrDefault() < 1 ? DefaultPageSize : query.PageSize!.Value;
        if (pageSize > MaxPageSize)
        {
            pageSize = MaxPageSize;
        }

        return (page, pageSize);
    }

    private static int ResolveUserId(long? userId)
    {
        if (!userId.HasValue || userId.Value <= 0)
        {
            return 0;
        }

        if (userId.Value > int.MaxValue)
        {
            return 0;
        }

        return (int)userId.Value;
    }

    private async Task<HashSet<long>> LoadReadSetAsync(int uid, IReadOnlyList<long> messageIds)
    {
        var set = new HashSet<long>();
        if (messageIds.Count == 0)
        {
            return set;
        }

        var rows = await _db.Queryable<MessageRead>()
            .Where(r => r.Uid == uid && messageIds.Contains(r.MsgId ?? 0))
            .Select(r => r.MsgId)
            .ToListAsync();

        foreach (var id in rows)
        {
            if (id.HasValue && id.Value > 0)
            {
                set.Add(id.Value);
            }
        }

        return set;
    }

    private ISugarQueryable<Message, MessageRead> BuildUnreadQuery(int uid)
    {
        return _db.Queryable<Message>()
            .LeftJoin<MessageRead>((m, r) => m.Id == r.MsgId && r.Uid == uid)
            .Where((m, r) => m.Receive == uid && r.MsgId == null);
    }

    private MessageItemDto BuildMessageItem(Message message, string language, bool isRead)
    {
        var type = message.Type ?? string.Empty;
        var typeLabel = ResolveTypeLabel(type, language);
        return new MessageItemDto
        {
            Id = message.Id,
            Type = type,
            TypeLabel = typeLabel,
            Title = NormalizeTitle(message.Title, type, typeLabel),
            Content = message.Content,
            Phone = message.PhoneContent,
            SiteId = message.SiteId,
            CreatedAt = message.CreateAt?.ToString("yyyy-MM-dd HH:mm:ss"),
            IsRead = isRead
        };
    }

    private string ResolveTypeLabel(string type, string language)
    {
        var key = ResolveTypeLabelKey(type);
        return _localizer.Translate(key, language);
    }

    private static string ResolveTypeLabelKey(string type)
    {
        return type switch
        {
            "package-expire" => "message.package_expire",
            "traffic-exceed" => "message.traffic_exceed",
            "connection-exceed" => "message.conn_exceed",
            "bandwidth-exceed" => "message.bandwidth_exceed",
            "cc-switch" => "message.rule_switch",
            "cert-expire" => "message.cert_expire",
            "refresh_url" => "message.refresh_url",
            "refresh_dir" => "message.refresh_dir",
            "preheat" => "message.preheat",
            _ => "message.other"
        };
    }

    private static string NormalizeTitle(string? title, string msgType, string typeLabel)
    {
        var trimmed = title?.Trim() ?? string.Empty;
        if (string.IsNullOrWhiteSpace(trimmed))
        {
            return typeLabel;
        }

        foreach (var ch in trimmed)
        {
            if (ch > 127)
            {
                return trimmed;
            }
        }

        return typeLabel;
    }
}
