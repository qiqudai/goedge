namespace Cnn.Agent.Config;

public sealed class AgentNodeState
{
    private int _enabled = 1;

    public bool Enabled => Volatile.Read(ref _enabled) == 1;

    public void SetEnabled(bool enabled)
    {
        Volatile.Write(ref _enabled, enabled ? 1 : 0);
    }
}
