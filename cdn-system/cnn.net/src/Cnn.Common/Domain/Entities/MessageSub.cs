using SqlSugar;

namespace Cnn.Domain.Entities;

[SugarTable("message_sub")]
public class MessageSub
{
    public int? Uid { get; set; }

    public string? MsgType { get; set; }

    public bool? Phone { get; set; }

    public bool? Email { get; set; }
}