using System.Collections.Generic;

namespace Cnn.Common.Contracts;

public static class ErrorCodes
{
    public const int Success = 200;

    public const int InvalidParam = 40001;
    public const int MissingParam = 40002;

    public const int AuthInvalid = 40101;
    public const int AuthExpired = 40102;

    public const int PermissionDenied = 40301;

    public const int NotFound = 40401;

    public const int AlreadyExists = 40901;
    public const int InUse = 40902;
    public const int StateConflict = 40903;

    public const int PreconditionFailed = 41201;

    public const int RateLimited = 42901;
    public const int QuotaExceeded = 42902;

    public const int InternalError = 50001;
    public const int DbError = 50002;
    public const int ConfigError = 50003;

    public const int ExternalProviderError = 50201;

    public const int ServiceUnavailable = 50301;
    public const int TaskQueueFull = 50302;

    public const int Timeout = 50401;

    public const int AgentOffline = 60001;
    public const int AgentAuthFailed = 60002;
    public const int AgentVersionMismatch = 60003;
    public const int AgentTaskReject = 60004;

    public const int WsNotConnected = 61001;
}

public static class ErrorCodeMessages
{
    private static readonly IReadOnlyDictionary<int, string> Map = new Dictionary<int, string>
    {
        { ErrorCodes.Success, "success" },
        { ErrorCodes.InvalidParam, "invalid_param" },
        { ErrorCodes.MissingParam, "missing_param" },
        { ErrorCodes.AuthInvalid, "auth_invalid" },
        { ErrorCodes.AuthExpired, "auth_expired" },
        { ErrorCodes.PermissionDenied, "permission_denied" },
        { ErrorCodes.NotFound, "not_found" },
        { ErrorCodes.AlreadyExists, "already_exists" },
        { ErrorCodes.InUse, "in_use" },
        { ErrorCodes.StateConflict, "state_conflict" },
        { ErrorCodes.PreconditionFailed, "precondition_failed" },
        { ErrorCodes.RateLimited, "rate_limited" },
        { ErrorCodes.QuotaExceeded, "quota_exceeded" },
        { ErrorCodes.InternalError, "internal_error" },
        { ErrorCodes.DbError, "db_error" },
        { ErrorCodes.ConfigError, "config_error" },
        { ErrorCodes.ExternalProviderError, "external_provider_error" },
        { ErrorCodes.ServiceUnavailable, "service_unavailable" },
        { ErrorCodes.TaskQueueFull, "task_queue_full" },
        { ErrorCodes.Timeout, "timeout" },
        { ErrorCodes.AgentOffline, "agent_offline" },
        { ErrorCodes.AgentAuthFailed, "agent_auth_failed" },
        { ErrorCodes.AgentVersionMismatch, "agent_version_mismatch" },
        { ErrorCodes.AgentTaskReject, "agent_task_reject" },
        { ErrorCodes.WsNotConnected, "ws_not_connected" },
    };

    public static string GetKey(int code)
    {
        return Map.TryGetValue(code, out var key) ? key : "internal_error";
    }
}
