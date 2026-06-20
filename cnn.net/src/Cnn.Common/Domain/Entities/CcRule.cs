using SqlSugar;

namespace Cnn.Domain.Entities;

[SugarTable("cc_rule")]
public class CcRule
{
    [SugarColumn(IsPrimaryKey = true, IsIdentity = true)]
    public int Id { get; set; }

    public int? Sort { get; set; }

    public int? Uid { get; set; }

    public string? Name { get; set; }

    public string? Des { get; set; }

    public string? Data { get; set; }

    public DateTime? CreateAt { get; set; }

    public DateTime? UpdateAt { get; set; }

    public bool? Internal { get; set; }

    public bool? Enable { get; set; }

    public bool? IsShow { get; set; }

    public long? TaskId { get; set; }

    public int? Version { get; set; }
}