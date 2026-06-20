using System.Text.Json.Serialization;

namespace Cnn.Common.Contracts;

public sealed class SystemInfoDto
{
    [JsonPropertyName("sys_name")]
    public string? SysName { get; set; }

    [JsonPropertyName("user_console_title")]
    public string? UserConsoleTitle { get; set; }

    [JsonPropertyName("admin_console_title")]
    public string? AdminConsoleTitle { get; set; }

    [JsonPropertyName("footer_link")]
    public string? FooterLink { get; set; }

    [JsonPropertyName("footer_copyright")]
    public string? FooterCopyright { get; set; }

    [JsonPropertyName("favicon_file")]
    public string? FaviconFile { get; set; }

    [JsonPropertyName("logo_file")]
    public string? LogoFile { get; set; }

    [JsonPropertyName("login_ad_file")]
    public string? LoginAdFile { get; set; }

    [JsonPropertyName("enable_email_login")]
    public bool EnableEmailLogin { get; set; }

    [JsonPropertyName("enable_sms_login")]
    public bool EnableSmsLogin { get; set; }

    [JsonPropertyName("allow_register")]
    public bool AllowRegister { get; set; }
}
