using Cnn.Common.Contracts;
using Cnn.Api.Services.Auth;
using Cnn.Domain.Entities;
using SqlSugar;

namespace Cnn.Api.Services.Common;

public interface IUserProfileService
{
    Task<ServiceResult<UserProfileDto>> GetAsync(long userId, CancellationToken cancellationToken);
    Task<ServiceResult<bool>> UpdateAsync(long userId, UpdateProfileRequest request, CancellationToken cancellationToken);
    Task<ServiceResult<bool>> UpdatePasswordAsync(long userId, UpdatePasswordRequest request, CancellationToken cancellationToken);
}

public sealed class UserProfileService : IUserProfileService
{
    private readonly ISqlSugarClient _db;

    public UserProfileService(ISqlSugarClient db)
    {
        _db = db;
    }

    public async Task<ServiceResult<UserProfileDto>> GetAsync(long userId, CancellationToken cancellationToken)
    {
        if (userId <= 0)
        {
            return ServiceResult<UserProfileDto>.Fail(ErrorCodes.InvalidParam, "user_id_required");
        }

        var user = await _db.Queryable<User>().Where(u => u.Id == userId).FirstAsync();
        if (user == null)
        {
            return ServiceResult<UserProfileDto>.Fail(ErrorCodes.NotFound, "user_not_found");
        }

        var dto = new UserProfileDto
        {
            Id = user.Id,
            Name = user.Name,
            Email = user.Email,
            Phone = user.Phone,
            Qq = user.Qq,
            Balance = user.Balance,
            CertName = user.CertName,
            CertNo = user.CertNo,
            CertVerified = user.CertVerified,
            WhiteIp = user.WhiteIp,
            LoginCaptcha = user.LoginCaptcha,
            CreateAt = user.CreateAt?.ToString("yyyy-MM-dd HH:mm:ss")
        };

        return ServiceResult<UserProfileDto>.Ok(dto);
    }

    public async Task<ServiceResult<bool>> UpdateAsync(long userId, UpdateProfileRequest request, CancellationToken cancellationToken)
    {
        if (userId <= 0 || request == null)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam, "user_id_required");
        }

        var rows = await _db.Updateable<User>()
            .SetColumns(u => new User
            {
                Email = (request.Email ?? string.Empty).Trim(),
                Phone = (request.Phone ?? string.Empty).Trim(),
                Qq = (request.Qq ?? string.Empty).Trim(),
                CertName = (request.CertName ?? string.Empty).Trim(),
                CertNo = (request.CertNo ?? string.Empty).Trim(),
                WhiteIp = (request.WhiteIp ?? string.Empty).Trim(),
                LoginCaptcha = (request.LoginCaptcha ?? string.Empty).Trim()
            })
            .Where(u => u.Id == userId)
            .ExecuteCommandAsync();

        if (rows <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.DbError, "db_save_error");
        }

        return ServiceResult<bool>.Ok(true);
    }

    public async Task<ServiceResult<bool>> UpdatePasswordAsync(long userId, UpdatePasswordRequest request, CancellationToken cancellationToken)
    {
        if (userId <= 0 || request == null)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam, "user_id_required");
        }

        var current = request.Current?.Trim() ?? string.Empty;
        var next = request.Next?.Trim() ?? string.Empty;
        if (string.IsNullOrWhiteSpace(current) || string.IsNullOrWhiteSpace(next))
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam, "invalid_param");
        }

        var providedHashed = string.Equals(request.PasswordHash?.Trim(), "sha256", StringComparison.OrdinalIgnoreCase);
        if (providedHashed && (!PasswordHasher.PasswordLooksHashed(current) || !PasswordHasher.PasswordLooksHashed(next)))
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam, "invalid_param");
        }

        if (!providedHashed && PasswordHasher.PasswordLooksHashed(current) && PasswordHasher.PasswordLooksHashed(next))
        {
            providedHashed = true;
        }

        var user = await _db.Queryable<User>().Where(u => u.Id == userId).FirstAsync();
        if (user == null)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.NotFound, "user_not_found");
        }

        var (ok, _) = PasswordHasher.VerifyPassword(user.Password, current, providedHashed);
        if (!ok)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam, "invalid_credentials");
        }

        var hashed = PasswordHasher.HashPasswordForStorage(next);
        if (string.IsNullOrWhiteSpace(hashed))
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InternalError, "internal_error");
        }

        var rows = await _db.Updateable<User>()
            .SetColumns(u => new User { Password = hashed })
            .Where(u => u.Id == userId)
            .ExecuteCommandAsync();

        if (rows <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.DbError, "db_save_error");
        }

        return ServiceResult<bool>.Ok(true);
    }
}
