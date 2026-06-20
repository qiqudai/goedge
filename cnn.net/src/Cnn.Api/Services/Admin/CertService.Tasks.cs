using System.Text.Json;
using Cnn.Common.Contracts;
using Cnn.Common.Contracts.Agent;
using Cnn.Domain.Entities;
using SqlSugar;
using TaskEntity = Cnn.Domain.Entities.Task;
using Task = System.Threading.Tasks.Task;

namespace Cnn.Api.Services.Admin;

public sealed partial class CertService
{
    public async Task<ServiceResult<bool>> UpdateIssuedCertAsync(AgentIssuedCertRequest request, CancellationToken cancellationToken)
    {
        if (request == null || request.CertId <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam, "cert_invalid_id");
        }
        if (string.IsNullOrWhiteSpace(request.CertPem) || string.IsNullOrWhiteSpace(request.KeyPem))
        {
            return ServiceResult<bool>.Fail(ErrorCodes.MissingParam, "cert_upload_required");
        }

        if (!TryParseCert(request.CertPem, out _, out var notBefore, out var notAfter))
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam, "cert_parse_failed");
        }

        var cert = await _db.Queryable<Cert>().Where(c => c.Id == request.CertId).FirstAsync();
        if (cert == null)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.NotFound, "cert_not_found");
        }

        var encryptedKey = EncryptKey(request.KeyPem);
        var now = DateTime.Now;

        var update = new Cert
        {
            CertPem = request.CertPem.Trim(),
            Key = encryptedKey,
            StartTime = notBefore,
            ExpireTime = notAfter,
            Enable = true,
            State = "ready",
            Ret = string.Empty,
            UpdateAt = now,
            AutoRenew = cert.AutoRenew ?? false
        };

        if (!string.Equals(cert.Type?.Trim(), "upload", StringComparison.OrdinalIgnoreCase) && cert.AutoRenew != true)
        {
            update.AutoRenew = true;
        }

        if (request.IssueTaskId > 0)
        {
            update.IssueTaskId = request.IssueTaskId;
            update.TaskId = request.IssueTaskId;
        }
        else
        {
            update.IssueTaskId = null;
        }

        var updater = _db.Updateable(update).Where(c => c.Id == request.CertId);
        if (request.IssueTaskId > 0)
        {
            updater = updater.UpdateColumns(c => new
            {
                c.CertPem,
                c.Key,
                c.StartTime,
                c.ExpireTime,
                c.Enable,
                c.State,
                c.Ret,
                c.UpdateAt,
                c.AutoRenew,
                c.IssueTaskId,
                c.TaskId
            });
        }
        else
        {
            updater = updater.UpdateColumns(c => new
            {
                c.CertPem,
                c.Key,
                c.StartTime,
                c.ExpireTime,
                c.Enable,
                c.State,
                c.Ret,
                c.UpdateAt,
                c.AutoRenew,
                c.IssueTaskId
            });
        }

        var rows = await updater.ExecuteCommandAsync();

        if (rows <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.DbError, "cert_update_failed");
        }

        if (request.IssueTaskId > 0)
        {
            await _db.Updateable<TaskEntity>()
                .SetColumns(t => new TaskEntity { State = "success", Ret = string.Empty, EndAt = now })
                .Where(t => t.Id == request.IssueTaskId)
                .ExecuteCommandAsync();
        }

        await CreateDeployTaskAsync(cert, request, now, cancellationToken);
        await _configVersionService.BumpAsync("cert", new[] { (long)request.CertId }, cancellationToken);
        return ServiceResult<bool>.Ok(true);
    }

    private async Task CreateDeployTaskAsync(Cert cert, AgentIssuedCertRequest request, DateTime now, CancellationToken cancellationToken)
    {
        if (cert == null || request == null || request.CertId <= 0)
        {
            return;
        }

        var certPem = request.CertPem?.Trim();
        var keyPem = request.KeyPem?.Trim();
        var domains = SplitCertDomains(cert.Domain);
        if (string.IsNullOrWhiteSpace(certPem) || string.IsNullOrWhiteSpace(keyPem) || domains.Count == 0)
        {
            return;
        }

        var payload = new DeployCertTaskPayload
        {
            CertId = request.CertId,
            CertPem = certPem,
            KeyPem = keyPem,
            Domains = domains
        };

        var task = new TaskEntity
        {
            Name = $"Deploy Cert {request.CertId}",
            Type = AgentTaskTypes.DeployCert,
            State = "waiting",
            Enable = true,
            Data = JsonSerializer.Serialize(payload, JsonOptions),
            CreateAt = now
        };

        var taskId = await _db.Insertable(task).ExecuteReturnIdentityAsync();
        if (taskId <= 0)
        {
            return;
        }

        await _db.Updateable<Cert>()
            .SetColumns(c => new Cert
            {
                TaskId = taskId,
                UpdateAt = now
            })
            .Where(c => c.Id == request.CertId)
            .ExecuteCommandAsync();
    }

    private async Task CreateIssueTasksAsync(
        IReadOnlyList<long> certIds,
        long batchId,
        bool allowMissingEmail,
        CancellationToken cancellationToken)
    {
        if (certIds.Count == 0)
        {
            return;
        }

        var certs = await _db.Queryable<Cert>().Where(c => certIds.Contains(c.Id)).ToListAsync();
        if (certs.Count == 0)
        {
            return;
        }

        var email = ResolveAcmeEmail();
        var emailMissing = string.IsNullOrWhiteSpace(email);
        if (emailMissing && !allowMissingEmail)
        {
            return;
        }

        var now = DateTime.Now;
        var pid = batchId > int.MaxValue ? int.MaxValue : (int)batchId;

        foreach (var cert in certs)
        {
            var domains = SplitCertDomains(cert.Domain);
            var state = "waiting";
            var ret = string.Empty;
            var certState = "waiting";
            var certRet = string.Empty;
            IssueCertTaskPayload? payload = null;
            var isLocal = RequiresDnsChallenge(cert, domains);

            if (domains.Count == 0)
            {
                state = "fail";
                ret = "cert domain is empty";
                certState = "fail";
                certRet = ret;
            }
            else if (emailMissing)
            {
                state = "fail";
                ret = "acme_email is required";
                certState = "fail";
                certRet = ret;
            }
            else
            {
                payload = BuildIssuePayload(cert, email!, domains);
            }

            var task = new TaskEntity
            {
                Pid = pid,
                Name = $"Issue Cert {cert.Id}",
                Type = "issue_cert",
                State = state,
                Enable = true,
                Data = payload == null ? string.Empty : JsonSerializer.Serialize(payload, JsonOptions),
                Ret = ret,
                Res = isLocal ? IssueCertTaskMeta.Build(0, true) : string.Empty,
                CreateAt = now,
                StartAt = state == "waiting" ? null : now,
                EndAt = state == "fail" ? now : null
            };

            var taskId = await _db.Insertable(task).ExecuteReturnIdentityAsync();
            if (taskId <= 0)
            {
                continue;
            }

            await _db.Updateable<Cert>()
                .SetColumns(c => new Cert
                {
                    IssueTaskId = taskId,
                    TaskId = taskId,
                    State = certState,
                    Ret = certRet,
                    UpdateAt = now
                })
                .Where(c => c.Id == cert.Id)
                .ExecuteCommandAsync();
        }
    }

    private IssueCertTaskPayload BuildIssuePayload(Cert cert, string email, IReadOnlyList<string> domains)
    {
        var ca = NormalizeCertType(cert.Type);
        if (string.IsNullOrWhiteSpace(ca))
        {
            ca = "letsencrypt";
        }

        return new IssueCertTaskPayload
        {
            Ca = ca,
            CaDirUrl = BuildCaDirUrl(ca),
            Email = email,
            Items = new List<IssueCertItem>
            {
                new IssueCertItem
                {
                    CertId = cert.Id,
                    Domains = domains
                }
            }
        };
    }

    private static bool RequiresDnsChallenge(Cert cert, IReadOnlyList<string> domains)
    {
        if (cert.Dnsapi is > 0)
        {
            return true;
        }

        return HasWildcard(domains);
    }

    private async Task<bool> IsCertReferencedBySitesAsync(IReadOnlyList<long> certIds)
    {
        if (certIds.Count == 0)
        {
            return false;
        }

        if (_db.DbMaintenance.IsAnyTable("site") && _db.DbMaintenance.IsAnyColumn("site", "cert_id"))
        {
            var idList = string.Join(',', certIds);
            var sql = $"select count(1) from `site` where `cert_id` in ({idList})";
            try
            {
                var rows = await _db.Ado.SqlQueryAsync<int>(sql);
                if (rows.Count > 0 && rows[0] > 0)
                {
                    return true;
                }
            }
            catch
            {
                return true;
            }
        }

        var ids = new HashSet<long>(certIds);
        var settingsRows = await _db.Queryable<Config>()
            .Where(c => c.Type == SiteSettingsType && c.ScopeName == SiteSettingsScope)
            .Select(c => c.Value)
            .ToListAsync();

        foreach (var raw in settingsRows)
        {
            if (string.IsNullOrWhiteSpace(raw))
            {
                continue;
            }

            if (!TryExtractCertId(raw, out var certId))
            {
                continue;
            }

            if (ids.Contains(certId))
            {
                return true;
            }
        }

        return false;
    }

    private static bool TryExtractCertId(string raw, out long certId)
    {
        certId = 0;
        try
        {
            using var doc = JsonDocument.Parse(raw);
            if (doc.RootElement.ValueKind != JsonValueKind.Object)
            {
                return false;
            }

            if (!doc.RootElement.TryGetProperty("https", out var https))
            {
                return false;
            }

            if (TryReadCertId(https, "certificate_id", out certId))
            {
                return certId > 0;
            }

            if (TryReadCertId(https, "cert_id", out certId))
            {
                return certId > 0;
            }

            if (TryReadCertId(https, "certId", out certId))
            {
                return certId > 0;
            }
        }
        catch
        {
            return false;
        }

        return false;
    }

    private static bool TryReadCertId(JsonElement element, string propertyName, out long certId)
    {
        certId = 0;
        if (!element.TryGetProperty(propertyName, out var idElement))
        {
            return false;
        }

        return TryParseLong(idElement, out certId) && certId > 0;
    }

    private string ResolveAcmeEmail()
    {
        return _configuration["Acme:Email"]?.Trim()
            ?? _configuration["App:AcmeEmail"]?.Trim()
            ?? string.Empty;
    }
}



