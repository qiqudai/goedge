using System.Threading;
using Cnn.Common.Contracts.Agent;

namespace Cnn.Agent.Config;

public sealed class EdgeConfigStore
{
    private EdgeConfigDto? _current;

    public EdgeConfigDto? Current => Volatile.Read(ref _current);

    public void Update(EdgeConfigDto config)
    {
        if (config == null)
        {
            return;
        }

        Volatile.Write(ref _current, config);
    }
}
