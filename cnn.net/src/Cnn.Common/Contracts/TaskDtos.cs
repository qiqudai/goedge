using System.Text.Json.Serialization;

namespace Cnn.Common.Contracts;

public sealed class TaskListQuery
{
    [JsonPropertyName("page")]
    public int? Page { get; set; }

    [JsonPropertyName("pageSize")]
    public int? PageSize { get; set; }

    [JsonPropertyName("keyword")]
    public string? Keyword { get; set; }

    [JsonPropertyName("type")]
    public string? Type { get; set; }

    [JsonPropertyName("user_id")]
    public long? UserId { get; set; }
}

public sealed class TaskCreateRequest
{
    [JsonPropertyName("type")]
    public string? Type { get; set; }

    [JsonPropertyName("urls")]
    public string? Urls { get; set; }

    [JsonPropertyName("user_id")]
    public long? UserId { get; set; }
}

public sealed record TaskListResult(
    [property: JsonPropertyName("list")] IReadOnlyList<TaskListItemDto> List,
    [property: JsonPropertyName("total")] long Total,
    [property: JsonPropertyName("page")] int Page
);

public sealed class TaskListItemDto
{
    [JsonPropertyName("id")]
    public long Id { get; set; }

    [JsonPropertyName("pid")]
    public long? Pid { get; set; }

    [JsonPropertyName("pry")]
    public int? Pry { get; set; }

    [JsonPropertyName("name")]
    public string? Name { get; set; }

    [JsonPropertyName("data")]
    public string? Data { get; set; }

    [JsonPropertyName("type")]
    public string? Type { get; set; }

    [JsonPropertyName("depend")]
    public string? Depend { get; set; }

    [JsonPropertyName("create_at")]
    public DateTime? CreateAt { get; set; }

    [JsonPropertyName("start_at")]
    public DateTime? StartAt { get; set; }

    [JsonPropertyName("end_at")]
    public DateTime? EndAt { get; set; }

    [JsonPropertyName("state")]
    public string? State { get; set; }

    [JsonPropertyName("err_times")]
    public int? ErrTimes { get; set; }

    [JsonPropertyName("retry_at")]
    public DateTime? RetryAt { get; set; }

    [JsonPropertyName("ret")]
    public string? Ret { get; set; }

    [JsonPropertyName("targets_json")]
    public string? TargetsJson { get; set; }

    [JsonPropertyName("progress")]
    public string? Progress { get; set; }
}

public sealed class TaskDetailDto
{
    [JsonPropertyName("id")]
    public long Id { get; set; }

    [JsonPropertyName("pid")]
    public long? Pid { get; set; }

    [JsonPropertyName("pry")]
    public int? Pry { get; set; }

    [JsonPropertyName("name")]
    public string? Name { get; set; }

    [JsonPropertyName("type")]
    public string? Type { get; set; }

    [JsonPropertyName("depend")]
    public string? Depend { get; set; }

    [JsonPropertyName("create_at")]
    public DateTime? CreateAt { get; set; }

    [JsonPropertyName("start_at")]
    public DateTime? StartAt { get; set; }

    [JsonPropertyName("end_at")]
    public DateTime? EndAt { get; set; }

    [JsonPropertyName("state")]
    public string? State { get; set; }

    [JsonPropertyName("err_times")]
    public int? ErrTimes { get; set; }

    [JsonPropertyName("progress")]
    public string? Progress { get; set; }

    [JsonPropertyName("ret")]
    public string? Ret { get; set; }
}

public sealed record TaskUsagePayload(
    [property: JsonPropertyName("limits")] TaskUsageLimit Limits,
    [property: JsonPropertyName("used")] TaskUsage Used,
    [property: JsonPropertyName("remaining")] TaskUsageLimit Remaining
);

public sealed class TaskUsage
{
    [JsonPropertyName("date")]
    public string Date { get; set; } = string.Empty;

    [JsonPropertyName("refresh_url")]
    public int RefreshUrl { get; set; }

    [JsonPropertyName("refresh_dir")]
    public int RefreshDir { get; set; }

    [JsonPropertyName("preheat")]
    public int Preheat { get; set; }
}

public sealed class TaskUsageLimit
{
    [JsonPropertyName("refresh_url")]
    public int RefreshUrl { get; set; }

    [JsonPropertyName("refresh_dir")]
    public int RefreshDir { get; set; }

    [JsonPropertyName("preheat")]
    public int Preheat { get; set; }
}

public sealed record TaskUpdateDto(
    [property: JsonPropertyName("id")] long Id,
    [property: JsonPropertyName("state")] string? State,
    [property: JsonPropertyName("progress")] string? Progress,
    [property: JsonPropertyName("ret")] string? Ret,
    [property: JsonPropertyName("start_at")] DateTime? StartAt,
    [property: JsonPropertyName("end_at")] DateTime? EndAt
);

public sealed class AgentTaskDto
{
    [JsonPropertyName("id")]
    public long Id { get; set; }

    [JsonPropertyName("pid")]
    public int? Pid { get; set; }

    [JsonPropertyName("pry")]
    public int? Pry { get; set; }

    [JsonPropertyName("name")]
    public string? Name { get; set; }

    [JsonPropertyName("type")]
    public string? Type { get; set; }

    [JsonPropertyName("res")]
    public string? Res { get; set; }

    [JsonPropertyName("data")]
    public string? Data { get; set; }

    [JsonPropertyName("targets_json")]
    public string? TargetsJson { get; set; }

    [JsonPropertyName("depend")]
    public string? Depend { get; set; }

    [JsonPropertyName("create_at")]
    public DateTime? CreateAt { get; set; }

    [JsonPropertyName("start_at")]
    public DateTime? StartAt { get; set; }

    [JsonPropertyName("end_at")]
    public DateTime? EndAt { get; set; }

    [JsonPropertyName("ret")]
    public string? Ret { get; set; }

    [JsonPropertyName("enable")]
    public bool? Enable { get; set; }

    [JsonPropertyName("state")]
    public string? State { get; set; }

    [JsonPropertyName("err_times")]
    public int? ErrTimes { get; set; }

    [JsonPropertyName("retry_at")]
    public DateTime? RetryAt { get; set; }

    [JsonPropertyName("progress")]
    public string? Progress { get; set; }
}

public sealed record AgentTaskListResult(
    [property: JsonPropertyName("tasks")] IReadOnlyList<AgentTaskDto> Tasks
);

public sealed class AgentTaskFinishRequest
{
    [JsonPropertyName("state")]
    public string? State { get; set; }

    [JsonPropertyName("ret")]
    public string? Ret { get; set; }
}
