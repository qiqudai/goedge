using System.IdentityModel.Tokens.Jwt;
using System.Security.Claims;
using System.Text;
using Microsoft.Extensions.Configuration;
using Microsoft.IdentityModel.Tokens;

namespace Cnn.Api.Services.Auth;

public sealed record TokenValidationResult(
    bool Success,
    bool Expired,
    ClaimsPrincipal? Principal,
    long UserId,
    string? Role
);

public interface IAuthTokenService
{
    string GenerateToken(long userId, string role, TimeSpan ttl);
    TokenValidationResult Validate(string token);
}

public sealed class AuthTokenService : IAuthTokenService
{
    private const string DefaultIssuer = "cdn-core-api";
    private const string DefaultSecret = "YOUR_SECRET_KEY_SHOULD_BE_IN_ENV";

    private readonly JwtSecurityTokenHandler _handler = new();
    private readonly IConfiguration _configuration;

    public AuthTokenService(IConfiguration configuration)
    {
        _configuration = configuration;
    }

    public string GenerateToken(long userId, string role, TimeSpan ttl)
    {
        if (ttl <= TimeSpan.Zero)
        {
            ttl = TimeSpan.FromHours(24);
        }

        var key = new SymmetricSecurityKey(Encoding.UTF8.GetBytes(ResolveSecret()));
        var creds = new SigningCredentials(key, SecurityAlgorithms.HmacSha256);
        var claims = new List<Claim>
        {
            new("uid", userId.ToString()),
            new("role", role),
            new(ClaimTypes.Role, role),
            new(ClaimTypes.NameIdentifier, userId.ToString())
        };

        var token = new JwtSecurityToken(
            issuer: DefaultIssuer,
            audience: null,
            claims: claims,
            expires: DateTime.UtcNow.Add(ttl),
            signingCredentials: creds
        );

        return _handler.WriteToken(token);
    }

    public TokenValidationResult Validate(string token)
    {
        if (string.IsNullOrWhiteSpace(token))
        {
            return new TokenValidationResult(false, false, null, 0, null);
        }

        var parameters = new TokenValidationParameters
        {
            ValidateIssuerSigningKey = true,
            IssuerSigningKey = new SymmetricSecurityKey(Encoding.UTF8.GetBytes(ResolveSecret())),
            ValidateIssuer = false,
            ValidateAudience = false,
            ValidateLifetime = true,
            ClockSkew = TimeSpan.Zero
        };

        try
        {
            var principal = _handler.ValidateToken(token, parameters, out _);
            var role = principal.FindFirst("role")?.Value ?? principal.FindFirst(ClaimTypes.Role)?.Value;
            var uidRaw = principal.FindFirst("uid")?.Value ?? principal.FindFirst(ClaimTypes.NameIdentifier)?.Value;
            if (!long.TryParse(uidRaw, out var userId) || userId <= 0)
            {
                return new TokenValidationResult(false, false, null, 0, null);
            }

            return new TokenValidationResult(true, false, principal, userId, role);
        }
        catch (SecurityTokenExpiredException)
        {
            return new TokenValidationResult(false, true, null, 0, null);
        }
        catch
        {
            return new TokenValidationResult(false, false, null, 0, null);
        }
    }

    private string ResolveSecret()
    {
        var env = Environment.GetEnvironmentVariable("JWT_SECRET");
        if (!string.IsNullOrWhiteSpace(env))
        {
            return env.Trim();
        }

        var cfg = _configuration["Jwt:Secret"];
        if (!string.IsNullOrWhiteSpace(cfg))
        {
            return cfg.Trim();
        }

        return DefaultSecret;
    }
}
