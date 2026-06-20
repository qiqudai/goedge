using System.Text.Json.Serialization;

namespace Cnn.Common.Contracts;

public class OrderListQuery
{
    [JsonPropertyName("keyword")]
    public string? Keyword { get; set; }

    [JsonPropertyName("page")]
    public int Page { get; set; } = 1;

    [JsonPropertyName("pageSize")]
    public int PageSize { get; set; } = 20;

    [JsonPropertyName("type")]
    public string? Type { get; set; }

    [JsonPropertyName("state")]
    public string? State { get; set; }
}

public sealed class UserOrderListQuery : OrderListQuery;

public sealed record OrderListResult<T>(
    [property: JsonPropertyName("list")] IReadOnlyList<T> List,
    [property: JsonPropertyName("total")] int Total
);

public sealed class AdminOrderDto
{
    [JsonPropertyName("id")]
    public long Id { get; set; }

    [JsonPropertyName("user_id")]
    public long UserId { get; set; }

    [JsonPropertyName("amount")]
    public decimal Amount { get; set; }

    [JsonPropertyName("status")]
    public int Status { get; set; }

    [JsonPropertyName("created_at")]
    public string? CreatedAt { get; set; }

    [JsonPropertyName("pay_type")]
    public string? PayType { get; set; }

    [JsonPropertyName("order_no")]
    public string? OrderNo { get; set; }

    [JsonPropertyName("type")]
    public string? Type { get; set; }

    [JsonPropertyName("remark")]
    public string? Remark { get; set; }
}

public sealed class UserOrderDto
{
    [JsonPropertyName("id")]
    public long Id { get; set; }

    [JsonPropertyName("type")]
    public string? Type { get; set; }

    [JsonPropertyName("type_label")]
    public string? TypeLabel { get; set; }

    [JsonPropertyName("remark")]
    public string? Remark { get; set; }

    [JsonPropertyName("price")]
    public string? Price { get; set; }

    [JsonPropertyName("pay")]
    public string? Pay { get; set; }

    [JsonPropertyName("more")]
    public string? More { get; set; }

    [JsonPropertyName("pay_type")]
    public string? PayType { get; set; }

    [JsonPropertyName("order_no")]
    public string? OrderNo { get; set; }

    [JsonPropertyName("created_at")]
    public string? CreatedAt { get; set; }

    [JsonPropertyName("paid")]
    public bool Paid { get; set; }
}

public sealed class AdminRechargeRequest
{
    [JsonPropertyName("user_id")]
    public long? UserId { get; set; }

    [JsonPropertyName("amount")]
    public decimal Amount { get; set; }

    [JsonPropertyName("remark")]
    public string? Remark { get; set; }
}

public sealed class UserRechargeRequest
{
    [JsonPropertyName("amount")]
    public decimal Amount { get; set; }

    [JsonPropertyName("remark")]
    public string? Remark { get; set; }

    [JsonPropertyName("pay_type")]
    public string? PayType { get; set; }
}

public sealed class BalanceLogListQuery
{
    [JsonPropertyName("user_id")]
    public long? UserId { get; set; }

    [JsonPropertyName("page")]
    public int Page { get; set; } = 1;

    [JsonPropertyName("pageSize")]
    public int PageSize { get; set; } = 20;
}

public sealed class BalanceLogDto
{
    [JsonPropertyName("id")]
    public long Id { get; set; }

    [JsonPropertyName("user_id")]
    public long UserId { get; set; }

    [JsonPropertyName("order_id")]
    public long OrderId { get; set; }

    [JsonPropertyName("action")]
    public string? Action { get; set; }

    [JsonPropertyName("source")]
    public string? Source { get; set; }

    [JsonPropertyName("reason")]
    public string? Reason { get; set; }

    [JsonPropertyName("amount_before")]
    public long AmountBefore { get; set; }

    [JsonPropertyName("amount_change")]
    public long AmountChange { get; set; }

    [JsonPropertyName("amount_after")]
    public long AmountAfter { get; set; }

    [JsonPropertyName("operator_id")]
    public long OperatorId { get; set; }

    [JsonPropertyName("operator_role")]
    public string? OperatorRole { get; set; }

    [JsonPropertyName("created_at")]
    public string? CreatedAt { get; set; }
}

public sealed class AdminAdjustBalanceRequest
{
    [JsonPropertyName("user_id")]
    public long UserId { get; set; }

    [JsonPropertyName("amount")]
    public decimal Amount { get; set; }

    [JsonPropertyName("action")]
    public string? Action { get; set; }

    [JsonPropertyName("reason")]
    public string? Reason { get; set; }

    [JsonPropertyName("source")]
    public string? Source { get; set; }
}

public sealed class AdminMarkOrderPaidRequest
{
    [JsonPropertyName("transaction_id")]
    public string? TransactionId { get; set; }

    [JsonPropertyName("reason")]
    public string? Reason { get; set; }
}

public sealed class UserPackageOpenOrderRequest
{
    [JsonPropertyName("package_id")]
    public long PackageId { get; set; }

    [JsonPropertyName("period")]
    public string? Period { get; set; }

    [JsonPropertyName("months")]
    public int Months { get; set; }

    [JsonPropertyName("pay_type")]
    public string? PayType { get; set; }

    [JsonPropertyName("remark")]
    public string? Remark { get; set; }
}

public sealed class UserPackageRenewOrderRequest
{
    [JsonPropertyName("user_package_id")]
    public long UserPackageId { get; set; }

    [JsonPropertyName("period")]
    public string? Period { get; set; }

    [JsonPropertyName("months")]
    public int Months { get; set; }

    [JsonPropertyName("pay_type")]
    public string? PayType { get; set; }

    [JsonPropertyName("remark")]
    public string? Remark { get; set; }
}

public sealed class OrderCreateResult
{
    [JsonPropertyName("order_id")]
    public long OrderId { get; set; }

    [JsonPropertyName("order_no")]
    public string? OrderNo { get; set; }

    [JsonPropertyName("paid")]
    public bool Paid { get; set; }

    [JsonPropertyName("pay_type")]
    public string? PayType { get; set; }

    [JsonPropertyName("amount")]
    public decimal? Amount { get; set; }

    [JsonPropertyName("created_at")]
    public string? CreatedAt { get; set; }

    [JsonPropertyName("pay_info")]
    public Dictionary<string, object?>? PayInfo { get; set; }
}

public sealed class ShkeeperCallbackPayload
{
    [JsonPropertyName("external_id")]
    public string? ExternalId { get; set; }

    [JsonPropertyName("crypto")]
    public string? Crypto { get; set; }

    [JsonPropertyName("addr")]
    public string? Addr { get; set; }

    [JsonPropertyName("fiat")]
    public string? Fiat { get; set; }

    [JsonPropertyName("balance_fiat")]
    public object? BalanceFiat { get; set; }

    [JsonPropertyName("balance_crypto")]
    public object? BalanceCrypto { get; set; }

    [JsonPropertyName("paid")]
    public bool Paid { get; set; }

    [JsonPropertyName("status")]
    public string? Status { get; set; }

    [JsonPropertyName("transactions")]
    public List<ShkeeperCallbackTransaction>? Transactions { get; set; }

    [JsonPropertyName("fee_percent")]
    public object? FeePercent { get; set; }

    [JsonPropertyName("overpaid_fiat")]
    public object? OverpaidFiat { get; set; }
}

public sealed class ShkeeperCallbackTransaction
{
    [JsonPropertyName("txid")]
    public string? TxId { get; set; }

    [JsonPropertyName("date")]
    public string? Date { get; set; }

    [JsonPropertyName("amount_crypto")]
    public object? AmountCrypto { get; set; }

    [JsonPropertyName("amount_fiat")]
    public object? AmountFiat { get; set; }

    [JsonPropertyName("trigger")]
    public bool Trigger { get; set; }

    [JsonPropertyName("crypto")]
    public string? Crypto { get; set; }
}
