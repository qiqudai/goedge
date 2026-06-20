using Microsoft.Extensions.Logging;

namespace Cnn.Api.Services.Authz;

public sealed class AuthorizationAuditLoggerSink : IAuthorizationAuditSink
{
    private readonly ILogger<AuthorizationAuditLoggerSink> _logger;

    public AuthorizationAuditLoggerSink(ILogger<AuthorizationAuditLoggerSink> logger)
    {
        _logger = logger;
    }

    public void Write(AuthorizationAuditRecord record)
    {
        _logger.LogInformation(
            "authz audit allowed={Allowed} role={Role} user={UserId} permission={Permission} resource={Resource} reason={Reason} trace={Trace} node={Node} duration_us={DurationUs}",
            record.Allowed,
            record.Role,
            record.UserId,
            record.Permission,
            record.ResourceId ?? string.Empty,
            record.Reason,
            record.TraceId ?? string.Empty,
            record.NodeId ?? string.Empty,
            record.DurationMicroseconds);
    }
}
