namespace Cnn.Api.Services.Deletion;

public sealed class CertificateDeletionGuard : IDeletionGuard
{
    private readonly SqlSugar.ISqlSugarClient _db;

    public CertificateDeletionGuard(SqlSugar.ISqlSugarClient db)
    {
        _db = db;
    }

    public string ResourceType => ResourceTypes.Certificate;

    public async Task<DeleteGuardResult> CheckAsync(long resourceId, CancellationToken cancellationToken)
    {
        if (resourceId <= 0)
        {
            return DeleteGuardResult.Deny("INVALID_RESOURCE_ID", "证书 ID 无效");
        }

        var refs = await CertificateUsageInspector.FindSiteUsagesAsync(_db, resourceId, cancellationToken);

        if (refs.Count == 0)
        {
            return DeleteGuardResult.Allow();
        }

        var items = refs.Select(r => new DeleteReferenceItem
        {
            ResourceType = ResourceTypes.Site,
            ResourceId = r.Id,
            DisplayName = $"{r.Username} / {r.PrimaryDomain}",
            Relation = "site.certificate"
        }).ToList();

        return DeleteGuardResult.Deny(
            "CERTIFICATE_IN_USE",
            "证书仍被站点绑定，请先解绑站点证书或删除站点后再删除。",
            items);
    }
}
