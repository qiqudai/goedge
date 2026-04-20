using SqlSugar;

namespace Cnn.Domain.Entities;

[SugarTable("order")]
public class Order
{
    [SugarColumn(IsPrimaryKey = true, IsIdentity = true)]
    public int Id { get; set; }

    public int? Uid { get; set; }

    public string? Type { get; set; }

    public string? Des { get; set; }

    public string? Data { get; set; }

    public DateTime? CreateAt { get; set; }

    public DateTime? PayAt { get; set; }

    public long? Amount { get; set; }

    public string? PayType { get; set; }

    public string? MchOrderNo { get; set; }

    public string? TransactionId { get; set; }

    public string? State { get; set; }
}