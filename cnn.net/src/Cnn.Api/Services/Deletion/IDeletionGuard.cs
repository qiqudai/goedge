namespace Cnn.Api.Services.Deletion;

public interface IDeletionGuard
{
    string ResourceType { get; }

    Task<DeleteGuardResult> CheckAsync(long resourceId, CancellationToken cancellationToken);
}
