namespace Cnn.Api.Services.Users;

public interface IUserPurgePlanner
{
    Task<UserPurgePlan> PlanAsync(long userId, CancellationToken cancellationToken);
}
