using SqlSugar;

namespace Cnn.Domain.Entities;

[SugarTable("balance_ledger")]
public class BalanceLedger
{
    [SugarColumn(IsPrimaryKey = true, IsIdentity = true)]
    public long Id { get; set; }

    [SugarColumn(ColumnName = "uid")]
    public long UserId { get; set; }

    [SugarColumn(ColumnName = "order_id")]
    public long OrderId { get; set; }

    [SugarColumn(ColumnName = "amount_before")]
    public long AmountBefore { get; set; }

    [SugarColumn(ColumnName = "amount_change")]
    public long AmountChange { get; set; }

    [SugarColumn(ColumnName = "amount_after")]
    public long AmountAfter { get; set; }

    public string? Action { get; set; }

    public string? Source { get; set; }

    public string? Reason { get; set; }

    [SugarColumn(ColumnName = "operator_id")]
    public long OperatorId { get; set; }

    [SugarColumn(ColumnName = "operator_role")]
    public string? OperatorRole { get; set; }

    [SugarColumn(ColumnName = "create_at")]
    public DateTime CreatedAt { get; set; }
}
