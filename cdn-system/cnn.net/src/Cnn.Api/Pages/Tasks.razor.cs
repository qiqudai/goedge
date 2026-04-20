using Cnn.Api.Services;
using Cnn.Common.Contracts;
using Microsoft.JSInterop;

namespace Cnn.Api.Pages;

public partial class Tasks : IDisposable
{
    private const string TableId = "tasks-table-main";
    private static readonly string StateFilterStorageKey = $"table:{TableId}:state_filter";

    private static readonly HashSet<string> AllowedFilters = new(StringComparer.OrdinalIgnoreCase)
    {
        "all", "fail", "retrying", "running", "waiting", "success"
    };

    private bool _loading = true;
    private List<TaskListItemDto> _items = new();
    private string _stateFilter = "all";
    private readonly HashSet<long> _expandedTaskIds = new();
    private readonly HashSet<long> _failedOnlyTaskIds = new();

    protected override async Task OnInitializedAsync()
    {
        Hub.TaskUpdated += OnTaskUpdated;
        await Hub.StartAsync();

        var data = await Api.GetTasksAsync();
        _items = data.List.ToList();

        await LoadFilterFromStorageAsync();
        _loading = false;
    }

    private async Task LoadFilterFromStorageAsync()
    {
        try
        {
            var value = await JS.InvokeAsync<string?>("localStorage.getItem", StateFilterStorageKey);
            if (string.IsNullOrWhiteSpace(value))
            {
                return;
            }

            var normalized = value.Trim().ToLowerInvariant();
            if (AllowedFilters.Contains(normalized))
            {
                _stateFilter = normalized;
            }
        }
        catch
        {
            // ignore localStorage read failures
        }
    }

    private void OnTaskUpdated(TaskUpdateDto update)
    {
        var index = _items.FindIndex(x => x.Id == update.Id);
        if (index >= 0)
        {
            var current = _items[index];
            _items[index] = new TaskListItemDto
            {
                Id = current.Id,
                Pid = current.Pid,
                Pry = current.Pry,
                Name = current.Name,
                Type = current.Type,
                Depend = current.Depend,
                CreateAt = current.CreateAt,
                StartAt = update.StartAt ?? current.StartAt,
                EndAt = update.EndAt ?? current.EndAt,
                State = update.State ?? current.State,
                ErrTimes = current.ErrTimes,
                RetryAt = current.RetryAt,
                Ret = update.Ret ?? current.Ret,
                TargetsJson = current.TargetsJson,
                Progress = update.Progress ?? current.Progress
            };
        }
        else
        {
            _items.Insert(0, new TaskListItemDto
            {
                Id = update.Id,
                StartAt = update.StartAt,
                EndAt = update.EndAt,
                State = update.State,
                Ret = update.Ret,
                TargetsJson = null,
                Progress = update.Progress
            });
        }

        InvokeAsync(StateHasChanged);
    }

    public void Dispose()
    {
        Hub.TaskUpdated -= OnTaskUpdated;
    }

    private static string Truncate(string? value, int maxLen)
    {
        if (string.IsNullOrWhiteSpace(value))
        {
            return "-";
        }

        var trimmed = value.Trim();
        if (trimmed.Length <= maxLen)
        {
            return trimmed;
        }

        return trimmed[..maxLen] + "...";
    }

    private static string FormatDate(DateTime? value)
    {
        return value?.ToString("yyyy-MM-dd HH:mm:ss", global::System.Globalization.CultureInfo.InvariantCulture) ?? "-";
    }

    private static string FormatUnix(long? unixSeconds)
    {
        if (!unixSeconds.HasValue || unixSeconds.Value <= 0)
        {
            return "-";
        }

        try
        {
            return DateTimeOffset.FromUnixTimeSeconds(unixSeconds.Value)
                .LocalDateTime
                .ToString("yyyy-MM-dd HH:mm:ss", global::System.Globalization.CultureInfo.InvariantCulture);
        }
        catch
        {
            return "-";
        }
    }

    private void ToggleExpand(long taskId)
    {
        if (_expandedTaskIds.Contains(taskId))
        {
            _expandedTaskIds.Remove(taskId);
            return;
        }

        _expandedTaskIds.Add(taskId);
    }

    private async Task SetStateFilterAsync(string filter)
    {
        var normalized = filter?.Trim().ToLowerInvariant() ?? "all";
        if (!AllowedFilters.Contains(normalized))
        {
            normalized = "all";
        }

        if (string.Equals(_stateFilter, normalized, StringComparison.OrdinalIgnoreCase))
        {
            return;
        }

        _stateFilter = normalized;
        try
        {
            await JS.InvokeVoidAsync("localStorage.setItem", StateFilterStorageKey, _stateFilter);
        }
        catch
        {
            // ignore localStorage write failures
        }
    }

    private string GetFilterButtonClass(string filter)
    {
        return string.Equals(_stateFilter, filter, StringComparison.OrdinalIgnoreCase)
            ? "btn-primary"
            : "btn-outline-primary";
    }

    private static string GetFilterLabel(string filter)
    {
        var normalized = filter?.Trim().ToLowerInvariant();
        return normalized switch
        {
            "fail" => "失败",
            "retrying" => "重试中",
            "running" => "运行中",
            "waiting" => "等待中",
            "success" => "成功",
            _ => "全部"
        };
    }

    private List<TaskListItemDto> FilterByState(IReadOnlyList<TaskListItemDto> items)
    {
        if (items == null || items.Count == 0)
        {
            return new List<TaskListItemDto>();
        }

        var filter = _stateFilter?.Trim().ToLowerInvariant();
        if (string.IsNullOrWhiteSpace(filter) || string.Equals(filter, "all", StringComparison.Ordinal))
        {
            return items.ToList();
        }

        return items
            .Where(x => string.Equals(x.State?.Trim(), filter, StringComparison.OrdinalIgnoreCase))
            .ToList();
    }

    private void ToggleFailedOnly(long taskId, object? checkedValue)
    {
        var enabled = checkedValue is bool b && b;
        if (enabled)
        {
            _failedOnlyTaskIds.Add(taskId);
        }
        else
        {
            _failedOnlyTaskIds.Remove(taskId);
        }
    }

    private static string GetStateBadgeClass(string? state)
    {
        var normalized = state?.Trim().ToLowerInvariant();
        return normalized switch
        {
            "success" => "badge bg-success",
            "failed_final" => "badge bg-danger",
            "running" => "badge bg-primary",
            "waiting" => "badge bg-secondary",
            _ => "badge bg-light text-dark"
        };
    }
}
