using System.Net;
using Cnn.Common.Contracts.Agent;

namespace Cnn.Agent.Stream;

public sealed class StreamListenerPlan
{
    public long StreamId { get; init; }
    public string Key { get; init; } = string.Empty;
    public IPAddress ListenIp { get; init; } = IPAddress.Any;
    public int ListenPort { get; init; }
    public string BalanceWay { get; init; } = "round_robin";
    public int MaxConns { get; init; } = 20_000;
    public TimeSpan ConnectTimeout { get; init; } = TimeSpan.FromSeconds(3);
    public TimeSpan IdleTimeout { get; init; } = TimeSpan.FromSeconds(60);
    public IReadOnlyList<EdgeStreamTargetDto> Targets { get; init; } = Array.Empty<EdgeStreamTargetDto>();
}

public sealed record StreamApplyResult(
    bool Success,
    int Started,
    int Stopped,
    int Restarted,
    IReadOnlyList<string> Errors,
    int Received = 0,
    int Planned = 0,
    int Applied = 0,
    int Skipped = 0,
    IReadOnlyList<string>? SkipReasons = null);

public sealed record StreamListenerState(
    string Key,
    long StreamId,
    string Listen,
    bool Running,
    int ActiveConnections,
    string? LastError);

public sealed record StreamRuntimeReport(
    string ConfiguredMode,
    string ActiveMode,
    bool NatActive,
    string? LastError,
    long LastConfigVersion,
    int LastReceived,
    int LastPlanned,
    int LastApplied,
    int LastSkipped,
    IReadOnlyList<string> LastSkipReasons,
    IReadOnlyCollection<StreamListenerState> States);
