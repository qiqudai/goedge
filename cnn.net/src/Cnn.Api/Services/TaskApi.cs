using System.Net.Http.Json;
using Cnn.Common.Contracts;

namespace Cnn.Api.Services;

public sealed class TaskApi
{
    private readonly HttpClient _http;

    public TaskApi(HttpClient http)
    {
        _http = http;
    }

    public async Task<TaskListData> GetTasksAsync(int page = 1, int pageSize = 20)
    {
        var url = $"/api/tasks?page={page}&pageSize={pageSize}";
        var response = await _http.GetFromJsonAsync<ApiResponse<TaskListData>>(url);
        return response?.Data ?? new TaskListData(Array.Empty<TaskListItemDto>(), 0, page);
    }

    public sealed record TaskListData(IReadOnlyList<TaskListItemDto> List, long Total, int Page);
}
