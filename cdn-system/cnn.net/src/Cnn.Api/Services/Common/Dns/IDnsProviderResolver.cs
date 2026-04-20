namespace Cnn.Api.Services.Common.Dns;

public interface IDnsProviderResolver
{
    IDnsRecordProvider? Resolve(string? type, string? credentials);
}

public sealed class DnsProviderResolver : IDnsProviderResolver
{
    public IDnsRecordProvider? Resolve(string? type, string? credentials)
    {
        return DnsProviderFactory.TryCreate(type, credentials);
    }
}

