using SqlSugar;

namespace Cnn.Domain.Entities;

[SugarTable("stream_group")]
public class StreamGroup
{
    [SugarColumn(IsPrimaryKey = true, IsIdentity = true)]
    public int Id { get; set; }

    public int? Uid { get; set; }

    public string? Name { get; set; }

    public string? Des { get; set; }
}