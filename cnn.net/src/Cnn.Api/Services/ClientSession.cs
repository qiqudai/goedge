using Cnn.Api.Services.Auth;

namespace Cnn.Api.Services;

public sealed class ClientSession
{
    private readonly IAuthTokenService _authService;

    public ClientSession(IAuthTokenService authService)
    {
        _authService = authService;
    }

    public string? Token { get; private set; }
    public string Role { get; private set; } = "admin";
    public string? Username { get; private set; }
    public long? UserId { get; private set; }
    public bool Initialized { get; private set; }

    public bool IsAuthenticated => !string.IsNullOrWhiteSpace(Token);
    public bool IsAdmin => string.Equals(Role, "admin", StringComparison.OrdinalIgnoreCase);

    public void Set(string? token, string? role, string? username)
    {
        Token = string.IsNullOrWhiteSpace(token) ? null : token;
        Role = string.IsNullOrWhiteSpace(role) ? "user" : role.Trim();
        Username = string.IsNullOrWhiteSpace(username) ? null : username.Trim();

        if (Token != null)
        {
            var res = _authService.Validate(Token);
            if (res.Success && res.UserId > 0)
            {
                UserId = res.UserId;
            }
            else
            {
                UserId = null;
            }
        }
        else
        {
            UserId = null;
        }
    }

    public void Clear()
    {
        Token = null;
        Role = "user";
        Username = null;
        UserId = null;
        Initialized = true;
    }

    public void MarkInitialized()
    {
        Initialized = true;
    }
}
