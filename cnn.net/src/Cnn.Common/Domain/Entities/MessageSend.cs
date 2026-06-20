using SqlSugar;

namespace Cnn.Domain.Entities;

[SugarTable("message_send")]
public class MessageSend
{
    [SugarColumn(IsPrimaryKey = true, IsIdentity = true)]
    public long Id { get; set; }

    public int? Uid { get; set; }

    public int? MsgId { get; set; }

    public string? Media { get; set; }

    public int? FailedTimes { get; set; }

    public string? State { get; set; }

    public string? Ret { get; set; }

    public DateTime? CreateAt { get; set; }
}