using Cnn.Common.Contracts;
using DomainEntity = Cnn.Domain.Entities.Domain;
using DomainOriginEntity = Cnn.Domain.Entities.DomainOrigin;
using SqlSugar;

namespace Cnn.Api.Services.Common;

public interface IDomainService
{
    Task<ServiceResult<DomainListResult>> ListAdminAsync(DomainListQuery query, CancellationToken cancellationToken);
    Task<ServiceResult<DomainListResult>> ListUserAsync(DomainListQuery query, long userId, CancellationToken cancellationToken);
    Task<ServiceResult<DomainDto>> CreateAsync(long userId, CreateDomainRequest request, CancellationToken cancellationToken);
    Task<ServiceResult<DomainConfigDto>> GetConfigAsync(long userId, long domainId, CancellationToken cancellationToken);
}

public sealed class DomainService : IDomainService
{
    private const int DefaultPageSize = 20;
    private const int MaxPageSize = 200;
    private const string DomainsTable = "domains";
    private const string DomainOriginsTable = "domain_origins";

    private readonly ISqlSugarClient _db;

    public DomainService(ISqlSugarClient db)
    {
        _db = db;
    }

    public async Task<ServiceResult<DomainListResult>> ListAdminAsync(DomainListQuery query, CancellationToken cancellationToken)
    {
        await EnsureTablesAsync();

        query ??= new DomainListQuery();
        var (page, pageSize) = ResolvePaging(query.Page, query.PageSize);
        var keyword = query.Keyword?.Trim();

        var q = _db.Queryable<DomainEntity>();
        if (!string.IsNullOrWhiteSpace(keyword))
        {
            var lowered = keyword!.ToLowerInvariant();
            q = q.Where(d => SqlFunc.Contains(SqlFunc.ToLower(d.Name), lowered));
        }

        var total = await q.CountAsync();
        var domains = await q.OrderBy(d => d.Id, OrderByType.Desc)
            .Skip((page - 1) * pageSize)
            .Take(pageSize)
            .ToListAsync();

        var list = await MapDomainDtosAsync(domains);
        return ServiceResult<DomainListResult>.Ok(new DomainListResult(list, (int)total));
    }

    public async Task<ServiceResult<DomainListResult>> ListUserAsync(DomainListQuery query, long userId, CancellationToken cancellationToken)
    {
        if (userId <= 0)
        {
            return ServiceResult<DomainListResult>.Fail(ErrorCodes.AuthInvalid);
        }

        await EnsureTablesAsync();

        query ??= new DomainListQuery();
        var (page, pageSize) = ResolvePaging(query.Page, query.PageSize);
        var keyword = query.Keyword?.Trim();

        var q = _db.Queryable<DomainEntity>().Where(d => d.UserId == userId);
        if (!string.IsNullOrWhiteSpace(keyword))
        {
            var lowered = keyword!.ToLowerInvariant();
            q = q.Where(d => SqlFunc.Contains(SqlFunc.ToLower(d.Name), lowered));
        }

        var total = await q.CountAsync();
        var domains = await q.OrderBy(d => d.Id, OrderByType.Desc)
            .Skip((page - 1) * pageSize)
            .Take(pageSize)
            .ToListAsync();

        var list = await MapDomainDtosAsync(domains);
        return ServiceResult<DomainListResult>.Ok(new DomainListResult(list, (int)total));
    }

    public async Task<ServiceResult<DomainDto>> CreateAsync(long userId, CreateDomainRequest request, CancellationToken cancellationToken)
    {
        if (userId <= 0)
        {
            return ServiceResult<DomainDto>.Fail(ErrorCodes.AuthInvalid);
        }

        await EnsureTablesAsync();

        var name = request?.Name?.Trim() ?? string.Empty;
        if (string.IsNullOrWhiteSpace(name))
        {
            return ServiceResult<DomainDto>.Fail(ErrorCodes.MissingParam, "domain_name_required");
        }

        var existed = await _db.Queryable<DomainEntity>().Where(d => d.UserId == userId && d.Name == name).AnyAsync();
        if (existed)
        {
            return ServiceResult<DomainDto>.Fail(ErrorCodes.AlreadyExists, "domain_exists");
        }

        var now = DateTime.Now;
        var entity = new DomainEntity
        {
            UserId = userId,
            Name = name,
            Cname = name + ".cdn.node.com",
            Status = 1,
            CreatedAt = now,
            UpdatedAt = now
        };

        var origins = BuildOrigins(request?.Origins, now);

        var tran = await _db.Ado.UseTranAsync(async () =>
        {
            var newId = await _db.Insertable(entity).ExecuteReturnBigIdentityAsync();
            entity.Id = newId;
            if (origins.Count > 0)
            {
                foreach (var origin in origins)
                {
                    origin.DomainId = newId;
                }
                await _db.Insertable(origins).ExecuteCommandAsync();
            }
        });

        if (!tran.IsSuccess)
        {
            return ServiceResult<DomainDto>.Fail(ErrorCodes.DbError, "db_save_error");
        }

        entity.Origins = origins;
        var originDtos = origins.Select(MapOriginDto).ToList();
        var dto = MapDomainDto(entity, originDtos);
        return ServiceResult<DomainDto>.Ok(dto);
    }

    public async Task<ServiceResult<DomainConfigDto>> GetConfigAsync(long userId, long domainId, CancellationToken cancellationToken)
    {
        if (userId <= 0)
        {
            return ServiceResult<DomainConfigDto>.Fail(ErrorCodes.AuthInvalid);
        }

        await EnsureTablesAsync();

        var domain = await _db.Queryable<DomainEntity>()
            .Where(d => d.Id == domainId && d.UserId == userId)
            .FirstAsync();
        if (domain == null)
        {
            return ServiceResult<DomainConfigDto>.Fail(ErrorCodes.NotFound, "domain_not_found");
        }

        var origins = await LoadOriginsAsync(new List<long> { domain.Id });

        var config = new DomainConfigDto
        {
            Domain = domain.Name,
            Origins = origins,
            HttpsOn = true,
            CacheRules = new List<DomainCacheRuleDto>
            {
                new DomainCacheRuleDto { Ext = ".jpg", Ttl = 3600 }
            }
        };

        return ServiceResult<DomainConfigDto>.Ok(config);
    }

    private async Task EnsureTablesAsync()
    {
        if (!_db.DbMaintenance.IsAnyTable(DomainsTable))
        {
            const string sql = """
CREATE TABLE IF NOT EXISTS domains (
  id BIGINT NOT NULL AUTO_INCREMENT,
  user_id BIGINT NOT NULL DEFAULT 0,
  name VARCHAR(255) NOT NULL,
  cname VARCHAR(255) DEFAULT '',
  status INT NOT NULL DEFAULT 0,
  origins LONGTEXT DEFAULT NULL,
  created_at DATETIME DEFAULT NULL,
  updated_at DATETIME DEFAULT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY idx_user_domain (user_id, name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
""";
            await _db.Ado.ExecuteCommandAsync(sql);
        }

        if (!_db.DbMaintenance.IsAnyTable(DomainOriginsTable))
        {
            const string sql = """
CREATE TABLE IF NOT EXISTS domain_origins (
  id BIGINT NOT NULL AUTO_INCREMENT,
  domain_id BIGINT NOT NULL,
  addr VARCHAR(255) DEFAULT '',
  port INT NOT NULL DEFAULT 80,
  weight INT NOT NULL DEFAULT 1,
  protocol VARCHAR(20) DEFAULT 'http',
  created_at DATETIME DEFAULT NULL,
  updated_at DATETIME DEFAULT NULL,
  PRIMARY KEY (id),
  KEY idx_domain_id (domain_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
""";
            await _db.Ado.ExecuteCommandAsync(sql);
        }
    }

    private async Task<IReadOnlyList<DomainDto>> MapDomainDtosAsync(IReadOnlyList<DomainEntity> domains)
    {
        if (domains.Count == 0)
        {
            return Array.Empty<DomainDto>();
        }

        var ids = domains.Select(d => d.Id).ToList();
        var origins = await LoadOriginsAsync(ids);
        var originMap = origins.GroupBy(o => o.DomainId).ToDictionary(g => g.Key, g => g.ToList());

        return domains.Select(domain =>
        {
            originMap.TryGetValue(domain.Id, out var list);
            return MapDomainDto(domain, list);
        }).ToList();
    }

    private async Task<List<DomainOriginDto>> LoadOriginsAsync(IReadOnlyList<long> domainIds)
    {
        if (domainIds.Count == 0)
        {
            return new List<DomainOriginDto>();
        }

        var items = await _db.Queryable<DomainOriginEntity>()
            .Where(o => domainIds.Contains(o.DomainId ?? 0))
            .OrderBy(o => o.Id, OrderByType.Asc)
            .ToListAsync();

        return items.Select(MapOriginDto).ToList();
    }

    private static DomainDto MapDomainDto(DomainEntity domain, IReadOnlyList<DomainOriginDto>? origins)
    {
        return new DomainDto
        {
            Id = domain.Id,
            UserId = domain.UserId ?? 0,
            Name = domain.Name,
            Cname = domain.Cname,
            Status = domain.Status ?? 0,
            Origins = origins ?? Array.Empty<DomainOriginDto>(),
            CreatedAt = FormatTime(domain.CreatedAt),
            UpdatedAt = FormatTime(domain.UpdatedAt)
        };
    }

    private static DomainOriginDto MapOriginDto(DomainOriginEntity origin)
    {
        return new DomainOriginDto
        {
            Id = origin.Id,
            DomainId = origin.DomainId ?? 0,
            Addr = origin.Addr,
            Port = origin.Port ?? 0,
            Weight = origin.Weight ?? 0,
            Protocol = origin.Protocol,
            CreatedAt = FormatTime(origin.CreatedAt),
            UpdatedAt = FormatTime(origin.UpdatedAt)
        };
    }

    private static List<DomainOriginEntity> BuildOrigins(IReadOnlyList<DomainOriginDto>? origins, DateTime now)
    {
        if (origins == null || origins.Count == 0)
        {
            return new List<DomainOriginEntity>();
        }

        var list = new List<DomainOriginEntity>(origins.Count);
        foreach (var origin in origins)
        {
            list.Add(new DomainOriginEntity
            {
                Addr = origin.Addr?.Trim(),
                Port = origin.Port > 0 ? origin.Port : 80,
                Weight = origin.Weight > 0 ? origin.Weight : 1,
                Protocol = string.IsNullOrWhiteSpace(origin.Protocol) ? "http" : origin.Protocol!.Trim().ToLowerInvariant(),
                CreatedAt = now,
                UpdatedAt = now
            });
        }

        return list;
    }

    private static (int Page, int PageSize) ResolvePaging(int page, int pageSize)
    {
        if (page < 1) page = 1;
        if (pageSize < 1) pageSize = DefaultPageSize;
        if (pageSize > MaxPageSize) pageSize = MaxPageSize;
        return (page, pageSize);
    }

    private static string? FormatTime(DateTime? time)
    {
        return time?.ToString("yyyy-MM-dd HH:mm:ss");
    }
}
