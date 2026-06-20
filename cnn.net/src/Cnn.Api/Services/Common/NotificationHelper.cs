using System.Text.Json;
using Cnn.Domain.Entities;
using SqlSugar;
using Task = System.Threading.Tasks.Task;

namespace Cnn.Api.Services.Common;

internal static class NotificationHelper
{
    private static readonly Dictionary<string, string> NotifyConfigKeyMap = new(StringComparer.OrdinalIgnoreCase)
    {
        ["traffic-exceed"] = "traffic-exceed-notify",
        ["traffic-exceeding"] = "traffic-exceeding-notify",
        ["package-expire"] = "package-expire-notify",
        ["package-expiring"] = "package-expiring-notify",
        ["cc-switch"] = "cc-switch-notify",
        ["bandwidth-exceed"] = "bandwidth-exceed-notify",
        ["conn-exceed"] = "conn-exceed-notify",
        ["cert-expire"] = "cert-expire-notify",
        ["cert-expiring"] = "cert-expiring-notify",
        ["account-auth2"] = "account-auth2-notify"
    };

    public static async Task CreateUserMessageAsync(
        ISqlSugarClient db,
        ISystemConfigService systemConfigService,
        long userId,
        string msgType,
        string title,
        string content,
        long userPackageId,
        long siteId,
        CancellationToken cancellationToken)
    {
        if (userId <= 0 || string.IsNullOrWhiteSpace(msgType))
        {
            return;
        }

        var now = DateTime.Now;
        var config = await systemConfigService.LoadSystemConfigAsync(cancellationToken);
        var policy = ResolveNotifyPolicy(config, msgType, now, systemConfigService);
        if (!policy.Enabled)
        {
            return;
        }

        var (phone, email, ok) = await LookupMessageSubscriptionAsync(db, userId, msgType);
        if (!ok)
        {
            return;
        }

        var allowEmail = email && policy.AllowEmail && policy.InPeriod;
        var allowPhone = phone && policy.AllowSms && policy.InPeriod;
        allowPhone = false;

        var message = new Message
        {
            Type = msgType,
            Receive = (int)userId,
            Title = title,
            Content = content,
            PhoneContent = content,
            UserPackageId = (int)userPackageId,
            SiteId = (int)siteId,
            IsShow = true,
            EmailNeedSend = allowEmail,
            PhoneNeedSend = allowPhone,
            CreateAt = now,
            UpdateAt = now
        };

        await db.Insertable(message).ExecuteCommandAsync();
    }

    private static async Task<(bool Phone, bool Email, bool Ok)> LookupMessageSubscriptionAsync(
        ISqlSugarClient db,
        long userId,
        string msgType)
    {
        if (userId <= 0 || string.IsNullOrWhiteSpace(msgType))
        {
            return (false, false, false);
        }

        var sub = await db.Queryable<MessageSub>()
            .Where(s => s.Uid == userId && s.MsgType == msgType)
            .FirstAsync();
        if (sub != null)
        {
            return (sub.Phone ?? false, sub.Email ?? false, true);
        }

        var count = await db.Queryable<MessageSub>()
            .Where(s => s.Uid == userId)
            .CountAsync();
        if (count == 0)
        {
            return (true, true, true);
        }

        return (false, false, true);
    }

    private static NotifyPolicy ResolveNotifyPolicy(
        IReadOnlyDictionary<string, string> cfg,
        string msgType,
        DateTime now,
        ISystemConfigService systemConfigService)
    {
        var policy = new NotifyPolicy
        {
            Enabled = true,
            AllowEmail = true,
            AllowSms = true,
            InPeriod = IsWithinNotificationPeriod(cfg, now)
        };

        var (globalEmail, globalSms) = ParseGlobalNotifyMethods(cfg.TryGetValue("notify-method", out var raw) ? raw : string.Empty);
        if (!globalEmail)
        {
            policy.AllowEmail = false;
        }
        if (!globalSms)
        {
            policy.AllowSms = false;
        }
        policy.AllowSms = false;

        if (!NotifyConfigKeyMap.TryGetValue(msgType, out var key))
        {
            return policy;
        }

        if (!cfg.TryGetValue(key, out var configRaw) || string.IsNullOrWhiteSpace(configRaw))
        {
            return policy;
        }

        var item = ParseNotifyItemConfig(configRaw, systemConfigService);
        if (!item.Enabled)
        {
            policy.Enabled = false;
            return policy;
        }

        if (item.Methods.Count > 0)
        {
            policy.AllowEmail = item.Methods.TryGetValue("email", out var email) && email && policy.AllowEmail;
            policy.AllowSms = item.Methods.TryGetValue("sms", out var sms) && sms && policy.AllowSms;
        }

        return policy;
    }

    private static ParsedNotifyItem ParseNotifyItemConfig(string raw, ISystemConfigService systemConfigService)
    {
        var item = new ParsedNotifyItem
        {
            Enabled = true,
            Methods = new Dictionary<string, bool>(StringComparer.OrdinalIgnoreCase)
        };

        try
        {
            using var doc = JsonDocument.Parse(raw);
            if (doc.RootElement.ValueKind != JsonValueKind.Object)
            {
                return item;
            }

            var root = doc.RootElement;
            var hasModern = root.TryGetProperty("enable", out _) ||
                            root.TryGetProperty("methods", out _) ||
                            root.TryGetProperty("email_template", out _) ||
                            root.TryGetProperty("sms_template", out _);

            if (hasModern)
            {
                if (root.TryGetProperty("enable", out var enableValue))
                {
                    item.Enabled = ParseBoolValue(enableValue, systemConfigService);
                }

                if (root.TryGetProperty("methods", out var methods) && methods.ValueKind == JsonValueKind.Array)
                {
                    foreach (var entry in methods.EnumerateArray())
                    {
                        if (entry.ValueKind == JsonValueKind.String)
                        {
                            var method = entry.GetString();
                            if (!string.IsNullOrWhiteSpace(method))
                            {
                                item.Methods[method.Trim()] = true;
                            }
                        }
                    }
                }

                return item;
            }

            if (root.TryGetProperty("state", out var stateValue))
            {
                item.Enabled = ParseBoolValue(stateValue, systemConfigService);
            }

            item.Methods["email"] = true;
            item.Methods["sms"] = true;
            return item;
        }
        catch
        {
            return item;
        }
    }

    private static bool ParseBoolValue(JsonElement value, ISystemConfigService systemConfigService)
    {
        switch (value.ValueKind)
        {
            case JsonValueKind.True:
                return true;
            case JsonValueKind.False:
                return false;
            case JsonValueKind.Number:
                return value.TryGetInt64(out var number) && number != 0;
            case JsonValueKind.String:
                return systemConfigService.ParseBoolFlag(value.GetString());
            default:
                return false;
        }
    }

    private static (bool Email, bool Sms) ParseGlobalNotifyMethods(string raw)
    {
        raw = raw?.Trim() ?? string.Empty;
        if (string.IsNullOrWhiteSpace(raw))
        {
            return (true, false);
        }

        if (raw.StartsWith("{", StringComparison.Ordinal))
        {
            try
            {
                using var doc = JsonDocument.Parse(raw);
                if (doc.RootElement.TryGetProperty("email", out var emailValue))
                {
                    var allowEmail = emailValue.ValueKind == JsonValueKind.True ||
                                     (emailValue.ValueKind == JsonValueKind.Number && emailValue.GetInt32() != 0);
                    var allowSms = false;
                    if (doc.RootElement.TryGetProperty("phone", out var phone))
                    {
                        allowSms = phone.ValueKind == JsonValueKind.True ||
                                   (phone.ValueKind == JsonValueKind.Number && phone.GetInt32() != 0);
                    }
                    return (allowEmail, allowSms);
                }
            }
            catch
            {
                return (true, false);
            }
        }

        var parts = raw.Split(new[] { ',', ';', ' ', '\n', '\t' }, StringSplitOptions.RemoveEmptyEntries);
        var email = false;
        var sms = false;
        foreach (var part in parts)
        {
            switch (part.Trim().ToLowerInvariant())
            {
                case "email":
                    email = true;
                    break;
                case "sms":
                case "phone":
                    sms = true;
                    break;
            }
        }

        if (!email && !sms)
        {
            return (true, false);
        }

        return (email, sms);
    }

    private static bool IsWithinNotificationPeriod(IReadOnlyDictionary<string, string> cfg, DateTime now)
    {
        var mode = cfg.TryGetValue("notification-period", out var raw) ? raw : string.Empty;
        mode = (mode ?? string.Empty).Trim().ToLowerInvariant();
        if (string.IsNullOrWhiteSpace(mode) || mode == "all")
        {
            return true;
        }
        if (mode != "custom")
        {
            return true;
        }

        if (!cfg.TryGetValue("notification-period-custom", out var customRaw))
        {
            return true;
        }

        var parts = customRaw.Split('-', StringSplitOptions.RemoveEmptyEntries);
        if (parts.Length != 2)
        {
            return true;
        }

        if (!int.TryParse(parts[0].Trim(), out var start) || !int.TryParse(parts[1].Trim(), out var end))
        {
            return true;
        }

        if (start < 0 || start > 23 || end < 0 || end > 23)
        {
            return true;
        }

        var hour = now.Hour;
        if (start <= end)
        {
            return hour >= start && hour <= end;
        }

        return hour >= start || hour <= end;
    }

    private sealed class NotifyPolicy
    {
        public bool Enabled { get; set; }
        public bool AllowEmail { get; set; }
        public bool AllowSms { get; set; }
        public bool InPeriod { get; set; }
    }

    private sealed class ParsedNotifyItem
    {
        public bool Enabled { get; set; }
        public Dictionary<string, bool> Methods { get; set; } = new(StringComparer.OrdinalIgnoreCase);
    }
}
