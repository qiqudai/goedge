using SqlSugar;

namespace Cnn.Domain.Entities;

[SugarTable("merge_stream_group")]
public class MergeStreamGroup
{
    public int? StreamId { get; set; }

    public int? GroupId { get; set; }
}