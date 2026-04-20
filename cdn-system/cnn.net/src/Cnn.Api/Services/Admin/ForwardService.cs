using System.Text.Json;
using System.Text.Json.Serialization;
using Cnn.Common.Contracts;
using Cnn.Common.Contracts.Admin;
using Cnn.Api.Services.Common;
using Cnn.Api.Services.Tasks.Workflow;
using Cnn.Domain.Entities;
using SqlSugar;
using Stream = Cnn.Domain.Entities.Stream;

namespace Cnn.Api.Services.Admin;

public sealed class ForwardService : IForwardService
{
    private const string ForwardDefaultKey = "forward_default_settings";
    private const string DefaultCnameDomain = "cdn.node.com";

    private static readonly JsonSerializerOptions JsonOptions = new(JsonSerializerDefaults.Web)
    {
        PropertyNameCaseInsensitive = true,
        DefaultIgnoreCondition = JsonIgnoreCondition.WhenWritingNull
    };

    private readonly ISqlSugarClient _db;
    private readonly IConfigVersionService _configVersionService;
    private readonly IForwardCnameSyncService _forwardCnameSyncService;
    private readonly IResourceActionRequestService _resourceActionRequestService;

    public ForwardService(
        ISqlSugarClient db,
        IConfigVersionService configVersionService,
        IForwardCnameSyncService forwardCnameSyncService,
        IResourceActionRequestService resourceActionRequestService)
    {
        _db = db;
        _configVersionService = configVersionService;
        _forwardCnameSyncService = forwardCnameSyncService;
        _resourceActionRequestService = resourceActionRequestService;
    }

    public async Task<ServiceResult<ForwardListResult>> ListAsync(
        ForwardListQuery query,
        long? userId,
        bool isAdmin,
        CancellationToken cancellationToken)
    {
        if (!isAdmin && (!userId.HasValue || userId <= 0))
        {
            return ServiceResult<ForwardListResult>.Fail(ErrorCodes.InvalidParam, "user_id_required");
        }

        var q = _db.Queryable<Stream>();
        if (!isAdmin)
        {
            q = q.Where(s => s.Uid == (int)userId!.Value);
        }
        else if (query.UserId is > 0)
        {
            q = q.Where(s => s.Uid == (int)query.UserId.Value);
        }

        if (query.UserPackageId is > 0)
        {
            q = q.Where(s => s.UserPackage == (int)query.UserPackageId.Value);
        }

        if (query.GroupId is > 0)
        {
            var ids = await FindForwardIdsByGroupIdAsync(query.GroupId.Value);
            if (ids.Count == 0)
            {
                return ServiceResult<ForwardListResult>.Ok(new ForwardListResult(Array.Empty<ForwardListItem>(), 0));
            }

            q = q.Where(s => ids.Contains(s.Id));
        }

        var status = query.Status?.Trim().ToLowerInvariant();
        if (!string.IsNullOrWhiteSpace(status))
        {
            if (status is "enabled" or "enable" or "1" or "true")
            {
                q = q.Where(s => s.Enable == true);
            }
            else if (status is "disabled" or "disable" or "0" or "false")
            {
                q = q.Where(s => s.Enable == false);
            }
        }

        var keyword = query.Keyword?.Trim() ?? string.Empty;
        var searchField = query.SearchField?.Trim().ToLowerInvariant() ?? "all";
        if (!string.IsNullOrWhiteSpace(keyword))
        {
            q = await ApplySearchAsync(q, keyword, searchField);
        }

        var page = query.Page < 1 ? 1 : query.Page;
        var pageSize = query.PageSize < 1 ? 10 : query.PageSize;

        var total = await q.CountAsync();
        var forwards = await q.OrderBy(s => s.Id, OrderByType.Desc)
            .ToPageListAsync(page, pageSize);

        var items = await BuildForwardListItemsAsync(forwards);
        return ServiceResult<ForwardListResult>.Ok(new ForwardListResult(items, total));
    }
    public async Task<ServiceResult<ForwardDetailDto>> CreateAsync(
        ForwardCreateRequest request,
        long? userId,
        bool isAdmin,
        CancellationToken cancellationToken)
    {
        var targetUserId = ResolveUserId(request.UserId, userId, isAdmin);
        if (targetUserId <= 0)
        {
            return ServiceResult<ForwardDetailDto>.Fail(ErrorCodes.InvalidParam, "user_id_required");
        }

        var listenPorts = ResolveListenPorts(request.ListenPorts, request.ListenPortsInput);
        if (listenPorts.Count == 0)
        {
            return ServiceResult<ForwardDetailDto>.Fail(ErrorCodes.MissingParam, "listen_ports_required");
        }

        var origins = ParseOrigins(request.OriginInput);
        if (origins.Count == 0)
        {
            return ServiceResult<ForwardDetailDto>.Fail(ErrorCodes.MissingParam, "origin_required");
        }

        var nodeGroupId = await ResolveNodeGroupFromPackageAsync(request.UserPackageId, request.NodeGroupId);
        if (nodeGroupId == 0)
        {
            nodeGroupId = await ResolveDefaultNodeGroupIdAsync();
        }

        var regionId = await ResolveForwardRegionIdAsync(nodeGroupId);
        var now = DateTime.Now;

        var forward = new Stream
        {
            Uid = (int)targetUserId,
            UserPackage = request.UserPackageId > 0 ? (int?)request.UserPackageId : null,
            RegionId = regionId > 0 ? (int?)regionId : null,
            NodeGroupId = nodeGroupId > 0 ? (int?)nodeGroupId : null,
            Listen = EncodeStringList(listenPorts),
            Backend = EncodeOrigins(origins),
            BackendPort = ExtractBackendPort(origins),
            Enable = true,
            State = "running",
            CreateAt = now,
            UpdateAt = now
        };

        var defaults = await LoadStreamDefaultMapAsync(targetUserId);
        var settings = new Dictionary<string, object?>();
        ApplyForwardDefaults(forward, defaults, settings);

        if (!string.IsNullOrWhiteSpace(request.Remark))
        {
            settings["remark"] = request.Remark!.Trim();
        }

        if (settings.Count > 0)
        {
            forward.Acl = JsonSerializer.Serialize(settings, JsonOptions);
        }

        UserPackage? pkg = null;
        if (forward.UserPackage is > 0)
        {
            pkg = await _db.Queryable<UserPackage>().Where(p => p.Id == forward.UserPackage.Value).FirstAsync();
        }

        if (!await ApplyForwardCnameAsync(forward, pkg))
        {
            return ServiceResult<ForwardDetailDto>.Fail(ErrorCodes.InternalError, "forward_cname_generate_failed");
        }

        var groupIds = ResolveGroupIds(request.GroupIds, request.GroupId);

        await _db.Ado.UseTranAsync(async () =>
        {
            var id = await _db.Insertable(forward).ExecuteReturnIdentityAsync();
            forward.Id = id;
            if (groupIds.Count > 0)
            {
                var relations = groupIds
                    .Where(gid => gid > 0)
                    .Distinct()
                    .Select(gid => new MergeStreamGroup { StreamId = id, GroupId = (int)gid })
                    .ToList();
                if (relations.Count > 0)
                {
                    await _db.Insertable(relations).ExecuteCommandAsync();
                }
            }
        });

        await _configVersionService.BumpAsync("forward", new[] { (long)forward.Id }, cancellationToken);
        _ = await _forwardCnameSyncService.SyncAsync(forward, cancellationToken);

        return ServiceResult<ForwardDetailDto>.Ok(ToDetailDto(forward));
    }
    public async Task<ServiceResult<ForwardDetailDto>> UpdateAsync(
        long id,
        ForwardUpdateRequest request,
        long? userId,
        bool isAdmin,
        CancellationToken cancellationToken)
    {
        if (id <= 0)
        {
            return ServiceResult<ForwardDetailDto>.Fail(ErrorCodes.InvalidParam);
        }

        var targetUserId = ResolveUserId(request.UserId, userId, isAdmin);
        if (!isAdmin && targetUserId <= 0)
        {
            return ServiceResult<ForwardDetailDto>.Fail(ErrorCodes.InvalidParam, "user_id_required");
        }

        var query = _db.Queryable<Stream>().Where(s => s.Id == id);
        if (!isAdmin)
        {
            query = query.Where(s => s.Uid == (int)targetUserId);
        }

        var forward = await query.FirstAsync();
        if (forward == null)
        {
            return ServiceResult<ForwardDetailDto>.Fail(ErrorCodes.NotFound, "forward_not_found");
        }

        var oldPackageId = forward.UserPackage ?? 0;

        if (isAdmin && request.UserId > 0)
        {
            forward.Uid = (int)request.UserId;
        }

        if (request.UserPackageId > 0)
        {
            forward.UserPackage = (int)request.UserPackageId;
        }

        var listenInput = string.IsNullOrWhiteSpace(request.ListenPortsInput)
            ? request.ListenPorts
            : request.ListenPortsInput;
        var listenPorts = ResolveListenPorts(null, listenInput);
        if (listenPorts.Count == 0)
        {
            return ServiceResult<ForwardDetailDto>.Fail(ErrorCodes.MissingParam, "listen_ports_required");
        }

        var originInput = string.IsNullOrWhiteSpace(request.OriginInput)
            ? request.Origin
            : request.OriginInput;
        var origins = ParseOrigins(originInput);
        if (origins.Count == 0)
        {
            return ServiceResult<ForwardDetailDto>.Fail(ErrorCodes.MissingParam, "origin_required");
        }

        forward.Listen = EncodeStringList(listenPorts);
        forward.Backend = EncodeOrigins(origins);
        forward.BackendPort = ExtractBackendPort(origins);

        if (!string.IsNullOrWhiteSpace(request.Remark))
        {
            forward.Acl = UpdateForwardRemark(forward.Acl, request.Remark!.Trim());
        }

        var nodeGroupId = (long)(forward.NodeGroupId ?? 0);
        var requestPackageId = forward.UserPackage ?? 0;
        if (requestPackageId != 0 && (nodeGroupId == 0 || (request.UserPackageId != 0 && request.UserPackageId != oldPackageId)))
        {
            var resolved = await ResolveNodeGroupFromPackageAsync(requestPackageId, 0);
            if (resolved != 0)
            {
                nodeGroupId = resolved;
            }
        }

        if (nodeGroupId != 0)
        {
            forward.NodeGroupId = (int)nodeGroupId;
            var regionId = await ResolveForwardRegionIdAsync(nodeGroupId);
            if (regionId > 0)
            {
                forward.RegionId = (int)regionId;
            }
        }

        UserPackage? pkg = null;
        if (forward.UserPackage is > 0)
        {
            pkg = await _db.Queryable<UserPackage>().Where(p => p.Id == forward.UserPackage.Value).FirstAsync();
        }

        if (!await ApplyForwardCnameAsync(forward, pkg))
        {
            return ServiceResult<ForwardDetailDto>.Fail(ErrorCodes.InternalError, "forward_cname_generate_failed");
        }

        forward.UpdateAt = DateTime.Now;

        var groupIds = ResolveGroupIds(request.GroupIds, request.GroupId);

        await _db.Ado.UseTranAsync(async () =>
        {
            await _db.Updateable(forward).ExecuteCommandAsync();

            await _db.Deleteable<MergeStreamGroup>()
                .Where(r => r.StreamId == forward.Id)
                .ExecuteCommandAsync();

            if (groupIds.Count > 0)
            {
                var relations = groupIds
                    .Where(gid => gid > 0)
                    .Distinct()
                    .Select(gid => new MergeStreamGroup { StreamId = forward.Id, GroupId = (int)gid })
                    .ToList();
                if (relations.Count > 0)
                {
                    await _db.Insertable(relations).ExecuteCommandAsync();
                }
            }
        });

        await _configVersionService.BumpAsync("forward", new[] { (long)forward.Id }, cancellationToken);
        _ = await _forwardCnameSyncService.SyncAsync(forward, cancellationToken);

        return ServiceResult<ForwardDetailDto>.Ok(ToDetailDto(forward));
    }
    public async Task<ServiceResult<ForwardBatchCreateResult>> BatchCreateAsync(
        ForwardBatchCreateRequest request,
        long? userId,
        bool isAdmin,
        CancellationToken cancellationToken)
    {
        var targetUserId = ResolveUserId(request.UserId, userId, isAdmin);
        if (targetUserId <= 0)
        {
            return ServiceResult<ForwardBatchCreateResult>.Fail(ErrorCodes.InvalidParam, "user_id_required");
        }

        if (string.IsNullOrWhiteSpace(request.Data))
        {
            return ServiceResult<ForwardBatchCreateResult>.Fail(ErrorCodes.MissingParam, "forward_batch_data_required");
        }

        var defaults = await LoadStreamDefaultMapAsync(targetUserId);

        UserPackage? pkg = null;
        if (request.UserPackageId != 0)
        {
            pkg = await _db.Queryable<UserPackage>().Where(p => p.Id == request.UserPackageId).FirstAsync();
        }

        var lines = SplitLines(request.Data);
        var created = 0;
        var createdIds = new List<long>();

        foreach (var line in lines)
        {
            var parsed = ParseForwardBatchLine(line);
            if (!parsed.Success)
            {
                if (request.IgnoreError)
                {
                    continue;
                }
                return ServiceResult<ForwardBatchCreateResult>.Fail(ErrorCodes.InvalidParam, parsed.ErrorKey);
            }

            var nodeGroupId = await ResolveNodeGroupFromPackageAsync(request.UserPackageId, 0);
            if (nodeGroupId == 0)
            {
                nodeGroupId = await ResolveDefaultNodeGroupIdAsync();
            }

            var regionId = await ResolveForwardRegionIdAsync(nodeGroupId);

            var now = DateTime.Now;
            var forward = new Stream
            {
                Uid = (int)targetUserId,
                UserPackage = request.UserPackageId > 0 ? (int?)request.UserPackageId : null,
                RegionId = regionId > 0 ? (int?)regionId : null,
                NodeGroupId = nodeGroupId > 0 ? (int?)nodeGroupId : null,
                Listen = EncodeStringList(parsed.ListenPorts),
                Backend = EncodeOrigins(parsed.Origins),
                BackendPort = ExtractBackendPort(parsed.Origins),
                Enable = true,
                State = "running",
                CreateAt = now,
                UpdateAt = now
            };

            var settings = new Dictionary<string, object?>();
            ApplyForwardDefaults(forward, defaults, settings);
            if (!string.IsNullOrWhiteSpace(request.Remark))
            {
                settings["remark"] = request.Remark!.Trim();
            }
            if (settings.Count > 0)
            {
                forward.Acl = JsonSerializer.Serialize(settings, JsonOptions);
            }

            if (!await ApplyForwardCnameAsync(forward, pkg))
            {
                if (request.IgnoreError)
                {
                    continue;
                }
                return ServiceResult<ForwardBatchCreateResult>.Fail(ErrorCodes.InternalError, "forward_cname_generate_failed");
            }

            var groupIds = ResolveGroupIds(request.GroupIds, request.GroupId);
            try
            {
                await _db.Ado.UseTranAsync(async () =>
                {
                    var id = await _db.Insertable(forward).ExecuteReturnIdentityAsync();
                    forward.Id = id;
                    if (groupIds.Count > 0)
                    {
                        var relations = groupIds
                            .Where(gid => gid > 0)
                            .Distinct()
                            .Select(gid => new MergeStreamGroup { StreamId = id, GroupId = (int)gid })
                            .ToList();
                        if (relations.Count > 0)
                        {
                            await _db.Insertable(relations).ExecuteCommandAsync();
                        }
                    }
                });
            }
            catch
            {
                if (request.IgnoreError)
                {
                    continue;
                }
                return ServiceResult<ForwardBatchCreateResult>.Fail(ErrorCodes.InternalError, "forward_create_failed");
            }

            created++;
            createdIds.Add(forward.Id);
            _ = await _forwardCnameSyncService.SyncAsync(forward, cancellationToken);
        }

        if (createdIds.Count > 0)
        {
            await _configVersionService.BumpAsync("forward", createdIds, cancellationToken);
        }

        return ServiceResult<ForwardBatchCreateResult>.Ok(new ForwardBatchCreateResult(created));
    }
    public async Task<ServiceResult<bool>> BatchUpdateAsync(
        ForwardBatchUpdateRequest request,
        long? userId,
        bool isAdmin,
        CancellationToken cancellationToken)
    {
        if (request.Ids == null || request.Ids.Count == 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.MissingParam, "ids_required");
        }

        var ids = request.Ids.ToList();
        if (!isAdmin)
        {
            if (!userId.HasValue || userId <= 0)
            {
                return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam, "user_id_required");
            }
            ids = await FilterForwardIdsForUserAsync(ids, userId.Value);
            if (ids.Count == 0)
            {
                return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam, "no_forwards_available");
            }
        }

        var listenValue = request.ListenPorts != null ? EncodeStringList(request.ListenPorts) : null;
        var backendValue = request.Origins != null ? EncodeOrigins(request.Origins) : null;
        var backendPortValue = request.Origins != null ? ExtractBackendPort(request.Origins) : null;
        var settingsJson = request.Settings != null ? JsonSerializer.Serialize(request.Settings, JsonOptions) : null;
        var originUpdate = request.Settings != null ? BuildOriginUpdate(request.Settings) : null;
        var remarkJson = request.Remark != null && request.Settings == null
            ? JsonSerializer.Serialize(new Dictionary<string, object?> { ["remark"] = request.Remark }, JsonOptions)
            : null;

        await _db.Ado.UseTranAsync(async () =>
        {
            if (request.UserPackageId.HasValue)
            {
                var pkgId = (int)request.UserPackageId.Value;
                await _db.Updateable<Stream>()
                    .SetColumns(s => new Stream { UserPackage = pkgId })
                    .Where(s => ids.Contains(s.Id))
                    .ExecuteCommandAsync();
            }

            if (listenValue != null)
            {
                var value = listenValue;
                await _db.Updateable<Stream>()
                    .SetColumns(s => new Stream { Listen = value })
                    .Where(s => ids.Contains(s.Id))
                    .ExecuteCommandAsync();
            }

            if (backendValue != null)
            {
                var value = backendValue;
                var portValue = backendPortValue;
                await _db.Updateable<Stream>()
                    .SetColumns(s => new Stream { Backend = value, BackendPort = portValue })
                    .Where(s => ids.Contains(s.Id))
                    .ExecuteCommandAsync();
            }

            if (settingsJson != null)
            {
                var aclValue = settingsJson;
                await _db.Updateable<Stream>()
                    .SetColumns(s => new Stream { Acl = aclValue })
                    .Where(s => ids.Contains(s.Id))
                    .ExecuteCommandAsync();
            }
            else if (remarkJson != null)
            {
                var aclValue = remarkJson;
                await _db.Updateable<Stream>()
                    .SetColumns(s => new Stream { Acl = aclValue })
                    .Where(s => ids.Contains(s.Id))
                    .ExecuteCommandAsync();
            }

            if (originUpdate != null)
            {
                if (!string.IsNullOrWhiteSpace(originUpdate.BalanceWay))
                {
                    var balanceWay = originUpdate.BalanceWay;
                    await _db.Updateable<Stream>()
                        .SetColumns(s => new Stream { BalanceWay = balanceWay })
                        .Where(s => ids.Contains(s.Id))
                        .ExecuteCommandAsync();
                }

                if (originUpdate.ProxyProtocol.HasValue)
                {
                    var proxyProtocol = originUpdate.ProxyProtocol.Value;
                    await _db.Updateable<Stream>()
                        .SetColumns(s => new Stream { ProxyProtocol = proxyProtocol })
                        .Where(s => ids.Contains(s.Id))
                        .ExecuteCommandAsync();
                }

                if (!string.IsNullOrWhiteSpace(originUpdate.Backend))
                {
                    var backend = originUpdate.Backend;
                    await _db.Updateable<Stream>()
                        .SetColumns(s => new Stream { Backend = backend })
                        .Where(s => ids.Contains(s.Id))
                        .ExecuteCommandAsync();
                }

                if (!string.IsNullOrWhiteSpace(originUpdate.BackendPort))
                {
                    var backendPort = originUpdate.BackendPort;
                    await _db.Updateable<Stream>()
                        .SetColumns(s => new Stream { BackendPort = backendPort })
                        .Where(s => ids.Contains(s.Id))
                        .ExecuteCommandAsync();
                }
            }

            var groupIds = request.GroupIds;
            if (groupIds == null && request.GroupId.HasValue)
            {
                groupIds = new List<long> { request.GroupId.Value };
            }

            if (groupIds != null)
            {
                await _db.Deleteable<MergeStreamGroup>()
                    .Where(r => ids.Contains(r.StreamId ?? 0))
                    .ExecuteCommandAsync();

                if (groupIds.Count > 0)
                {
                    var relations = new List<MergeStreamGroup>();
                    foreach (var sid in ids)
                    {
                        foreach (var gid in groupIds)
                        {
                            if (gid > 0)
                            {
                                relations.Add(new MergeStreamGroup { StreamId = (int)sid, GroupId = (int)gid });
                            }
                        }
                    }
                    if (relations.Count > 0)
                    {
                        await _db.Insertable(relations).ExecuteCommandAsync();
                    }
                }
            }
        });

        await _configVersionService.BumpAsync("forward", ids, cancellationToken);
        return ServiceResult<bool>.Ok(true);
    }

    public async Task<ServiceResult<ForwardBatchActionResult>> BatchActionAsync(
        ForwardBatchActionRequest request,
        long? userId,
        bool isAdmin,
        CancellationToken cancellationToken)
    {
        if (request.Ids == null || request.Ids.Count == 0)
        {
            return ServiceResult<ForwardBatchActionResult>.Fail(ErrorCodes.MissingParam, "ids_required");
        }

        var action = request.Action?.Trim().ToLowerInvariant();
        if (string.IsNullOrWhiteSpace(action))
        {
            return ServiceResult<ForwardBatchActionResult>.Fail(ErrorCodes.MissingParam);
        }

        var ids = request.Ids.ToList();
        if (!isAdmin)
        {
            if (!userId.HasValue || userId <= 0)
            {
                return ServiceResult<ForwardBatchActionResult>.Fail(ErrorCodes.InvalidParam, "user_id_required");
            }
            ids = await FilterForwardIdsForUserAsync(ids, userId.Value);
            if (ids.Count == 0)
            {
                return ServiceResult<ForwardBatchActionResult>.Fail(ErrorCodes.InvalidParam, "no_forwards_available");
            }
        }

        switch (action)
        {
            case "enable":
            {
                var taskResult = await _resourceActionRequestService.RequestAsync(
                    StreamActionCommandFactory.CreateStatusChange(ids, true, userId, userId),
                    cancellationToken);
                if (!taskResult.Success)
                {
                    return ServiceResult<ForwardBatchActionResult>.Fail(taskResult.ErrorCode, taskResult.MessageKey);
                }

                return ServiceResult<ForwardBatchActionResult>.Ok(new ForwardBatchActionResult(taskResult.Data!.TaskId));
            }
            case "disable":
            {
                var taskResult = await _resourceActionRequestService.RequestAsync(
                    StreamActionCommandFactory.CreateStatusChange(ids, false, userId, userId),
                    cancellationToken);
                if (!taskResult.Success)
                {
                    return ServiceResult<ForwardBatchActionResult>.Fail(taskResult.ErrorCode, taskResult.MessageKey);
                }

                return ServiceResult<ForwardBatchActionResult>.Ok(new ForwardBatchActionResult(taskResult.Data!.TaskId));
            }
            case "delete":
            {
                var taskResult = await _resourceActionRequestService.RequestAsync(
                    StreamActionCommandFactory.CreateDelete(ids, userId, userId),
                    cancellationToken);
                if (!taskResult.Success)
                {
                    return ServiceResult<ForwardBatchActionResult>.Fail(taskResult.ErrorCode, taskResult.MessageKey);
                }

                return ServiceResult<ForwardBatchActionResult>.Ok(new ForwardBatchActionResult(taskResult.Data!.TaskId));
            }
            default:
                return ServiceResult<ForwardBatchActionResult>.Fail(ErrorCodes.InvalidParam, "unknown_action");
        }
    }
    private static long ResolveUserId(long requestUserId, long? userId, bool isAdmin)
    {
        if (!isAdmin)
        {
            return userId ?? 0;
        }

        return requestUserId > 0 ? requestUserId : (userId ?? 0);
    }

    private async Task<ISugarQueryable<Stream>> ApplySearchAsync(ISugarQueryable<Stream> query, string keyword, string searchField)
    {
        switch (searchField)
        {
            case "forward_id":
                if (long.TryParse(keyword, out var id) && id > 0)
                {
                    query = query.Where(s => s.Id == id);
                }
                else
                {
                    query = query.Where(_ => false);
                }
                break;
            case "listen":
                query = query.Where(s => SqlFunc.Contains(s.Listen, keyword));
                break;
            case "origin":
                query = query.Where(s => SqlFunc.Contains(s.Backend, keyword));
                break;
            case "cname":
                query = query.Where(s => SqlFunc.Contains(s.CnameHostname, keyword));
                break;
            case "package":
                {
                    var ids = await FindUserPackageIdsByNameAsync(keyword);
                    if (ids.Count == 0)
                    {
                        query = query.Where(_ => false);
                    }
                    else
                    {
                        query = query.Where(s => ids.Contains(s.UserPackage ?? 0));
                    }
                }
                break;
            case "group":
                {
                    var ids = await FindForwardIdsByGroupNameAsync(keyword);
                    if (ids.Count == 0)
                    {
                        query = query.Where(_ => false);
                    }
                    else
                    {
                        query = query.Where(s => ids.Contains(s.Id));
                    }
                }
                break;
            case "user":
                {
                    var userIds = await FindUserIdsByKeywordAsync(keyword);
                    if (userIds.Count == 0)
                    {
                        query = query.Where(_ => false);
                    }
                    else
                    {
                        query = query.Where(s => s.Uid.HasValue && userIds.Contains(s.Uid.Value));
                    }
                }
                break;
            default:
                {
                    var cond = Expressionable.Create<Stream>();
                    cond.Or(s => SqlFunc.Contains(s.Listen, keyword) ||
                                 SqlFunc.Contains(s.Backend, keyword) ||
                                 SqlFunc.Contains(s.CnameHostname, keyword));
                    if (long.TryParse(keyword, out var keywordId) && keywordId > 0)
                    {
                        cond.Or(s => s.Id == keywordId);
                    }
                    var userIds = await FindUserIdsByKeywordAsync(keyword);
                    if (userIds.Count > 0)
                    {
                        cond.Or(s => s.Uid.HasValue && userIds.Contains(s.Uid.Value));
                    }
                    var pkgIds = await FindUserPackageIdsByNameAsync(keyword);
                    if (pkgIds.Count > 0)
                    {
                        cond.Or(s => s.UserPackage.HasValue && pkgIds.Contains(s.UserPackage.Value));
                    }
                    var forwardIds = await FindForwardIdsByGroupNameAsync(keyword);
                    if (forwardIds.Count > 0)
                    {
                        cond.Or(s => forwardIds.Contains(s.Id));
                    }
                    query = query.Where(cond.ToExpression());
                }
                break;
        }
        return query;
    }
    private async Task<IReadOnlyList<ForwardListItem>> BuildForwardListItemsAsync(IReadOnlyList<Stream> forwards)
    {
        if (forwards.Count == 0)
        {
            return Array.Empty<ForwardListItem>();
        }

        var userMap = await LoadUsersForForwardAsync(forwards);
        var pkgMap = await LoadUserPackagesForForwardAsync(forwards);
        var (groupMap, relMap) = await LoadForwardGroupsAsync(forwards);
        var nodeGroupMap = await LoadNodeGroupsForForwardAsync(forwards);

        var items = new List<ForwardListItem>(forwards.Count);
        foreach (var forward in forwards)
        {
            var origins = ParseForwardOrigins(forward.Backend);
            var originDisplay = string.Empty;
            var originInput = string.Empty;
            if (origins.Count > 0)
            {
                var addresses = origins.Select(o => o.Address).Where(a => !string.IsNullOrWhiteSpace(a)).ToList();
                originDisplay = string.Join(",", addresses);
                originInput = string.Join(" ", addresses);
            }
            else if (!string.IsNullOrWhiteSpace(forward.Backend))
            {
                originDisplay = forward.Backend!;
                originInput = forward.Backend!;
            }

            UserPackage? pkg = null;
            var pkgName = string.Empty;
            if (forward.UserPackage.HasValue && pkgMap.TryGetValue(forward.UserPackage.Value, out var stored))
            {
                pkg = stored;
                pkgName = stored.Name ?? string.Empty;
            }

            var beforeCname = forward.CnameHostname?.Trim() ?? string.Empty;
            if (await ApplyForwardCnameAsync(forward, pkg))
            {
                if (string.IsNullOrWhiteSpace(beforeCname) && !string.IsNullOrWhiteSpace(forward.CnameHostname) && forward.Id > 0)
                {
                    await _db.Updateable<Stream>()
                        .SetColumns(s => new Stream { CnameHostname = forward.CnameHostname, CnameDomain = forward.CnameDomain })
                        .Where(s => s.Id == forward.Id)
                        .ExecuteCommandAsync();
                }
            }

            var cname = string.IsNullOrWhiteSpace(forward.CnameHostname) ? "-" : forward.CnameHostname!.Trim();

            var groupIds = relMap.TryGetValue(forward.Id, out var rel) ? rel : Array.Empty<long>();
            var groupNames = new List<string>();
            var primaryGroupId = 0L;
            if (groupIds.Count > 0)
            {
                primaryGroupId = groupIds[0];
                foreach (var gid in groupIds)
                {
                    if (groupMap.TryGetValue(gid, out var groupName) && !string.IsNullOrWhiteSpace(groupName))
                    {
                        groupNames.Add(groupName);
                    }
                }
            }

            var remark = ExtractRemark(forward.Acl);
            var listenPorts = SplitFieldsFromRaw(forward.Listen);

            items.Add(new ForwardListItem
            {
                Id = forward.Id,
                UserId = forward.Uid ?? 0,
                UserName = userMap.TryGetValue(forward.Uid ?? 0, out var name) ? name : string.Empty,
                ListenPorts = string.Join(" ", listenPorts),
                OriginDisplay = originDisplay,
                Origin = originInput,
                UserPackageId = forward.UserPackage ?? 0,
                UserPackageName = pkgName,
                GroupId = primaryGroupId,
                GroupIds = groupIds,
                GroupName = string.Join(",", groupNames),
                NodeGroupId = forward.NodeGroupId ?? 0,
                NodeGroupName = nodeGroupMap.TryGetValue(forward.NodeGroupId ?? 0, out var ngName) ? ngName : string.Empty,
                Cname = cname,
                Status = forward.Enable ?? false,
                Remark = remark,
                CreatedAt = forward.CreateAt
            });
        }

        return items;
    }

    private static ForwardDetailDto ToDetailDto(Stream forward)
    {
        var origins = ParseForwardOrigins(forward.Backend);
        return new ForwardDetailDto
        {
            Id = forward.Id,
            UserId = forward.Uid ?? 0,
            UserPackageId = forward.UserPackage ?? 0,
            RegionId = forward.RegionId ?? 0,
            NodeGroupId = forward.NodeGroupId ?? 0,
            BackupNodeGroup = forward.BackupNodeGroup ?? 0,
            EnableBackupGroup = forward.EnableBackupGroup ?? false,
            Enable = forward.Enable ?? false,
            State = forward.State,
            Remark = ExtractRemark(forward.Acl),
            CnameDomain = forward.CnameDomain,
            CnameHostname2 = forward.CnameHostname2,
            CnameMode = forward.CnameMode,
            Cname = forward.CnameHostname,
            ListenPorts = SplitFieldsFromRaw(forward.Listen),
            Origins = origins,
            BackendPort = forward.BackendPort,
            BalanceWay = forward.BalanceWay,
            ProxyProtocol = forward.ProxyProtocol ?? false,
            ConnLimit = forward.ConnLimit,
            Settings = ParseSettings(forward.Acl),
            CreatedAt = forward.CreateAt,
            UpdatedAt = forward.UpdateAt
        };
    }

    private async Task<Dictionary<long, string>> LoadUsersForForwardAsync(IReadOnlyList<Stream> items)
    {
        var ids = items.Select(s => s.Uid ?? 0).Where(id => id > 0).Distinct().ToList();
        var result = new Dictionary<long, string>();
        if (ids.Count == 0)
        {
            return result;
        }

        var users = await _db.Queryable<User>().Where(u => ids.Contains(u.Id)).ToListAsync();
        foreach (var user in users)
        {
            result[user.Id] = user.Name ?? string.Empty;
        }
        return result;
    }

    private async Task<Dictionary<long, UserPackage>> LoadUserPackagesForForwardAsync(IReadOnlyList<Stream> items)
    {
        var ids = items.Select(s => s.UserPackage ?? 0).Where(id => id > 0).Distinct().ToList();
        var result = new Dictionary<long, UserPackage>();
        if (ids.Count == 0)
        {
            return result;
        }

        var pkgs = await _db.Queryable<UserPackage>().Where(p => ids.Contains(p.Id)).ToListAsync();
        foreach (var pkg in pkgs)
        {
            result[pkg.Id] = pkg;
        }
        return result;
    }

    private async Task<(Dictionary<long, string> GroupMap, Dictionary<long, IReadOnlyList<long>> RelMap)> LoadForwardGroupsAsync(IReadOnlyList<Stream> items)
    {
        var ids = items.Select(s => s.Id).Where(id => id > 0).Distinct().ToList();
        var groupMap = new Dictionary<long, string>();
        var relMap = new Dictionary<long, IReadOnlyList<long>>();
        if (ids.Count == 0)
        {
            return (groupMap, relMap);
        }

        var relations = await _db.Queryable<MergeStreamGroup>().Where(r => ids.Contains(r.StreamId ?? 0)).ToListAsync();
        var groupIds = new HashSet<long>();
        var relBuffer = new Dictionary<long, List<long>>();
        foreach (var rel in relations)
        {
            var sid = rel.StreamId ?? 0;
            var gid = rel.GroupId ?? 0;
            if (sid == 0 || gid == 0)
            {
                continue;
            }
            if (!relBuffer.TryGetValue(sid, out var list))
            {
                list = new List<long>();
                relBuffer[sid] = list;
            }
            list.Add(gid);
            groupIds.Add(gid);
        }

        foreach (var pair in relBuffer)
        {
            relMap[pair.Key] = pair.Value;
        }

        if (groupIds.Count == 0)
        {
            return (groupMap, relMap);
        }

        var groups = await _db.Queryable<StreamGroup>().Where(g => groupIds.Contains(g.Id)).ToListAsync();
        foreach (var group in groups)
        {
            groupMap[group.Id] = group.Name ?? string.Empty;
        }

        return (groupMap, relMap);
    }

    private async Task<Dictionary<long, string>> LoadNodeGroupsForForwardAsync(IReadOnlyList<Stream> items)
    {
        var ids = items.Select(s => s.NodeGroupId ?? 0).Where(id => id > 0).Distinct().ToList();
        var result = new Dictionary<long, string>();
        if (ids.Count == 0)
        {
            return result;
        }

        var groups = await _db.Queryable<NodeGroup>().Where(g => ids.Contains(g.Id)).ToListAsync();
        foreach (var group in groups)
        {
            result[group.Id] = group.Name ?? string.Empty;
        }
        return result;
    }
    private async Task<List<long>> FindUserIdsByKeywordAsync(string keyword)
    {
        var users = await _db.Queryable<User>()
            .Where(u => SqlFunc.Contains(u.Name, keyword) || SqlFunc.Contains(u.Email, keyword) || SqlFunc.Contains(u.Phone, keyword))
            .ToListAsync();
        return users.Select(u => (long)u.Id).ToList();
    }

    private async Task<List<long>> FindUserPackageIdsByNameAsync(string keyword)
    {
        var pkgs = await _db.Queryable<UserPackage>()
            .Where(p => SqlFunc.Contains(p.Name, keyword))
            .ToListAsync();
        return pkgs.Select(p => (long)p.Id).ToList();
    }

    private async Task<List<long>> FindForwardIdsByGroupNameAsync(string keyword)
    {
        var groups = await _db.Queryable<StreamGroup>()
            .Where(g => SqlFunc.Contains(g.Name, keyword))
            .ToListAsync();
        if (groups.Count == 0)
        {
            return new List<long>();
        }
        var ids = groups.Select(g => (long)g.Id).ToList();
        return await FindForwardIdsByGroupIdsAsync(ids);
    }

    private async Task<List<long>> FindForwardIdsByGroupIdAsync(long groupId)
    {
        return await FindForwardIdsByGroupIdsAsync(new List<long> { groupId });
    }

    private async Task<List<long>> FindForwardIdsByGroupIdsAsync(IReadOnlyList<long> groupIds)
    {
        if (groupIds.Count == 0)
        {
            return new List<long>();
        }
        var relations = await _db.Queryable<MergeStreamGroup>()
            .Where(r => r.GroupId.HasValue && groupIds.Contains(r.GroupId.Value))
            .ToListAsync();
        return relations.Select(r => (long)(r.StreamId ?? 0)).Where(id => id > 0).ToList();
    }

    private async Task<List<long>> FilterForwardIdsForUserAsync(IReadOnlyList<long> ids, long userId)
    {
        if (ids.Count == 0)
        {
            return new List<long>();
        }

        return await _db.Queryable<Stream>()
            .Where(s => s.Uid == (int)userId && ids.Contains(s.Id))
            .Select(s => (long)s.Id)
            .ToListAsync();
    }

    private static List<string> ResolveListenPorts(IReadOnlyList<string>? ports, string? input)
    {
        if (ports != null && ports.Count > 0)
        {
            return ports.Select(p => p?.Trim() ?? string.Empty)
                .Where(p => !string.IsNullOrWhiteSpace(p))
                .ToList();
        }

        if (!string.IsNullOrWhiteSpace(input))
        {
            return SplitFields(input);
        }

        return new List<string>();
    }

    private static List<long> ResolveGroupIds(IReadOnlyList<long>? groupIds, long groupId)
    {
        if (groupIds != null && groupIds.Count > 0)
        {
            return groupIds.ToList();
        }
        if (groupId > 0)
        {
            return new List<long> { groupId };
        }
        return new List<long>();
    }

    private static string EncodeStringList(IEnumerable<string> items)
    {
        var list = items.Select(item => item?.Trim() ?? string.Empty)
            .Where(item => !string.IsNullOrWhiteSpace(item))
            .ToList();
        return list.Count == 0 ? string.Empty : JsonSerializer.Serialize(list, JsonOptions);
    }

    private static string EncodeOrigins(IEnumerable<ForwardOriginDto> origins)
    {
        var list = origins.Where(o => !string.IsNullOrWhiteSpace(o.Address))
            .Select(o => new ForwardOriginDto
            {
                Address = o.Address!.Trim(),
                Weight = o.Weight <= 0 ? 1 : o.Weight,
                Enable = o.Enable
            })
            .ToList();
        return list.Count == 0 ? string.Empty : JsonSerializer.Serialize(list, JsonOptions);
    }

    private static string EncodeOrigins(IEnumerable<ForwardOrigin> origins)
    {
        var list = origins.Where(o => !string.IsNullOrWhiteSpace(o.Address))
            .Select(o => new ForwardOriginDto
            {
                Address = o.Address!.Trim(),
                Weight = o.Weight <= 0 ? 1 : o.Weight,
                Enable = o.Enable
            })
            .ToList();
        return list.Count == 0 ? string.Empty : JsonSerializer.Serialize(list, JsonOptions);
    }

    private static List<ForwardOrigin> ParseOrigins(string? input)
    {
        var origins = new List<ForwardOrigin>();
        if (string.IsNullOrWhiteSpace(input))
        {
            return origins;
        }

        foreach (var item in SplitFields(input))
        {
            origins.Add(new ForwardOrigin(item, 1, true));
        }
        return origins;
    }

    private static ForwardBatchParseResult ParseForwardBatchLine(string line)
    {
        var parts = line.Split('|');
        if (parts.Length < 2)
        {
            return ForwardBatchParseResult.Fail("forward_batch_data_invalid");
        }

        var listenPorts = SplitFields(parts[0]);
        if (listenPorts.Count == 0)
        {
            return ForwardBatchParseResult.Fail("listen_ports_required");
        }

        var origins = ParseOrigins(parts[1]);
        if (origins.Count == 0)
        {
            return ForwardBatchParseResult.Fail("origin_required");
        }

        return ForwardBatchParseResult.Ok(listenPorts, origins);
    }
    private static string ExtractBackendPort(IReadOnlyList<ForwardOrigin> origins)
    {
        if (origins.Count == 0)
        {
            return string.Empty;
        }

        var addr = origins[0].Address?.Trim() ?? string.Empty;
        if (string.IsNullOrWhiteSpace(addr))
        {
            return string.Empty;
        }

        if (addr.Count(c => c == ':') == 1)
        {
            var parts = addr.Split(':');
            if (parts.Length == 2 && !string.IsNullOrWhiteSpace(parts[1]))
            {
                return parts[1];
            }
        }

        var idx = addr.IndexOf("]:", StringComparison.Ordinal);
        if (idx >= 0)
        {
            var portPart = addr[(idx + 2)..];
            if (!string.IsNullOrWhiteSpace(portPart))
            {
                return portPart;
            }
        }

        return string.Empty;
    }

    private static string ExtractBackendPort(IReadOnlyList<ForwardOriginDto> origins)
    {
        var list = origins.Select(o => new ForwardOrigin(o.Address ?? string.Empty, o.Weight, o.Enable)).ToList();
        return ExtractBackendPort(list);
    }

    private static List<string> SplitLines(string raw)
    {
        return raw.Split('\n')
            .Select(part => part.Trim())
            .Where(part => !string.IsNullOrWhiteSpace(part))
            .ToList();
    }

    private static List<string> SplitFields(string raw)
    {
        var normalized = raw.Replace(",", " ").Replace(";", " ").Replace("\n", " ").Replace("\r", " ");
        return normalized.Split(' ', StringSplitOptions.RemoveEmptyEntries)
            .Select(part => part.Trim())
            .Where(part => !string.IsNullOrWhiteSpace(part))
            .ToList();
    }

    private static List<string> SplitFieldsFromRaw(string? raw)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            return new List<string>();
        }

        var trimmed = raw.Trim();
        if (trimmed.StartsWith("["))
        {
            try
            {
                var list = JsonSerializer.Deserialize<List<string>>(trimmed, JsonOptions);
                if (list != null)
                {
                    return list.Select(item => item?.Trim() ?? string.Empty).Where(item => !string.IsNullOrWhiteSpace(item)).ToList();
                }
            }
            catch
            {
            }
        }

        return SplitFields(trimmed);
    }

    private static IReadOnlyList<ForwardOriginDto> ParseForwardOrigins(string? raw)
    {
        var result = new List<ForwardOriginDto>();
        if (string.IsNullOrWhiteSpace(raw))
        {
            return result;
        }

        var trimmed = raw.Trim();
        if (trimmed.StartsWith("["))
        {
            try
            {
                using var doc = JsonDocument.Parse(trimmed);
                if (doc.RootElement.ValueKind == JsonValueKind.Array)
                {
                    foreach (var item in doc.RootElement.EnumerateArray())
                    {
                        if (item.ValueKind == JsonValueKind.Object)
                        {
                            var address = ReadJsonString(item, "address") ?? ReadJsonString(item, "addr");
                            if (string.IsNullOrWhiteSpace(address))
                            {
                                continue;
                            }
                            var weight = ReadJsonInt(item, "weight", 1);
                            var enable = ReadJsonBool(item, "enable", true);
                            result.Add(new ForwardOriginDto { Address = address, Weight = weight, Enable = enable });
                            continue;
                        }

                        if (item.ValueKind == JsonValueKind.String)
                        {
                            var addr = item.GetString();
                            if (!string.IsNullOrWhiteSpace(addr))
                            {
                                result.Add(new ForwardOriginDto { Address = addr, Weight = 1, Enable = true });
                            }
                        }
                    }
                }
                return result;
            }
            catch
            {
            }
        }

        foreach (var item in SplitFields(trimmed))
        {
            result.Add(new ForwardOriginDto { Address = item, Weight = 1, Enable = true });
        }

        return result;
    }

    private static Dictionary<string, JsonElement>? ParseSettings(string? raw)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            return null;
        }

        try
        {
            return JsonSerializer.Deserialize<Dictionary<string, JsonElement>>(raw, JsonOptions);
        }
        catch
        {
            return null;
        }
    }

    private static string ExtractRemark(string? raw)
    {
        var settings = ParseSettings(raw);
        if (settings == null)
        {
            return string.Empty;
        }
        if (settings.TryGetValue("remark", out var value))
        {
            return value.ValueKind == JsonValueKind.String ? value.GetString() ?? string.Empty : value.ToString();
        }
        return string.Empty;
    }

    private static string UpdateForwardRemark(string? raw, string remark)
    {
        Dictionary<string, object?> map;
        if (string.IsNullOrWhiteSpace(raw))
        {
            map = new Dictionary<string, object?>();
        }
        else
        {
            try
            {
                map = JsonSerializer.Deserialize<Dictionary<string, object?>>(raw, JsonOptions) ??
                      new Dictionary<string, object?>();
            }
            catch
            {
                map = new Dictionary<string, object?>();
            }
        }

        map["remark"] = remark;
        return JsonSerializer.Serialize(map, JsonOptions);
    }

    private static OriginUpdate? BuildOriginUpdate(Dictionary<string, JsonElement> settings)
    {
        if (settings.Count == 0)
        {
            return null;
        }

        if (!settings.TryGetValue("origin", out var origin) || origin.ValueKind != JsonValueKind.Object)
        {
            return null;
        }

        string? balanceWay = null;
        bool? proxyProtocol = null;
        string? backend = null;
        string? backendPort = null;

        if (origin.TryGetProperty("balance_way", out var balanceValue))
        {
            var value = balanceValue.ValueKind == JsonValueKind.String ? balanceValue.GetString() : balanceValue.ToString();
            if (!string.IsNullOrWhiteSpace(value))
            {
                balanceWay = value;
            }
        }

        if (origin.TryGetProperty("proxy_protocol", out var proxyValue))
        {
            proxyProtocol = ParseBool(proxyValue, false);
        }

        if (origin.TryGetProperty("backsource_port", out var portValue))
        {
            var value = portValue.ValueKind == JsonValueKind.String ? portValue.GetString() : portValue.ToString();
            if (!string.IsNullOrWhiteSpace(value))
            {
                backendPort = value;
            }
        }

        if (origin.TryGetProperty("origins", out var originsValue))
        {
            var encoded = EncodeOriginsFromJson(originsValue);
            if (!string.IsNullOrWhiteSpace(encoded))
            {
                backend = encoded;
            }
        }

        if (string.IsNullOrWhiteSpace(balanceWay) && !proxyProtocol.HasValue &&
            string.IsNullOrWhiteSpace(backend) && string.IsNullOrWhiteSpace(backendPort))
        {
            return null;
        }

        return new OriginUpdate(balanceWay, proxyProtocol, backend, backendPort);
    }

    private static void ApplyOriginSettings(Dictionary<string, JsonElement> settings, Dictionary<string, object?> updates)
    {
        if (!settings.TryGetValue("origin", out var origin) || origin.ValueKind != JsonValueKind.Object)
        {
            return;
        }

        if (origin.TryGetProperty("balance_way", out var balanceValue))
        {
            var balanceWay = balanceValue.ValueKind == JsonValueKind.String ? balanceValue.GetString() : balanceValue.ToString();
            if (!string.IsNullOrWhiteSpace(balanceWay))
            {
                updates["balance_way"] = balanceWay;
            }
        }

        if (origin.TryGetProperty("proxy_protocol", out var proxyValue))
        {
            updates["proxy_protocol"] = ParseBool(proxyValue, false);
        }

        if (origin.TryGetProperty("backsource_port", out var portValue))
        {
            var port = portValue.ValueKind == JsonValueKind.String ? portValue.GetString() : portValue.ToString();
            if (!string.IsNullOrWhiteSpace(port))
            {
                updates["backend_port"] = port;
            }
        }

        if (origin.TryGetProperty("origins", out var originsValue))
        {
            var encoded = EncodeOriginsFromJson(originsValue);
            if (!string.IsNullOrWhiteSpace(encoded))
            {
                updates["backend"] = encoded;
            }
        }
    }

    private static string EncodeOriginsFromJson(JsonElement originsValue)
    {
        if (originsValue.ValueKind != JsonValueKind.Array)
        {
            return string.Empty;
        }

        var origins = new List<ForwardOriginDto>();
        foreach (var item in originsValue.EnumerateArray())
        {
            if (item.ValueKind != JsonValueKind.Object)
            {
                continue;
            }
            var address = ReadJsonString(item, "address") ?? string.Empty;
            if (string.IsNullOrWhiteSpace(address))
            {
                continue;
            }
            var weight = ReadJsonInt(item, "weight", 1);
            var enable = ReadJsonBool(item, "enable", true);
            origins.Add(new ForwardOriginDto { Address = address, Weight = weight, Enable = enable });
        }

        return EncodeOrigins(origins);
    }
    private async Task<long> ResolveNodeGroupFromPackageAsync(long userPackageId, long requestedId)
    {
        if (requestedId != 0)
        {
            return requestedId;
        }

        if (userPackageId == 0)
        {
            return 0;
        }

        var pkg = await _db.Queryable<UserPackage>()
            .Where(p => p.Id == userPackageId)
            .Select(p => new { p.NodeGroupId, p.Package })
            .FirstAsync();
        if (pkg == null)
        {
            return 0;
        }

        if (pkg.NodeGroupId is > 0)
        {
            return pkg.NodeGroupId.Value;
        }

        if (pkg.Package is > 0)
        {
            var plan = await _db.Queryable<Package>()
                .Where(p => p.Id == pkg.Package.Value)
                .Select(p => new { p.NodeGroupId })
                .FirstAsync();
            if (plan?.NodeGroupId is > 0)
            {
                return plan.NodeGroupId.Value;
            }
        }

        return 0;
    }

    private async Task<long> ResolveForwardRegionIdAsync(long nodeGroupId)
    {
        if (nodeGroupId != 0)
        {
            var group = await _db.Queryable<NodeGroup>()
                .Where(g => g.Id == nodeGroupId)
                .Select(g => g.RegionId)
                .FirstAsync();
            if (group.HasValue && group.Value > 0)
            {
                return group.Value;
            }
        }

        var region = await _db.Queryable<Region>()
            .OrderBy(r => r.Id, OrderByType.Asc)
            .Select(r => r.Id)
            .FirstAsync();
        return region;
    }

    private async Task<long> ResolveDefaultNodeGroupIdAsync()
    {
        var line = await _db.Queryable<Line>()
            .OrderBy(l => l.Id, OrderByType.Asc)
            .Select(l => l.NodeGroupId)
            .FirstAsync();
        if (line.HasValue && line.Value > 0)
        {
            return line.Value;
        }

        var group = await _db.Queryable<NodeGroup>()
            .OrderBy(g => g.Id, OrderByType.Asc)
            .Select(g => g.Id)
            .FirstAsync();
        return group;
    }

    private async Task<Dictionary<string, string>> LoadStreamDefaultMapAsync(long userId)
    {
        var global = await LoadConfigMapAsync("stream_default_config", "global", 0);
        var forwardDefaults = await LoadForwardDefaultMapAsync();
        if (forwardDefaults.Count > 0)
        {
            foreach (var pair in forwardDefaults)
            {
                global[pair.Key] = pair.Value;
            }
        }

        var userDefaults = await LoadConfigMapAsync("stream_default_config", "user", (int)userId);
        if (userDefaults.Count == 0)
        {
            return global;
        }

        foreach (var pair in userDefaults)
        {
            global[pair.Key] = pair.Value;
        }
        return global;
    }

    private async Task<Dictionary<string, string>> LoadConfigMapAsync(string type, string scopeName, int scopeId)
    {
        var items = await _db.Queryable<Config>()
            .Where(c => c.Type == type && c.ScopeName == scopeName && c.ScopeId == scopeId)
            .ToListAsync();
        var map = new Dictionary<string, string>(StringComparer.OrdinalIgnoreCase);
        foreach (var item in items)
        {
            if (string.IsNullOrWhiteSpace(item.Name))
            {
                continue;
            }
            map[item.Name] = item.Value ?? string.Empty;
        }
        return map;
    }

    private async Task<Dictionary<string, string>> LoadForwardDefaultMapAsync()
    {
        var cfg = await _db.Queryable<Config>()
            .Where(c => c.Name == ForwardDefaultKey && c.Type == "system")
            .FirstAsync();
        if (cfg == null || string.IsNullOrWhiteSpace(cfg.Value))
        {
            return new Dictionary<string, string>(StringComparer.OrdinalIgnoreCase);
        }

        try
        {
            using var doc = JsonDocument.Parse(cfg.Value);
            if (doc.RootElement.ValueKind != JsonValueKind.Array)
            {
                return new Dictionary<string, string>(StringComparer.OrdinalIgnoreCase);
            }

            var map = new Dictionary<string, string>(StringComparer.OrdinalIgnoreCase);
            foreach (var item in doc.RootElement.EnumerateArray())
            {
                if (item.ValueKind != JsonValueKind.Object)
                {
                    continue;
                }

                var key = ReadJsonString(item, "key");
                if (string.IsNullOrWhiteSpace(key))
                {
                    continue;
                }

                var scope = ReadJsonString(item, "scope");
                if (!string.IsNullOrWhiteSpace(scope) &&
                    !string.Equals(scope.Trim(), "global", StringComparison.OrdinalIgnoreCase))
                {
                    continue;
                }

                if (!item.TryGetProperty("value", out var value))
                {
                    map[key] = string.Empty;
                    continue;
                }

                map[key] = EncodeForwardDefaultValue(value);
            }

            return map;
        }
        catch
        {
            return new Dictionary<string, string>(StringComparer.OrdinalIgnoreCase);
        }
    }

    private static string EncodeForwardDefaultValue(JsonElement value)
    {
        return value.ValueKind switch
        {
            JsonValueKind.String => value.GetString() ?? string.Empty,
            JsonValueKind.True => "true",
            JsonValueKind.False => "false",
            JsonValueKind.Number => EncodeNumber(value),
            JsonValueKind.Null => string.Empty,
            _ => JsonSerializer.Serialize(value, JsonOptions)
        };
    }

    private static string EncodeNumber(JsonElement value)
    {
        if (value.TryGetInt64(out var parsed))
        {
            return parsed.ToString();
        }

        if (value.TryGetDouble(out var dbl))
        {
            return ((long)dbl).ToString();
        }

        return value.ToString();
    }

    private static void ApplyForwardDefaults(Stream forward, Dictionary<string, string> defaults, Dictionary<string, object?> settings)
    {
        if (defaults.Count == 0)
        {
            return;
        }

        if (!settings.ContainsKey("listen_protocol") && defaults.TryGetValue("listen_protocol", out var listenProtocol) && !string.IsNullOrWhiteSpace(listenProtocol))
        {
            settings["listen_protocol"] = listenProtocol;
        }

        if (!settings.TryGetValue("origin", out var originValue) || originValue is not Dictionary<string, object?> originMap)
        {
            originMap = new Dictionary<string, object?>();
            settings["origin"] = originMap;
        }

        if (!originMap.ContainsKey("balance_way") && defaults.TryGetValue("balance_way", out var balanceWay))
        {
            originMap["balance_way"] = balanceWay;
        }

        if (defaults.TryGetValue("proxy_protocol", out var proxyProtocolRaw) && !string.IsNullOrWhiteSpace(proxyProtocolRaw))
        {
            if (!originMap.ContainsKey("proxy_protocol"))
            {
                originMap["proxy_protocol"] = ParseBool(proxyProtocolRaw);
            }

            forward.ProxyProtocol = ParseBool(proxyProtocolRaw);
        }

        if (string.IsNullOrWhiteSpace(forward.BalanceWay) && defaults.TryGetValue("balance_way", out var balanceDefault))
        {
            forward.BalanceWay = balanceDefault;
        }

        if (string.IsNullOrWhiteSpace(forward.BackendPort) && defaults.TryGetValue("backsource_port", out var backPort))
        {
            forward.BackendPort = backPort;
        }
    }

    private async Task<bool> ApplyForwardCnameAsync(Stream forward, UserPackage? pkg)
    {
        if (forward == null)
        {
            return true;
        }

        var mode = ResolveForwardCnameMode(forward, pkg);
        var domain = ResolveForwardCnameDomain(forward, pkg);

        if (string.Equals(mode, "package", StringComparison.OrdinalIgnoreCase))
        {
            var host = ResolvePackageCnameHost(pkg);
            if (string.IsNullOrWhiteSpace(host))
            {
                host = forward.CnameHostname?.Trim() ?? string.Empty;
            }

            var cname = BuildCnameHostname(host, domain);
            if (!string.IsNullOrWhiteSpace(cname) && !string.Equals(cname, forward.CnameHostname, StringComparison.OrdinalIgnoreCase))
            {
                forward.CnameHostname = cname;
            }

            if (!string.IsNullOrWhiteSpace(domain) && !string.Equals(domain, forward.CnameDomain, StringComparison.OrdinalIgnoreCase))
            {
                forward.CnameDomain = domain;
            }

            return true;
        }

        var current = forward.CnameHostname?.Trim() ?? string.Empty;
        if (string.IsNullOrWhiteSpace(current))
        {
            var host = await GenerateUniqueForwardHostnameAsync(domain);
            if (string.IsNullOrWhiteSpace(host))
            {
                return false;
            }
            var cname = BuildCnameHostname(host, domain);
            if (!string.IsNullOrWhiteSpace(cname))
            {
                forward.CnameHostname = cname;
            }
        }

        if (!string.IsNullOrWhiteSpace(domain) && !string.Equals(domain, forward.CnameDomain, StringComparison.OrdinalIgnoreCase))
        {
            forward.CnameDomain = domain;
        }

        return true;
    }

    private static string ResolveForwardCnameMode(Stream forward, UserPackage? pkg)
    {
        var mode = forward.CnameMode?.Trim().ToLowerInvariant() ?? string.Empty;
        if (string.IsNullOrWhiteSpace(mode) && pkg != null)
        {
            mode = pkg.CnameMode?.Trim().ToLowerInvariant() ?? string.Empty;
        }
        return mode;
    }

    private static string ResolveForwardCnameDomain(Stream forward, UserPackage? pkg)
    {
        var domain = forward.CnameDomain?.Trim() ?? string.Empty;
        if (string.IsNullOrWhiteSpace(domain) && pkg != null)
        {
            domain = pkg.CnameDomain?.Trim() ?? string.Empty;
        }
        if (string.IsNullOrWhiteSpace(domain))
        {
            domain = DefaultCnameDomain;
        }
        return DomainHelper.NormalizeDomainInput(domain);
    }

    private static string ResolvePackageCnameHost(UserPackage? pkg)
    {
        if (pkg == null)
        {
            return string.Empty;
        }
        if (!string.IsNullOrWhiteSpace(pkg.CnameHostname))
        {
            return pkg.CnameHostname.Trim();
        }
        if (!string.IsNullOrWhiteSpace(pkg.CnameHostname2))
        {
            return pkg.CnameHostname2.Trim();
        }
        if (!string.IsNullOrWhiteSpace(pkg.RecordId))
        {
            return pkg.RecordId.Trim();
        }
        return string.Empty;
    }

    private static string BuildCnameHostname(string host, string domain)
    {
        host = host.Trim().TrimEnd('.');
        domain = domain.Trim().TrimEnd('.');
        if (string.IsNullOrWhiteSpace(host))
        {
            return string.IsNullOrWhiteSpace(domain) ? string.Empty : domain;
        }
        if (host == "@")
        {
            return domain;
        }
        if (string.IsNullOrWhiteSpace(domain))
        {
            return host;
        }
        if (string.Equals(host, domain, StringComparison.OrdinalIgnoreCase))
        {
            return domain;
        }
        var suffix = "." + domain;
        if (host.EndsWith(suffix, StringComparison.OrdinalIgnoreCase))
        {
            return host;
        }
        return host + suffix;
    }

    private async Task<string?> GenerateUniqueForwardHostnameAsync(string domain)
    {
        domain = DomainHelper.NormalizeDomainInput(domain);
        for (var i = 0; i < 5; i++)
        {
            var token = DomainHelper.GenerateToken(8);
            var full = BuildCnameHostname(token, domain);
            if (string.IsNullOrWhiteSpace(full))
            {
                continue;
            }

            var siteCount = await _db.Queryable<Site>().Where(s => s.CnameHostname == full).CountAsync();
            if (siteCount != 0)
            {
                continue;
            }
            var forwardCount = await _db.Queryable<Stream>().Where(s => s.CnameHostname == full).CountAsync();
            if (forwardCount == 0)
            {
                return token;
            }
        }
        return null;
    }

    private static string? ReadJsonString(JsonElement element, string key)
    {
        if (element.ValueKind != JsonValueKind.Object)
        {
            return null;
        }

        if (!element.TryGetProperty(key, out var value))
        {
            return null;
        }

        return value.ValueKind == JsonValueKind.String ? value.GetString() : value.ToString();
    }

    private static int ReadJsonInt(JsonElement element, string key, int fallback)
    {
        if (element.ValueKind != JsonValueKind.Object)
        {
            return fallback;
        }

        if (!element.TryGetProperty(key, out var value))
        {
            return fallback;
        }

        return value.ValueKind switch
        {
            JsonValueKind.Number when value.TryGetInt32(out var parsed) => parsed,
            JsonValueKind.String when int.TryParse(value.GetString(), out var parsed) => parsed,
            _ => fallback
        };
    }

    private static bool ReadJsonBool(JsonElement element, string key, bool fallback)
    {
        if (element.ValueKind != JsonValueKind.Object)
        {
            return fallback;
        }

        if (!element.TryGetProperty(key, out var value))
        {
            return fallback;
        }

        return value.ValueKind switch
        {
            JsonValueKind.True => true,
            JsonValueKind.False => false,
            JsonValueKind.Number => value.TryGetInt32(out var parsed) && parsed != 0,
            JsonValueKind.String => ParseBool(value.GetString(), fallback),
            _ => fallback
        };
    }

    private static bool ParseBool(string? raw, bool fallback = false)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            return fallback;
        }

        var normalized = raw.Trim().ToLowerInvariant();
        return normalized is "1" or "true" or "yes" or "on";
    }

    private static bool ParseBool(JsonElement value, bool fallback)
    {
        return value.ValueKind switch
        {
            JsonValueKind.True => true,
            JsonValueKind.False => false,
            JsonValueKind.Number => value.TryGetInt32(out var parsed) && parsed != 0,
            JsonValueKind.String => ParseBool(value.GetString(), fallback),
            _ => fallback
        };
    }

    private sealed record OriginUpdate(string? BalanceWay, bool? ProxyProtocol, string? Backend, string? BackendPort);

    private sealed record ForwardOrigin(string Address, int Weight, bool Enable);

    private sealed class ForwardBatchParseResult
    {
        public bool Success { get; private init; }
        public IReadOnlyList<string> ListenPorts { get; private init; } = Array.Empty<string>();
        public IReadOnlyList<ForwardOrigin> Origins { get; private init; } = Array.Empty<ForwardOrigin>();
        public string ErrorKey { get; private init; } = string.Empty;

        public static ForwardBatchParseResult Ok(IReadOnlyList<string> listenPorts, IReadOnlyList<ForwardOrigin> origins)
        {
            return new ForwardBatchParseResult { Success = true, ListenPorts = listenPorts, Origins = origins };
        }

        public static ForwardBatchParseResult Fail(string errorKey)
        {
            return new ForwardBatchParseResult { Success = false, ErrorKey = errorKey };
        }
    }
}
