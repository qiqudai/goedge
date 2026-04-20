using System.Text.Json.Serialization;

namespace Cnn.Common.Contracts;

public sealed class DashboardUserDto
{
    [JsonPropertyName("role")]
    public string? Role { get; set; }

    [JsonPropertyName("username")]
    public string? Username { get; set; }

    [JsonPropertyName("id")]
    public long Id { get; set; }

    [JsonPropertyName("level")]
    public string? Level { get; set; }

    [JsonPropertyName("auth_state")]
    public string? AuthState { get; set; }

    [JsonPropertyName("last_login")]
    public string? LastLogin { get; set; }

    [JsonPropertyName("login_ip")]
    public string? LoginIp { get; set; }

    [JsonPropertyName("avatar")]
    public string? Avatar { get; set; }
}

public sealed class DashboardOverviewDto
{
    [JsonPropertyName("bandwidth_peak")]
    public string? BandwidthPeak { get; set; }

    [JsonPropertyName("node_bandwidth_peak")]
    public string? NodeBandwidthPeak { get; set; }

    [JsonPropertyName("requests")]
    public string? Requests { get; set; }

    [JsonPropertyName("traffic")]
    public string? Traffic { get; set; }

    [JsonPropertyName("blocked_ips")]
    public string? BlockedIps { get; set; }
}

public sealed class DashboardChartDto
{
    [JsonPropertyName("x_axis")]
    public IReadOnlyList<string> XAxis { get; set; } = Array.Empty<string>();

    [JsonPropertyName("bandwidth")]
    public IReadOnlyList<double> Bandwidth { get; set; } = Array.Empty<double>();

    [JsonPropertyName("requests")]
    public IReadOnlyList<double> Requests { get; set; } = Array.Empty<double>();

    [JsonPropertyName("traffic")]
    public IReadOnlyList<double> Traffic { get; set; } = Array.Empty<double>();

    [JsonPropertyName("blocked")]
    public IReadOnlyList<double> Blocked { get; set; } = Array.Empty<double>();
}

public sealed class DashboardTopItemDto
{
    [JsonPropertyName("name")]
    public string? Name { get; set; }

    [JsonPropertyName("count")]
    public ulong Count { get; set; }

    [JsonPropertyName("traffic")]
    public string? Traffic { get; set; }
}

public sealed class DashboardAnnouncementDto
{
    [JsonPropertyName("id")]
    public long Id { get; set; }

    [JsonPropertyName("title")]
    public string? Title { get; set; }

    [JsonPropertyName("time")]
    public string? Time { get; set; }
}

public sealed class DashboardPackageDto
{
    [JsonPropertyName("name")]
    public string? Name { get; set; }

    [JsonPropertyName("desc")]
    public string? Desc { get; set; }

    [JsonPropertyName("percent")]
    public int Percent { get; set; }
}

public sealed class DashboardResourceDto
{
    [JsonPropertyName("domains")]
    public long Domains { get; set; }

    [JsonPropertyName("forward")]
    public long Forward { get; set; }

    [JsonPropertyName("certs")]
    public long Certs { get; set; }

    [JsonPropertyName("packages")]
    public long Packages { get; set; }
}

public sealed class DashboardOpsSummaryDto
{
    [JsonPropertyName("users")]
    public long Users { get; set; }

    [JsonPropertyName("packages")]
    public long Packages { get; set; }

    [JsonPropertyName("recharge")]
    public string? Recharge { get; set; }
}

public sealed class DashboardOpsDto
{
    [JsonPropertyName("summary")]
    public DashboardOpsSummaryDto? Summary { get; set; }
}

public sealed class DashboardSystemStatusDto
{
    [JsonPropertyName("master")]
    public bool Master { get; set; }

    [JsonPropertyName("ck")]
    public bool? Ck { get; set; }

    [JsonPropertyName("ck_tips")]
    public IReadOnlyList<string> CkTips { get; set; } = Array.Empty<string>();

    [JsonPropertyName("elastic")]
    public bool? Elastic { get; set; }

    [JsonPropertyName("agent")]
    public bool Agent { get; set; }

    [JsonPropertyName("agent_total")]
    public long AgentTotal { get; set; }

    [JsonPropertyName("agent_online")]
    public long AgentOnline { get; set; }

    [JsonPropertyName("checked_at")]
    public string? CheckedAt { get; set; }
}

public sealed class DashboardLicenseDto
{
    [JsonPropertyName("total_nodes")]
    public long TotalNodes { get; set; }

    [JsonPropertyName("current_nodes")]
    public long CurrentNodes { get; set; }

    [JsonPropertyName("expire_at")]
    public string? ExpireAt { get; set; }
}

public sealed class DashboardResultDto
{
    [JsonPropertyName("user")]
    public DashboardUserDto? User { get; set; }

    [JsonPropertyName("stats")]
    public DashboardOverviewDto? Stats { get; set; }

    [JsonPropertyName("charts")]
    public DashboardChartDto? Charts { get; set; }

    [JsonPropertyName("top_domains")]
    public IReadOnlyList<DashboardTopItemDto> TopDomains { get; set; } = Array.Empty<DashboardTopItemDto>();

    [JsonPropertyName("top_urls")]
    public IReadOnlyList<DashboardTopItemDto> TopUrls { get; set; } = Array.Empty<DashboardTopItemDto>();

    [JsonPropertyName("top_ips")]
    public IReadOnlyList<DashboardTopItemDto> TopIps { get; set; } = Array.Empty<DashboardTopItemDto>();

    [JsonPropertyName("top_countries")]
    public IReadOnlyList<DashboardTopItemDto> TopCountries { get; set; } = Array.Empty<DashboardTopItemDto>();

    [JsonPropertyName("announcements")]
    public IReadOnlyList<DashboardAnnouncementDto> Announcements { get; set; } = Array.Empty<DashboardAnnouncementDto>();

    [JsonPropertyName("package")]
    public DashboardPackageDto? Package { get; set; }

    [JsonPropertyName("resources")]
    public DashboardResourceDto? Resources { get; set; }

    [JsonPropertyName("ops")]
    public DashboardOpsDto? Ops { get; set; }

    [JsonPropertyName("system_status")]
    public DashboardSystemStatusDto? SystemStatus { get; set; }

    [JsonPropertyName("license")]
    public DashboardLicenseDto? License { get; set; }
}
