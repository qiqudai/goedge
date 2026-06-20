using System.Text.Json.Serialization;

namespace Cnn.Common.Contracts;

public sealed class UserProfileDto
{
    [JsonPropertyName("id")]
    public long Id { get; set; }

    [JsonPropertyName("name")]
    public string? Name { get; set; }

    [JsonPropertyName("email")]
    public string? Email { get; set; }

    [JsonPropertyName("phone")]
    public string? Phone { get; set; }

    [JsonPropertyName("qq")]
    public string? Qq { get; set; }

    [JsonPropertyName("balance")]
    public long? Balance { get; set; }

    [JsonPropertyName("cert_name")]
    public string? CertName { get; set; }

    [JsonPropertyName("cert_no")]
    public string? CertNo { get; set; }

    [JsonPropertyName("cert_verified")]
    public bool? CertVerified { get; set; }

    [JsonPropertyName("white_ip")]
    public string? WhiteIp { get; set; }

    [JsonPropertyName("login_captcha")]
    public string? LoginCaptcha { get; set; }

    [JsonPropertyName("create_at")]
    public string? CreateAt { get; set; }
}

public sealed class UpdateProfileRequest
{
    [JsonPropertyName("email")]
    public string? Email { get; set; }

    [JsonPropertyName("phone")]
    public string? Phone { get; set; }

    [JsonPropertyName("qq")]
    public string? Qq { get; set; }

    [JsonPropertyName("cert_name")]
    public string? CertName { get; set; }

    [JsonPropertyName("cert_no")]
    public string? CertNo { get; set; }

    [JsonPropertyName("white_ip")]
    public string? WhiteIp { get; set; }

    [JsonPropertyName("login_captcha")]
    public string? LoginCaptcha { get; set; }
}

public sealed class UpdatePasswordRequest
{
    [JsonPropertyName("current")]
    public string? Current { get; set; }

    [JsonPropertyName("next")]
    public string? Next { get; set; }

    [JsonPropertyName("password_hash")]
    public string? PasswordHash { get; set; }
}
