using System.Text.Json;

namespace Cnn.Api.Services.Common;

public static class DomainParser
{
    public static IReadOnlyList<string> ParseDomains(string? raw)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            return Array.Empty<string>();
        }

        try
        {
            using var doc = JsonDocument.Parse(raw);
            if (doc.RootElement.ValueKind == JsonValueKind.Array)
            {
                var list = new List<string>();
                foreach (var item in doc.RootElement.EnumerateArray())
                {
                    if (item.ValueKind == JsonValueKind.String)
                    {
                        var value = item.GetString();
                        if (!string.IsNullOrWhiteSpace(value))
                        {
                            list.Add(value);
                        }
                    }
                }
                return list;
            }
        }
        catch
        {
            // ignore invalid JSON
        }

        return raw.Split(new[] { '\n', '\r', ',', ' ' }, StringSplitOptions.RemoveEmptyEntries);
    }

    public static string NormalizeDomain(string? input)
    {
        if (string.IsNullOrWhiteSpace(input))
        {
            return string.Empty;
        }

        var host = input.Trim().ToLowerInvariant();
        host = host.Replace("http://", string.Empty, StringComparison.OrdinalIgnoreCase)
            .Replace("https://", string.Empty, StringComparison.OrdinalIgnoreCase);

        var slashIndex = host.IndexOfAny(new[] { '/', '?', '#' });
        if (slashIndex >= 0)
        {
            host = host[..slashIndex];
        }

        var colonIndex = host.IndexOf(':');
        if (colonIndex >= 0)
        {
            host = host[..colonIndex];
        }

        host = host.TrimStart('*', '.').TrimEnd('.');
        return host;
    }
}
