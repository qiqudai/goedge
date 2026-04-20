namespace Cnn.Api.Services.Deletion;

public sealed class DeleteReferenceItem
{
    public string ResourceType { get; init; } = default!;
    public long ResourceId { get; init; }
    public string DisplayName { get; init; } = default!;
    public string Relation { get; init; } = default!;
}
