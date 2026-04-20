using System.Security.Cryptography;
using System.Text.Json;
using System.Text.Json.Serialization;
using Cnn.Common.Contracts;
using Cnn.Common.Contracts.Admin;
using Cnn.Api.Services.Common;
using Cnn.Domain.Entities;
using SqlSugar;
using TaskEntity = Cnn.Domain.Entities.Task;
using AdminAgentPackageItemDto = Cnn.Common.Contracts.Admin.AgentPackageItemDto;

namespace Cnn.Api.Services.Admin;

public interface IAgentPackageService
{
    Task<ServiceResult<AgentPackageListResult>> ListAsync(CancellationToken cancellationToken);
    Task<ServiceResult<AdminAgentPackageItemDto>> UploadAsync(string? version, IFormFile? file, CancellationToken cancellationToken);
    Task<ServiceResult<bool>> UpdateGrayAsync(AgentPackageGrayRequest request, CancellationToken cancellationToken);
    Task<ServiceResult<bool>> SetStableAsync(AgentPackageStableRequest request, CancellationToken cancellationToken);
    Task<ServiceResult<AgentPackageNodeListResult>> ListNodesAsync(string? preferredVersion, CancellationToken cancellationToken);
    Task<ServiceResult<AgentPackageUpgradeResult>> UpgradeAsync(AgentPackageUpgradeRequest request, string? apiBaseUrl, CancellationToken cancellationToken);
    Task<ServiceResult<AgentPackageUpgradeStatusResult>> UpgradeStatusAsync(long taskId, CancellationToken cancellationToken);
    Task<ServiceResult<AgentPackageDownloadResult>> ResolveDownloadAsync(string? version, CancellationToken cancellationToken);
}

public sealed class AgentPackageService : IAgentPackageService
{
    private const string PackageType = "agent_package";
    private const string PackageScopeName = "global";
    private const int PackageScopeId = 0;

    private static readonly JsonSerializerOptions JsonOptions = new(JsonSerializerDefaults.Web)
    {
        PropertyNameCaseInsensitive = true
    };

    private readonly ISqlSugarClient _db;
    private readonly INodeStatusService _nodeStatus;
    private readonly INodeConfigService _nodeConfigService;

    public AgentPackageService(ISqlSugarClient db, INodeStatusService nodeStatus, INodeConfigService nodeConfigService)
    {
        _db = db;
        _nodeStatus = nodeStatus;
        _nodeConfigService = nodeConfigService;
    }

    public async Task<ServiceResult<AgentPackageListResult>> ListAsync(CancellationToken cancellationToken)
    {
        var items = await LoadPackagesAsync();
        var list = items.Select(MapPackageDto).ToList();
        return ServiceResult<AgentPackageListResult>.Ok(new AgentPackageListResult(list));
    }

    public async Task<ServiceResult<AdminAgentPackageItemDto>> UploadAsync(string? version, IFormFile? file, CancellationToken cancellationToken)
    {
        var normalizedVersion = version?.Trim() ?? string.Empty;
        if (string.IsNullOrWhiteSpace(normalizedVersion))
        {
            return ServiceResult<AdminAgentPackageItemDto>.Fail(ErrorCodes.MissingParam, "version_required");
        }

        if (!IsValidVersionToken(normalizedVersion))
        {
            return ServiceResult<AdminAgentPackageItemDto>.Fail(ErrorCodes.InvalidParam, "invalid_version_format");
        }

        if (file == null || file.Length == 0)
        {
            return ServiceResult<AdminAgentPackageItemDto>.Fail(ErrorCodes.MissingParam, "file_required");
        }

        var ext = NormalizePackageExt(file.FileName);
        if (string.IsNullOrWhiteSpace(ext))
        {
            return ServiceResult<AdminAgentPackageItemDto>.Fail(ErrorCodes.InvalidParam, "invalid_file_type");
        }

        var dir = EnsurePackageDir();
        if (string.IsNullOrWhiteSpace(dir))
        {
            return ServiceResult<AdminAgentPackageItemDto>.Fail(ErrorCodes.InternalError, "upload_save_failed");
        }

        var filename = $"agent_{normalizedVersion}{ext}";
        var targetPath = Path.Combine(dir, filename);
        var tmpPath = targetPath + ".tmp";

        try
        {
            await using var stream = new FileStream(tmpPath, FileMode.Create, FileAccess.Write, FileShare.None, 4096, true);
            await file.CopyToAsync(stream, cancellationToken);
            if (File.Exists(targetPath))
            {
                File.Delete(targetPath);
            }
            File.Move(tmpPath, targetPath);
        }
        catch
        {
            SafeDelete(tmpPath);
            return ServiceResult<AdminAgentPackageItemDto>.Fail(ErrorCodes.InternalError, "upload_save_failed");
        }

        var (size, sha256) = ComputeFileMeta(targetPath);
        var pkg = new AgentPackageRecord
        {
            Version = normalizedVersion,
            Filename = filename,
            Status = "normal",
            GrayPercent = 0,
            UploadTime = DateTime.Now,
            Size = size,
            Sha256 = sha256
        };

        var upserted = await UpsertPackageAsync(pkg);
        if (!upserted)
        {
            return ServiceResult<AdminAgentPackageItemDto>.Fail(ErrorCodes.DbError, "db_save_error");
        }

        return ServiceResult<AdminAgentPackageItemDto>.Ok(MapPackageDto(pkg));
    }

    public async Task<ServiceResult<bool>> UpdateGrayAsync(AgentPackageGrayRequest request, CancellationToken cancellationToken)
    {
        if (request == null)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam);
        }

        var version = request?.Version?.Trim() ?? string.Empty;
        if (string.IsNullOrWhiteSpace(version))
        {
            return ServiceResult<bool>.Fail(ErrorCodes.MissingParam, "version_required");
        }

        var pkg = await GetPackageAsync(version);
        if (pkg == null)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.NotFound, "version_not_found");
        }

        var percent = request?.Percent ?? 0;
        if (percent < 0) percent = 0;
        if (percent > 100) percent = 100;

        pkg.Status = "gray";
        pkg.GrayPercent = percent;
        if (pkg.UploadTime == null || pkg.UploadTime.Value == DateTime.MinValue)
        {
            pkg.UploadTime = DateTime.Now;
        }

        if (!await UpsertPackageAsync(pkg))
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InternalError, "update_failed");
        }

        return ServiceResult<bool>.Ok(true);
    }

    public async Task<ServiceResult<bool>> SetStableAsync(AgentPackageStableRequest request, CancellationToken cancellationToken)
    {
        if (request == null)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam);
        }

        var version = request?.Version?.Trim() ?? string.Empty;
        if (string.IsNullOrWhiteSpace(version))
        {
            return ServiceResult<bool>.Fail(ErrorCodes.MissingParam, "version_required");
        }

        var items = await LoadPackagesAsync();
        if (items.Count == 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.NotFound, "version_not_found");
        }

        var found = false;
        foreach (var item in items)
        {
            if (string.Equals(item.Version, version, StringComparison.OrdinalIgnoreCase))
            {
                item.Status = "stable";
                found = true;
            }
            else if (string.Equals(item.Status, "stable", StringComparison.OrdinalIgnoreCase))
            {
                item.Status = "history";
            }

            if (item.UploadTime == null || item.UploadTime.Value == DateTime.MinValue)
            {
                item.UploadTime = DateTime.Now;
            }
        }

        if (!found)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.NotFound, "version_not_found");
        }

        foreach (var item in items)
        {
            if (!await UpsertPackageAsync(item))
            {
                return ServiceResult<bool>.Fail(ErrorCodes.InternalError, "update_failed");
            }
        }

        return ServiceResult<bool>.Ok(true);
    }

    public async Task<ServiceResult<AgentPackageNodeListResult>> ListNodesAsync(string? preferredVersion, CancellationToken cancellationToken)
    {
        var latest = await ResolveLatestVersionAsync(preferredVersion);

        var nodes = await _db.Queryable<Node>()
            .Where(n => n.Pid == 0)
            .Select(n => new Node
            {
                Id = n.Id,
                Name = n.Name,
                Ip = n.Ip,
                RegionId = n.RegionId,
                Enable = n.Enable
            })
            .OrderBy(n => n.Id, OrderByType.Asc)
            .ToListAsync();

        var nodeIds = nodes.Select(n => (long)n.Id).ToList();
        var regionIds = nodes.Where(n => n.RegionId.HasValue && n.RegionId.Value > 0)
            .Select(n => n.RegionId!.Value)
            .Distinct()
            .ToList();

        var regionMap = new Dictionary<int, string>();
        if (regionIds.Count > 0)
        {
            var regions = await _db.Queryable<Region>().Where(r => regionIds.Contains(r.Id)).ToListAsync();
            foreach (var region in regions)
            {
                if (!string.IsNullOrWhiteSpace(region.Name))
                {
                    regionMap[region.Id] = region.Name!;
                }
            }
        }

        var groupNameMap = await LoadNodeGroupNamesAsync(nodeIds);
        var versionMap = await _nodeConfigService.GetMapAsync("agent_version", cancellationToken);

        var result = new List<AgentPackageNodeDto>(nodes.Count);
        foreach (var node in nodes)
        {
            var current = versionMap.TryGetValue(node.Id, out var value) ? value?.Trim() : string.Empty;
            var status = "idle";
            if (!string.IsNullOrWhiteSpace(latest) && !string.IsNullOrWhiteSpace(current))
            {
                status = CompareVersion(latest, current) > 0 ? "upgrade_available" : "up_to_date";
            }

            var regionName = node.RegionId.HasValue && regionMap.TryGetValue(node.RegionId.Value, out var name)
                ? name
                : string.Empty;

            groupNameMap.TryGetValue(node.Id, out var groupName);

            result.Add(new AgentPackageNodeDto
            {
                Id = node.Id,
                Name = node.Name,
                Ip = node.Ip,
                RegionId = node.RegionId,
                RegionName = regionName,
                GroupName = groupName,
                CurrentVersion = current,
                LatestVersion = latest,
                Status = status,
                Online = _nodeStatus.IsOnline(node.Id, TimeSpan.FromSeconds(30))
            });
        }

        return ServiceResult<AgentPackageNodeListResult>.Ok(new AgentPackageNodeListResult(result));
    }

    public async Task<ServiceResult<AgentPackageUpgradeResult>> UpgradeAsync(AgentPackageUpgradeRequest request, string? apiBaseUrl, CancellationToken cancellationToken)
    {
        if (request == null)
        {
            return ServiceResult<AgentPackageUpgradeResult>.Fail(ErrorCodes.InvalidParam);
        }

        var version = request?.Version?.Trim() ?? string.Empty;
        if (string.IsNullOrWhiteSpace(version))
        {
            return ServiceResult<AgentPackageUpgradeResult>.Fail(ErrorCodes.InvalidParam, "version_required");
        }

        var pkg = await GetPackageAsync(version);
        if (pkg == null)
        {
            return ServiceResult<AgentPackageUpgradeResult>.Fail(ErrorCodes.NotFound, "version_not_found");
        }

        var nodeSet = new HashSet<long>();
        if (request?.NodeIds != null)
        {
            foreach (var id in request.NodeIds)
            {
                if (id > 0)
                {
                    nodeSet.Add(id);
                }
            }
        }

        if (request?.GroupIds?.Count > 0)
        {
            var groupNodeIds = await _db.Queryable<Line>()
                .Where(l => request.GroupIds.Contains(l.NodeGroupId ?? 0))
                .Select(l => l.NodeId)
                .Distinct()
                .ToListAsync();
            foreach (var id in groupNodeIds)
            {
                if (id.HasValue && id.Value > 0)
                {
                    nodeSet.Add(id.Value);
                }
            }
        }

        if (request?.RegionIds?.Count > 0)
        {
            var regionNodeIds = await _db.Queryable<Node>()
                .Where(n => n.Pid == 0 && request.RegionIds.Contains(n.RegionId ?? 0))
                .Select(n => n.Id)
                .ToListAsync();
            foreach (var id in regionNodeIds)
            {
                if (id > 0)
                {
                    nodeSet.Add(id);
                }
            }
        }

        if (nodeSet.Count == 0)
        {
            return ServiceResult<AgentPackageUpgradeResult>.Fail(ErrorCodes.MissingParam, "node_ids_required");
        }

        var nodeIds = nodeSet.OrderBy(id => id).ToList();
        var downloadUrl = string.Empty;
        if (!string.IsNullOrWhiteSpace(apiBaseUrl))
        {
            var encoded = UrlEncodeLikeGo(pkg.Version ?? version);
            downloadUrl = $"{apiBaseUrl.TrimEnd('/')}/api/v1/agent/upgrade/package?version={encoded}";
        }

        var payload = new
        {
            version = pkg.Version,
            file_name = pkg.Filename,
            sha256 = pkg.Sha256,
            download_url = downloadUrl
        };
        var payloadRaw = JsonSerializer.Serialize(payload, JsonOptions);
        var targets = TaskTargets.Create(nodeIds).Marshal();

        var task = new TaskEntity
        {
            Type = "agent_upgrade",
            Name = "Agent Upgrade " + (pkg.Version ?? version),
            Data = payloadRaw,
            TargetsJson = targets,
            State = "waiting",
            Enable = true,
            CreateAt = DateTime.Now
        };

        var taskId = await _db.Insertable(task).ExecuteReturnBigIdentityAsync();
        return ServiceResult<AgentPackageUpgradeResult>.Ok(new AgentPackageUpgradeResult { TaskId = taskId });
    }

    public async Task<ServiceResult<AgentPackageUpgradeStatusResult>> UpgradeStatusAsync(long taskId, CancellationToken cancellationToken)
    {
        if (taskId <= 0)
        {
            return ServiceResult<AgentPackageUpgradeStatusResult>.Fail(ErrorCodes.MissingParam, "task_id_required");
        }

        var task = await _db.Queryable<TaskEntity>().Where(t => t.Id == taskId).FirstAsync();
        if (task == null)
        {
            return ServiceResult<AgentPackageUpgradeStatusResult>.Fail(ErrorCodes.NotFound, "task_not_found");
        }

        var targets = ParseTargets(task.TargetsJson);
        var nodes = new List<AgentPackageUpgradeNodeDto>();
        foreach (var pair in targets.Nodes)
        {
            var target = pair.Value ?? new TaskTarget();
            var progress = target.Progress ?? 0;
            var message = target.Message;
            if (progress == 0 && !string.IsNullOrWhiteSpace(target.Ret))
            {
                if (TryParseProgressPayload(target.Ret!, out var parsedProgress, out var parsedMessage))
                {
                    if (parsedProgress > 0)
                    {
                        progress = parsedProgress;
                    }
                    if (!string.IsNullOrWhiteSpace(parsedMessage))
                    {
                        message = parsedMessage;
                    }
                }
            }

            nodes.Add(new AgentPackageUpgradeNodeDto
            {
                NodeId = pair.Key,
                State = target.State,
                Progress = progress,
                Message = message,
                Ret = target.Ret,
                LastAt = target.LastAt
            });
        }

        var result = new AgentPackageUpgradeStatusResult
        {
            TaskId = task.Id,
            State = task.State,
            Nodes = nodes
        };

        return ServiceResult<AgentPackageUpgradeStatusResult>.Ok(result);
    }

    public async Task<ServiceResult<AgentPackageDownloadResult>> ResolveDownloadAsync(string? version, CancellationToken cancellationToken)
    {
        var normalized = version?.Trim() ?? string.Empty;
        if (string.IsNullOrWhiteSpace(normalized))
        {
            return ServiceResult<AgentPackageDownloadResult>.Fail(ErrorCodes.MissingParam, "version_required");
        }

        var pkg = await GetPackageAsync(normalized);
        if (pkg == null)
        {
            return ServiceResult<AgentPackageDownloadResult>.Fail(ErrorCodes.NotFound, "version_not_found");
        }

        var dir = ResolvePackageDir();
        if (string.IsNullOrWhiteSpace(dir))
        {
            return ServiceResult<AgentPackageDownloadResult>.Fail(ErrorCodes.NotFound, "file_not_found");
        }

        var path = Path.Combine(dir, pkg.Filename ?? string.Empty);
        if (!File.Exists(path))
        {
            return ServiceResult<AgentPackageDownloadResult>.Fail(ErrorCodes.NotFound, "file_not_found");
        }

        return ServiceResult<AgentPackageDownloadResult>.Ok(new AgentPackageDownloadResult
        {
            FilePath = path,
            FileName = pkg.Filename
        });
    }

    private async Task<List<AgentPackageRecord>> LoadPackagesAsync()
    {
        var items = await _db.Queryable<Config>()
            .Where(c => c.Type == PackageType && c.ScopeName == PackageScopeName && c.ScopeId == PackageScopeId)
            .ToListAsync();

        var list = new List<AgentPackageRecord>(items.Count);
        foreach (var item in items)
        {
            var pkg = new AgentPackageRecord();
            if (!string.IsNullOrWhiteSpace(item.Value))
            {
                try
                {
                    pkg = JsonSerializer.Deserialize<AgentPackageRecord>(item.Value!, JsonOptions) ?? new AgentPackageRecord();
                }
                catch
                {
                    pkg = new AgentPackageRecord();
                }
            }

            if (string.IsNullOrWhiteSpace(pkg.Version))
            {
                pkg.Version = item.Name?.Trim();
            }

            if (string.IsNullOrWhiteSpace(pkg.Version))
            {
                continue;
            }

            if (pkg.UploadTime == null || pkg.UploadTime.Value == DateTime.MinValue)
            {
                pkg.UploadTime = item.CreateAt;
            }

            list.Add(pkg);
        }

        list.Sort((a, b) =>
        {
            var at = a.UploadTime ?? DateTime.MinValue;
            var bt = b.UploadTime ?? DateTime.MinValue;
            var timeCompare = bt.CompareTo(at);
            if (timeCompare != 0)
            {
                return timeCompare;
            }
            return CompareVersion(b.Version, a.Version);
        });

        return list;
    }

    private async Task<AgentPackageRecord?> GetPackageAsync(string version)
    {
        var normalized = version?.Trim() ?? string.Empty;
        if (string.IsNullOrWhiteSpace(normalized))
        {
            return null;
        }

        var item = await _db.Queryable<Config>()
            .Where(c => c.Type == PackageType && c.ScopeName == PackageScopeName && c.ScopeId == PackageScopeId && c.Name == normalized)
            .FirstAsync();
        if (item == null)
        {
            return null;
        }

        var pkg = new AgentPackageRecord();
        if (!string.IsNullOrWhiteSpace(item.Value))
        {
            try
            {
                pkg = JsonSerializer.Deserialize<AgentPackageRecord>(item.Value!, JsonOptions) ?? new AgentPackageRecord();
            }
            catch
            {
                pkg = new AgentPackageRecord();
            }
        }

        if (string.IsNullOrWhiteSpace(pkg.Version))
        {
            pkg.Version = normalized;
        }
        if (pkg.UploadTime == null || pkg.UploadTime.Value == DateTime.MinValue)
        {
            pkg.UploadTime = item.CreateAt;
        }

        return pkg;
    }

    private async Task<bool> UpsertPackageAsync(AgentPackageRecord pkg)
    {
        if (string.IsNullOrWhiteSpace(pkg.Version))
        {
            return false;
        }

        var raw = JsonSerializer.Serialize(pkg, JsonOptions);
        var existing = await _db.Queryable<Config>()
            .Where(c => c.Type == PackageType && c.ScopeName == PackageScopeName && c.ScopeId == PackageScopeId && c.Name == pkg.Version)
            .FirstAsync();

        var now = DateTime.Now;
        if (existing != null)
        {
            await _db.Updateable<Config>()
                .SetColumns(c => new Config
                {
                    Value = raw,
                    UpdateAt = now,
                    Enable = true
                })
                .Where(c => c.Type == PackageType && c.ScopeName == PackageScopeName && c.ScopeId == PackageScopeId && c.Name == pkg.Version)
                .ExecuteCommandAsync();
            return true;
        }

        await _db.Insertable(new Config
        {
            Name = pkg.Version,
            Value = raw,
            Type = PackageType,
            ScopeName = PackageScopeName,
            ScopeId = PackageScopeId,
            Enable = true,
            CreateAt = now,
            UpdateAt = now
        }).ExecuteCommandAsync();
        return true;
    }

    private async Task<string> ResolveLatestVersionAsync(string? preferredVersion)
    {
        if (!string.IsNullOrWhiteSpace(preferredVersion))
        {
            return preferredVersion!.Trim();
        }

        var items = await LoadPackagesAsync();
        var stable = items.FirstOrDefault(p => string.Equals(p.Status, "stable", StringComparison.OrdinalIgnoreCase) && !string.IsNullOrWhiteSpace(p.Version));
        if (stable != null)
        {
            return stable.Version ?? string.Empty;
        }

        var best = string.Empty;
        foreach (var item in items)
        {
            if (string.IsNullOrWhiteSpace(item.Version))
            {
                continue;
            }

            if (string.IsNullOrWhiteSpace(best) || CompareVersion(item.Version, best) > 0)
            {
                best = item.Version!;
            }
        }

        return best;
    }

    private async Task<Dictionary<long, string>> LoadNodeGroupNamesAsync(IReadOnlyList<long> nodeIds)
    {
        var result = new Dictionary<long, string>();
        if (nodeIds == null || nodeIds.Count == 0)
        {
            return result;
        }

        var lines = await _db.Queryable<Line>()
            .Where(l => nodeIds.Contains(l.NodeId ?? 0))
            .Select(l => new { l.NodeId, l.NodeGroupId })
            .ToListAsync();

        var groupIds = new HashSet<int>();
        var nodeGroupMap = new Dictionary<long, HashSet<int>>();
        foreach (var line in lines)
        {
            if (!line.NodeId.HasValue || line.NodeId.Value <= 0 || !line.NodeGroupId.HasValue || line.NodeGroupId.Value <= 0)
            {
                continue;
            }

            if (!nodeGroupMap.TryGetValue(line.NodeId.Value, out var set))
            {
                set = new HashSet<int>();
                nodeGroupMap[line.NodeId.Value] = set;
            }

            if (set.Add(line.NodeGroupId.Value))
            {
                groupIds.Add(line.NodeGroupId.Value);
            }
        }

        if (groupIds.Count == 0)
        {
            return result;
        }

        var groups = await _db.Queryable<NodeGroup>()
            .Where(g => groupIds.Contains(g.Id))
            .Select(g => new { g.Id, g.Name })
            .ToListAsync();

        var nameLookup = groups.Where(g => !string.IsNullOrWhiteSpace(g.Name))
            .ToDictionary(g => g.Id, g => g.Name ?? string.Empty);

        foreach (var pair in nodeGroupMap)
        {
            var names = pair.Value.Select(id => nameLookup.TryGetValue(id, out var name) ? name : string.Empty)
                .Where(name => !string.IsNullOrWhiteSpace(name))
                .Distinct()
                .OrderBy(name => name)
                .ToList();

            result[pair.Key] = names.Count == 0 ? string.Empty : string.Join(", ", names);
        }

        return result;
    }

    private static AdminAgentPackageItemDto MapPackageDto(AgentPackageRecord pkg)
    {
        return new AdminAgentPackageItemDto
        {
            Version = pkg.Version,
            Status = pkg.Status,
            GrayPercent = pkg.GrayPercent,
            UploadTime = FormatTime(pkg.UploadTime),
            Filename = pkg.Filename,
            Size = pkg.Size,
            Sha256 = pkg.Sha256
        };
    }

    private static string? FormatTime(DateTime? time)
    {
        return time?.ToString("yyyy-MM-dd HH:mm:ss");
    }

    private static string? ResolvePackageDir()
    {
        var baseDir = AppContext.BaseDirectory;
        if (string.IsNullOrWhiteSpace(baseDir))
        {
            return null;
        }

        return Path.Combine(baseDir, "agent");
    }

    private static string? EnsurePackageDir()
    {
        var dir = ResolvePackageDir();
        if (string.IsNullOrWhiteSpace(dir))
        {
            return null;
        }

        Directory.CreateDirectory(dir);
        return dir;
    }

    private static void SafeDelete(string path)
    {
        try
        {
            if (File.Exists(path))
            {
                File.Delete(path);
            }
        }
        catch
        {
            // ignore
        }
    }

    private static string NormalizePackageExt(string? name)
    {
        var lower = (name ?? string.Empty).Trim().ToLowerInvariant();
        if (lower.EndsWith(".tar.gz", StringComparison.Ordinal))
        {
            return ".tar.gz";
        }

        if (lower.EndsWith(".zip", StringComparison.Ordinal))
        {
            return ".zip";
        }

        return string.Empty;
    }

    private static (long Size, string Sha256) ComputeFileMeta(string path)
    {
        if (string.IsNullOrWhiteSpace(path) || !File.Exists(path))
        {
            return (0, string.Empty);
        }

        var info = new FileInfo(path);
        try
        {
            using var stream = File.OpenRead(path);
            using var sha = SHA256.Create();
            var hash = sha.ComputeHash(stream);
            return (info.Length, Convert.ToHexString(hash).ToLowerInvariant());
        }
        catch
        {
            return (info.Length, string.Empty);
        }
    }

    private static bool IsValidVersionToken(string version)
    {
        if (string.IsNullOrWhiteSpace(version))
        {
            return false;
        }

        foreach (var ch in version)
        {
            if ((ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '.' || ch == '-' || ch == '_')
            {
                continue;
            }
            return false;
        }

        return true;
    }

    private static int CompareVersion(string? a, string? b)
    {
        var left = ParseVersionSegments(a);
        var right = ParseVersionSegments(b);
        if (left.Count == 0 || right.Count == 0)
        {
            return 0;
        }

        var max = Math.Max(left.Count, right.Count);
        for (var i = 0; i < max; i++)
        {
            var lv = i < left.Count ? left[i] : 0;
            var rv = i < right.Count ? right[i] : 0;
            if (lv > rv)
            {
                return 1;
            }
            if (lv < rv)
            {
                return -1;
            }
        }

        return 0;
    }

    private static List<int> ParseVersionSegments(string? raw)
    {
        var value = raw?.Trim().TrimStart('v', 'V') ?? string.Empty;
        if (string.IsNullOrWhiteSpace(value))
        {
            return new List<int>();
        }

        var parts = value.Split('.', StringSplitOptions.RemoveEmptyEntries);
        var list = new List<int>(parts.Length);
        foreach (var part in parts)
        {
            var digits = LeadingDigits(part.Trim());
            if (string.IsNullOrWhiteSpace(digits))
            {
                break;
            }

            if (!int.TryParse(digits, out var number))
            {
                return new List<int>();
            }

            list.Add(number);
        }

        return list;
    }

    private static string LeadingDigits(string raw)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            return string.Empty;
        }

        for (var i = 0; i < raw.Length; i++)
        {
            var ch = raw[i];
            if (ch < '0' || ch > '9')
            {
                return raw.Substring(0, i);
            }
        }

        return raw;
    }

    private static string UrlEncodeLikeGo(string value)
    {
        return value.Replace(" ", "%20").Replace("+", "%2B");
    }

    private static TaskTargets ParseTargets(string? raw)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            return new TaskTargets();
        }

        try
        {
            var targets = JsonSerializer.Deserialize<TaskTargets>(raw, JsonOptions);
            return targets ?? new TaskTargets();
        }
        catch
        {
            return new TaskTargets();
        }
    }

    private static bool TryParseProgressPayload(string raw, out int progress, out string message)
    {
        progress = 0;
        message = string.Empty;
        if (string.IsNullOrWhiteSpace(raw))
        {
            return false;
        }

        try
        {
            var payload = JsonSerializer.Deserialize<TaskProgressPayload>(raw, JsonOptions);
            if (payload == null)
            {
                return false;
            }
            progress = payload.Progress;
            message = payload.Message ?? string.Empty;
            return true;
        }
        catch
        {
            return false;
        }
    }

    private sealed class TaskProgressPayload
    {
        [JsonPropertyName("progress")]
        public int Progress { get; set; }

        [JsonPropertyName("message")]
        public string? Message { get; set; }
    }

    private sealed class AgentPackageRecord
    {
        [JsonPropertyName("version")]
        public string? Version { get; set; }

        [JsonPropertyName("filename")]
        public string? Filename { get; set; }

        [JsonPropertyName("status")]
        public string? Status { get; set; }

        [JsonPropertyName("gray_percent")]
        public int GrayPercent { get; set; }

        [JsonPropertyName("upload_time")]
        public DateTime? UploadTime { get; set; }

        [JsonPropertyName("size")]
        public long Size { get; set; }

        [JsonPropertyName("sha256")]
        public string? Sha256 { get; set; }
    }
}
