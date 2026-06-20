using SqlSugar;

namespace Cnn.Domain.Entities;

[SugarTable("user_package")]
public class UserPackage
{
    [SugarColumn(IsPrimaryKey = true, IsIdentity = true)]
    public int Id { get; set; }

    public int? Uid { get; set; }

    public string? Name { get; set; }

    public int? Package { get; set; }

    public int? RegionId { get; set; }

    public int? NodeGroupId { get; set; }

    public int? BackupNodeGroup { get; set; }

    public bool? EnableBackupGroup { get; set; }

    public string? CnameDomain { get; set; }

    public string? CnameHostname2 { get; set; }

    public string? CnameHostname { get; set; }

    public string? CnameMode { get; set; }

    public string? RecordId { get; set; }

    public int? Traffic { get; set; }

    public string? Bandwidth { get; set; }

    public int? Connection { get; set; }

    public int? Domain { get; set; }

    public int? MainDomainLimit { get; set; }

    public int? HttpPort { get; set; }

    public int? StreamPort { get; set; }

    public bool? CustomCcRule { get; set; }

    public bool? Websocket { get; set; }

    public bool? L2Origin { get; set; }

    public long? MonthPrice { get; set; }

    public long? QuarterPrice { get; set; }

    public long? YearPrice { get; set; }

    public DateTime? CreateAt { get; set; }

    public DateTime? StartAt { get; set; }

    public DateTime? EndAt { get; set; }

    public long? TaskId { get; set; }

    public int? Version { get; set; }

    public bool? IsExpired { get; set; }
}
