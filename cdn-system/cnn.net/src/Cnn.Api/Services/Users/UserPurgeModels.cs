namespace Cnn.Api.Services.Users;

public sealed class UserPurgeSummary
{
    public long UserId { get; init; }
    public string? Username { get; init; }
    public int SiteCount { get; init; }
    public int StreamCount { get; init; }
    public int CertificateCount { get; init; }
    public int RuleCount { get; init; }
    public int SiteGroupCount { get; init; }
    public int SubscriptionCount { get; init; }
    public int DefaultConfigCount { get; init; }
    public int TaskCount { get; init; }
}

public sealed class UserPurgePlan
{
    public UserPurgeSummary Summary { get; init; } = new();
    public IReadOnlyList<string> Steps { get; init; } = Array.Empty<string>();
}
