using Cnn.Common.Contracts;
using Cnn.Domain.Entities;
using SqlSugar;

namespace Cnn.Api.Services.Common;

public sealed class UserPackagePermissionService : IUserPackagePermissionService
{
    private readonly ISqlSugarClient _db;

    public UserPackagePermissionService(ISqlSugarClient db)
    {
        _db = db;
    }

    public async Task<ServiceResult<bool>> UserHasCustomCcRuleAsync(long userId, CancellationToken cancellationToken)
    {
        if (userId <= 0)
        {
            return ServiceResult<bool>.Ok(false);
        }

        try
        {
            var packages = await _db.Queryable<UserPackage>()
                .Where(p => p.Uid == userId && p.CustomCcRule == true)
                .ToListAsync();

            if (packages.Count == 0)
            {
                return ServiceResult<bool>.Ok(false);
            }

            var now = DateTime.Now;
            foreach (var pack in packages)
            {
                if (!pack.EndAt.HasValue || pack.EndAt.Value == DateTime.MinValue || pack.EndAt.Value > now)
                {
                    return ServiceResult<bool>.Ok(true);
                }
            }

            return ServiceResult<bool>.Ok(false);
        }
        catch
        {
            return ServiceResult<bool>.Fail(ErrorCodes.DbError, "custom_cc_rule_check_failed");
        }
    }
}
