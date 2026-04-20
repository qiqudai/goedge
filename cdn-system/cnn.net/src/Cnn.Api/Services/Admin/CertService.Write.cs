using Cnn.Api.Services.Deletion;
using Cnn.Api.Services.Tasks.Workflow;
using Cnn.Common.Contracts;
using Cnn.Common.Contracts.Admin;
using Cnn.Domain.Entities;
using SqlSugar;
using Task = System.Threading.Tasks.Task;

namespace Cnn.Api.Services.Admin;

public sealed partial class CertService
{
    public async Task<ServiceResult<CertItemDto>> CreateAsync(
        CertCreateRequest request,
        long? userId,
        bool isAdmin,
        CancellationToken cancellationToken)
    {
        if (request == null)
        {
            return ServiceResult<CertItemDto>.Fail(ErrorCodes.InvalidParam, "invalid_param");
        }

        var type = NormalizeCertType(request.Type);
        if (string.IsNullOrWhiteSpace(type))
        {
            type = "upload";
        }

        var uid = request.UserId;
        if (!isAdmin || uid <= 0)
        {
            uid = userId ?? 0;
        }
        if (uid <= 0)
        {
            return ServiceResult<CertItemDto>.Fail(ErrorCodes.InvalidParam, "user_id_required");
        }

        var now = DateTime.Now;
        var certEntity = new Cert
        {
            Uid = (int)uid,
            Name = request.Name?.Trim(),
            Des = request.Description,
            Type = type,
            Domain = request.Domain?.Trim(),
            Dnsapi = request.DnsApi > 0 ? request.DnsApi : null,
            AutoRenew = request.AutoRenew,
            Enable = true,
            CreateAt = now,
            UpdateAt = now
        };

        if (string.Equals(type, "upload", StringComparison.OrdinalIgnoreCase))
        {
            if (string.IsNullOrWhiteSpace(request.CertPem) || string.IsNullOrWhiteSpace(request.KeyPem))
            {
                return ServiceResult<CertItemDto>.Fail(ErrorCodes.MissingParam, "cert_upload_required");
            }

            if (!TryParseCert(request.CertPem, out var parsedDomains, out var notBefore, out var notAfter))
            {
                return ServiceResult<CertItemDto>.Fail(ErrorCodes.InvalidParam, "cert_parse_failed");
            }

            certEntity.CertPem = request.CertPem?.Trim();
            certEntity.Key = EncryptKey(request.KeyPem);
            certEntity.StartTime = notBefore;
            certEntity.ExpireTime = notAfter;

            if (string.IsNullOrWhiteSpace(certEntity.Domain))
            {
                certEntity.Domain = string.Join(',', parsedDomains);
            }

            if (string.IsNullOrWhiteSpace(certEntity.Name) && parsedDomains.Count > 0)
            {
                certEntity.Name = DefaultCertName(parsedDomains[0]);
            }

            certEntity.State = "ready";
            certEntity.Ret = string.Empty;
        }
        else
        {
            var domains = NormalizeDomainsFromInput(request.Domain, out var domainError);
            if (domains.Count == 0)
            {
                return ServiceResult<CertItemDto>.Fail(ErrorCodes.MissingParam, string.IsNullOrWhiteSpace(domainError) ? "cert_domain_required" : domainError);
            }

            certEntity.Domain = string.Join(',', domains);
            certEntity.CertPem = string.Empty;
            certEntity.Key = string.Empty;
            certEntity.State = "waiting";
            certEntity.Ret = string.Empty;

            if (string.IsNullOrWhiteSpace(certEntity.Name))
            {
                certEntity.Name = DefaultCertName(domains[0]);
            }
        }

        var id = await _db.Insertable(certEntity).ExecuteReturnIdentityAsync();
        if (id <= 0)
        {
            return ServiceResult<CertItemDto>.Fail(ErrorCodes.DbError, "db_create_error");
        }

        certEntity.Id = id;
        var state = string.Empty;
        if (!string.Equals(type, "upload", StringComparison.OrdinalIgnoreCase))
        {
            await CreateIssueTasksAsync(new[] { (long)id }, DateTimeOffset.UtcNow.ToUnixTimeSeconds(), allowMissingEmail: true, cancellationToken);
            state = "waiting";
        }
        else
        {
            state = "ready";
        }
        await _configVersionService.BumpAsync("cert", new[] { (long)id }, cancellationToken);

        return ServiceResult<CertItemDto>.Ok(new CertItemDto
        {
            Id = certEntity.Id,
            UserId = certEntity.Uid ?? 0,
            Name = certEntity.Name,
            Description = certEntity.Des,
            Type = certEntity.Type,
            Domain = certEntity.Domain,
            DnsApi = certEntity.Dnsapi ?? 0,
            CertPem = certEntity.CertPem,
            KeyPem = DecryptKey(certEntity.Key),
            StartTime = certEntity.StartTime,
            ExpireTime = certEntity.ExpireTime,
            AutoRenew = certEntity.AutoRenew ?? false,
            CreateAt = certEntity.CreateAt,
            UpdateAt = certEntity.UpdateAt,
            Enable = certEntity.Enable ?? false,
            TaskId = certEntity.TaskId,
            State = state,
            Version = certEntity.Version
        });
    }

    public async Task<ServiceResult<bool>> UpdateAsync(
        long id,
        CertUpdateRequest request,
        long? userId,
        bool isAdmin,
        CancellationToken cancellationToken)
    {
        if (id <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam, "cert_invalid_id");
        }

        if (request == null)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam, "invalid_param");
        }

        var query = _db.Queryable<Cert>().Where(c => c.Id == id);
        if (!isAdmin)
        {
            var uid = userId ?? 0;
            if (uid <= 0)
            {
                return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam, "user_id_required");
            }
            query = query.Where(c => c.Uid == (int)uid);
        }

        var existing = await query.FirstAsync();
        if (existing == null)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.NotFound, "cert_not_found");
        }

        var type = string.IsNullOrWhiteSpace(request.Type)
            ? NormalizeCertType(existing.Type) ?? "upload"
            : NormalizeCertType(request.Type);
        if (string.IsNullOrWhiteSpace(type))
        {
            type = "upload";
        }

        var now = DateTime.Now;
        var update = new Cert
        {
            Name = request.Name?.Trim(),
            Des = request.Description,
            Type = type,
            Domain = request.Domain?.Trim(),
            Dnsapi = request.DnsApi > 0 ? request.DnsApi : null,
            AutoRenew = request.AutoRenew,
            State = existing.State,
            Ret = existing.Ret,
            UpdateAt = now
        };

        if (string.Equals(type, "upload", StringComparison.OrdinalIgnoreCase))
        {
            if (string.IsNullOrWhiteSpace(request.CertPem) || string.IsNullOrWhiteSpace(request.KeyPem))
            {
                return ServiceResult<bool>.Fail(ErrorCodes.MissingParam, "cert_upload_required");
            }

            if (!TryParseCert(request.CertPem, out var parsedDomains, out var notBefore, out var notAfter))
            {
                return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam, "cert_parse_failed");
            }

            update.CertPem = request.CertPem?.Trim();
            update.Key = EncryptKey(request.KeyPem);
            update.StartTime = notBefore;
            update.ExpireTime = notAfter;
            update.State = "ready";
            update.Ret = string.Empty;

            if (string.IsNullOrWhiteSpace(update.Domain))
            {
                update.Domain = string.Join(',', parsedDomains);
            }
        }
        else
        {
            var domains = NormalizeDomainsFromInput(request.Domain, out var domainError);
            if (domains.Count == 0)
            {
                return ServiceResult<bool>.Fail(ErrorCodes.MissingParam, string.IsNullOrWhiteSpace(domainError) ? "cert_domain_required" : domainError);
            }
            update.Domain = string.Join(',', domains);
        }

        var rows = await _db.Updateable(update)
            .UpdateColumns(c => new
            {
                c.Name,
                c.Des,
                c.Type,
                c.Domain,
                c.Dnsapi,
                c.CertPem,
                c.Key,
                c.StartTime,
                c.ExpireTime,
                c.AutoRenew,
                c.State,
                c.Ret,
                c.UpdateAt
            })
            .Where(c => c.Id == id)
            .ExecuteCommandAsync();

        if (rows <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.DbError, "cert_update_failed");
        }

        await _configVersionService.BumpAsync("cert", new[] { id }, cancellationToken);
        return ServiceResult<bool>.Ok(true);
    }

    public async Task<ServiceResult<bool>> DeleteAsync(
        long id,
        long? userId,
        bool isAdmin,
        CancellationToken cancellationToken)
    {
        if (id <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam, "cert_invalid_id");
        }

        var query = _db.Queryable<Cert>().Where(c => c.Id == id);
        if (!isAdmin)
        {
            var uid = userId ?? 0;
            if (uid <= 0)
            {
                return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam, "user_id_required");
            }
            query = query.Where(c => c.Uid == (int)uid);
        }

        var cert = await query.FirstAsync();
        if (cert == null)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.NotFound, "cert_not_found");
        }

        if (cert.Enable == true)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.StateConflict, "cert_in_use_disable_first");
        }

        var rows = await _db.Deleteable<Cert>().Where(c => c.Id == id).ExecuteCommandAsync();
        if (rows <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.DbError, "cert_delete_failed");
        }

        await _configVersionService.BumpAsync("cert", new[] { id }, cancellationToken);
        return ServiceResult<bool>.Ok(true);
    }

    public async Task<ServiceResult<CertBatchCreateResult>> BatchCreateAsync(
        CertBatchCreateRequest request,
        long? userId,
        bool isAdmin,
        CancellationToken cancellationToken)
    {
        if (request == null)
        {
            return ServiceResult<CertBatchCreateResult>.Fail(ErrorCodes.InvalidParam, "invalid_param");
        }

        var type = NormalizeCertType(request.Type);
        if (string.IsNullOrWhiteSpace(type))
        {
            return ServiceResult<CertBatchCreateResult>.Fail(ErrorCodes.MissingParam, "cert_type_required");
        }

        var uid = request.UserId;
        if (!isAdmin || uid <= 0)
        {
            uid = userId ?? 0;
        }
        if (uid <= 0)
        {
            return ServiceResult<CertBatchCreateResult>.Fail(ErrorCodes.InvalidParam, "user_id_required");
        }

        var domains = NormalizeDomainsFromJson(request.Domains, out var domainError);
        if (domains.Count == 0)
        {
            return ServiceResult<CertBatchCreateResult>.Fail(ErrorCodes.MissingParam, string.IsNullOrWhiteSpace(domainError) ? "cert_batch_domains_required" : domainError);
        }

        if (HasWildcard(domains) && request.DnsApi <= 0)
        {
            return ServiceResult<CertBatchCreateResult>.Fail(ErrorCodes.InvalidParam, "wildcard_requires_dnsapi");
        }

        var batchId = DateTimeOffset.UtcNow.ToUnixTimeSeconds();
        var now = DateTime.Now;
        var ids = new List<long>();

        var tran = await _db.Ado.UseTranAsync(async () =>
        {
            foreach (var domain in domains)
            {
                var cert = new Cert
                {
                    Uid = (int)uid,
                    Name = DefaultCertName(domain),
                    Type = type,
                    Domain = domain,
                    Dnsapi = request.DnsApi > 0 ? request.DnsApi : null,
                    AutoRenew = request.AutoRenew,
                    Enable = true,
                    CreateAt = now,
                    UpdateAt = now
                };

                var id = await _db.Insertable(cert).ExecuteReturnIdentityAsync();
                if (id <= 0)
                {
                    throw new InvalidOperationException("db_create_error");
                }

                ids.Add(id);
            }
        });

        if (!tran.IsSuccess)
        {
            var key = string.Equals(tran.ErrorMessage, "db_create_error", StringComparison.Ordinal) ? "db_create_error" : "cert_save_failed";
            return ServiceResult<CertBatchCreateResult>.Fail(ErrorCodes.DbError, key);
        }

        if (ids.Count > 0)
        {
            await CreateIssueTasksAsync(ids, batchId, allowMissingEmail: true, cancellationToken);
            await _configVersionService.BumpAsync("cert", ids, cancellationToken);
        }

        return ServiceResult<CertBatchCreateResult>.Ok(new CertBatchCreateResult(batchId.ToString(), ids.Count, ids));
    }

    public async Task<ServiceResult<CertWildcardResult>> WildcardCreateAsync(
        CertWildcardRequest request,
        long? userId,
        bool isAdmin,
        CancellationToken cancellationToken)
    {
        if (request == null)
        {
            return ServiceResult<CertWildcardResult>.Fail(ErrorCodes.InvalidParam, "invalid_param");
        }

        var domain = NormalizeDomain(request.Domain);
        if (string.IsNullOrWhiteSpace(domain) || !domain.StartsWith("*.", StringComparison.Ordinal))
        {
            return ServiceResult<CertWildcardResult>.Fail(ErrorCodes.MissingParam, "cert_wildcard_required");
        }

        if (IsIpDomain(domain))
        {
            return ServiceResult<CertWildcardResult>.Fail(ErrorCodes.InvalidParam, "invalid_domain");
        }

        var type = NormalizeCertType(request.Type);
        if (string.IsNullOrWhiteSpace(type))
        {
            return ServiceResult<CertWildcardResult>.Fail(ErrorCodes.MissingParam, "cert_type_required");
        }

        var uid = request.UserId;
        if (!isAdmin || uid <= 0)
        {
            uid = userId ?? 0;
        }
        if (uid <= 0)
        {
            return ServiceResult<CertWildcardResult>.Fail(ErrorCodes.InvalidParam, "user_id_required");
        }

        var now = DateTime.Now;
        var cert = new Cert
        {
            Uid = (int)uid,
            Name = DefaultCertName(domain),
            Type = type,
            Domain = domain,
            Dnsapi = request.DnsApi > 0 ? request.DnsApi : null,
            AutoRenew = request.AutoRenew,
            Enable = true,
            CreateAt = now,
            UpdateAt = now
        };

        var id = await _db.Insertable(cert).ExecuteReturnIdentityAsync();
        if (id <= 0)
        {
            return ServiceResult<CertWildcardResult>.Fail(ErrorCodes.DbError, "db_create_error");
        }

        await CreateIssueTasksAsync(new[] { (long)id }, DateTimeOffset.UtcNow.ToUnixTimeSeconds(), allowMissingEmail: true, cancellationToken);
        await _configVersionService.BumpAsync("cert", new[] { (long)id }, cancellationToken);
        return ServiceResult<CertWildcardResult>.Ok(new CertWildcardResult(id));
    }

    public async Task<ServiceResult<CertBatchActionResult>> BatchActionAsync(
        CertBatchActionRequest request,
        long? userId,
        bool isAdmin,
        CancellationToken cancellationToken)
    {
        if (request?.Ids == null || request.Ids.Count == 0)
        {
            return ServiceResult<CertBatchActionResult>.Fail(ErrorCodes.MissingParam, "ids_required");
        }

        var action = request.Action?.Trim().ToLowerInvariant();
        if (string.IsNullOrWhiteSpace(action))
        {
            return ServiceResult<CertBatchActionResult>.Fail(ErrorCodes.InvalidParam, "cert_unknown_action");
        }

        var ids = request.Ids.Where(id => id > 0).Distinct().ToList();
        if (ids.Count == 0)
        {
            return ServiceResult<CertBatchActionResult>.Fail(ErrorCodes.MissingParam, "ids_required");
        }

        if (!isAdmin)
        {
            var uid = userId ?? 0;
            if (uid <= 0)
            {
                return ServiceResult<CertBatchActionResult>.Fail(ErrorCodes.InvalidParam, "user_id_required");
            }

            ids = await _db.Queryable<Cert>()
                .Where(c => c.Uid == (int)uid && ids.Contains(c.Id))
                .Select(c => (long)c.Id)
                .ToListAsync();

            if (ids.Count == 0)
            {
                return ServiceResult<CertBatchActionResult>.Fail(ErrorCodes.NotFound, "cert_not_found");
            }
        }

        switch (action)
        {
            case "enable":
            {
                var taskResult = await _resourceActionRequestService.RequestAsync(
                    CertificateActionCommandFactory.CreateStatusChange(ids, true, false, userId, userId),
                    cancellationToken);
                if (!taskResult.Success)
                {
                    return ServiceResult<CertBatchActionResult>.Fail(taskResult.ErrorCode, taskResult.MessageKey);
                }

                return ServiceResult<CertBatchActionResult>.Ok(new CertBatchActionResult(taskResult.Data!.TaskId));
            }
            case "auto_renew_enable":
            {
                var rows = await _db.Updateable<Cert>()
                    .SetColumns(c => new Cert { AutoRenew = true })
                    .Where(c => ids.Contains(c.Id))
                    .ExecuteCommandAsync();
                if (rows <= 0)
                {
                    return ServiceResult<CertBatchActionResult>.Fail(ErrorCodes.DbError, "cert_action_failed");
                }
                break;
            }
            case "auto_renew_disable":
            {
                var rows = await _db.Updateable<Cert>()
                    .SetColumns(c => new Cert { AutoRenew = false })
                    .Where(c => ids.Contains(c.Id))
                    .ExecuteCommandAsync();
                if (rows <= 0)
                {
                    return ServiceResult<CertBatchActionResult>.Fail(ErrorCodes.DbError, "cert_action_failed");
                }
                break;
            }
            case "disable":
            case "force_disable":
            {
                foreach (var certId in ids)
                {
                    var usages = await CertificateUsageInspector.FindSiteUsagesAsync(_db, certId, cancellationToken);
                    if (usages.Count > 0)
                    {
                        return ServiceResult<CertBatchActionResult>.Fail(ErrorCodes.InUse, "cert_site_ref_disable_first");
                    }
                }

                var taskResult = await _resourceActionRequestService.RequestAsync(
                    CertificateActionCommandFactory.CreateStatusChange(ids, false, action == "force_disable", userId, userId),
                    cancellationToken);
                if (!taskResult.Success)
                {
                    return ServiceResult<CertBatchActionResult>.Fail(taskResult.ErrorCode, taskResult.MessageKey);
                }

                return ServiceResult<CertBatchActionResult>.Ok(new CertBatchActionResult(taskResult.Data!.TaskId));
            }
            case "delete":
            {
                var enabledCount = await _db.Queryable<Cert>()
                    .Where(c => ids.Contains(c.Id) && c.Enable == true)
                    .CountAsync();
                if (enabledCount > 0)
                {
                    return ServiceResult<CertBatchActionResult>.Fail(ErrorCodes.StateConflict, "cert_selected_enabled_disable_first");
                }

                var rows = await _db.Deleteable<Cert>().Where(c => ids.Contains(c.Id)).ExecuteCommandAsync();
                if (rows <= 0)
                {
                    return ServiceResult<CertBatchActionResult>.Fail(ErrorCodes.DbError, "cert_delete_failed");
                }
                break;
            }
            default:
                return ServiceResult<CertBatchActionResult>.Fail(ErrorCodes.InvalidParam, "cert_unknown_action");
        }

        await _configVersionService.BumpAsync("cert", ids, cancellationToken);
        return ServiceResult<CertBatchActionResult>.Ok(new CertBatchActionResult(0));
    }

    public async Task<ServiceResult<bool>> ReissueAsync(CertReissueRequest request, CancellationToken cancellationToken)
    {
        if (request?.Ids == null || request.Ids.Count == 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.MissingParam, "ids_required");
        }

        if (string.IsNullOrWhiteSpace(ResolveAcmeEmail()))
        {
            return ServiceResult<bool>.Fail(ErrorCodes.MissingParam, "acme_email_required");
        }

        var ids = request.Ids.Where(id => id > 0).Distinct().ToList();
        if (ids.Count == 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.MissingParam, "ids_required");
        }

        await CreateIssueTasksAsync(ids, DateTimeOffset.UtcNow.ToUnixTimeSeconds(), allowMissingEmail: false, cancellationToken);
        await _configVersionService.BumpAsync("cert", ids, cancellationToken);
        return ServiceResult<bool>.Ok(true);
    }
}

