using SqlSugar;

namespace Cnn.Domain.Entities;

[SugarTable("stream")]
public class Stream
{
    [SugarColumn(IsPrimaryKey = true, IsIdentity = true)]
    public int Id { get; set; }

    public int? Uid { get; set; }

    public int? UserPackage { get; set; }

    public int? RegionId { get; set; }

    public int? NodeGroupId { get; set; }

    public int? BackupNodeGroup { get; set; }

    public bool? EnableBackupGroup { get; set; }

    public string? CnameDomain { get; set; }

    public string? CnameHostname2 { get; set; }

    public string? CnameMode { get; set; }

    public string? CnameHostname { get; set; }

    public string? Listen { get; set; }

    public string? BalanceWay { get; set; }

    public bool? ProxyProtocol { get; set; }

    public string? BackendPort { get; set; }

    public string? Backend { get; set; }

    public string? ConnLimit { get; set; }

    public string? Acl { get; set; }

    public DateTime? CreateAt { get; set; }

    public DateTime? UpdateAt { get; set; }

    public int? Version { get; set; }

    public bool? Enable { get; set; }

    public long? TaskId { get; set; }

    public long? CnameTaskId { get; set; }

    public string? RecordId { get; set; }

    public string? State { get; set; }
}