namespace Cnn.Agent.Stream;

public sealed class StreamRuntimeOptions
{
    public string Mode { get; set; } = "userspace";
    public bool FallbackToUserspaceOnNatFailure { get; set; } = true;
    public string IptablesBinary { get; set; } = "iptables";
    public int CommandTimeoutMs { get; set; } = 3000;
}
