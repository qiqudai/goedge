using System.Text.Json;
using Cnn.Common.Contracts;
using Cnn.Api.Services.Common;
using Cnn.Domain.Entities;
using SqlSugar;
using Task = System.Threading.Tasks.Task;

namespace Cnn.Api.Services.Auth;

public interface IAuthService
{
    Task<ServiceResult<object>> LoginAsync(LoginRequest request, string clientIp, string? requestHost, CancellationToken cancellationToken);
    Task<ServiceResult<object>> SendLoginCaptchaAsync(LoginCaptchaRequest request, string clientIp, CancellationToken cancellationToken);
    Task<ServiceResult<object>> RegisterAsync(RegisterRequest request, CancellationToken cancellationToken);
}

public sealed class AuthService : IAuthService
{
    private const string CaptchaTemplateKey = "email_captcha_templ";
    private const string AllowRegisterKey = "allow_register";
    private const string AllowEmailCaptchaKey = "allow-enable-email-captcha-login";
    private const string AllowSmsCaptchaKey = "allow-enable-sms-captcha-login";
    private const string LoginSessionKey = "login_session_valid_time";
    private const string BindMasterHostKey = "bind-master-host";
    private const string LimitAdminLoginKey = "limit_admin_login_domain";
    private const string LimitUserLoginKey = "limit_user_login_domain";

    private static readonly InMemoryLimiter LoginLimiter = new(5, TimeSpan.FromMinutes(5), TimeSpan.FromMinutes(10));
    private static readonly InMemoryLimiter CaptchaLimiter = new(3, TimeSpan.FromMinutes(10), TimeSpan.FromMinutes(30));

    private static readonly JsonSerializerOptions JsonOptions = new(JsonSerializerDefaults.Web)
    {
        PropertyNameCaseInsensitive = true
    };

    private readonly ISqlSugarClient _db;
    private readonly ISystemConfigService _systemConfigService;
    private readonly ICaptchaService _captchaService;
    private readonly IEmailService _emailService;
    private readonly IAuthTokenService _tokenService;

    public AuthService(
        ISqlSugarClient db,
        ISystemConfigService systemConfigService,
        ICaptchaService captchaService,
        IEmailService emailService,
        IAuthTokenService tokenService)
    {
        _db = db;
        _systemConfigService = systemConfigService;
        _captchaService = captchaService;
        _emailService = emailService;
        _tokenService = tokenService;
    }

    public async Task<ServiceResult<object>> LoginAsync(LoginRequest request, string clientIp, string? requestHost, CancellationToken cancellationToken)
    {
        if (request == null || string.IsNullOrWhiteSpace(request.Username) || string.IsNullOrWhiteSpace(request.Password))
        {
            await WriteLoginLogAsync(0, clientIp, false, "invalid request", cancellationToken);
            return ServiceResult<object>.Fail(ErrorCodes.InvalidParam);
        }

        var (allowed, cooldown) = LoginLimiter.Allow(BuildLimiterKey("login", request.Username, clientIp));
        if (!allowed)
        {
            await WriteLoginLogAsync(0, clientIp, false, "rate limited", cancellationToken);
            return ServiceResult<object>.FailWithData(
                ErrorCodes.RateLimited,
                new RateLimitPayload { RateLimited = true, RateCooldown = (int)Math.Ceiling(cooldown.TotalSeconds) },
                "rate_limited");
        }

        var user = await _db.Queryable<User>()
            .Where(u => u.Name == request.Username || u.Email == request.Username)
            .FirstAsync();

        if (user == null)
        {
            await WriteLoginLogAsync(0, clientIp, false, "user not found: " + request.Username, cancellationToken);
            return ServiceResult<object>.Fail(ErrorCodes.AuthInvalid, "invalid_credentials");
        }

        if (user.Enable != true)
        {
            await WriteLoginLogAsync(user.Id, clientIp, false, "user disabled", cancellationToken);
            return ServiceResult<object>.Fail(ErrorCodes.PermissionDenied, "user_disabled");
        }

        var role = user.Type == 1 ? "admin" : "user";
        if (!await IsLoginHostAllowedAsync(role, requestHost, cancellationToken))
        {
            await WriteLoginLogAsync(0, clientIp, false, "login host not allowed", cancellationToken);
            return ServiceResult<object>.Fail(ErrorCodes.AuthInvalid, "invalid_credentials");
        }

        var providedHashed = string.Equals(request.PasswordHash?.Trim(), "sha256", StringComparison.OrdinalIgnoreCase);
        if (providedHashed && !PasswordHasher.PasswordLooksHashed(request.Password))
        {
            await WriteLoginLogAsync(0, clientIp, false, "invalid password hash", cancellationToken);
            return ServiceResult<object>.Fail(ErrorCodes.InvalidParam);
        }
        if (!providedHashed)
        {
            providedHashed = PasswordHasher.PasswordLooksHashed(request.Password);
        }

        var (ok, upgrade) = PasswordHasher.VerifyPassword(user.Password, request.Password, providedHashed);
        if (!ok)
        {
            await WriteLoginLogAsync(user.Id, clientIp, false, "password mismatch", cancellationToken);
            return ServiceResult<object>.Fail(ErrorCodes.AuthInvalid, "invalid_credentials");
        }

        if (upgrade)
        {
            var hashed = PasswordHasher.HashPasswordForStorage(request.Password);
            if (!string.IsNullOrWhiteSpace(hashed))
            {
                await _db.Updateable<User>()
                    .SetColumns(u => new User { Password = hashed })
                    .Where(u => u.Id == user.Id)
                    .ExecuteCommandAsync();
            }
        }

        var (needCaptcha, captchaType) = await ResolveLoginCaptchaRequirementAsync(user, request.CaptchaType, cancellationToken);
        if (needCaptcha)
        {
            if (captchaType == "email" && string.IsNullOrWhiteSpace(user.Email))
            {
                await WriteLoginLogAsync(user.Id, clientIp, false, "email missing for captcha", cancellationToken);
                return ServiceResult<object>.Fail(ErrorCodes.InvalidParam, "email_required_for_captcha");
            }

            if (captchaType == "sms" && string.IsNullOrWhiteSpace(user.Phone))
            {
                await WriteLoginLogAsync(user.Id, clientIp, false, "phone missing for captcha", cancellationToken);
                return ServiceResult<object>.Fail(ErrorCodes.InvalidParam, "phone_required_for_captcha");
            }

            var email = captchaType == "email" ? user.Email : string.Empty;
            var phone = captchaType == "sms" ? user.Phone : string.Empty;
            if (!await _captchaService.VerifyAsync(email, phone, request.Captcha, cancellationToken))
            {
                await WriteLoginLogAsync(user.Id, clientIp, false, "captcha mismatch", cancellationToken);
                return ServiceResult<object>.Fail(ErrorCodes.AuthInvalid, "invalid_captcha");
            }
        }

        var ttl = await ResolveLoginSessionTtlAsync(cancellationToken);
        var token = _tokenService.GenerateToken(user.Id, role, ttl);
        await WriteLoginLogAsync(user.Id, clientIp, true, "ok", cancellationToken);

        var payload = new LoginResponse
        {
            Token = token,
            Role = role,
            Uid = user.Id,
            Name = user.Name
        };

        return ServiceResult<object>.Ok(payload);
    }

    public async Task<ServiceResult<object>> SendLoginCaptchaAsync(LoginCaptchaRequest request, string clientIp, CancellationToken cancellationToken)
    {
        if (request == null || string.IsNullOrWhiteSpace(request.Username))
        {
            return ServiceResult<object>.Fail(ErrorCodes.InvalidParam);
        }

        var (allowed, cooldown) = CaptchaLimiter.Allow(BuildLimiterKey("captcha", request.Username, clientIp));
        if (!allowed)
        {
            return ServiceResult<object>.FailWithData(
                ErrorCodes.RateLimited,
                new RateLimitPayload { RateLimited = true, RateCooldown = (int)Math.Ceiling(cooldown.TotalSeconds) },
                "rate_limited");
        }

        var user = await _db.Queryable<User>()
            .Where(u => u.Name == request.Username || u.Email == request.Username)
            .FirstAsync();

        if (user == null)
        {
            return ServiceResult<object>.Fail(ErrorCodes.AuthInvalid, "invalid_credentials");
        }

        if (user.Enable != true)
        {
            return ServiceResult<object>.Fail(ErrorCodes.PermissionDenied, "user_disabled");
        }

        var (needCaptcha, captchaType) = await ResolveLoginCaptchaRequirementAsync(user, request.Type, cancellationToken);
        if (!needCaptcha)
        {
            return ServiceResult<object>.Fail(ErrorCodes.InvalidParam, "captcha_not_enabled");
        }

        var code = _captchaService.GenerateCode(6);
        if (captchaType == "email")
        {
            var email = user.Email?.Trim();
            if (string.IsNullOrWhiteSpace(email))
            {
                return ServiceResult<object>.Fail(ErrorCodes.InvalidParam, "email_required_for_captcha");
            }

            var (title, body) = await ResolveEmailCaptchaTemplateAsync(code, user.Name, cancellationToken);
            var sent = await _emailService.SendAsync(email, title, body, cancellationToken);
            if (!sent)
            {
                return ServiceResult<object>.Fail(ErrorCodes.ExternalProviderError, "email_send_failed");
            }

            var stored = await _captchaService.StoreAsync(email, string.Empty, clientIp, code, cancellationToken);
            if (!stored)
            {
                return ServiceResult<object>.Fail(ErrorCodes.InternalError, "captcha_store_failed");
            }

            return ServiceResult<object>.Ok(null!);
        }

        if (captchaType == "sms")
        {
            var phone = user.Phone?.Trim();
            if (string.IsNullOrWhiteSpace(phone))
            {
                return ServiceResult<object>.Fail(ErrorCodes.InvalidParam, "phone_required_for_captcha");
            }

            return ServiceResult<object>.Fail(ErrorCodes.ConfigError, "sms_not_configured");
        }

        return ServiceResult<object>.Fail(ErrorCodes.InvalidParam);
    }

    public async Task<ServiceResult<object>> RegisterAsync(RegisterRequest request, CancellationToken cancellationToken)
    {
        if (request == null || string.IsNullOrWhiteSpace(request.Username) || string.IsNullOrWhiteSpace(request.Password))
        {
            return ServiceResult<object>.Fail(ErrorCodes.InvalidParam);
        }

        var cfg = await _systemConfigService.LoadSystemConfigAsync(cancellationToken);
        if (!cfg.TryGetValue(AllowRegisterKey, out var allowRaw) || !_systemConfigService.ParseBoolFlag(allowRaw))
        {
            return ServiceResult<object>.Fail(ErrorCodes.PermissionDenied, "register_disabled");
        }

        var username = request.Username.Trim();
        var password = request.Password.Trim();
        if (string.Equals(request.PasswordHash?.Trim(), "sha256", StringComparison.OrdinalIgnoreCase) &&
            !PasswordHasher.PasswordLooksHashed(password))
        {
            return ServiceResult<object>.Fail(ErrorCodes.InvalidParam);
        }

        var exists = await _db.Queryable<User>()
            .Where(u => u.Name == username || u.Email == request.Email)
            .AnyAsync();
        if (exists)
        {
            return ServiceResult<object>.Fail(ErrorCodes.AlreadyExists, "user_exists");
        }

        var hashed = PasswordHasher.HashPasswordForStorage(password);
        if (string.IsNullOrWhiteSpace(hashed))
        {
            return ServiceResult<object>.Fail(ErrorCodes.InternalError, "user_create_failed");
        }

        var user = new User
        {
            Name = username,
            Email = request.Email?.Trim(),
            Phone = request.Phone?.Trim(),
            Password = hashed,
            Enable = true,
            Type = 2,
            CreateAt = DateTime.Now
        };

        var rows = await _db.Insertable(user).ExecuteCommandAsync();
        if (rows <= 0)
        {
            return ServiceResult<object>.Fail(ErrorCodes.InternalError, "user_create_failed");
        }

        return ServiceResult<object>.Ok(null!);
    }

    private async Task<(bool NeedCaptcha, string Type)> ResolveLoginCaptchaRequirementAsync(User user, string? requestedType, CancellationToken cancellationToken)
    {
        var cfg = await _systemConfigService.LoadSystemConfigAsync(cancellationToken);
        var emailEnabled = _systemConfigService.ParseBoolFlag(Get(cfg, AllowEmailCaptchaKey));
        var smsEnabled = _systemConfigService.ParseBoolFlag(Get(cfg, AllowSmsCaptchaKey));
        if (!emailEnabled && !smsEnabled)
        {
            return (false, string.Empty);
        }

        requestedType = requestedType?.Trim().ToLowerInvariant() ?? string.Empty;
        if (requestedType == "email" && emailEnabled)
        {
            return (true, "email");
        }
        if (requestedType == "sms" && smsEnabled)
        {
            return (true, "sms");
        }

        var pref = user.LoginCaptcha?.Trim().ToLowerInvariant() ?? string.Empty;
        if (pref == "email" && emailEnabled)
        {
            return (true, "email");
        }
        if (pref == "sms" && smsEnabled)
        {
            return (true, "sms");
        }

        if (emailEnabled)
        {
            return (true, "email");
        }
        if (smsEnabled)
        {
            return (true, "sms");
        }

        return (false, string.Empty);
    }

    private async Task<TimeSpan> ResolveLoginSessionTtlAsync(CancellationToken cancellationToken)
    {
        var cfg = await _systemConfigService.LoadSystemConfigAsync(cancellationToken);
        if (!cfg.TryGetValue(LoginSessionKey, out var raw))
        {
            return TimeSpan.FromHours(24);
        }

        raw = raw?.Trim();
        if (string.IsNullOrWhiteSpace(raw) || !int.TryParse(raw, out var seconds) || seconds <= 0)
        {
            return TimeSpan.FromHours(24);
        }

        return TimeSpan.FromSeconds(seconds);
    }

    private async Task<(string Title, string Body)> ResolveEmailCaptchaTemplateAsync(string code, string? username, CancellationToken cancellationToken)
    {
        var title = "验证码";
        var body = "您的验证码是 " + code;

        var cfg = await _systemConfigService.LoadSystemConfigAsync(cancellationToken);
        if (!cfg.TryGetValue(CaptchaTemplateKey, out var raw))
        {
            return (title, body);
        }

        raw = raw?.Trim();
        if (string.IsNullOrWhiteSpace(raw))
        {
            return (title, body);
        }

        try
        {
            var payload = JsonSerializer.Deserialize<CaptchaTemplate>(raw, JsonOptions);
            if (payload != null)
            {
                if (!string.IsNullOrWhiteSpace(payload.Title))
                {
                    title = payload.Title.Trim();
                }
                if (!string.IsNullOrWhiteSpace(payload.Data))
                {
                    body = payload.Data.Trim();
                }
            }
        }
        catch
        {
            return (title, body);
        }

        username = username ?? string.Empty;
        body = body.Replace("{{captcha}}", code, StringComparison.OrdinalIgnoreCase)
            .Replace("{{code}}", code, StringComparison.OrdinalIgnoreCase)
            .Replace("{{username}}", username, StringComparison.OrdinalIgnoreCase);

        return (title, body);
    }

    private async Task<bool> IsLoginHostAllowedAsync(string role, string? requestHost, CancellationToken cancellationToken)
    {
        var cfg = await _systemConfigService.LoadSystemConfigAsync(cancellationToken);
        var host = NormalizeHost(requestHost);
        if (string.IsNullOrWhiteSpace(host))
        {
            return true;
        }

        var bindHosts = SplitHostList(Get(cfg, BindMasterHostKey));
        var limitValue = role == "admin" ? Get(cfg, LimitAdminLoginKey) : Get(cfg, LimitUserLoginKey);
        return HostAllowedByLimit(host, limitValue, bindHosts);
    }

    private static string BuildLimiterKey(string prefix, string username, string clientIp)
    {
        var userPart = (username ?? string.Empty).Trim().ToLowerInvariant();
        var ipPart = (clientIp ?? string.Empty).Trim();
        return $"{prefix}|{ipPart}|{userPart}";
    }

    private async Task WriteLoginLogAsync(long userId, string clientIp, bool success, string postContent, CancellationToken cancellationToken)
    {
        var log = new LoginLog
        {
            Uid = userId > 0 ? (int?)userId : null,
            Ip = clientIp,
            Success = success,
            PostContent = postContent,
            CreateAt = DateTime.Now
        };

        await _db.Insertable(log).ExecuteCommandAsync();
    }

    private static string? Get(Dictionary<string, string> map, string key)
    {
        return map.TryGetValue(key, out var value) ? value : null;
    }

    private static List<string> SplitHostList(string? raw)
    {
        raw = raw?.Trim();
        if (string.IsNullOrWhiteSpace(raw))
        {
            return new List<string>();
        }

        var parts = raw.Split(new[] { ' ', '\n', '\r', '\t', ',', ';' }, StringSplitOptions.RemoveEmptyEntries);
        var list = new List<string>();
        var seen = new HashSet<string>(StringComparer.OrdinalIgnoreCase);
        foreach (var part in parts)
        {
            var host = NormalizeHost(part);
            if (string.IsNullOrWhiteSpace(host))
            {
                continue;
            }

            if (seen.Add(host))
            {
                list.Add(host);
            }
        }

        return list;
    }

    private static string NormalizeHost(string? raw)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            return string.Empty;
        }

        var host = raw.Trim().ToLowerInvariant();
        if (host.StartsWith("[", StringComparison.Ordinal))
        {
            var idx = host.IndexOf(']');
            if (idx >= 0)
            {
                return host[..(idx + 1)];
            }
        }

        var colon = host.IndexOf(':');
        if (colon >= 0)
        {
            host = host[..colon];
        }

        host = host.TrimEnd('.');
        return host;
    }

    private static bool HostAllowedByLimit(string host, string? limitValue, List<string> bindHosts)
    {
        if (string.IsNullOrWhiteSpace(limitValue))
        {
            return true;
        }

        host = NormalizeHost(host);
        if (string.IsNullOrWhiteSpace(host))
        {
            return true;
        }

        var limits = SplitHostList(limitValue);
        if (limits.Count == 0)
        {
            return true;
        }

        var hasDot = limits.Any(item => item.Contains('.'));
        if (hasDot)
        {
            return limits.Any(item => string.Equals(host, item, StringComparison.OrdinalIgnoreCase));
        }

        if (bindHosts.Count == 0)
        {
            return true;
        }

        foreach (var prefix in limits)
        {
            var normalized = prefix.TrimEnd('.');
            if (string.IsNullOrWhiteSpace(normalized))
            {
                continue;
            }

            foreach (var baseHost in bindHosts)
            {
                if (string.IsNullOrWhiteSpace(baseHost))
                {
                    continue;
                }

                if (string.Equals(host, normalized + "." + baseHost, StringComparison.OrdinalIgnoreCase))
                {
                    return true;
                }
            }
        }

        return false;
    }

    private sealed class CaptchaTemplate
    {
        public string? Title { get; set; }
        public string? Data { get; set; }
    }
}
