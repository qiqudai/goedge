using System.Net;
using System.Net.Mail;
using System.Text.Json;
using Cnn.Api.Services.Common;

namespace Cnn.Api.Services.Auth;

public sealed class SmtpConfig
{
    public string? Ip { get; set; }
    public int Port { get; set; }
    public string? User { get; set; }
    public string? Pwd { get; set; }
    public bool UseSsl { get; set; }
    public string? ProxyIp { get; set; }
    public int ProxyPort { get; set; }
    public string? ProxyUser { get; set; }
    public string? ProxyPwd { get; set; }
    public bool ProxyState { get; set; }
}

public interface IEmailService
{
    Task<bool> SendAsync(string to, string subject, string htmlBody, CancellationToken cancellationToken);
}

public sealed class EmailService : IEmailService
{
    private static readonly JsonSerializerOptions JsonOptions = new(JsonSerializerDefaults.Web)
    {
        PropertyNameCaseInsensitive = true
    };

    private readonly ISystemConfigService _systemConfigService;

    public EmailService(ISystemConfigService systemConfigService)
    {
        _systemConfigService = systemConfigService;
    }

    public async Task<bool> SendAsync(string to, string subject, string htmlBody, CancellationToken cancellationToken)
    {
        to = to.Trim();
        if (string.IsNullOrWhiteSpace(to))
        {
            return false;
        }

        var config = await LoadConfigAsync(cancellationToken);
        if (config == null)
        {
            return false;
        }

        if (string.IsNullOrWhiteSpace(config.Ip) || config.Port <= 0)
        {
            return false;
        }

        using var message = new MailMessage
        {
            From = new MailAddress(config.User ?? string.Empty),
            Subject = subject,
            Body = htmlBody,
            IsBodyHtml = true
        };
        message.To.Add(new MailAddress(to));

        using var client = new SmtpClient(config.Ip, config.Port)
        {
            EnableSsl = config.UseSsl
        };
        if (!string.IsNullOrWhiteSpace(config.User) && !string.IsNullOrWhiteSpace(config.Pwd))
        {
            client.Credentials = new NetworkCredential(config.User, config.Pwd);
        }

        try
        {
            await client.SendMailAsync(message, cancellationToken);
            return true;
        }
        catch
        {
            return false;
        }
    }

    private async Task<SmtpConfig?> LoadConfigAsync(CancellationToken cancellationToken)
    {
        var cfg = await _systemConfigService.LoadSystemConfigAsync(cancellationToken);
        if (!cfg.TryGetValue("smtp", out var raw))
        {
            return null;
        }

        raw = raw?.Trim();
        if (string.IsNullOrWhiteSpace(raw))
        {
            return null;
        }

        try
        {
            return JsonSerializer.Deserialize<SmtpConfig>(raw, JsonOptions);
        }
        catch
        {
            return null;
        }
    }
}
