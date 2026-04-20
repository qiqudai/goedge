namespace Cnn.Agent.Sync;

public interface IConfigVersionTracker
{
    long ReadAppliedVersion();
    bool ShouldApply(long newVersion, bool force);
    void MarkApplied(long version);
}

public sealed class ConfigVersionTracker : IConfigVersionTracker
{
    private readonly ISyncStateStore _stateStore;

    public ConfigVersionTracker(ISyncStateStore stateStore)
    {
        _stateStore = stateStore;
    }

    public long ReadAppliedVersion()
    {
        return _stateStore.Read().LastAppliedVersion;
    }

    public bool ShouldApply(long newVersion, bool force)
    {
        if (force)
        {
            return true;
        }

        if (newVersion <= 0)
        {
            return true;
        }

        return newVersion > ReadAppliedVersion();
    }

    public void MarkApplied(long version)
    {
        _stateStore.MarkApplied(version);
    }
}
