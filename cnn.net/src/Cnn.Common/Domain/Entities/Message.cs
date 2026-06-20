using SqlSugar;

namespace Cnn.Domain.Entities;

[SugarTable("message")]
public class Message
{
    [SugarColumn(IsPrimaryKey = true, IsIdentity = true)]
    public long Id { get; set; }

    public string? Type { get; set; }

    public int? PubUser { get; set; }

    public int? Receive { get; set; }

    public string? Title { get; set; }

    public string? Content { get; set; }

    public string? PhoneContent { get; set; }

    public string? EventId { get; set; }

    public int? UserPackageId { get; set; }

    public int? SiteId { get; set; }

    public bool? IsShow { get; set; }

    public bool? IsRed { get; set; }

    public bool? IsBold { get; set; }

    public bool? IsExternal { get; set; }

    public bool? IsPopup { get; set; }

    public bool? EmailNeedSend { get; set; }

    public bool? PhoneNeedSend { get; set; }

    public bool? EmailIsSent { get; set; }

    public bool? PhoneIsSent { get; set; }

    public string? Url { get; set; }

    public int? Sort { get; set; }

    public DateTime? CreateAt { get; set; }

    public DateTime? UpdateAt { get; set; }
}