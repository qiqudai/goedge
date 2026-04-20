namespace Cnn.Api.Services.Common;

public interface IUserPackagePermissionService
{
    Task<ServiceResult<bool>> UserHasCustomCcRuleAsync(long userId, CancellationToken cancellationToken);
}
