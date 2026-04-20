namespace Cnn.Api.Services.Authz;

public sealed record AuthorizationAuditRecord(
    DateTimeOffset Timestamp,
    string Role,
    long? UserId,
    string Permission,
    string? ResourceId,
    bool Allowed,
    string Reason,
    string? TraceId,
    string? NodeId,
    long DurationMicroseconds);

public interface IAuthorizationAuditSink
{
    void Write(AuthorizationAuditRecord record);
}
