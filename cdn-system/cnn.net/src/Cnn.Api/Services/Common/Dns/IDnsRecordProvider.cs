namespace Cnn.Api.Services.Common.Dns;

public interface IDnsRecordProvider
{
    Task<IReadOnlyList<DnsRecord>> GetRecordsAsync(string domain);
    Task AddRecordAsync(string domain, DnsRecord record);
    Task DeleteRecordAsync(string domain, DnsRecord record);
}

public interface IDnsRecordSetUpdater
{
    Task UpsertRecordSetAsync(string domain, DnsRecord record, IReadOnlyList<string> values);
}

public interface IDnsRecordValueReplacer
{
    Task ReplaceRecordValueAsync(string domain, DnsRecord record, string newValue);
}

public interface IDnsLineRecordDeleter
{
    Task DeleteRecordsByLineAsync(string domain, DnsRecord record);
}
