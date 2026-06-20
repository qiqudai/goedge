using SqlSugar;

namespace Cnn.Domain.Entities;

[SugarTable("acl")]
public class Acl
{
    [SugarColumn(IsPrimaryKey = true, IsIdentity = true)]
    public int Id { get; set; }

    public int? Uid { get; set; }

    public string? Name { get; set; }

    public string? Des { get; set; }

    public string? DefaultAction { get; set; }

    public string? Data { get; set; }

    public DateTime? CreateAt { get; set; }

    public DateTime? UpdateAt { get; set; }

    public bool? Enable { get; set; }

    public long? TaskId { get; set; }

    public int? Version { get; set; }
}