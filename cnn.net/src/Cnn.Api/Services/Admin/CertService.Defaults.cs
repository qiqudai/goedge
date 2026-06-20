using System.Text.Json;
using Cnn.Common.Contracts;
using Cnn.Common.Contracts.Admin;
using Cnn.Domain.Entities;
using SqlSugar;
using Task = System.Threading.Tasks.Task;

namespace Cnn.Api.Services.Admin;

public sealed partial class CertService
{
    public async Task<ServiceResult<CertDefaultSettingsDto>> GetDefaultSettingsAsync(
        long? userId,
        bool isAdmin,
        CancellationToken cancellationToken)
    {
        long targetUserId = 0;
        if (isAdmin && userId is > 0)
        {
            targetUserId = userId.Value;
        }
        else if (!isAdmin && userId is > 0)
        {
            targetUserId = userId.Value;
        }

        if (targetUserId > 0)
        {
            var userSettings = await LoadDefaultSettingsAsync(CertDefaultUserScope, (int)targetUserId, cancellationToken);
            if (userSettings != null)
            {
                return ServiceResult<CertDefaultSettingsDto>.Ok(userSettings);
            }
        }

        var globalSettings = await LoadDefaultSettingsAsync(CertDefaultScope, 0, cancellationToken);
        if (globalSettings != null)
        {
            return ServiceResult<CertDefaultSettingsDto>.Ok(globalSettings);
        }

        var created = new CertDefaultSettingsDto { Type = "system", DnsApi = 0 };
        await SaveDefaultSettingsAsync(CertDefaultScope, 0, created, cancellationToken);
        return ServiceResult<CertDefaultSettingsDto>.Ok(created);
    }

    public async Task<ServiceResult<bool>> UpdateDefaultSettingsAsync(
        CertDefaultSettingsRequest request,
        long? userId,
        bool isAdmin,
        CancellationToken cancellationToken)
    {
        if (request == null)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam, "invalid_param");
        }

        var type = string.IsNullOrWhiteSpace(request.Type) ? "system" : request.Type.Trim();
        var targetUserId = 0L;

        if (isAdmin)
        {
            targetUserId = request.UserId ?? 0;
        }
        else
        {
            targetUserId = userId ?? 0;
        }

        var scopeName = targetUserId > 0 ? CertDefaultUserScope : CertDefaultScope;
        var scopeId = targetUserId > 0 ? (int)targetUserId : 0;

        var payload = new CertDefaultSettingsDto
        {
            Type = type,
            DnsApi = request.DnsApi
        };

        await SaveDefaultSettingsAsync(scopeName, scopeId, payload, cancellationToken);
        return ServiceResult<bool>.Ok(true);
    }

    private async Task<CertDefaultSettingsDto?> LoadDefaultSettingsAsync(string scopeName, int scopeId, CancellationToken cancellationToken)
    {
        var cfg = await _db.Queryable<Config>()
            .Where(c => c.Name == CertDefaultSettingsKey && c.Type == CertDefaultSettingsType && c.ScopeName == scopeName && c.ScopeId == scopeId)
            .FirstAsync();

        if (cfg == null || string.IsNullOrWhiteSpace(cfg.Value))
        {
            return null;
        }

        try
        {
            var settings = JsonSerializer.Deserialize<CertDefaultSettingsDto>(cfg.Value, JsonOptions);
            if (settings == null)
            {
                return null;
            }

            if (string.IsNullOrWhiteSpace(settings.Type))
            {
                settings.Type = "system";
            }

            return settings;
        }
        catch
        {
            return null;
        }
    }

    private async Task SaveDefaultSettingsAsync(string scopeName, int scopeId, CertDefaultSettingsDto payload, CancellationToken cancellationToken)
    {
        var raw = JsonSerializer.Serialize(payload, JsonOptions);
        var existing = await _db.Queryable<Config>()
            .Where(c => c.Name == CertDefaultSettingsKey && c.Type == CertDefaultSettingsType && c.ScopeName == scopeName && c.ScopeId == scopeId)
            .FirstAsync();

        var now = DateTime.Now;
        if (existing == null)
        {
            var entity = new Config
            {
                Name = CertDefaultSettingsKey,
                Type = CertDefaultSettingsType,
                ScopeName = scopeName,
                ScopeId = scopeId,
                Value = raw,
                Enable = true,
                CreateAt = now,
                UpdateAt = now
            };
            await _db.Insertable(entity).ExecuteCommandAsync();
            return;
        }

        await _db.Updateable<Config>()
            .SetColumns(c => new Config { Value = raw, UpdateAt = now })
            .Where(c => c.Name == CertDefaultSettingsKey && c.Type == CertDefaultSettingsType && c.ScopeName == scopeName && c.ScopeId == scopeId)
            .ExecuteCommandAsync();
    }
}


