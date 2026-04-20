using System.Text.Json.Serialization;

namespace Cnn.Common.Contracts;

public sealed class LoginRequest
{
    [JsonPropertyName("username")]
    public string? Username { get; set; }

    [JsonPropertyName("password")]
    public string? Password { get; set; }

    [JsonPropertyName("password_hash")]
    public string? PasswordHash { get; set; }

    [JsonPropertyName("captcha")]
    public string? Captcha { get; set; }

    [JsonPropertyName("captcha_type")]
    public string? CaptchaType { get; set; }
}

public sealed class LoginCaptchaRequest
{
    [JsonPropertyName("username")]
    public string? Username { get; set; }

    [JsonPropertyName("type")]
    public string? Type { get; set; }
}

public sealed class RegisterRequest
{
    [JsonPropertyName("username")]
    public string? Username { get; set; }

    [JsonPropertyName("password")]
    public string? Password { get; set; }

    [JsonPropertyName("password_hash")]
    public string? PasswordHash { get; set; }

    [JsonPropertyName("email")]
    public string? Email { get; set; }

    [JsonPropertyName("phone")]
    public string? Phone { get; set; }
}

public sealed class LoginResponse
{
    [JsonPropertyName("token")]
    public string? Token { get; set; }

    [JsonPropertyName("role")]
    public string? Role { get; set; }

    [JsonPropertyName("uid")]
    public long Uid { get; set; }

    [JsonPropertyName("name")]
    public string? Name { get; set; }
}

public sealed class RateLimitPayload
{
    [JsonPropertyName("rate_limited")]
    public bool RateLimited { get; set; }

    [JsonPropertyName("rate_cooldown")]
    public int RateCooldown { get; set; }
}
