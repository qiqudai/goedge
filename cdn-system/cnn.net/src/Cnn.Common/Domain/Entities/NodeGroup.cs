using SqlSugar;

namespace Cnn.Domain.Entities;

[SugarTable("node_group")]
public class NodeGroup
{
    [SugarColumn(IsPrimaryKey = true, IsIdentity = true)]
    public int Id { get; set; }

    public int? RegionId { get; set; }

    public string? CnameHostname { get; set; }

    public string? CnameDomain { get; set; }

    public string? Name { get; set; }

    public string? Des { get; set; }

    public string? BackupSwitchType { get; set; }

    public string? BackupSwitchPolicy { get; set; }

    public DateTime? CreateAt { get; set; }

    public DateTime? UpdateAt { get; set; }
}
