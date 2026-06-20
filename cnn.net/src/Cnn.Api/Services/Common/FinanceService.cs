using System.Globalization;
using System.Net;
using System.Net.Http.Headers;
using System.Security.Cryptography;
using System.Text;
using System.Text.Json;
using Cnn.Common.Contracts;
using Cnn.Common.Localization;
using Cnn.Domain.Entities;
using SqlSugar;
using Task = System.Threading.Tasks.Task;

namespace Cnn.Api.Services.Common;

public interface IFinanceService
{
    Task<ServiceResult<OrderListResult<AdminOrderDto>>> ListAdminAsync(OrderListQuery query, CancellationToken cancellationToken);
    Task<ServiceResult<OrderListResult<UserOrderDto>>> ListUserAsync(UserOrderListQuery query, long? userId, string language, CancellationToken cancellationToken);
    Task<ServiceResult<OrderListResult<BalanceLogDto>>> ListAdminBalanceLogsAsync(BalanceLogListQuery query, CancellationToken cancellationToken);
    Task<ServiceResult<OrderListResult<BalanceLogDto>>> ListUserBalanceLogsAsync(BalanceLogListQuery query, long userId, CancellationToken cancellationToken);
    Task<ServiceResult<bool>> AdminRechargeAsync(AdminRechargeRequest request, CancellationToken cancellationToken);
    Task<ServiceResult<OrderCreateResult>> UserRechargeAsync(long userId, UserRechargeRequest request, string? callbackBaseUrl, CancellationToken cancellationToken);
    Task<ServiceResult<bool>> AdminAdjustBalanceAsync(AdminAdjustBalanceRequest request, long operatorId, CancellationToken cancellationToken);
    Task<ServiceResult<bool>> MarkOrderPaidAsync(long orderId, AdminMarkOrderPaidRequest request, long operatorId, CancellationToken cancellationToken);
    Task<ServiceResult<OrderCreateResult>> CreateUserPackageOpenOrderAsync(long userId, UserPackageOpenOrderRequest request, string? callbackBaseUrl, CancellationToken cancellationToken);
    Task<ServiceResult<OrderCreateResult>> CreateUserPackageRenewOrderAsync(long userId, UserPackageRenewOrderRequest request, string? callbackBaseUrl, CancellationToken cancellationToken);
    Task<ServiceResult<bool>> HandleShkeeperCallbackAsync(ShkeeperCallbackPayload payload, string? callbackApiKey, CancellationToken cancellationToken);
}

public sealed class FinanceService : IFinanceService
{
    private const int DefaultPageSize = 20;
    private const int MaxPageSize = 200;

    private readonly ISqlSugarClient _db;
    private readonly IMessageLocalizer _localizer;
    private readonly ISystemConfigService _systemConfigService;
    private readonly IUserPackageSyncService _userPackageSyncService;

    public FinanceService(
        ISqlSugarClient db,
        IMessageLocalizer localizer,
        ISystemConfigService systemConfigService,
        IUserPackageSyncService userPackageSyncService)
    {
        _db = db;
        _localizer = localizer;
        _systemConfigService = systemConfigService;
        _userPackageSyncService = userPackageSyncService;
    }

    public async Task<ServiceResult<OrderListResult<AdminOrderDto>>> ListAdminAsync(OrderListQuery query, CancellationToken cancellationToken)
    {
        query ??= new OrderListQuery();
        var (page, pageSize) = ResolvePaging(query.Page, query.PageSize);
        var keyword = query.Keyword?.Trim();
        var orderType = query.Type?.Trim();
        var state = query.State?.Trim();

        var q = _db.Queryable<Order>();
        if (!string.IsNullOrWhiteSpace(keyword))
        {
            if (long.TryParse(keyword, out var uid))
            {
                q = q.Where(o => SqlFunc.Contains(o.MchOrderNo, keyword!) || SqlFunc.Contains(o.Des, keyword!) || o.Uid == uid);
            }
            else
            {
                q = q.Where(o => SqlFunc.Contains(o.MchOrderNo, keyword!) || SqlFunc.Contains(o.Des, keyword!));
            }
        }

        if (!string.IsNullOrWhiteSpace(orderType))
        {
            q = q.Where(o => o.Type == orderType);
        }

        if (!string.IsNullOrWhiteSpace(state))
        {
            q = q.Where(o => o.State == state);
        }

        var total = await q.CountAsync();
        var orders = await q.OrderBy(o => o.Id, OrderByType.Desc)
            .Skip((page - 1) * pageSize)
            .Take(pageSize)
            .ToListAsync();

        var list = orders.Select(o => new AdminOrderDto
        {
            Id = o.Id,
            UserId = o.Uid ?? 0,
            Amount = ToAmount(o.Amount),
            Status = IsPaid(o.State) ? 1 : 0,
            CreatedAt = FormatTime(o.CreateAt),
            PayType = o.PayType,
            OrderNo = o.MchOrderNo,
            Type = o.Type,
            Remark = o.Des
        }).ToList();

        return ServiceResult<OrderListResult<AdminOrderDto>>.Ok(new OrderListResult<AdminOrderDto>(list, (int)total));
    }

    public async Task<ServiceResult<OrderListResult<UserOrderDto>>> ListUserAsync(UserOrderListQuery query, long? userId, string language, CancellationToken cancellationToken)
    {
        var uid = userId.GetValueOrDefault();
        if (uid <= 0)
        {
            return ServiceResult<OrderListResult<UserOrderDto>>.Fail(ErrorCodes.AuthInvalid);
        }

        query ??= new UserOrderListQuery();
        var (page, pageSize) = ResolvePaging(query.Page, query.PageSize);
        var keyword = query.Keyword?.Trim();
        var orderType = query.Type?.Trim();

        var q = _db.Queryable<Order>().Where(o => o.Uid == uid);
        if (!string.IsNullOrWhiteSpace(orderType))
        {
            q = q.Where(o => o.Type == orderType);
        }
        if (!string.IsNullOrWhiteSpace(keyword))
        {
            q = q.Where(o => SqlFunc.Contains(o.MchOrderNo, keyword!) || SqlFunc.Contains(o.Des, keyword!));
        }

        var total = await q.CountAsync();
        var orders = await q.OrderBy(o => o.Id, OrderByType.Desc)
            .Skip((page - 1) * pageSize)
            .Take(pageSize)
            .ToListAsync();

        var list = orders.Select(o =>
        {
            var amountText = FormatAmountText(o.Amount);
            return new UserOrderDto
            {
                Id = o.Id,
                Type = o.Type,
                TypeLabel = ResolveOrderTypeLabel(o.Type, language),
                Remark = o.Des,
                Price = amountText,
                Pay = amountText,
                More = SummarizeOrderMore(o.Data),
                PayType = o.PayType,
                OrderNo = o.MchOrderNo,
                CreatedAt = FormatTime(o.CreateAt),
                Paid = IsPaid(o.State)
            };
        }).ToList();

        return ServiceResult<OrderListResult<UserOrderDto>>.Ok(new OrderListResult<UserOrderDto>(list, (int)total));
    }

    public async Task<ServiceResult<OrderListResult<BalanceLogDto>>> ListAdminBalanceLogsAsync(BalanceLogListQuery query, CancellationToken cancellationToken)
    {
        query ??= new BalanceLogListQuery();
        var (page, pageSize) = ResolvePaging(query.Page, query.PageSize);

        var q = _db.Queryable<BalanceLedger>();
        if (query.UserId is > 0)
        {
            q = q.Where(l => l.UserId == query.UserId.Value);
        }

        var total = await q.CountAsync();
        var rows = await q.OrderBy(l => l.Id, OrderByType.Desc)
            .Skip((page - 1) * pageSize)
            .Take(pageSize)
            .ToListAsync();

        var list = rows.Select(MapBalanceLog).ToList();
        return ServiceResult<OrderListResult<BalanceLogDto>>.Ok(new OrderListResult<BalanceLogDto>(list, (int)total));
    }

    public async Task<ServiceResult<OrderListResult<BalanceLogDto>>> ListUserBalanceLogsAsync(BalanceLogListQuery query, long userId, CancellationToken cancellationToken)
    {
        if (userId <= 0)
        {
            return ServiceResult<OrderListResult<BalanceLogDto>>.Fail(ErrorCodes.AuthInvalid);
        }

        query ??= new BalanceLogListQuery();
        var (page, pageSize) = ResolvePaging(query.Page, query.PageSize);

        var q = _db.Queryable<BalanceLedger>().Where(l => l.UserId == userId);
        var total = await q.CountAsync();
        var rows = await q.OrderBy(l => l.Id, OrderByType.Desc)
            .Skip((page - 1) * pageSize)
            .Take(pageSize)
            .ToListAsync();

        var list = rows.Select(MapBalanceLog).ToList();
        return ServiceResult<OrderListResult<BalanceLogDto>>.Ok(new OrderListResult<BalanceLogDto>(list, (int)total));
    }

    public async Task<ServiceResult<bool>> AdminRechargeAsync(AdminRechargeRequest request, CancellationToken cancellationToken)
    {
        var userId = request.UserId.GetValueOrDefault();
        if (userId <= 0 || request.Amount <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam);
        }

        return await AdjustBalanceWithOrderAsync(
            userId,
            request.Amount,
            "credit",
            request.Remark,
            "admin_manual",
            operatorId: 0,
            operatorRole: "admin",
            cancellationToken);
    }

    public async Task<ServiceResult<OrderCreateResult>> UserRechargeAsync(long userId, UserRechargeRequest request, string? callbackBaseUrl, CancellationToken cancellationToken)
    {
        if (userId <= 0)
        {
            return ServiceResult<OrderCreateResult>.Fail(ErrorCodes.AuthInvalid);
        }

        if (request.Amount <= 0)
        {
            return ServiceResult<OrderCreateResult>.Fail(ErrorCodes.InvalidParam);
        }

        var amountCents = ToCents(request.Amount);
        if (amountCents <= 0)
        {
            return ServiceResult<OrderCreateResult>.Fail(ErrorCodes.InvalidParam);
        }

        var payType = NormalizePayType(request.PayType);
        var now = DateTime.Now;
        var merchantOrder = GenerateMerchantOrder("recharge");

        Dictionary<string, object?>? payInfo = null;
        var orderData = string.Empty;
        if (IsShkeeperPayType(payType))
        {
            var shkeeper = await CreateShkeeperPayInfoAsync(merchantOrder, amountCents, callbackBaseUrl, cancellationToken);
            if (!shkeeper.Success)
            {
                return ServiceResult<OrderCreateResult>.Fail(shkeeper.ErrorCode, shkeeper.MessageKey);
            }

            payInfo = shkeeper.Data;
            orderData = MarshalJson(payInfo);
        }

        var order = new Order
        {
            Uid = (int)userId,
            Type = "recharge",
            Des = request.Remark?.Trim(),
            Data = orderData,
            CreateAt = now,
            Amount = amountCents,
            PayType = payType,
            MchOrderNo = merchantOrder,
            TransactionId = string.Empty,
            State = "pending"
        };

        var orderId = await _db.Insertable(order).ExecuteReturnIdentityAsync();
        order.Id = orderId;

        return ServiceResult<OrderCreateResult>.Ok(new OrderCreateResult
        {
            OrderId = order.Id,
            OrderNo = order.MchOrderNo,
            Paid = false,
            PayType = order.PayType,
            Amount = ToAmount(order.Amount),
            CreatedAt = FormatTime(order.CreateAt),
            PayInfo = payInfo
        });
    }

    public async Task<ServiceResult<bool>> AdminAdjustBalanceAsync(AdminAdjustBalanceRequest request, long operatorId, CancellationToken cancellationToken)
    {
        if (request.UserId <= 0 || request.Amount <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam);
        }

        var action = (request.Action ?? string.Empty).Trim().ToLowerInvariant();
        if (action != "debit")
        {
            action = "credit";
        }

        var source = request.Source?.Trim();
        if (string.IsNullOrWhiteSpace(source))
        {
            source = action == "debit" ? "admin_deduct" : "admin_recharge";
        }

        return await AdjustBalanceWithOrderAsync(
            request.UserId,
            request.Amount,
            action,
            request.Reason,
            source,
            operatorId,
            "admin",
            cancellationToken);
    }

    public async Task<ServiceResult<bool>> MarkOrderPaidAsync(long orderId, AdminMarkOrderPaidRequest request, long operatorId, CancellationToken cancellationToken)
    {
        if (orderId <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam);
        }

        List<long> syncIds = new();
        var reason = string.IsNullOrWhiteSpace(request.Reason) ? "admin debug mark paid" : request.Reason!.Trim();

        var tran = await _db.Ado.UseTranAsync(async () =>
        {
            syncIds = await ApplyOrderPaidTxAsync(
                orderId,
                request.TransactionId,
                "admin_mark_paid",
                reason,
                operatorId,
                "admin",
                callbackPayload: null,
                cancellationToken);
        });

        if (!tran.IsSuccess)
        {
            return MapFinanceTranError<bool>(tran.ErrorException);
        }

        await SyncUserPackagesAsync(syncIds, "payment", cancellationToken);
        return ServiceResult<bool>.Ok(true);
    }

    public async Task<ServiceResult<OrderCreateResult>> CreateUserPackageOpenOrderAsync(long userId, UserPackageOpenOrderRequest request, string? callbackBaseUrl, CancellationToken cancellationToken)
    {
        if (userId <= 0)
        {
            return ServiceResult<OrderCreateResult>.Fail(ErrorCodes.AuthInvalid);
        }

        if (request.PackageId <= 0)
        {
            return ServiceResult<OrderCreateResult>.Fail(ErrorCodes.InvalidParam, "package_id_required");
        }

        var pkg = await _db.Queryable<Package>()
            .Where(p => p.Id == request.PackageId && p.Enable == true)
            .FirstAsync();
        if (pkg == null)
        {
            return ServiceResult<OrderCreateResult>.Fail(ErrorCodes.NotFound, "package_not_found");
        }

        var months = PeriodToMonths(request.Period, request.Months);
        if (months <= 0)
        {
            return ServiceResult<OrderCreateResult>.Fail(ErrorCodes.InvalidParam, "invalid_period");
        }

        var amountRes = PackageAmountByMonths(pkg.MonthPrice, pkg.QuarterPrice, pkg.YearPrice, months);
        if (!amountRes.Success)
        {
            return ServiceResult<OrderCreateResult>.Fail(ErrorCodes.InvalidParam, amountRes.MessageKey);
        }

        var payType = NormalizePayType(request.PayType);
        if (string.IsNullOrWhiteSpace(payType))
        {
            payType = "balance";
        }

        var now = DateTime.Now;
        var merchantOrder = GenerateMerchantOrder("purchase");
        var data = new Dictionary<string, object?>
        {
            ["package_id"] = request.PackageId,
            ["months"] = months,
            ["auto_renew"] = true
        };

        var order = new Order
        {
            Uid = (int)userId,
            Type = "purchase",
            Des = request.Remark?.Trim(),
            Data = MarshalJson(data),
            CreateAt = now,
            Amount = amountRes.Data,
            PayType = payType,
            MchOrderNo = merchantOrder,
            State = "pending"
        };

        if (string.Equals(payType, "balance", StringComparison.OrdinalIgnoreCase))
        {
            List<long> syncIds = new();
            var tran = await _db.Ado.UseTranAsync(async () =>
            {
                var orderId = await _db.Insertable(order).ExecuteReturnIdentityAsync();
                order.Id = orderId;
                syncIds = await ApplyOrderPaidTxAsync(
                    order.Id,
                    transactionId: null,
                    source: "balance_pay",
                    reason: "purchase by balance",
                    operatorId: userId,
                    operatorRole: "user",
                    callbackPayload: null,
                    cancellationToken);
            });

            if (!tran.IsSuccess)
            {
                return MapFinanceTranError<OrderCreateResult>(tran.ErrorException);
            }

            await SyncUserPackagesAsync(syncIds, "purchase", cancellationToken);
            return ServiceResult<OrderCreateResult>.Ok(new OrderCreateResult
            {
                OrderId = order.Id,
                OrderNo = order.MchOrderNo,
                Paid = true,
                PayType = order.PayType,
                Amount = ToAmount(order.Amount)
            });
        }

        Dictionary<string, object?>? payInfo = null;
        if (IsShkeeperPayType(payType))
        {
            var shkeeper = await CreateShkeeperPayInfoAsync(merchantOrder, amountRes.Data, callbackBaseUrl, cancellationToken);
            if (!shkeeper.Success)
            {
                return ServiceResult<OrderCreateResult>.Fail(shkeeper.ErrorCode, shkeeper.MessageKey);
            }

            payInfo = shkeeper.Data ?? new Dictionary<string, object?>();
            foreach (var pair in payInfo)
            {
                data[pair.Key] = pair.Value;
            }

            order.Data = MarshalJson(data);
        }

        var id = await _db.Insertable(order).ExecuteReturnIdentityAsync();
        order.Id = id;

        return ServiceResult<OrderCreateResult>.Ok(new OrderCreateResult
        {
            OrderId = order.Id,
            OrderNo = order.MchOrderNo,
            Paid = false,
            PayType = order.PayType,
            Amount = ToAmount(order.Amount),
            PayInfo = payInfo
        });
    }

    public async Task<ServiceResult<OrderCreateResult>> CreateUserPackageRenewOrderAsync(long userId, UserPackageRenewOrderRequest request, string? callbackBaseUrl, CancellationToken cancellationToken)
    {
        if (userId <= 0)
        {
            return ServiceResult<OrderCreateResult>.Fail(ErrorCodes.AuthInvalid);
        }

        if (request.UserPackageId <= 0)
        {
            return ServiceResult<OrderCreateResult>.Fail(ErrorCodes.InvalidParam, "user_package_id_required");
        }

        var userPackage = await _db.Queryable<UserPackage>()
            .Where(p => p.Id == request.UserPackageId && p.Uid == userId)
            .FirstAsync();
        if (userPackage == null)
        {
            return ServiceResult<OrderCreateResult>.Fail(ErrorCodes.NotFound, "package_not_found");
        }

        var months = PeriodToMonths(request.Period, request.Months);
        if (months <= 0)
        {
            return ServiceResult<OrderCreateResult>.Fail(ErrorCodes.InvalidParam, "invalid_period");
        }

        var amountRes = PackageAmountByMonths(userPackage.MonthPrice, userPackage.QuarterPrice, userPackage.YearPrice, months);
        if (!amountRes.Success)
        {
            return ServiceResult<OrderCreateResult>.Fail(ErrorCodes.InvalidParam, amountRes.MessageKey);
        }

        var payType = NormalizePayType(request.PayType);
        if (string.IsNullOrWhiteSpace(payType))
        {
            payType = "balance";
        }

        var now = DateTime.Now;
        var merchantOrder = GenerateMerchantOrder("renew");
        var data = new Dictionary<string, object?>
        {
            ["user_package_id"] = request.UserPackageId,
            ["months"] = months,
            ["auto_renew"] = true
        };

        var order = new Order
        {
            Uid = (int)userId,
            Type = "renew",
            Des = request.Remark?.Trim(),
            Data = MarshalJson(data),
            CreateAt = now,
            Amount = amountRes.Data,
            PayType = payType,
            MchOrderNo = merchantOrder,
            State = "pending"
        };

        if (string.Equals(payType, "balance", StringComparison.OrdinalIgnoreCase))
        {
            List<long> syncIds = new();
            var tran = await _db.Ado.UseTranAsync(async () =>
            {
                var orderId = await _db.Insertable(order).ExecuteReturnIdentityAsync();
                order.Id = orderId;
                syncIds = await ApplyOrderPaidTxAsync(
                    order.Id,
                    transactionId: null,
                    source: "balance_pay",
                    reason: "renew by balance",
                    operatorId: userId,
                    operatorRole: "user",
                    callbackPayload: null,
                    cancellationToken);
            });

            if (!tran.IsSuccess)
            {
                return MapFinanceTranError<OrderCreateResult>(tran.ErrorException);
            }

            await SyncUserPackagesAsync(syncIds, "renew", cancellationToken);
            return ServiceResult<OrderCreateResult>.Ok(new OrderCreateResult
            {
                OrderId = order.Id,
                OrderNo = order.MchOrderNo,
                Paid = true,
                PayType = order.PayType,
                Amount = ToAmount(order.Amount)
            });
        }

        Dictionary<string, object?>? payInfo = null;
        if (IsShkeeperPayType(payType))
        {
            var shkeeper = await CreateShkeeperPayInfoAsync(merchantOrder, amountRes.Data, callbackBaseUrl, cancellationToken);
            if (!shkeeper.Success)
            {
                return ServiceResult<OrderCreateResult>.Fail(shkeeper.ErrorCode, shkeeper.MessageKey);
            }

            payInfo = shkeeper.Data ?? new Dictionary<string, object?>();
            foreach (var pair in payInfo)
            {
                data[pair.Key] = pair.Value;
            }

            order.Data = MarshalJson(data);
        }

        var id = await _db.Insertable(order).ExecuteReturnIdentityAsync();
        order.Id = id;

        return ServiceResult<OrderCreateResult>.Ok(new OrderCreateResult
        {
            OrderId = order.Id,
            OrderNo = order.MchOrderNo,
            Paid = false,
            PayType = order.PayType,
            Amount = ToAmount(order.Amount),
            PayInfo = payInfo
        });
    }

    public async Task<ServiceResult<bool>> HandleShkeeperCallbackAsync(ShkeeperCallbackPayload payload, string? callbackApiKey, CancellationToken cancellationToken)
    {
        if (payload == null)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam, "invalid_callback_payload");
        }

        var settings = await LoadShkeeperSettingsAsync(cancellationToken);
        if (!settings.Enable)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.ConfigError, "shkeeper_disabled");
        }

        if (string.IsNullOrWhiteSpace(settings.CallbackApiKey) || !string.Equals(settings.CallbackApiKey, callbackApiKey?.Trim(), StringComparison.Ordinal))
        {
            return ServiceResult<bool>.Fail(ErrorCodes.PermissionDenied, "invalid_callback_key");
        }

        var externalId = payload.ExternalId?.Trim();
        if (string.IsNullOrWhiteSpace(externalId))
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam, "external_id_required");
        }

        var paid = payload.Paid;
        var status = payload.Status?.Trim().ToUpperInvariant();
        if (status is "PAID" or "OVERPAID")
        {
            paid = true;
        }

        if (!paid)
        {
            return ServiceResult<bool>.Ok(true);
        }

        var order = await _db.Queryable<Order>().Where(o => o.MchOrderNo == externalId).FirstAsync();
        if (order == null)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.NotFound, "order_not_found");
        }

        string? transactionId = null;
        if (payload.Transactions != null)
        {
            foreach (var tx in payload.Transactions)
            {
                if (!string.IsNullOrWhiteSpace(tx.TxId))
                {
                    transactionId = tx.TxId.Trim();
                    if (tx.Trigger)
                    {
                        break;
                    }
                }
            }
        }

        List<long> syncIds = new();
        var tran = await _db.Ado.UseTranAsync(async () =>
        {
            syncIds = await ApplyOrderPaidTxAsync(
                order.Id,
                transactionId,
                "shkeeper_callback",
                "shkeeper callback paid",
                0,
                "system",
                payload,
                cancellationToken);
        });

        if (!tran.IsSuccess)
        {
            return MapFinanceTranError<bool>(tran.ErrorException);
        }

        await SyncUserPackagesAsync(syncIds, "payment", cancellationToken);
        return ServiceResult<bool>.Ok(true);
    }

    private async Task<ServiceResult<bool>> AdjustBalanceWithOrderAsync(
        long userId,
        decimal amount,
        string action,
        string? reason,
        string source,
        long operatorId,
        string operatorRole,
        CancellationToken cancellationToken)
    {
        var amountCents = ToCents(amount);
        if (amountCents <= 0)
        {
            return ServiceResult<bool>.Fail(ErrorCodes.InvalidParam);
        }

        var change = string.Equals(action, "debit", StringComparison.OrdinalIgnoreCase) ? -amountCents : amountCents;
        var now = DateTime.Now;

        var tran = await _db.Ado.UseTranAsync(async () =>
        {
            var order = new Order
            {
                Uid = (int)userId,
                Type = change > 0 ? "recharge" : "adjust",
                Des = reason?.Trim(),
                Data = string.Empty,
                CreateAt = now,
                PayAt = now,
                Amount = change,
                PayType = source,
                MchOrderNo = GenerateMerchantOrder("adj"),
                TransactionId = string.Empty,
                State = "paid"
            };
            var orderId = await _db.Insertable(order).ExecuteReturnIdentityAsync();
            order.Id = orderId;

            await AdjustUserBalanceWithLedgerAsync(
                userId,
                order.Id,
                change,
                reason,
                source,
                operatorId,
                operatorRole,
                cancellationToken);
        });

        if (!tran.IsSuccess)
        {
            return MapFinanceTranError<bool>(tran.ErrorException);
        }

        return ServiceResult<bool>.Ok(true);
    }

    private async Task<List<long>> ApplyOrderPaidTxAsync(
        long orderId,
        string? transactionId,
        string source,
        string reason,
        long operatorId,
        string operatorRole,
        object? callbackPayload,
        CancellationToken cancellationToken)
    {
        var order = await _db.Queryable<Order>().Where(o => o.Id == orderId).FirstAsync();
        if (order == null)
        {
            throw new InvalidOperationException("order_not_found");
        }

        if (IsPaid(order.State))
        {
            return new List<long>();
        }

        var syncIds = new List<long>();
        var orderData = ParseOrderData(order.Data);
        var needBalanceDebit = string.Equals(NormalizePayType(order.PayType), "balance", StringComparison.OrdinalIgnoreCase);
        var orderType = order.Type?.Trim().ToLowerInvariant() ?? string.Empty;

        switch (orderType)
        {
            case "recharge":
            case "adjust":
                await AdjustUserBalanceWithLedgerAsync(
                    order.Uid ?? 0,
                    order.Id,
                    order.Amount ?? 0,
                    reason,
                    source,
                    operatorId,
                    operatorRole,
                    cancellationToken);
                break;

            case "purchase":
            {
                var packageId = GetLongFromObject(orderData.TryGetValue("package_id", out var packageRaw) ? packageRaw : null);
                var months = (int)GetLongFromObject(orderData.TryGetValue("months", out var monthsRaw) ? monthsRaw : null);
                var autoRenew = GetBoolFromObject(orderData.TryGetValue("auto_renew", out var autoRaw) ? autoRaw : null);

                if (packageId <= 0 || months <= 0)
                {
                    throw new InvalidOperationException("invalid_order_data");
                }

                if (needBalanceDebit)
                {
                    await AdjustUserBalanceWithLedgerAsync(
                        order.Uid ?? 0,
                        order.Id,
                        -(order.Amount ?? 0),
                        reason,
                        source,
                        operatorId,
                        operatorRole,
                        cancellationToken);
                }

                var userPackageId = await CreateUserPackageFromPlanAsync(order.Uid ?? 0, packageId, months, cancellationToken);
                syncIds.Add(userPackageId);
                orderData["package_id"] = packageId;
                orderData["months"] = months;
                orderData["auto_renew"] = autoRenew;
                orderData["user_package_id"] = userPackageId;
                break;
            }

            case "renew":
            {
                var userPackageId = GetLongFromObject(orderData.TryGetValue("user_package_id", out var userPackRaw) ? userPackRaw : null);
                var months = (int)GetLongFromObject(orderData.TryGetValue("months", out var monthsRaw) ? monthsRaw : null);
                var autoRenew = GetBoolFromObject(orderData.TryGetValue("auto_renew", out var autoRaw) ? autoRaw : null);

                if (userPackageId <= 0 || months <= 0)
                {
                    throw new InvalidOperationException("invalid_order_data");
                }

                if (needBalanceDebit)
                {
                    await AdjustUserBalanceWithLedgerAsync(
                        order.Uid ?? 0,
                        order.Id,
                        -(order.Amount ?? 0),
                        reason,
                        source,
                        operatorId,
                        operatorRole,
                        cancellationToken);
                }

                var renewedId = await RenewUserPackageAsync(order.Uid ?? 0, userPackageId, months, cancellationToken);
                syncIds.Add(renewedId);
                orderData["user_package_id"] = renewedId;
                orderData["months"] = months;
                orderData["auto_renew"] = autoRenew;
                break;
            }

            default:
                throw new InvalidOperationException("unsupported_order_type");
        }

        if (callbackPayload != null)
        {
            orderData["callback_payload"] = callbackPayload;
        }

        var paidAt = DateTime.Now;
        var txId = string.IsNullOrWhiteSpace(transactionId) ? order.TransactionId : transactionId.Trim();
        var nextData = (orderType == "purchase" || orderType == "renew" || callbackPayload != null)
            ? MarshalJson(orderData)
            : order.Data;

        await _db.Updateable<Order>()
            .SetColumns(o => new Order
            {
                State = "paid",
                PayAt = paidAt,
                TransactionId = txId,
                Data = nextData
            })
            .Where(o => o.Id == order.Id)
            .ExecuteCommandAsync();
        return syncIds;
    }

    private async Task<long> CreateUserPackageFromPlanAsync(long userId, long packageId, int months, CancellationToken cancellationToken)
    {
        var pkg = await _db.Queryable<Package>().Where(p => p.Id == packageId).FirstAsync();
        if (pkg == null)
        {
            throw new InvalidOperationException("package_not_found");
        }

        var now = DateTime.Now;
        var recordId = await GenerateUniqueRecordIdAsync(cancellationToken);

        var userPackage = new UserPackage
        {
            Uid = (int)userId,
            Name = pkg.Name,
            Package = pkg.Id,
            RegionId = pkg.RegionId,
            NodeGroupId = pkg.NodeGroupId,
            BackupNodeGroup = pkg.BackupNodeGroup,
            EnableBackupGroup = false,
            CnameDomain = pkg.CnameDomain,
            CnameHostname2 = pkg.CnameHostname2,
            CnameMode = pkg.CnameMode,
            RecordId = recordId,
            Traffic = pkg.Traffic,
            Bandwidth = pkg.Bandwidth,
            Connection = pkg.Connection,
            Domain = pkg.Domain,
            HttpPort = pkg.HttpPort,
            StreamPort = pkg.StreamPort,
            CustomCcRule = pkg.CustomCcRule,
            Websocket = pkg.Websocket,
            L2Origin = pkg.L2Origin,
            MonthPrice = pkg.MonthPrice,
            QuarterPrice = pkg.QuarterPrice,
            YearPrice = pkg.YearPrice,
            CreateAt = now,
            StartAt = now,
            EndAt = now.AddMonths(months),
            IsExpired = false,
            Version = 1
        };

        var id = await _db.Insertable(userPackage).ExecuteReturnIdentityAsync();
        if (id <= 0)
        {
            throw new InvalidOperationException("db_create_error");
        }

        return id;
    }

    private async Task<long> RenewUserPackageAsync(long userId, long userPackageId, int months, CancellationToken cancellationToken)
    {
        var userPackage = await _db.Queryable<UserPackage>()
            .Where(p => p.Id == userPackageId && p.Uid == userId)
            .FirstAsync();
        if (userPackage == null)
        {
            throw new InvalidOperationException("user_package_not_found");
        }

        var now = DateTime.Now;
        var baseTime = userPackage.EndAt.HasValue && userPackage.EndAt.Value > now ? userPackage.EndAt.Value : now;
        var newEnd = baseTime.AddMonths(months);

        await _db.Updateable<UserPackage>()
            .SetColumns(p => new UserPackage
            {
                EndAt = newEnd,
                IsExpired = false
            })
            .Where(p => p.Id == userPackageId)
            .ExecuteCommandAsync();

        return userPackageId;
    }

    private async Task AdjustUserBalanceWithLedgerAsync(
        long userId,
        long orderId,
        long amountChange,
        string? reason,
        string source,
        long operatorId,
        string operatorRole,
        CancellationToken cancellationToken)
    {
        if (userId <= 0)
        {
            throw new InvalidOperationException("invalid_user_id");
        }

        if (amountChange == 0)
        {
            throw new InvalidOperationException("invalid_amount_change");
        }

        var user = await _db.Queryable<User>().Where(u => u.Id == userId).FirstAsync();
        if (user == null)
        {
            throw new InvalidOperationException("user_not_found");
        }

        var before = user.Balance ?? 0;
        var after = before + amountChange;
        if (after < 0)
        {
            throw new InvalidOperationException("insufficient_balance");
        }

        await _db.Updateable<User>()
            .SetColumns(u => new User { Balance = after })
            .Where(u => u.Id == userId)
            .ExecuteCommandAsync();

        var action = amountChange < 0 ? "debit" : "credit";
        var ledger = new BalanceLedger
        {
            UserId = userId,
            OrderId = orderId,
            AmountBefore = before,
            AmountChange = amountChange,
            AmountAfter = after,
            Action = action,
            Source = source,
            Reason = reason?.Trim(),
            OperatorId = operatorId,
            OperatorRole = operatorRole,
            CreatedAt = DateTime.Now
        };

        await _db.Insertable(ledger).ExecuteCommandAsync();
    }

    private async Task SyncUserPackagesAsync(IReadOnlyList<long> ids, string trigger, CancellationToken cancellationToken)
    {
        if (ids.Count == 0)
        {
            return;
        }

        var seen = new HashSet<long>();
        foreach (var id in ids)
        {
            if (id <= 0 || !seen.Add(id))
            {
                continue;
            }

            await _userPackageSyncService.SyncAsync(id, trigger, cancellationToken);
        }
    }

    private async Task<string> GenerateUniqueRecordIdAsync(CancellationToken cancellationToken)
    {
        for (var i = 0; i < 5; i++)
        {
            var candidate = RandomToken(8);
            var exists = await _db.Queryable<UserPackage>().AnyAsync(p => p.RecordId == candidate);
            if (!exists)
            {
                return candidate;
            }
        }

        throw new InvalidOperationException("failed_allocate_record_id");
    }

    private static string RandomToken(int length)
    {
        const string chars = "abcdefghijklmnopqrstuvwxyz0123456789";
        var bytes = RandomNumberGenerator.GetBytes(length);
        var sb = new StringBuilder(length);
        for (var i = 0; i < length; i++)
        {
            sb.Append(chars[bytes[i] % chars.Length]);
        }

        return sb.ToString();
    }

    private static (int Page, int PageSize) ResolvePaging(int page, int pageSize)
    {
        if (page < 1) page = 1;
        if (pageSize < 1) pageSize = DefaultPageSize;
        if (pageSize > MaxPageSize) pageSize = MaxPageSize;
        return (page, pageSize);
    }

    private static bool IsPaid(string? state)
    {
        if (string.IsNullOrWhiteSpace(state))
        {
            return false;
        }

        return state.Equals("paid", StringComparison.OrdinalIgnoreCase)
            || state.Equals("success", StringComparison.OrdinalIgnoreCase)
            || state.Equals("done", StringComparison.OrdinalIgnoreCase);
    }

    private static decimal ToAmount(long? amountCents)
    {
        if (!amountCents.HasValue)
        {
            return 0m;
        }
        return amountCents.Value / 100m;
    }

    private static long ToCents(decimal amount)
    {
        return (long)Math.Round(amount * 100m, MidpointRounding.AwayFromZero);
    }

    private static string FormatAmountText(long? amountCents)
    {
        var amount = ToAmount(amountCents);
        return amount.ToString("F2", CultureInfo.InvariantCulture);
    }

    private static string? FormatTime(DateTime? time)
    {
        return time?.ToString("yyyy-MM-dd HH:mm:ss");
    }

    private string ResolveOrderTypeLabel(string? type, string language)
    {
        if (string.IsNullOrWhiteSpace(type))
        {
            return string.Empty;
        }

        return type.Trim().ToLowerInvariant() switch
        {
            "purchase" => _localizer.Translate("order.purchase", language),
            "renew" => _localizer.Translate("order.renew", language),
            "recharge" => _localizer.Translate("order.recharge", language),
            "adjust" => _localizer.Translate("order.adjust", language),
            _ => _localizer.Translate("order.other", language)
        };
    }

    private static string NormalizePayType(string? raw)
    {
        var value = (raw ?? string.Empty).Trim().ToLowerInvariant();
        return value switch
        {
            "" or "usdt" or "trc20" or "usdt_trc20" or "usdt-trc20" or "shkeeper" or "shkeeper_trc20" => "usdt_trc20",
            _ => value
        };
    }

    private static bool IsShkeeperPayType(string? payType)
    {
        return string.Equals(NormalizePayType(payType), "usdt_trc20", StringComparison.OrdinalIgnoreCase);
    }

    private static int PeriodToMonths(string? period, int months)
    {
        if (months > 0)
        {
            return months;
        }

        return (period ?? string.Empty).Trim().ToLowerInvariant() switch
        {
            "month" => 1,
            "quarter" => 3,
            "year" => 12,
            _ => 0
        };
    }

    private static ServiceResult<long> PackageAmountByMonths(long? monthPrice, long? quarterPrice, long? yearPrice, int months)
    {
        if (months <= 0)
        {
            return ServiceResult<long>.Fail(ErrorCodes.InvalidParam, "invalid_months");
        }

        if (months == 1 && monthPrice is > 0)
        {
            return ServiceResult<long>.Ok(monthPrice.Value);
        }

        if (months == 3 && quarterPrice is > 0)
        {
            return ServiceResult<long>.Ok(quarterPrice.Value);
        }

        if (months == 12 && yearPrice is > 0)
        {
            return ServiceResult<long>.Ok(yearPrice.Value);
        }

        if (monthPrice is > 0)
        {
            return ServiceResult<long>.Ok(monthPrice.Value * months);
        }

        if (quarterPrice is > 0)
        {
            return ServiceResult<long>.Ok((long)Math.Round(quarterPrice.Value * months / 3d, MidpointRounding.AwayFromZero));
        }

        if (yearPrice is > 0)
        {
            return ServiceResult<long>.Ok((long)Math.Round(yearPrice.Value * months / 12d, MidpointRounding.AwayFromZero));
        }

        return ServiceResult<long>.Fail(ErrorCodes.InvalidParam, "no_valid_price");
    }

    private static string GenerateMerchantOrder(string prefix)
    {
        var now = DateTime.Now.ToString("yyyyMMddHHmmss");
        var token = RandomToken(6);
        return $"{prefix}-{now}-{token}";
    }

    private static string MarshalJson(object? value)
    {
        return JsonSerializer.Serialize(value ?? new { });
    }

    private static Dictionary<string, object?> ParseOrderData(string? raw)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            return new Dictionary<string, object?>(StringComparer.OrdinalIgnoreCase);
        }

        try
        {
            var map = JsonSerializer.Deserialize<Dictionary<string, object?>>(raw);
            return map ?? new Dictionary<string, object?>(StringComparer.OrdinalIgnoreCase);
        }
        catch
        {
            return new Dictionary<string, object?>(StringComparer.OrdinalIgnoreCase);
        }
    }

    private static long GetLongFromObject(object? value)
    {
        if (value == null)
        {
            return 0;
        }

        if (value is JsonElement json)
        {
            if (json.ValueKind == JsonValueKind.Number && json.TryGetInt64(out var number))
            {
                return number;
            }

            if (json.ValueKind == JsonValueKind.String && long.TryParse(json.GetString(), out var parsed))
            {
                return parsed;
            }

            return 0;
        }

        if (value is long l) return l;
        if (value is int i) return i;
        if (value is decimal d) return (long)d;
        if (value is double db) return (long)db;

        return long.TryParse(value.ToString(), out var result) ? result : 0;
    }

    private static bool GetBoolFromObject(object? value)
    {
        if (value == null)
        {
            return false;
        }

        if (value is JsonElement json)
        {
            return json.ValueKind switch
            {
                JsonValueKind.True => true,
                JsonValueKind.False => false,
                JsonValueKind.Number => json.TryGetInt32(out var n) && n != 0,
                JsonValueKind.String => ParseBoolFlag(json.GetString()),
                _ => false
            };
        }

        if (value is bool b)
        {
            return b;
        }

        return ParseBoolFlag(value.ToString());
    }

    private static bool ParseBoolFlag(string? raw)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            return false;
        }

        return raw.Trim().ToLowerInvariant() is "1" or "true" or "yes" or "on";
    }

    private static string SummarizeOrderMore(string? raw)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            return string.Empty;
        }

        try
        {
            using var doc = JsonDocument.Parse(raw);
            var root = doc.RootElement;
            var parts = new List<string>(4);

            if (root.TryGetProperty("channel", out var channel) && channel.ValueKind == JsonValueKind.String)
            {
                var text = channel.GetString();
                if (!string.IsNullOrWhiteSpace(text))
                {
                    parts.Add($"channel={text}");
                }
            }

            if (root.TryGetProperty("crypto", out var crypto) && crypto.ValueKind == JsonValueKind.String)
            {
                var text = crypto.GetString();
                if (!string.IsNullOrWhiteSpace(text))
                {
                    parts.Add($"crypto={text}");
                }
            }

            if (root.TryGetProperty("expected_amount", out var expected))
            {
                var text = expected.ToString();
                if (!string.IsNullOrWhiteSpace(text))
                {
                    parts.Add($"crypto_amount={text}");
                }
            }

            if (root.TryGetProperty("wallet", out var wallet) && wallet.ValueKind == JsonValueKind.String)
            {
                var text = wallet.GetString();
                if (!string.IsNullOrWhiteSpace(text))
                {
                    parts.Add($"wallet={text}");
                }
            }

            return parts.Count == 0 ? raw : string.Join(", ", parts);
        }
        catch
        {
            return raw;
        }
    }

    private static BalanceLogDto MapBalanceLog(BalanceLedger row)
    {
        return new BalanceLogDto
        {
            Id = row.Id,
            UserId = row.UserId,
            OrderId = row.OrderId,
            Action = row.Action,
            Source = row.Source,
            Reason = row.Reason,
            AmountBefore = row.AmountBefore,
            AmountChange = row.AmountChange,
            AmountAfter = row.AmountAfter,
            OperatorId = row.OperatorId,
            OperatorRole = row.OperatorRole,
            CreatedAt = row.CreatedAt.ToString("yyyy-MM-dd HH:mm:ss")
        };
    }

    private static ServiceResult<T> MapFinanceTranError<T>(Exception? ex)
    {
        var message = ex?.Message?.Trim().ToLowerInvariant() ?? string.Empty;
        if (message.Contains("user_not_found") || message.Contains("order_not_found") || message.Contains("package_not_found") || message.Contains("user_package_not_found"))
        {
            return ServiceResult<T>.Fail(ErrorCodes.NotFound);
        }

        if (message.Contains("insufficient_balance") || message.Contains("unsupported_order_type") || message.Contains("invalid_order_data"))
        {
            return ServiceResult<T>.Fail(ErrorCodes.InvalidParam, ex?.Message);
        }

        return ServiceResult<T>.Fail(ErrorCodes.InternalError);
    }

    private async Task<ServiceResult<Dictionary<string, object?>>> CreateShkeeperPayInfoAsync(string merchantOrder, long amountCents, string? callbackBaseUrl, CancellationToken cancellationToken)
    {
        var settings = await LoadShkeeperSettingsAsync(cancellationToken);
        if (!settings.Enable)
        {
            return ServiceResult<Dictionary<string, object?>>.Fail(ErrorCodes.ConfigError, "shkeeper_disabled");
        }

        if (string.IsNullOrWhiteSpace(settings.BaseUrl) || string.IsNullOrWhiteSpace(settings.ApiKey) || string.IsNullOrWhiteSpace(settings.CryptoName))
        {
            return ServiceResult<Dictionary<string, object?>>.Fail(ErrorCodes.ConfigError, "shkeeper_config_invalid");
        }

        var callbackUrl = settings.CallbackUrl;
        if (string.IsNullOrWhiteSpace(callbackUrl) && !string.IsNullOrWhiteSpace(callbackBaseUrl))
        {
            callbackUrl = callbackBaseUrl!.TrimEnd('/') + "/api/v1/pay/shkeeper/callback";
        }

        if (string.IsNullOrWhiteSpace(callbackUrl))
        {
            return ServiceResult<Dictionary<string, object?>>.Fail(ErrorCodes.ConfigError, "callback_url_empty");
        }

        var endpoint = $"{settings.BaseUrl.TrimEnd('/')}/api/v1/{Uri.EscapeDataString(settings.CryptoName)}/payment_request";
        var requestBody = new
        {
            external_id = merchantOrder,
            fiat = settings.Fiat,
            amount = FormatAmountText(amountCents),
            callback_url = callbackUrl
        };

        using var client = new HttpClient
        {
            Timeout = TimeSpan.FromSeconds(settings.TimeoutSec <= 0 ? 12 : settings.TimeoutSec)
        };

        using var req = new HttpRequestMessage(HttpMethod.Post, endpoint)
        {
            Content = new StringContent(JsonSerializer.Serialize(requestBody), Encoding.UTF8, "application/json")
        };
        req.Headers.Accept.Add(new MediaTypeWithQualityHeaderValue("application/json"));
        req.Headers.TryAddWithoutValidation("X-Shkeeper-Api-Key", settings.ApiKey);

        HttpResponseMessage response;
        try
        {
            response = await client.SendAsync(req, cancellationToken);
        }
        catch
        {
            return ServiceResult<Dictionary<string, object?>>.Fail(ErrorCodes.ExternalProviderError, "shkeeper_request_failed");
        }

        await using var stream = await response.Content.ReadAsStreamAsync(cancellationToken);

        ShkeeperInvoiceResponse? parsed;
        try
        {
            parsed = await JsonSerializer.DeserializeAsync<ShkeeperInvoiceResponse>(stream, cancellationToken: cancellationToken);
        }
        catch
        {
            return ServiceResult<Dictionary<string, object?>>.Fail(ErrorCodes.ExternalProviderError, "shkeeper_response_invalid");
        }

        if (response.StatusCode is < HttpStatusCode.OK or >= HttpStatusCode.MultipleChoices)
        {
            return ServiceResult<Dictionary<string, object?>>.Fail(ErrorCodes.ExternalProviderError, string.IsNullOrWhiteSpace(parsed?.Message) ? "shkeeper_create_failed" : parsed!.Message!);
        }

        if (!string.Equals(parsed?.Status?.Trim(), "success", StringComparison.OrdinalIgnoreCase))
        {
            return ServiceResult<Dictionary<string, object?>>.Fail(ErrorCodes.ExternalProviderError, string.IsNullOrWhiteSpace(parsed?.Message) ? "shkeeper_create_failed" : parsed!.Message!);
        }

        var data = new Dictionary<string, object?>
        {
            ["channel"] = "shkeeper",
            ["crypto"] = settings.CryptoName,
            ["fiat"] = settings.Fiat,
            ["invoice_id"] = parsed?.Id,
            ["wallet"] = parsed?.Wallet,
            ["expected_amount"] = parsed?.Amount,
            ["exchange_rate"] = parsed?.ExchangeRate,
            ["display_name"] = parsed?.DisplayName,
            ["status"] = parsed?.Status
        };

        return ServiceResult<Dictionary<string, object?>>.Ok(data);
    }

    private async Task<ShkeeperSettings> LoadShkeeperSettingsAsync(CancellationToken cancellationToken)
    {
        var cfg = await _systemConfigService.LoadSystemConfigAsync(cancellationToken);

        var settings = new ShkeeperSettings
        {
            Enable = _systemConfigService.ParseBoolFlag(cfg.GetValueOrDefault("pay_shkeeper_enable")),
            BaseUrl = (cfg.GetValueOrDefault("pay_shkeeper_base_url") ?? string.Empty).Trim().TrimEnd('/'),
            ApiKey = (cfg.GetValueOrDefault("pay_shkeeper_api_key") ?? string.Empty).Trim(),
            CallbackApiKey = (cfg.GetValueOrDefault("pay_shkeeper_callback_api_key") ?? string.Empty).Trim(),
            CryptoName = (cfg.GetValueOrDefault("pay_shkeeper_crypto") ?? "TRX-USDT").Trim(),
            Fiat = (cfg.GetValueOrDefault("pay_shkeeper_fiat") ?? "USD").Trim().ToUpperInvariant(),
            CallbackUrl = (cfg.GetValueOrDefault("pay_shkeeper_callback_url") ?? string.Empty).Trim(),
            TimeoutSec = ParsePositiveInt(cfg.GetValueOrDefault("pay_shkeeper_timeout_sec"), 12)
        };

        if (string.IsNullOrWhiteSpace(settings.CryptoName))
        {
            settings.CryptoName = "TRX-USDT";
        }

        if (string.IsNullOrWhiteSpace(settings.Fiat))
        {
            settings.Fiat = "USD";
        }

        if (settings.TimeoutSec <= 0)
        {
            settings.TimeoutSec = 12;
        }

        return settings;
    }

    private static int ParsePositiveInt(string? raw, int fallback)
    {
        if (!int.TryParse((raw ?? string.Empty).Trim(), out var value) || value <= 0)
        {
            return fallback;
        }

        return value;
    }

    private sealed class ShkeeperSettings
    {
        public bool Enable { get; set; }
        public string BaseUrl { get; set; } = string.Empty;
        public string ApiKey { get; set; } = string.Empty;
        public string CallbackApiKey { get; set; } = string.Empty;
        public string CryptoName { get; set; } = "TRX-USDT";
        public string Fiat { get; set; } = "USD";
        public string CallbackUrl { get; set; } = string.Empty;
        public int TimeoutSec { get; set; } = 12;
    }

    private sealed class ShkeeperInvoiceResponse
    {
        public string? Amount { get; set; }
        public string? DisplayName { get; set; }
        public string? ExchangeRate { get; set; }
        public long Id { get; set; }
        public string? Status { get; set; }
        public string? Wallet { get; set; }
        public string? Message { get; set; }
    }
}
