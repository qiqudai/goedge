using Task = System.Threading.Tasks.Task;
using Cnn.Common.Contracts;
using Cnn.Common.Contracts.Admin;
using Cnn.Api.Services.Common;
using Cnn.Domain.Entities;
using SqlSugar;

namespace Cnn.Api.Services.Admin;

public sealed class CnameDomainService : ICnameDomainService
{
    private const string CnameTable = "cname_domains";

    private readonly ISqlSugarClient _db;

    public CnameDomainService(ISqlSugarClient db)
    {
        _db = db;
    }

    public async Task<ServiceResult<CnameDomainListResult>> ListAsync(CancellationToken cancellationToken)
    {
        await EnsureCnameTableAsync();

        var list = await _db.Queryable<CnameDomains>()
            .OrderBy(d => d.Id, OrderByType.Desc)
            .ToListAsync();

        var items = list.Select(item => new CnameDomainItem
        {
            Id = item.Id,
            Domain = item.Domain,
            DnsProviderId = item.DnsProviderId,
            Note = item.Note,
            CreatedAt = item.CreateAt,
            UpdatedAt = item.UpdateAt
        }).ToList();

        return ServiceResult<CnameDomainListResult>.Ok(new CnameDomainListResult(items));
    }

    public async Task<ServiceResult<bool>> CreateAsync(CnameDomainUpsertRequest request, CancellationToken cancellationToken)
    {
        await EnsureCnameTableAsync();

        var domain = DomainHelper.NormalizeDomainInput(request.Domain);
        if (string.IsNullOrWhiteSpace(domain) || !DomainHelper.IsValidDomain(domain))
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam, "invalid_domain");
        }

        var dnsProviderId = request.DnsProviderId.GetValueOrDefault();
        if (dnsProviderId == 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.MissingParam, "dns_provider_required");
        }

        var providerExists = await _db.Queryable<Dnsapi>().Where(p => p.Id == dnsProviderId).AnyAsync();
        if (!providerExists)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.NotFound, "dns_provider_not_found");
        }

        var existed = await _db.Queryable<CnameDomains>().Where(d => d.Domain == domain).AnyAsync();
        if (existed)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.AlreadyExists, "cname_domain_exists");
        }

        var now = DateTime.Now;
        var item = new CnameDomains
        {
            Domain = domain,
            DnsProviderId = (int)dnsProviderId,
            Note = request.Note?.Trim(),
            CreateAt = now,
            UpdateAt = now
        };

        await _db.Insertable(item).ExecuteCommandAsync();
        return ServiceResult<bool>.Ok(true);
    }

    public async Task<ServiceResult<bool>> UpdateAsync(long id, CnameDomainUpsertRequest request, CancellationToken cancellationToken)
    {
        await EnsureCnameTableAsync();

        if (id <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam);
        }

        var existing = await _db.Queryable<CnameDomains>().Where(d => d.Id == id).FirstAsync();
        if (existing == null)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.NotFound);
        }

        var domain = DomainHelper.NormalizeDomainInput(request.Domain);
        if (string.IsNullOrWhiteSpace(domain) || !DomainHelper.IsValidDomain(domain))
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam, "invalid_domain");
        }

        var dnsProviderId = request.DnsProviderId.GetValueOrDefault();
        if (dnsProviderId == 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.MissingParam, "dns_provider_required");
        }

        var providerExists = await _db.Queryable<Dnsapi>().Where(p => p.Id == dnsProviderId).AnyAsync();
        if (!providerExists)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.NotFound, "dns_provider_not_found");
        }

        if (!string.Equals(existing.Domain, domain, StringComparison.OrdinalIgnoreCase))
        {
            var existed = await _db.Queryable<CnameDomains>().Where(d => d.Domain == domain).AnyAsync();
            if (existed)
            {
                return ServiceResult<bool>.Fail(ErrorCodes.AlreadyExists, "cname_domain_exists");
            }
        }

        var now = DateTime.Now;
        var note = request.Note?.Trim();
        await _db.Updateable<CnameDomains>()
            .SetColumns(d => new CnameDomains
            {
                Domain = domain,
                DnsProviderId = (int)dnsProviderId,
                Note = note,
                UpdateAt = now
            })
            .Where(d => d.Id == id)
            .ExecuteCommandAsync();

        return ServiceResult<bool>.Ok(true);
    }

    public async Task<ServiceResult<bool>> DeleteAsync(long id, CancellationToken cancellationToken)
    {
        await EnsureCnameTableAsync();

        if (id <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam);
        }

        var existing = await _db.Queryable<CnameDomains>().Where(d => d.Id == id).FirstAsync();
        if (existing == null)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.NotFound);
        }

        var (inUse, errorKey) = await IsDomainInUseAsync(existing.Domain);
        if (errorKey != null)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InternalError, errorKey);
        }
        if (inUse)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InUse, "cname_domain_in_use");
        }

        await _db.Deleteable<CnameDomains>().Where(d => d.Id == id).ExecuteCommandAsync();
        return ServiceResult<bool>.Ok(true);
    }

    private async Task EnsureCnameTableAsync()
    {
        if (!_db.DbMaintenance.IsAnyTable(CnameTable))
        {
            const string sql = """
CREATE TABLE IF NOT EXISTS cname_domains (
  id INT(11) NOT NULL AUTO_INCREMENT,
  domain VARCHAR(255) NOT NULL,
  dns_provider_id BIGINT NOT NULL DEFAULT 0,
  note VARCHAR(255) DEFAULT '',
  create_at DATETIME DEFAULT NULL,
  update_at DATETIME DEFAULT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY idx_cname_domains_domain (domain)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
""";
            await _db.Ado.ExecuteCommandAsync(sql);
        }

        if (_db.DbMaintenance.IsAnyColumn(CnameTable, "dns_provider_id"))
        {
            return;
        }

        await Task.Run(() =>
        {
            _db.DbMaintenance.AddColumn(CnameTable, new DbColumnInfo
            {
                DbColumnName = "dns_provider_id",
                DataType = "bigint",
                IsNullable = false,
                DefaultValue = "0"
            });
        });
    }

    private async Task<(bool InUse, string? ErrorKey)> IsDomainInUseAsync(string? domain)
    {
        if (string.IsNullOrWhiteSpace(domain))
        {
            return (false, null);
        }

        var checks = new (string Table, string Column)[]
        {
            ("site", "cname_domain"),
            ("stream", "cname_domain"),
            ("node_group", "cname_domain"),
            ("package", "cname_domain"),
            ("user_package", "cname_domain"),
            ("plan", "cname_domain")
        };

        foreach (var (table, column) in checks)
        {
            if (!_db.DbMaintenance.IsAnyTable(table))
            {
                continue;
            }
            if (!_db.DbMaintenance.IsAnyColumn(table, column))
            {
                continue;
            }

            try
            {
                var sql = $"select count(1) from `{table}` where `{column}` = @domain limit 1";
                var rows = await _db.Ado.SqlQueryAsync<int>(sql, new { domain });
                if (rows.Count > 0 && rows[0] > 0)
                {
                    return (true, null);
                }
            }
            catch
            {
                return (false, "cname_domain_check_failed");
            }
        }

        return (false, null);
    }
}
