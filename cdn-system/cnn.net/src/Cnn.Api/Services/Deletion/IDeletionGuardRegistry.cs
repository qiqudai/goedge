namespace Cnn.Api.Services.Deletion;

public interface IDeletionGuardRegistry
{
    IDeletionGuard Resolve(string resourceType);
}
