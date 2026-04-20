namespace Cnn.Agent.Security;

public sealed record SecurityDecision(
    bool Allowed,
    int StatusCode,
    string? ErrorPageKey,
    string? RuleType,
    string? RuleId,
    string? Reason)
{
    public static SecurityDecision Allow() => new(true, StatusCodes.Status200OK, null, null, null, null);

    public static SecurityDecision Block(
        int statusCode,
        string ruleType,
        string? ruleId,
        string? reason,
        string? errorPageKey = null)
    {
        if (statusCode < 400)
        {
            statusCode = StatusCodes.Status403Forbidden;
        }

        return new SecurityDecision(false, statusCode, errorPageKey, ruleType, ruleId, reason);
    }
}
