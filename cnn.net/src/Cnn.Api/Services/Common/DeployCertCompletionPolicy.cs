namespace Cnn.Api.Services.Common;

public static class DeployCertCompletionPolicy
{
    public const string ConfigKey = "deploy_cert_completion_policy";
    public const string StrictAllSuccess = "strict_all_success";
    public const string AllowPartialFailures = "allow_partial_failures";

    public static async Task<string> ResolvePolicyAsync(ISystemConfigService? systemConfigService, CancellationToken cancellationToken)
    {
        if (systemConfigService == null)
        {
            return StrictAllSuccess;
        }

        try
        {
            var config = await systemConfigService.LoadSystemConfigAsync(cancellationToken);
            return config.TryGetValue(ConfigKey, out var raw) ? Normalize(raw) : StrictAllSuccess;
        }
        catch
        {
            return StrictAllSuccess;
        }
    }

    public static bool IsAllowPartial(string? policy)
    {
        return string.Equals(Normalize(policy), AllowPartialFailures, StringComparison.OrdinalIgnoreCase);
    }

    public static string Normalize(string? raw)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            return StrictAllSuccess;
        }

        var value = raw.Trim().ToLowerInvariant();
        if (value is "1" or "true" or "yes" or "on" or "allow_partial" or "partial" or "tolerant" or AllowPartialFailures)
        {
            return AllowPartialFailures;
        }

        return StrictAllSuccess;
    }
}
