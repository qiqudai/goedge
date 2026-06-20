using System.Security.Claims;

namespace Cnn.Api.Services;

public sealed record AdminIdentity(long UserId, bool IsAdmin);

public interface IAdminIdentityResolver
{
    AdminIdentity? Resolve(ClaimsPrincipal? user);
}

public sealed class AdminIdentityResolver : IAdminIdentityResolver
{
    public AdminIdentity? Resolve(ClaimsPrincipal? user)
    {
        if (user == null || user.Identity?.IsAuthenticated != true)
        {
            return null;
        }

        var role = user.FindFirst("role")?.Value ?? user.FindFirst(ClaimTypes.Role)?.Value;
        var isAdmin = string.Equals(role, "admin", StringComparison.OrdinalIgnoreCase);

        var userIdRaw = user.FindFirst("uid")?.Value
            ?? user.FindFirst(ClaimTypes.NameIdentifier)?.Value
            ?? user.FindFirst("user_id")?.Value;

        if (!long.TryParse(userIdRaw, out var userId) || userId <= 0)
        {
            return null;
        }

        return new AdminIdentity(userId, isAdmin);
    }
}
