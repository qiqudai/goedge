using Cnn.Common.Contracts;
using Cnn.Common.Localization;
using Microsoft.AspNetCore.Http;

namespace Cnn.Api.Responses;

public static class ApiResponseFactory
{
    public static ApiResponse<T> Ok<T>(HttpContext context, IMessageLocalizer localizer, T? data)
    {
        return Build(context, localizer, ErrorCodes.Success, null, data, null);
    }

    public static ApiResponse<T> Fail<T>(
        HttpContext context,
        IMessageLocalizer localizer,
        int code,
        string? messageKey = null,
        ApiError? error = null,
        T? data = default
    )
    {
        return Build(context, localizer, code, messageKey, data, error);
    }

    private static ApiResponse<T> Build<T>(
        HttpContext context,
        IMessageLocalizer localizer,
        int code,
        string? messageKey,
        T? data,
        ApiError? error
    )
    {
        var key = string.IsNullOrWhiteSpace(messageKey) ? ErrorCodeMessages.GetKey(code) : messageKey!;
        var language = LanguageResolver.Resolve(context, localizer.DefaultLanguage);
        var message = localizer.Translate(key, language);
        var traceId = context.TraceIdentifier ?? string.Empty;

        return new ApiResponse<T>(code, message, data, traceId, error);
    }
}
