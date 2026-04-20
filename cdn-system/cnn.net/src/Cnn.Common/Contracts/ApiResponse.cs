using System.Text.Json.Serialization;

namespace Cnn.Common.Contracts;

public sealed record ApiError(
    [property: JsonPropertyName("field")] string? Field,
    [property: JsonPropertyName("reason")] string? Reason,
    [property: JsonPropertyName("detail")] string? Detail
);

public sealed class ApiResponse<T>
{
    [JsonPropertyName("code")]
    public int Code { get; init; }

    [JsonPropertyName("message")]
    public string Message { get; init; } = string.Empty;

    [JsonPropertyName("data")]
    public T? Data { get; init; }

    [JsonPropertyName("trace_id")]
    public string TraceId { get; init; } = string.Empty;

    [JsonPropertyName("error")]
    public ApiError? Error { get; init; }

    public ApiResponse(int code, string message, T? data, string traceId, ApiError? error)
    {
        Code = code;
        Message = message;
        Data = data;
        TraceId = traceId;
        Error = error;
    }
}
