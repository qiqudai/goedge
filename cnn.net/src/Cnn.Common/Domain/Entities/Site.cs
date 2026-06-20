using SqlSugar;

namespace Cnn.Domain.Entities;

[SugarTable("site")]
public class Site
{
    [SugarColumn(IsPrimaryKey = true, IsIdentity = true)]
    public int Id { get; set; }

    public int? Uid { get; set; }

    public int? UserPackage { get; set; }

    public int? RegionId { get; set; }

    public int? NodeGroupId { get; set; }

    public int? BackupNodeGroup { get; set; }

    public bool? EnableBackupGroup { get; set; }

    public int? DnsProviderId { get; set; }

    public string? PlatformDnsRecordId { get; set; }

    public string? UserDnsRecordId { get; set; }

    public string? CnameDomain { get; set; }

    public string? CnameHostname2 { get; set; }

    public string? CnameMode { get; set; }

    public string? CnameHostname { get; set; }

    public string? Domain { get; set; }

    public string? HttpListen { get; set; }

    public string? HttpsListen { get; set; }

    [SugarColumn(IsIgnore = true)]
    public int? CertId { get; set; }

    public string? BalanceWay { get; set; }

    public string? Backend { get; set; }

    public string? BackendProtocol { get; set; }

    public string? BackendHttpsPort { get; set; }

    public string? BackendHttpPort { get; set; }

    public string? ProxyTimeout { get; set; }

    public bool? BackendPortMapping { get; set; }

    public string? HealthCheck { get; set; }

    public bool? UpsKeepalive { get; set; }

    public int? UpsKeepaliveConn { get; set; }

    public int? UpsKeepaliveTimeout { get; set; }

    public string? ProxyHttpVersion { get; set; }

    public string? ProxySslProtocols { get; set; }

    public string? BackendHost { get; set; }

    public bool? Range { get; set; }

    public string? ProxyCache { get; set; }

    public int? CcDefaultRule { get; set; }

    public string? CcSwitch { get; set; }

    public string? ExtraCcRule { get; set; }

    public bool? BlockProxy { get; set; }

    public string? BlockRegion { get; set; }

    public string? BlackIp { get; set; }

    public string? WhiteIp { get; set; }

    public string? SpiderAllow { get; set; }

    public int? Acl { get; set; }

    public string? Hotlink { get; set; }

    public string? Cors { get; set; }

    public string? RespHeader { get; set; }

    public string? ReqHeader { get; set; }

    [SugarColumn(ColumnName = "page_404")]
    public string? Page404 { get; set; }

    [SugarColumn(ColumnName = "page_50x")]
    public string? Page50x { get; set; }

    public string? UrlRewrite { get; set; }

    public bool? GzipEnable { get; set; }

    public string? GzipTypes { get; set; }

    public bool? WebsocketEnable { get; set; }

    public bool? AcmeProxyToOrgin { get; set; }

    public int? PostSizeLimit { get; set; }

    public DateTime? CreateAt { get; set; }

    public DateTime? UpdateAt { get; set; }

    public int? Version { get; set; }

    public bool? Enable { get; set; }

    public long? TaskId { get; set; }

    public long? CnameTaskId { get; set; }

    public string? RecordId { get; set; }

    public string? State { get; set; }
}
