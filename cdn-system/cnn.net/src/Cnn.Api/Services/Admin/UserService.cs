using System.Text.Json;
using Cnn.Common.Contracts;
using Cnn.Common.Contracts.Admin;
using Cnn.Api.Services.Auth;
using Cnn.Api.Services.Common;
using Cnn.Domain.Entities;
using SqlSugar;

namespace Cnn.Api.Services.Admin;

public interface IUserService
{
    Task<ServiceResult<UserListResult>> ListAsync(UserListQuery query, CancellationToken cancellationToken);
    Task<ServiceResult<UserItemDto>> GetAsync(long id, CancellationToken cancellationToken);
    Task<ServiceResult<UserItemDto>> CreateAsync(UserCreateRequest request, CancellationToken cancellationToken);
    Task<ServiceResult<bool>> ToggleStatusAsync(long id, UserStatusUpdateRequest request, CancellationToken cancellationToken);
    Task<ServiceResult<bool>> UpdateAsync(long id, UserUpdateRequest request, CancellationToken cancellationToken);
    Task<ServiceResult<bool>> DeleteAsync(long id, CancellationToken cancellationToken);
    Task<ServiceResult<bool>> ResetPurgeUsageAsync(long id, CancellationToken cancellationToken);
    Task<ServiceResult<LoginResponse>> ImpersonateAsync(long id, CancellationToken cancellationToken);
}

public sealed class UserService : IUserService
{
    private const int DefaultPageSize = 20;
    private const int MaxPageSize = 200;

    private readonly ISqlSugarClient _db;
    private readonly IAuthTokenService _tokenService;
    private readonly ISystemConfigService _systemConfigService;

    public UserService(ISqlSugarClient db, IAuthTokenService tokenService, ISystemConfigService systemConfigService)
    {
        _db = db;
        _tokenService = tokenService;
        _systemConfigService = systemConfigService;
    }

    public async Task<ServiceResult<UserListResult>> ListAsync(UserListQuery query, CancellationToken cancellationToken)
    {
        query ??= new UserListQuery();
        var (page, pageSize) = ResolvePaging(query);
        var keyword = query.Keyword?.Trim();

        var q = _db.Queryable<User>();
        if (!string.IsNullOrWhiteSpace(keyword))
        {
            var lower = keyword.ToLowerInvariant();
            if (long.TryParse(lower, out var id) && id > 0)
            {
                q = q.Where(u => u.Id == id
                                 || SqlFunc.ToLower(u.Name)!.Contains(lower)
                                 || SqlFunc.ToLower(u.Email)!.Contains(lower)
                                 || SqlFunc.ToLower(u.Phone)!.Contains(lower)
                                 || SqlFunc.ToLower(u.Qq)!.Contains(lower)
                                 || SqlFunc.ToLower(u.Des)!.Contains(lower));
            }
            else
            {
                q = q.Where(u => SqlFunc.ToLower(u.Name)!.Contains(lower)
                                 || SqlFunc.ToLower(u.Email)!.Contains(lower)
                                 || SqlFunc.ToLower(u.Phone)!.Contains(lower)
                                 || SqlFunc.ToLower(u.Qq)!.Contains(lower)
                                 || SqlFunc.ToLower(u.Des)!.Contains(lower));
            }
        }

        var total = await q.CountAsync();
        var users = await q.OrderBy(u => u.Id, OrderByType.Desc)
            .ToPageListAsync(page, pageSize);

        var list = users.Select(BuildUserItem).ToList();
        return ServiceResult<UserListResult>.Ok(new UserListResult(list, total));
    }

    public async Task<ServiceResult<UserItemDto>> GetAsync(long id, CancellationToken cancellationToken)
    {
        if (id <= 0)
        {
            return ServiceResult<UserItemDto>.Fail(ErrorCodes.InvalidParam);
        }

        var user = await _db.Queryable<User>().Where(u => u.Id == id).FirstAsync();
        if (user == null)
        {
            return ServiceResult<UserItemDto>.Fail(ErrorCodes.NotFound, "user_not_found");
        }

        return ServiceResult<UserItemDto>.Ok(BuildUserItem(user));
    }

    public async Task<ServiceResult<UserItemDto>> CreateAsync(UserCreateRequest request, CancellationToken cancellationToken)
    {
        if (request == null)
        {
            return ServiceResult<UserItemDto>.Fail(ErrorCodes.InvalidParam);
        }

        var email = (request.Email ?? string.Empty).Trim();
        var name = (request.Name ?? string.Empty).Trim();
        var password = (request.Password ?? string.Empty).Trim();
        var des = (request.Des ?? string.Empty).Trim();
        var phone = (request.Phone ?? string.Empty).Trim();
        var qq = (request.Qq ?? string.Empty).Trim();
        var certName = (request.CertName ?? string.Empty).Trim();
        var certNo = (request.CertNo ?? string.Empty).Trim();
        var whiteIp = (request.WhiteIp ?? string.Empty).Trim();
        var loginCaptcha = (request.LoginCaptcha ?? string.Empty).Trim().ToLowerInvariant();
        var enable = request.Enable ?? true;

        if (string.IsNullOrWhiteSpace(name) && !string.IsNullOrWhiteSpace(email))
        {
            name = email;
        }

        if (string.IsNullOrWhiteSpace(name) || string.IsNullOrWhiteSpace(password))
        {
            return ServiceResult<UserItemDto>.Fail(ErrorCodes.InvalidParam);
        }

        var nameExists = await _db.Queryable<User>().AnyAsync(u => u.Name == name);
        if (nameExists)
        {
            return ServiceResult<UserItemDto>.Fail(ErrorCodes.AlreadyExists, "user_exists");
        }

        if (!string.IsNullOrWhiteSpace(email))
        {
            var emailExists = await _db.Queryable<User>().AnyAsync(u => u.Email == email);
            if (emailExists)
            {
                return ServiceResult<UserItemDto>.Fail(ErrorCodes.AlreadyExists, "user_exists");
            }
        }

        var hashed = PasswordHasher.HashPasswordForStorage(password);
        if (string.IsNullOrWhiteSpace(hashed))
        {
            return ServiceResult<UserItemDto>.Fail(ErrorCodes.InternalError, "user_create_failed");
        }

        var user = new User
        {
            Email = email,
            Name = name,
            Des = des,
            Phone = phone,
            Qq = qq,
            Password = hashed,
            Enable = enable,
            CertName = certName,
            CertNo = certNo,
            LoginCaptcha = loginCaptcha,
            WhiteIp = whiteIp,
            CreateAt = DateTime.Now,
            Type = 2,
            Balance = 0,
            Freeze = 0
        };

        var inserted = await _db.Insertable(user).ExecuteReturnEntityAsync();
        if (inserted == null || inserted.Id <= 0)
        {
            return ServiceResult<UserItemDto>.Fail(ErrorCodes.DbError, "db_save_error");
        }

        return ServiceResult<UserItemDto>.Ok(BuildUserItem(inserted));
    }

    public async Task<ServiceResult<bool>> ToggleStatusAsync(long id, UserStatusUpdateRequest request, CancellationToken cancellationToken)
    {
        if (id <= 0 || request == null)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam);
        }

        var enable = request.Status == 1;
        var rows = await _db.Updateable<User>()
            .SetColumns(u => new User { Enable = enable })
            .Where(u => u.Id == id)
            .ExecuteCommandAsync();

        if (rows <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.NotFound, "user_not_found");
        }

        return ServiceResult<bool>.Ok(true);
    }

    public async Task<ServiceResult<bool>> UpdateAsync(long id, UserUpdateRequest request, CancellationToken cancellationToken)
    {
        if (id <= 0 || request == null)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam);
        }

        var exists = await _db.Queryable<User>().Where(u => u.Id == id).Select(u => u.Id).FirstAsync();
        if (exists == 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.NotFound, "user_not_found");
        }

        var email = (request.Email ?? string.Empty).Trim();
        var name = (request.Name ?? string.Empty).Trim();
        var des = (request.Des ?? string.Empty).Trim();
        var phone = (request.Phone ?? string.Empty).Trim();
        var qq = (request.Qq ?? string.Empty).Trim();
        var certName = (request.CertName ?? string.Empty).Trim();
        var certNo = (request.CertNo ?? string.Empty).Trim();
        var whiteIp = (request.WhiteIp ?? string.Empty).Trim();
        var loginCaptcha = (request.LoginCaptcha ?? string.Empty).Trim();
        var enable = request.Enable ?? false;

        var rows = await _db.Updateable<User>()
            .SetColumns(u => new User
            {
                Email = email,
                Name = name,
                Des = des,
                Phone = phone,
                Qq = qq,
                CertName = certName,
                CertNo = certNo,
                WhiteIp = whiteIp,
                LoginCaptcha = loginCaptcha,
                Enable = enable
            })
            .Where(u => u.Id == id)
            .ExecuteCommandAsync();

        if (rows <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.DbError, "db_save_error");
        }

        var password = request.Password?.Trim();
        if (!string.IsNullOrWhiteSpace(password))
        {
            var hashed = PasswordHasher.HashPasswordForStorage(password);
            if (string.IsNullOrWhiteSpace(hashed))
            {
                return ServiceResult<bool>.Fail(ErrorCodes.InternalError, "user_create_failed");
            }

            await _db.Updateable<User>()
                .SetColumns(u => new User { Password = hashed })
                .Where(u => u.Id == id)
                .ExecuteCommandAsync();
        }

        return ServiceResult<bool>.Ok(true);
    }

    public async Task<ServiceResult<bool>> DeleteAsync(long id, CancellationToken cancellationToken)
    {
        if (id <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam);
        }

        var rows = await _db.Deleteable<User>().Where(u => u.Id == id).ExecuteCommandAsync();
        if (rows <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.NotFound, "user_not_found");
        }

        return ServiceResult<bool>.Ok(true);
    }

    public async Task<ServiceResult<bool>> ResetPurgeUsageAsync(long id, CancellationToken cancellationToken)
    {
        if (id <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam);
        }

        var userExists = await _db.Queryable<User>().Where(u => u.Id == id).Select(u => u.Id).FirstAsync();
        if (userExists == 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.NotFound, "user_not_found");
        }

        var payload = new
        {
            date = DateTime.Now.ToString("yyyy-MM-dd"),
            refresh_url = 0,
            refresh_dir = 0,
            preheat = 0
        };

        var raw = JsonSerializer.Serialize(payload, new JsonSerializerOptions(JsonSerializerDefaults.Web));
        var query = _db.Queryable<Config>()
            .Where(c => c.Name == "purge_usage" && c.Type == "user" && c.ScopeName == "user" && c.ScopeId == id);

        var cfg = await query.FirstAsync();
        if (cfg == null)
        {
            cfg = new Config
            {
                Name = "purge_usage",
                Value = raw,
                Type = "user",
                ScopeId = (int)id,
                ScopeName = "user",
                Enable = true,
                CreateAt = DateTime.Now,
                UpdateAt = DateTime.Now
            };
            var inserted = await _db.Insertable(cfg).ExecuteCommandAsync();
            if (inserted <= 0)
            {
                return ServiceResult<bool>.Fail(ErrorCodes.DbError, "db_save_error");
            }
        }
        else
        {
            cfg.Value = raw;
            cfg.UpdateAt = DateTime.Now;
            var updated = await _db.Updateable(cfg)
                .UpdateColumns(c => new { c.Value, c.UpdateAt })
                .Where(c => c.Name == "purge_usage" && c.Type == "user" && c.ScopeName == "user" && c.ScopeId == id)
                .ExecuteCommandAsync();
            if (updated <= 0)
            {
                return ServiceResult<bool>.Fail(ErrorCodes.DbError, "db_save_error");
            }
        }

        return ServiceResult<bool>.Ok(true);
    }

    public async Task<ServiceResult<LoginResponse>> ImpersonateAsync(long id, CancellationToken cancellationToken)
    {
        if (id <= 0)
        {
            return ServiceResult<LoginResponse>.Fail(ErrorCodes.InvalidParam);
        }

        var user = await _db.Queryable<User>().Where(u => u.Id == id).FirstAsync();
        if (user == null)
        {
            return ServiceResult<LoginResponse>.Fail(ErrorCodes.NotFound, "user_not_found");
        }

        if (user.Enable != true)
        {
            return ServiceResult<LoginResponse>.Fail(ErrorCodes.PermissionDenied, "user_disabled");
        }

        var role = user.Type == 1 ? "admin" : "user";
        var ttl = await ResolveLoginSessionTtlAsync(cancellationToken);
        var token = _tokenService.GenerateToken(user.Id, role, ttl);

        var payload = new LoginResponse
        {
            Token = token,
            Role = role,
            Uid = user.Id,
            Name = user.Name
        };

        return ServiceResult<LoginResponse>.Ok(payload);
    }

    private static (int Page, int PageSize) ResolvePaging(UserListQuery query)
    {
        var page = query.Page.GetValueOrDefault() < 1 ? 1 : query.Page!.Value;
        var pageSize = query.PageSize.GetValueOrDefault();
        if (pageSize <= 0)
        {
            pageSize = query.Size.GetValueOrDefault();
        }
        if (pageSize <= 0)
        {
            pageSize = DefaultPageSize;
        }
        if (pageSize > MaxPageSize)
        {
            pageSize = MaxPageSize;
        }

        return (page, pageSize);
    }

    private static UserItemDto BuildUserItem(User user)
    {
        return new UserItemDto
        {
            Id = user.Id,
            Email = user.Email,
            Name = user.Name,
            Des = user.Des,
            Phone = user.Phone,
            Qq = user.Qq,
            CertId = user.CertId,
            CertName = user.CertName,
            CertNo = user.CertNo,
            CertVerified = user.CertVerified,
            WhiteIp = user.WhiteIp,
            LoginCaptcha = user.LoginCaptcha,
            Balance = user.Balance,
            Freeze = user.Freeze,
            CreateAt = user.CreateAt?.ToString("yyyy-MM-dd HH:mm:ss"),
            Enable = user.Enable,
            Type = user.Type
        };
    }

    private async Task<TimeSpan> ResolveLoginSessionTtlAsync(CancellationToken cancellationToken)
    {
        var cfg = await _systemConfigService.LoadSystemConfigAsync(cancellationToken);
        if (!cfg.TryGetValue("login_session_valid_time", out var raw))
        {
            return TimeSpan.FromHours(24);
        }

        raw = raw?.Trim();
        if (string.IsNullOrWhiteSpace(raw) || !int.TryParse(raw, out var seconds) || seconds <= 0)
        {
            return TimeSpan.FromHours(24);
        }

        return TimeSpan.FromSeconds(seconds);
    }
}
