using SqlSugar;

namespace Cnn.Domain.Entities;

[SugarTable("message_read")]
public class MessageRead
{
    public int? Uid { get; set; }

    public long? MsgId { get; set; }

    public DateTime? CreateAt { get; set; }
}