namespace Cnn.Api.Services.Deletion;

public interface IDeletionPreviewService
{
    Task<DeleteGuardResult> PreviewAsync(string resourceType, long resourceId, CancellationToken cancellationToken);
}
