namespace Cnn.Api.Services.Deletion;

public sealed class DeletionPreviewService : IDeletionPreviewService
{
    private readonly IDeletionGuardRegistry _registry;

    public DeletionPreviewService(IDeletionGuardRegistry registry)
    {
        _registry = registry;
    }

    public Task<DeleteGuardResult> PreviewAsync(string resourceType, long resourceId, CancellationToken cancellationToken)
    {
        var guard = _registry.Resolve(resourceType);
        return guard.CheckAsync(resourceId, cancellationToken);
    }
}
