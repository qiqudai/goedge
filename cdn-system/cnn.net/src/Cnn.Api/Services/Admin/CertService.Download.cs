using System.IO;
using System.IO.Compression;
using System.Text;
using Cnn.Common.Contracts;
using Cnn.Domain.Entities;
using SqlSugar;
using Task = System.Threading.Tasks.Task;

namespace Cnn.Api.Services.Admin;

public sealed partial class CertService
{
    public async Task<ServiceResult<CertDownloadPayload>> DownloadAsync(
        long id,
        long? userId,
        bool isAdmin,
        string? domain,
        CancellationToken cancellationToken)
    {
        if (id <= 0)
        {
            return ServiceResult<CertDownloadPayload>.Fail(ErrorCodes.InvalidParam, "cert_invalid_id");
        }

        var certQuery = _db.Queryable<Cert>().Where(c => c.Id == id);
        long uid = 0;
        if (!isAdmin)
        {
            uid = userId ?? 0;
            if (uid <= 0)
            {
                return ServiceResult<CertDownloadPayload>.Fail(ErrorCodes.PermissionDenied);
            }
            certQuery = certQuery.Where(c => c.Uid == (int)uid);
        }

        var cert = await certQuery.FirstAsync();
        if (cert == null)
        {
            return ServiceResult<CertDownloadPayload>.Fail(ErrorCodes.NotFound, "cert_not_found");
        }

        var domainKey = NormalizeCertDomainKey(domain);
        if (string.IsNullOrWhiteSpace(domainKey))
        {
            domainKey = NormalizeCertDomainKey(cert.Domain);
        }
        if (string.IsNullOrWhiteSpace(domainKey))
        {
            return ServiceResult<CertDownloadPayload>.Fail(ErrorCodes.MissingParam, "cert_domain_required");
        }

        var certs = await LoadCertsByDomainAsync(domainKey, uid, cancellationToken);
        if (certs.Count == 0)
        {
            return ServiceResult<CertDownloadPayload>.Fail(ErrorCodes.NotFound, "cert_not_found");
        }

        var payload = BuildCertZip(domainKey, certs);
        return ServiceResult<CertDownloadPayload>.Ok(payload);
    }

    private async Task<IReadOnlyList<Cert>> LoadCertsByDomainAsync(string domainKey, long uid, CancellationToken cancellationToken)
    {
        if (string.IsNullOrWhiteSpace(domainKey))
        {
            return Array.Empty<Cert>();
        }

        var query = _db.Queryable<Cert>().Where(c => SqlFunc.Contains(c.Domain, domainKey));
        if (uid > 0)
        {
            query = query.Where(c => c.Uid == (int)uid);
        }

        var list = await query.OrderBy(c => c.Id, OrderByType.Asc).ToListAsync();
        var result = new List<Cert>();
        foreach (var cert in list)
        {
            if (CertDomainMatches(domainKey, cert.Domain))
            {
                result.Add(cert);
            }
        }
        return result;
    }

    private CertDownloadPayload BuildCertZip(string domainKey, IReadOnlyList<Cert> certs)
    {
        var safeDomain = SanitizeFilename(domainKey);
        if (string.IsNullOrWhiteSpace(safeDomain))
        {
            safeDomain = "certs";
        }

        using var ms = new MemoryStream();
        using (var zip = new ZipArchive(ms, ZipArchiveMode.Create, true))
        {
            foreach (var cert in certs)
            {
                var baseName = SanitizeFilename($"{safeDomain}_{cert.Id}_{NormalizeCertType(cert.Type)}");
                if (string.IsNullOrWhiteSpace(baseName))
                {
                    baseName = $"cert_{cert.Id}";
                }

                var certPem = (cert.CertPem ?? string.Empty).Trim();
                var keyPem = DecryptKey(cert.Key)?.Trim() ?? string.Empty;

                WriteZipEntry(zip, baseName + ".pem", certPem + "\n");
                WriteZipEntry(zip, baseName + ".key", keyPem + "\n");
            }
        }

        var bytes = ms.ToArray();
        var fileName = Path.GetFileName(safeDomain + ".zip");
        return new CertDownloadPayload(bytes, fileName);
    }

    private static void WriteZipEntry(ZipArchive archive, string name, string content)
    {
        var entry = archive.CreateEntry(name);
        using var stream = entry.Open();
        using var writer = new StreamWriter(stream, Encoding.UTF8, leaveOpen: false);
        writer.Write(content);
    }
}


