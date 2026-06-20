using SqlSugar;

namespace Cnn.Domain.Entities;

[SugarTable("lets_account")]
public class LetsAccount
{
    [SugarColumn(IsPrimaryKey = true, IsIdentity = true)]
    public int Id { get; set; }

    public bool? Enable { get; set; }

    public DateTime? InvalidDate { get; set; }

    public bool? IsCreated { get; set; }

    public DateTime? CreateFailedAt { get; set; }
}