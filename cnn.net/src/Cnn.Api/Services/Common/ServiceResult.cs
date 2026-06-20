using Cnn.Common.Contracts;

namespace Cnn.Api.Services.Common;

public sealed record ServiceResult<T>(bool Success, T? Data, int ErrorCode, string? MessageKey)
{
    public static ServiceResult<T> Ok(T data)
    {
        return new ServiceResult<T>(true, data, ErrorCodes.Success, null);
    }

    public static ServiceResult<T> Fail(int errorCode, string? messageKey = null)
    {
        return new ServiceResult<T>(false, default, errorCode, messageKey);
    }

    public static ServiceResult<T> FailWithData(int errorCode, T data, string? messageKey = null)
    {
        return new ServiceResult<T>(false, data, errorCode, messageKey);
    }
}
