namespace Cnn.Agent.Network;

public interface IPackageBandwidthLimiter
{
    Task<PackageBandwidthApplyResult> ApplyAsync(
        IReadOnlyCollection<Cnn.Common.Contracts.Agent.AgentPackageConfigDto> packages,
        CancellationToken cancellationToken);
}

public sealed record PackageBandwidthApplyResult(
    bool Applied,
    string Interface,
    long LimitMbps,
    string Message);

