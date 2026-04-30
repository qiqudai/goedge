namespace Cnn.Agent.Network;

public interface INodeBandwidthLimiter
{
    Task<NodeBandwidthApplyResult> ApplyAsync(string? bwLimit, CancellationToken cancellationToken);
}

public sealed record NodeBandwidthApplyResult(
    bool Applied,
    string Interface,
    long LimitMbps,
    string Message);

