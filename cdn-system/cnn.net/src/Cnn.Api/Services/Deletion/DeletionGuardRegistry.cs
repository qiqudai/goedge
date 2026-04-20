namespace Cnn.Api.Services.Deletion;

public sealed class DeletionGuardRegistry : IDeletionGuardRegistry
{
    private readonly IReadOnlyDictionary<string, IDeletionGuard> _guards;

    public DeletionGuardRegistry(IEnumerable<IDeletionGuard> guards)
    {
        _guards = guards.ToDictionary(g => g.ResourceType, StringComparer.OrdinalIgnoreCase);
    }

    public IDeletionGuard Resolve(string resourceType)
    {
        if (string.IsNullOrWhiteSpace(resourceType))
        {
            throw new ArgumentException("resourceType is required.", nameof(resourceType));
        }

        if (_guards.TryGetValue(resourceType, out var guard))
        {
            return guard;
        }

        throw new KeyNotFoundException($"No deletion guard registered for resource type '{resourceType}'.");
    }
}
