using SqlSugar;

namespace Cnn.Domain.Entities;

[SugarTable("cc_filter")]
public class CcFilter
{
    [SugarColumn(IsPrimaryKey = true, IsIdentity = true)]
    public int Id { get; set; }

    public int? Uid { get; set; }

    public string? Name { get; set; }

    public string? Des { get; set; }

    public string? Type { get; set; }

    public int? WithinSecond { get; set; }

    public int? MaxReq { get; set; }

    public int? MaxReqPerUri { get; set; }

    public string? Extra { get; set; }

    public DateTime? CreateAt { get; set; }

    public DateTime? UpdateAt { get; set; }

    public bool? Internal { get; set; }

    public bool? Enable { get; set; }

    public long? TaskId { get; set; }

    public int? Version { get; set; }
}