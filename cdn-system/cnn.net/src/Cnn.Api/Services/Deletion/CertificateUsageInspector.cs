using System.Text.Json;
using Cnn.Domain.Entities;
using SqlSugar;

namespace Cnn.Api.Services.Deletion;

internal static class CertificateUsageInspector
{
    public static async Task<IReadOnlyList<CertificateSiteUsage>> FindSiteUsagesAsync(
        ISqlSugarClient db,
        long certificateId,
        CancellationToken cancellationToken)
    {
        if (certificateId <= 0)
        {
            return Array.Empty<CertificateSiteUsage>();
        }

        var usages = new Dictionary<long, CertificateSiteUsage>();

        if (db.DbMaintenance.IsAnyTable("site") && db.DbMaintenance.IsAnyColumn("site", "cert_id"))
        {
            var directRefs = await db.Ado.SqlQueryAsync<CertificateSiteUsage>(
                """
SELECT
  s.id,
  s.domain AS primary_domain,
  u.name AS username
FROM site s
JOIN `user` u ON u.id = s.uid
WHERE s.cert_id = @certificateId
ORDER BY s.id ASC
""",
                new { certificateId });

            foreach (var item in directRefs)
            {
                usages[item.Id] = item;
            }
        }

        var settingsRows = await db.Queryable<Config>()
            .Where(c => c.Type == "site_settings" && c.ScopeName == "site" && c.ScopeId != null)
            .ToListAsync();

        var configSiteIds = settingsRows
            .Where(row => row.ScopeId.HasValue && TryExtractCertId(row.Value, out var currentCertId) && currentCertId == certificateId)
            .Select(row => row.ScopeId!.Value)
            .Distinct()
            .ToList();

        if (configSiteIds.Count == 0)
        {
            return usages.Values.OrderBy(x => x.Id).ToList();
        }

        var configRefs = await db.Ado.SqlQueryAsync<CertificateSiteUsage>(
            $"""
SELECT
  s.id,
  s.domain AS primary_domain,
  u.name AS username
FROM site s
JOIN `user` u ON u.id = s.uid
WHERE s.id IN ({string.Join(',', configSiteIds)})
ORDER BY s.id ASC
""");

        foreach (var item in configRefs)
        {
            usages[item.Id] = item;
        }

        return usages.Values.OrderBy(x => x.Id).ToList();
    }

    public static bool TryExtractCertId(string? raw, out long certId)
    {
        certId = 0;
        if (string.IsNullOrWhiteSpace(raw))
        {
            return false;
        }

        try
        {
            using var doc = JsonDocument.Parse(raw);
            if (doc.RootElement.ValueKind != JsonValueKind.Object)
            {
                return false;
            }

            if (!doc.RootElement.TryGetProperty("https", out var https))
            {
                return false;
            }

            return TryReadCertId(https, "certificate_id", out certId)
                   || TryReadCertId(https, "cert_id", out certId)
                   || TryReadCertId(https, "certId", out certId);
        }
        catch
        {
            return false;
        }
    }

    private static bool TryReadCertId(JsonElement element, string propertyName, out long certId)
    {
        certId = 0;
        if (!element.TryGetProperty(propertyName, out var rawValue))
        {
            return false;
        }

        return rawValue.ValueKind switch
        {
            JsonValueKind.Number => rawValue.TryGetInt64(out certId) && certId > 0,
            JsonValueKind.String => long.TryParse(rawValue.GetString(), out certId) && certId > 0,
            _ => false
        };
    }

    internal sealed class CertificateSiteUsage
    {
        public long Id { get; init; }
        public string PrimaryDomain { get; init; } = string.Empty;
        public string Username { get; init; } = string.Empty;
    }
}
