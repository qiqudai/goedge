using System.Text.Json;
using TaskEntity = Cnn.Domain.Entities.Task;

namespace Cnn.Api.Services.Common.Tasks;

public sealed class JsonTaskMetadataAccessor : ITaskMetadataAccessor
{
    private static readonly JsonSerializerOptions JsonOptions = new(JsonSerializerDefaults.Web)
    {
        PropertyNameCaseInsensitive = true
    };

    public long GetOwnerUserId(TaskEntity task)
    {
        if (string.IsNullOrWhiteSpace(task.Res))
        {
            return 0;
        }

        try
        {
            using var doc = JsonDocument.Parse(task.Res);
            if (doc.RootElement.TryGetProperty("user_id", out var value))
            {
                if (value.ValueKind == JsonValueKind.Number && value.TryGetInt64(out var id))
                {
                    return id;
                }
                if (value.ValueKind == JsonValueKind.String && long.TryParse(value.GetString(), out var parsed))
                {
                    return parsed;
                }
            }
        }
        catch
        {
            return 0;
        }

        return 0;
    }

    public IReadOnlyList<int> GetSiteIds(TaskEntity task)
    {
        var ids = new List<int>();
        if (string.IsNullOrWhiteSpace(task.Data))
        {
            return ids;
        }

        try
        {
            using var doc = JsonDocument.Parse(task.Data);
            if (!doc.RootElement.TryGetProperty("site_ids", out var list))
            {
                return ids;
            }

            if (list.ValueKind != JsonValueKind.Array)
            {
                return ids;
            }

            foreach (var item in list.EnumerateArray())
            {
                if (item.ValueKind == JsonValueKind.Number && item.TryGetInt32(out var id) && id > 0)
                {
                    ids.Add(id);
                }
            }
        }
        catch
        {
            return ids;
        }

        return ids;
    }

    public string BuildOwnerMeta(long userId)
    {
        var meta = new Dictionary<string, long> { ["user_id"] = userId };
        return JsonSerializer.Serialize(meta, JsonOptions);
    }

    public string BuildTargetsJson(IEnumerable<TaskTargetItem> targets)
    {
        return JsonSerializer.Serialize(targets, JsonOptions);
    }
}
