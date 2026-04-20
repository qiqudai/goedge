namespace Cnn.Agent.Proxy;

public sealed record ProxyApplyResult(bool Success, string Status, string? Error, long Version)
{
    public static ProxyApplyResult Ok(long version) => new(true, "ok", null, version);

    public static ProxyApplyResult Skipped(long version) => new(true, "skipped", null, version);

    public static ProxyApplyResult Fail(long version, string error) => new(false, "fail", error, version);
}
