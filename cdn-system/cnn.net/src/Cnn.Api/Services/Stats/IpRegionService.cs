namespace Cnn.Api.Services.Stats;

public interface IIpRegionService
{
    (string Country, string Province) Lookup(string ip);
}

public sealed class IpRegionService : IIpRegionService
{
    public (string Country, string Province) Lookup(string ip)
    {
        return (string.Empty, string.Empty);
    }
}
