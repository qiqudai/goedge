using System.Text.Json;
using Cnn.Common.Contracts.Agent;
using Cnn.Common.Contracts.Admin;
using Cnn.Api.Services.Agent;
using Cnn.Api.Services.Common;
using Cnn.Domain.Entities;
using Certes;
using Certes.Acme;
using Certes.Acme.Resource;
using Microsoft.Extensions.Configuration;
using Microsoft.Extensions.Logging;
using SqlSugar;
using TaskEntity = Cnn.Domain.Entities.Task;
using Task = System.Threading.Tasks.Task;

namespace Cnn.Api.Services.Admin;

public interface ICertIssueProcessor
{
    Task ProcessAsync(CancellationToken cancellationToken);
}

public sealed class CertIssueProcessor : ICertIssueProcessor
{
    private const int MaxCertIssueAttempts = 3;
    private static readonly int[] RetryMinutes = { 5, 10, 20, 30, 60, 60, 60 };
    private static readonly JsonSerializerOptions JsonOptions = new(JsonSerializerDefaults.Web)
    {
        PropertyNameCaseInsensitive = true
    };

    private readonly ISqlSugarClient _db;
    private readonly IAgentConnectionManager _connections;
    private readonly INodeStatusService _nodeStatus;
    private readonly INodeRateLimitService _nodeRateLimit;
    private readonly ICryptoService _cryptoService;
    private readonly IConfigVersionService _configVersionService;
    private readonly IConfiguration _configuration;
    private readonly ISystemConfigService? _systemConfigService;
    private readonly ILogger<CertIssueProcessor> _logger;

    public CertIssueProcessor(
        ISqlSugarClient db,
        IAgentConnectionManager connections,
        INodeStatusService nodeStatus,
        INodeRateLimitService nodeRateLimit,
        ICryptoService cryptoService,
        IConfigVersionService configVersionService,
        IConfiguration configuration,
        ISystemConfigService? systemConfigService,
        ILogger<CertIssueProcessor> logger)
    {
        _db = db;
        _connections = connections;
        _nodeStatus = nodeStatus;
        _nodeRateLimit = nodeRateLimit;
        _cryptoService = cryptoService;
        _configVersionService = configVersionService;
        _configuration = configuration;
        _systemConfigService = systemConfigService;
        _logger = logger;
    }

    public async Task ProcessAsync(CancellationToken cancellationToken)
    {
        await DispatchPendingIssueTasksAsync(cancellationToken);
        await DispatchPendingDeployTasksAsync(cancellationToken);
        await ProcessLocalIssuesAsync(cancellationToken);
    }

    private async Task DispatchPendingDeployTasksAsync(CancellationToken cancellationToken)
    {
        var deployPolicy = await DeployCertCompletionPolicy.ResolvePolicyAsync(_systemConfigService, cancellationToken);
        var allowPartialFailures = DeployCertCompletionPolicy.IsAllowPartial(deployPolicy);

        var connected = _connections.GetConnectedNodeIds();
        if (connected.Count == 0)
        {
            return;
        }

        var availableNodeIds = connected
            .Select(static id => long.TryParse(id, out var parsed) ? parsed : 0)
            .Where(static id => id > 0)
            .Distinct()
            .OrderBy(static id => id)
            .ToArray();
        if (availableNodeIds.Length == 0)
        {
            return;
        }

        var now = DateTime.Now;
        var tasks = await _db.Queryable<TaskEntity>()
            .Where(t => t.Enable == true && t.Type == "deploy_cert" && (t.State == "waiting" || t.State == "retrying"))
            .OrderBy(t => t.Id, OrderByType.Asc)
            .Take(100)
            .ToListAsync();

        foreach (var task in tasks)
        {
            if (task.RetryAt.HasValue && task.RetryAt.Value > now && string.IsNullOrWhiteSpace(task.TargetsJson))
            {
                continue;
            }

            var targets = ParseTargets(task.TargetsJson);
            if (targets.Nodes.Count == 0)
            {
                targets = TaskTargets.Create(availableNodeIds);
            }

            var changed = false;
            var dispatched = false;

            foreach (var nodeId in availableNodeIds)
            {
                if (!targets.Nodes.TryGetValue(nodeId.ToString(), out var target) || target == null)
                {
                    continue;
                }

                var state = (target.State ?? string.Empty).Trim().ToLowerInvariant();
                if (state != TaskTargetState.Waiting)
                {
                    continue;
                }

                if (target.RetryAt > 0)
                {
                    var retryAt = DateTimeOffset.FromUnixTimeSeconds(target.RetryAt).LocalDateTime;
                    if (retryAt > now)
                    {
                        continue;
                    }
                }

                if (!_connections.TryGetSocket(nodeId.ToString(), out var socket) ||
                    socket.State != System.Net.WebSockets.WebSocketState.Open)
                {
                    continue;
                }

                var payload = new
                {
                    kind = "task_dispatch",
                    msg_id = $"task-{task.Id}-{nodeId}",
                    task = new
                    {
                        task_id = task.Id,
                        task_type = task.Type,
                        task_name = task.Name,
                        payload = task.Data ?? string.Empty
                    }
                };

                try
                {
                    await _connections.SendAsync(socket, payload, cancellationToken);
                    target.State = TaskTargetState.Running;
                    target.Tries = Math.Max(1, target.Tries + 1);
                    target.LastAt = DateTimeOffset.Now.ToUnixTimeSeconds();
                    target.RetryAt = 0;
                    changed = true;
                    dispatched = true;
                }
                catch (Exception ex)
                {
                    _logger.LogWarning(ex, "Dispatch deploy_cert task failed: {TaskId} node={NodeId}", task.Id, nodeId);
                }
            }

            targets.EnsureCounts();
            var allSettled = targets.Total > 0 && targets.Pending == 0;
            var nextState = task.State ?? "waiting";
            DateTime? nextRetryAt = null;
            var nextRet = task.Ret;
            if (targets.Success >= targets.Total && targets.Total > 0)
            {
                nextState = "success";
                nextRet = BuildDeployRetSummary("success", targets, allowPartialFailures, deployPolicy, task.Ret);
            }
            else if (!allowPartialFailures && targets.Fail > 0)
            {
                nextState = "fail";
                nextRet = BuildDeployRetSummary("fail", targets, allowPartialFailures, deployPolicy, task.Ret);
            }
            else if (allowPartialFailures && allSettled)
            {
                if (targets.Success > 0)
                {
                    nextState = "success";
                    nextRet = BuildDeployRetSummary("success", targets, allowPartialFailures, deployPolicy, task.Ret);
                }
                else if (targets.Fail > 0)
                {
                    nextState = "fail";
                    nextRet = BuildDeployRetSummary("fail", targets, allowPartialFailures, deployPolicy, task.Ret);
                }
            }
            else if (dispatched || targets.Nodes.Values.Any(t => string.Equals(t.State, TaskTargetState.Running, StringComparison.OrdinalIgnoreCase)))
            {
                nextState = "running";
            }
            else if (targets.Pending > 0)
            {
                nextState = "retrying";
                var minRetry = targets.Nodes.Values
                    .Where(t => string.Equals(t.State, TaskTargetState.Waiting, StringComparison.OrdinalIgnoreCase) && t.RetryAt > 0)
                    .Select(t => DateTimeOffset.FromUnixTimeSeconds(t.RetryAt).LocalDateTime)
                    .DefaultIfEmpty()
                    .Min();
                if (minRetry != default)
                {
                    nextRetryAt = minRetry;
                }
            }

            if (changed ||
                !string.Equals(task.State, nextState, StringComparison.OrdinalIgnoreCase) ||
                !string.Equals(task.Ret, nextRet, StringComparison.Ordinal) ||
                string.IsNullOrWhiteSpace(task.TargetsJson))
            {
                var startAt = task.StartAt ?? now;
                var endAt = nextState is "success" or "fail" ? now : task.EndAt;
                await _db.Updateable<TaskEntity>()
                    .SetColumns(t => new TaskEntity
                    {
                        State = nextState,
                        StartAt = startAt,
                        EndAt = endAt,
                        RetryAt = nextRetryAt,
                        Ret = nextRet,
                        TargetsJson = targets.Marshal()
                    })
                    .Where(t => t.Id == task.Id)
                    .ExecuteCommandAsync();
            }
        }
    }

    private async Task DispatchPendingIssueTasksAsync(CancellationToken cancellationToken)
    {
        var connected = _connections.GetConnectedNodeIds();
        if (connected.Count == 0)
        {
            return;
        }

        var now = DateTime.Now;
        var tasks = await _db.Queryable<TaskEntity>()
            .Where(t => t.Enable == true && t.Type == "issue_cert" && (t.State == "waiting" || t.State == "retrying"))
            .Where(t => t.RetryAt == null || t.RetryAt <= now)
            .OrderBy(t => t.Id, OrderByType.Asc)
            .Take(100)
            .ToListAsync();

        foreach (var task in tasks)
        {
            var meta = IssueCertTaskMeta.Parse(task.Res);
            if (meta?.Local == true)
            {
                continue;
            }

            var targetNodeId = meta?.TargetNodeId ?? 0;
            if (targetNodeId <= 0)
            {
                targetNodeId = await ResolveTargetNodeIdAsync(task);
                if (targetNodeId <= 0)
                {
                    await FailTaskAsync(task, "no available nodes for cert issue", now);
                    await _db.Updateable<Cert>()
                        .SetColumns(c => new Cert { State = "fail", Ret = "no available nodes for cert issue", UpdateAt = now })
                        .Where(c => c.IssueTaskId == task.Id)
                        .ExecuteCommandAsync();
                    continue;
                }

                var metaRaw = IssueCertTaskMeta.Build(targetNodeId, local: false);
                await _db.Updateable<TaskEntity>()
                    .SetColumns(t => new TaskEntity { Res = metaRaw })
                    .Where(t => t.Id == task.Id)
                    .ExecuteCommandAsync();
            }

            var nodeId = targetNodeId.ToString();
            if (!_connections.TryGetSocket(nodeId, out var socket) || socket.State != System.Net.WebSockets.WebSocketState.Open)
            {
                continue;
            }

            var payload = new
            {
                kind = "task_dispatch",
                msg_id = $"task-{task.Id}-{nodeId}",
                task = new
                {
                    task_id = task.Id,
                    task_type = task.Type,
                    task_name = task.Name,
                    payload = task.Data ?? string.Empty
                }
            };

            try
            {
                await _connections.SendAsync(socket, payload, cancellationToken);
                var startAt = task.StartAt ?? now;
                await _db.Updateable<TaskEntity>()
                    .SetColumns(t => new TaskEntity
                    {
                        State = "running",
                        StartAt = startAt
                    })
                    .Where(t => t.Id == task.Id)
                    .ExecuteCommandAsync();

                await _db.Updateable<Cert>()
                    .SetColumns(c => new Cert { State = "issuing", Ret = string.Empty, UpdateAt = now })
                    .Where(c => c.IssueTaskId == task.Id)
                    .ExecuteCommandAsync();
            }
            catch (Exception ex)
            {
                _logger.LogWarning(ex, "Dispatch issue_cert task failed: {TaskId}", task.Id);
                await MarkTaskRetryAsync(task, ex.Message, now);
            }
        }
    }

    private async Task ProcessLocalIssuesAsync(CancellationToken cancellationToken)
    {
        var now = DateTime.Now;
        var tasks = await _db.Queryable<TaskEntity>()
            .Where(t => t.Enable == true && t.Type == "issue_cert" && (t.State == "waiting" || t.State == "retrying"))
            .Where(t => t.RetryAt == null || t.RetryAt <= now)
            .OrderBy(t => t.Id, OrderByType.Asc)
            .Take(10)
            .ToListAsync();

        foreach (var task in tasks)
        {
            var meta = IssueCertTaskMeta.Parse(task.Res);
            if (meta?.Local != true)
            {
                continue;
            }

            await ProcessLocalIssueAsync(task, cancellationToken);
        }
    }

    private async Task ProcessLocalIssueAsync(TaskEntity task, CancellationToken cancellationToken)
    {
        var now = DateTime.Now;
        try
        {
            var cert = await _db.Queryable<Cert>().Where(c => c.IssueTaskId == task.Id).FirstAsync();
            if (cert == null)
            {
                await FailTaskAsync(task, "cert not found", now);
                return;
            }

            var domains = CertService.SplitCertDomains(cert.Domain);
            if (domains.Count == 0)
            {
                await FailIssueAsync(task, cert, "cert domain is empty", now);
                return;
            }

            var payload = ParseIssuePayload(task.Data);
            var email = payload?.Email?.Trim() ?? ResolveAcmeEmail();
            if (string.IsNullOrWhiteSpace(email))
            {
                await FailIssueAsync(task, cert, "acme_email is required", now);
                return;
            }

            var startAt = task.StartAt ?? now;
            await _db.Updateable<TaskEntity>()
                .SetColumns(t => new TaskEntity { State = "running", StartAt = startAt })
                .Where(t => t.Id == task.Id)
                .ExecuteCommandAsync();

            await _db.Updateable<Cert>()
                .SetColumns(c => new Cert { State = "issuing", Ret = string.Empty, UpdateAt = now })
                .Where(c => c.Id == cert.Id)
                .ExecuteCommandAsync();

            var ca = CertService.NormalizeCertType(payload?.Ca) ?? CertService.NormalizeCertType(cert.Type);
            if (string.IsNullOrWhiteSpace(ca))
            {
                ca = "letsencrypt";
            }

            var caDirUrl = payload?.CaDirUrl;
            if (string.IsNullOrWhiteSpace(caDirUrl))
            {
                caDirUrl = CertService.BuildCaDirUrl(ca);
            }

            var issued = await IssueDns01Async(cert, caDirUrl!, email, domains, cancellationToken);
            await UpdateIssuedCertAsync(cert, task, issued, now, cancellationToken);
        }
        catch (Exception ex)
        {
            await HandleIssueFailureAsync(task, ex.Message, now);
        }
    }

    private async Task<IssuedCertResult> IssueDns01Async(
        Cert cert,
        string caDirUrl,
        string email,
        IReadOnlyList<string> domains,
        CancellationToken cancellationToken)
    {
        var ctx = await CreateAcmeContextAsync(caDirUrl, email, cancellationToken);
        var order = await ctx.NewOrder(domains.ToList());
        var authz = await order.Authorizations();

        foreach (var auth in authz)
        {
            var resource = await auth.Resource();
            var domain = resource.Identifier.Value;
            var dnsChallenge = await auth.Dns();
            var recordValue = ctx.AccountKey.DnsTxt(dnsChallenge.Token);
            var info = BuildDnsChallengeInfo(domain, recordValue);

            await StoreDnsChallengeInfoAsync(cert.Id, info, cancellationToken);
            var timeout = cert.Dnsapi is > 0 ? TimeSpan.FromMinutes(30) : TimeSpan.FromMinutes(10);
            var interval = cert.Dnsapi is > 0 ? TimeSpan.FromSeconds(15) : TimeSpan.FromSeconds(10);

            var ok = await WaitForDnsRecordAsync(info.Fqdn ?? string.Empty, info.RecordValue ?? string.Empty, timeout, interval, cancellationToken);
            if (!ok)
            {
                throw new InvalidOperationException("DNS TXT record not found");
            }

            await dnsChallenge.Validate();
            await WaitAuthorizationValidAsync(auth, cancellationToken);
        }

        var key = KeyFactory.NewKey(KeyAlgorithm.RS256);
        var csr = new CsrInfo
        {
            CommonName = domains[0]
        };
        var certChain = await order.Generate(csr, key, null, 1);
        return new IssuedCertResult(certChain.ToPem(), key.ToPem());
    }

    private async Task<AcmeContext> CreateAcmeContextAsync(string caDirUrl, string email, CancellationToken cancellationToken)
    {
        var dir = _configuration["Acme:AccountPath"];
        if (string.IsNullOrWhiteSpace(dir))
        {
            dir = Path.Combine(AppContext.BaseDirectory, "acme");
        }

        System.IO.Directory.CreateDirectory(dir);
        var keyFile = Path.Combine(dir, $"{Sanitize(email)}-{Sanitize(caDirUrl)}.pem");
        IKey key;
        if (File.Exists(keyFile))
        {
            var pem = await File.ReadAllTextAsync(keyFile, cancellationToken);
            key = KeyFactory.FromPem(pem);
        }
        else
        {
            key = KeyFactory.NewKey(KeyAlgorithm.ES256);
            await File.WriteAllTextAsync(keyFile, key.ToPem(), cancellationToken);
        }

        var ctx = new AcmeContext(new Uri(caDirUrl), key);
        await ctx.NewAccount(email, true);
        return ctx;
    }

    private static async Task WaitAuthorizationValidAsync(IAuthorizationContext auth, CancellationToken cancellationToken)
    {
        for (var i = 0; i < 60; i++)
        {
            var res = await auth.Resource();
            if (res.Status == AuthorizationStatus.Valid)
            {
                return;
            }
            if (res.Status == AuthorizationStatus.Invalid)
            {
                throw new InvalidOperationException(res.Challenges?.FirstOrDefault()?.Error?.Detail ?? "authorization invalid");
            }

            await Task.Delay(TimeSpan.FromSeconds(5), cancellationToken);
        }

        throw new TimeoutException("authorization timeout");
    }

    private static async Task<bool> WaitForDnsRecordAsync(
        string fqdn,
        string recordValue,
        TimeSpan timeout,
        TimeSpan interval,
        CancellationToken cancellationToken)
    {
        var start = DateTime.UtcNow;
        while (DateTime.UtcNow - start <= timeout)
        {
            if (await LookupTxtRecordAsync(fqdn, recordValue, cancellationToken))
            {
                return true;
            }
            await Task.Delay(interval, cancellationToken);
        }
        return false;
    }

    private static async Task<bool> LookupTxtRecordAsync(string fqdn, string expected, CancellationToken cancellationToken)
    {
        fqdn = fqdn.Trim().TrimEnd('.');
        expected = expected.Trim();
        if (string.IsNullOrWhiteSpace(fqdn) || string.IsNullOrWhiteSpace(expected))
        {
            return false;
        }

        try
        {
            var records = await CertService.QueryTxtAsync(fqdn, cancellationToken);
            return records.Any(record => string.Equals(record.Trim(), expected, StringComparison.Ordinal));
        }
        catch
        {
            return false;
        }
    }

    private async Task StoreDnsChallengeInfoAsync(long certId, DnsChallengeInfoDto info, CancellationToken cancellationToken)
    {
        if (certId <= 0)
        {
            return;
        }

        var payload = JsonSerializer.Serialize(info, JsonOptions);
        await _db.Updateable<Cert>()
            .SetColumns(c => new Cert { State = "dns_pending", Ret = payload, UpdateAt = DateTime.Now })
            .Where(c => c.Id == certId)
            .ExecuteCommandAsync();
    }

    private async Task<long> ResolveTargetNodeIdAsync(TaskEntity task)
    {
        var nodes = await _db.Queryable<Node>()
            .Where(n => n.Pid == 0 && n.Enable == true)
            .OrderBy(n => n.Id, OrderByType.Asc)
            .ToListAsync();

        if (nodes.Count == 0)
        {
            return 0;
        }

        var candidates = nodes.Where(n => !_nodeRateLimit.IsLimited(n.Id)).ToList();
        if (candidates.Count == 0)
        {
            return 0;
        }

        var online = candidates
            .Where(n => _nodeStatus.IsOnline(n.Id, TimeSpan.FromSeconds(90)))
            .ToList();

        var pool = online.Count > 0 ? online : candidates;
        if (pool.Count == 0)
        {
            return 0;
        }

        var index = 0;
        var pid = task.Pid.GetValueOrDefault();
        if (pid > 0)
        {
            index = pid % pool.Count;
        }

        return pool[index].Id;
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

    private static string BuildDeployRetSummary(
        string nextState,
        TaskTargets targets,
        bool allowPartialFailures,
        string deployPolicy,
        string? fallback)
    {
        var failedNodes = targets.Nodes
            .Where(static kv => string.Equals(kv.Value?.State, TaskTargetState.FailedFinal, StringComparison.OrdinalIgnoreCase))
            .Select(static kv => kv.Key)
            .OrderBy(static x => x, StringComparer.Ordinal)
            .ToList();
        var failedNodesText = failedNodes.Count == 0 ? "-" : string.Join(",", failedNodes.Take(8));

        if (string.Equals(nextState, "success", StringComparison.OrdinalIgnoreCase) && targets.Fail > 0)
        {
            return $"deploy_cert partial success ({targets.Success}/{targets.Total}), failed={targets.Fail}, failed_nodes={failedNodesText}, policy={deployPolicy}";
        }

        if (string.Equals(nextState, "fail", StringComparison.OrdinalIgnoreCase) && targets.Fail > 0)
        {
            var reason = allowPartialFailures
                ? "deploy_cert failed: all target nodes failed"
                : "deploy_cert failed by strict policy";
            var suffix = string.IsNullOrWhiteSpace(fallback) ? string.Empty : $"; last={fallback}";
            return $"{reason} ({targets.Fail}/{targets.Total}), failed_nodes={failedNodesText}, policy={deployPolicy}{suffix}";
        }

        return fallback ?? string.Empty;
    }

    private static DnsChallengeInfoDto BuildDnsChallengeInfo(string domain, string recordValue)
    {
        var baseDomain = domain.Trim().TrimEnd('.');
        if (baseDomain.StartsWith("*.", StringComparison.Ordinal))
        {
            baseDomain = baseDomain[2..];
        }

        var fqdn = "_acme-challenge." + baseDomain;
        var zone = baseDomain;
        var recordName = ResolveRecordName(fqdn, zone);

        return new DnsChallengeInfoDto
        {
            Domain = domain,
            Fqdn = fqdn,
            RecordName = recordName,
            RecordValue = recordValue,
            RecordType = "TXT",
            Zone = zone
        };
    }

    private static string ResolveRecordName(string fqdn, string zone)
    {
        fqdn = fqdn.Trim().TrimEnd('.');
        zone = zone.Trim().TrimEnd('.');
        if (string.IsNullOrWhiteSpace(zone) || fqdn == zone)
        {
            return "@";
        }
        if (fqdn.EndsWith("." + zone, StringComparison.Ordinal))
        {
            var name = fqdn[..^(zone.Length + 1)];
            return string.IsNullOrWhiteSpace(name) ? "@" : name;
        }
        return fqdn;
    }

    private IssueCertTaskPayload? ParseIssuePayload(string? raw)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            return null;
        }

        try
        {
            return JsonSerializer.Deserialize<IssueCertTaskPayload>(raw, JsonOptions);
        }
        catch
        {
            return null;
        }
    }

    private string ResolveAcmeEmail()
    {
        return _configuration["Acme:Email"]?.Trim()
            ?? _configuration["App:AcmeEmail"]?.Trim()
            ?? string.Empty;
    }

    private async Task UpdateIssuedCertAsync(Cert cert, TaskEntity task, IssuedCertResult issued, DateTime now, CancellationToken cancellationToken)
    {
        if (!CertService.TryParseCert(issued.CertPem, out _, out var notBefore, out var notAfter))
        {
            throw new InvalidOperationException("cert parse failed");
        }

        var encryptedKey = _cryptoService.Encrypt(issued.KeyPem);
        if (string.IsNullOrWhiteSpace(encryptedKey))
        {
            encryptedKey = issued.KeyPem;
        }

        var nextAutoRenew = cert.AutoRenew ?? false;
        if (!string.Equals(cert.Type?.Trim(), "upload", StringComparison.OrdinalIgnoreCase) && nextAutoRenew != true)
        {
            nextAutoRenew = true;
        }

        var updates = new Cert
        {
            CertPem = issued.CertPem.Trim(),
            Key = encryptedKey,
            StartTime = notBefore,
            ExpireTime = notAfter,
            Enable = true,
            AutoRenew = nextAutoRenew,
            State = "ready",
            Ret = string.Empty,
            UpdateAt = now
        };

        await _db.Updateable(updates)
            .UpdateColumns(c => new
            {
                c.CertPem,
                c.Key,
                c.StartTime,
                c.ExpireTime,
                c.Enable,
                c.State,
                c.Ret,
                c.UpdateAt,
                c.AutoRenew
            })
            .Where(c => c.Id == cert.Id)
            .ExecuteCommandAsync();

        await _db.Updateable<TaskEntity>()
            .SetColumns(t => new TaskEntity { State = "success", Ret = string.Empty, EndAt = now })
            .Where(t => t.Id == task.Id)
            .ExecuteCommandAsync();

        await _configVersionService.BumpAsync("cert", new[] { (long)cert.Id }, cancellationToken);
    }

    private async Task FailIssueAsync(TaskEntity task, Cert cert, string message, DateTime now)
    {
        await _db.Updateable<Cert>()
            .SetColumns(c => new Cert { State = "fail", Ret = message, UpdateAt = now })
            .Where(c => c.Id == cert.Id)
            .ExecuteCommandAsync();

        await FailTaskAsync(task, message, now);
    }

    private async Task HandleIssueFailureAsync(TaskEntity task, string message, DateTime now)
    {
        var fatal = IsFatalIssueError(message);
        if (fatal)
        {
            await FailTaskAsync(task, message, now);
            await _db.Updateable<Cert>()
                .SetColumns(c => new Cert { State = "fail", Ret = message, UpdateAt = now })
                .Where(c => c.IssueTaskId == task.Id)
                .ExecuteCommandAsync();
            return;
        }

        await MarkTaskRetryAsync(task, message, now);
        await _db.Updateable<Cert>()
            .SetColumns(c => new Cert { State = "fail", Ret = message, UpdateAt = now })
            .Where(c => c.IssueTaskId == task.Id)
            .ExecuteCommandAsync();
    }

    private async Task MarkTaskRetryAsync(TaskEntity task, string message, DateTime now)
    {
        var nextErrTimes = (task.ErrTimes ?? 0) + 1;
        var delayMinutes = nextErrTimes - 1 < RetryMinutes.Length ? RetryMinutes[nextErrTimes - 1] : 60;
        var retryAt = now.AddMinutes(delayMinutes);

        if (nextErrTimes >= MaxCertIssueAttempts)
        {
            await FailTaskAsync(task, message, now);
            return;
        }

        await _db.Updateable<TaskEntity>()
            .SetColumns(t => new TaskEntity
            {
                State = "retrying",
                Ret = message,
                RetryAt = retryAt,
                ErrTimes = nextErrTimes
            })
            .Where(t => t.Id == task.Id)
            .ExecuteCommandAsync();
    }

    private async Task FailTaskAsync(TaskEntity task, string message, DateTime now)
    {
        await _db.Updateable<TaskEntity>()
            .SetColumns(t => new TaskEntity
            {
                State = "fail",
                Ret = message,
                EndAt = now
            })
            .Where(t => t.Id == task.Id)
            .ExecuteCommandAsync();
    }

    private static bool IsFatalIssueError(string message)
    {
        if (string.IsNullOrWhiteSpace(message))
        {
            return false;
        }

        var msg = message.Trim().ToLowerInvariant();
        return msg.Contains("cert not found")
            || msg.Contains("domain is empty")
            || msg.Contains("acme:error")
            || msg.Contains("dns problem")
            || msg.Contains("nxdomain")
            || msg.Contains("no such host")
            || msg.Contains("unauthorized")
            || msg.Contains("forbidden")
            || msg.Contains("connectex")
            || msg.Contains("connection refused")
            || msg.Contains("dial tcp")
            || msg.Contains("timeout");
    }

    private static string Sanitize(string input)
    {
        if (string.IsNullOrWhiteSpace(input))
        {
            return "default";
        }

        var chars = input.Select(ch => char.IsLetterOrDigit(ch) ? ch : '_').ToArray();
        return new string(chars);
    }

    private sealed record IssuedCertResult(string CertPem, string KeyPem);
}
