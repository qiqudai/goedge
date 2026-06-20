namespace Cnn.Api.Services.Users;

public interface IUserPurgeExecutor
{
    Task ExecuteAsync(long userId, CancellationToken cancellationToken);
}
