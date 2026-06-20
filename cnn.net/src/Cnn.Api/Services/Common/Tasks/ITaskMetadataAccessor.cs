using TaskEntity = Cnn.Domain.Entities.Task;

namespace Cnn.Api.Services.Common.Tasks;

public sealed class TaskTargetItem
{
    public string TargetType { get; init; } = default!;
    public string TargetValue { get; init; } = default!;
    public int SortOrder { get; init; }
}

public interface ITaskMetadataAccessor
{
    long GetOwnerUserId(TaskEntity task);
    IReadOnlyList<int> GetSiteIds(TaskEntity task);
    string BuildOwnerMeta(long userId);
    string BuildTargetsJson(IEnumerable<TaskTargetItem> targets);
}
