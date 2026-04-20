using System.Security.Cryptography;
using Cnn.Common.Contracts;
using Cnn.Domain.Entities;
using SqlSugar;

namespace Cnn.Api.Services.Common;

public interface IApiKeyService
{
    Task<ServiceResult<ApiKeyDto>> GetAsync(long userId, CancellationToken cancellationToken);
    Task<ServiceResult<bool>> UpdateAsync(long userId, ApiKeyUpdateRequest request, CancellationToken cancellationToken);
    Task<ServiceResult<ApiKeySecretDto>> ResetSecretAsync(long userId, CancellationToken cancellationToken);
}

public sealed class ApiKeyService : IApiKeyService
{
    private readonly ISqlSugarClient _db;

    public ApiKeyService(ISqlSugarClient db)
    {
        _db = db;
    }

    public async Task<ServiceResult<ApiKeyDto>> GetAsync(long userId, CancellationToken cancellationToken)
    {
        var key = await EnsureKeyAsync(userId, cancellationToken);
        if (key == null)
        {
            return ServiceResult<ApiKeyDto>.Fail(ErrorCodes.DbError, "db_save_error");
        }

        var dto = new ApiKeyDto
        {
            ApiKey = key.ApiKeyValue,
            ApiSecret = key.ApiSecret,
            ApiIp = key.ApiIp
        };

        return ServiceResult<ApiKeyDto>.Ok(dto);
    }

    public async Task<ServiceResult<bool>> UpdateAsync(long userId, ApiKeyUpdateRequest request, CancellationToken cancellationToken)
    {
        if (request == null)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam);
        }

        var key = await EnsureKeyAsync(userId, cancellationToken);
        if (key == null)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.DbError, "db_save_error");
        }

        var apiIp = request.ApiIp?.Trim() ?? string.Empty;
        var rows = await _db.Updateable<ApiKey>()
            .SetColumns(k => new ApiKey { ApiIp = apiIp })
            .Where(k => k.Id == key.Id)
            .ExecuteCommandAsync();

        if (rows <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.DbError, "db_save_error");
        }

        return ServiceResult<bool>.Ok(true);
    }

    public async Task<ServiceResult<ApiKeySecretDto>> ResetSecretAsync(long userId, CancellationToken cancellationToken)
    {
        var key = await EnsureKeyAsync(userId, cancellationToken);
        if (key == null)
        {
            return ServiceResult<ApiKeySecretDto>.Fail(ErrorCodes.DbError, "db_save_error");
        }

        var newSecret = RandomHex(15);
        var rows = await _db.Updateable<ApiKey>()
            .SetColumns(k => new ApiKey { ApiSecret = newSecret })
            .Where(k => k.Id == key.Id)
            .ExecuteCommandAsync();

        if (rows <= 0)
        {
            return ServiceResult<ApiKeySecretDto>.Fail(ErrorCodes.DbError, "db_save_error");
        }

        return ServiceResult<ApiKeySecretDto>.Ok(new ApiKeySecretDto { ApiSecret = newSecret });
    }

    private async Task<ApiKey?> EnsureKeyAsync(long userId, CancellationToken cancellationToken)
    {
        if (userId <= 0 || userId > int.MaxValue)
        {
            return null;
        }

        var uid = (int)userId;
        var key = await _db.Queryable<ApiKey>().Where(k => k.Uid == uid).FirstAsync();
        if (key != null)
        {
            return key;
        }

        key = new ApiKey
        {
            Uid = uid,
            ApiKeyValue = RandomHex(8),
            ApiSecret = RandomHex(15),
            ApiIp = string.Empty
        };

        var rows = await _db.Insertable(key).ExecuteCommandAsync();
        if (rows <= 0)
        {
            return null;
        }

        return key;
    }

    private static string RandomHex(int byteCount)
    {
        var bytes = new byte[byteCount];
        RandomNumberGenerator.Fill(bytes);
        return Convert.ToHexString(bytes).ToLowerInvariant();
    }
}
