using SqlSugar;

namespace Cnn.Domain.Entities;

[SugarTable("ip_switch_log")]
public class IpSwitchLog
{
    [SugarColumn(IsPrimaryKey = true, IsIdentity = true)]
    public long Id { get; set; }

    public DateTime? CreateAt { get; set; }

    public string? Type { get; set; }

    public int? NodeGroupId { get; set; }

    public int? NodeId { get; set; }

    public int? LineId { get; set; }

    public string? Ip { get; set; }

    public string? Action { get; set; }

    public bool? EmailNeedSend { get; set; }

    public bool? EmailIsSent { get; set; }

    public int? EmailFailTimes { get; set; }

    public string? EmailRet { get; set; }

    public DateTime? EmailTime { get; set; }

    public string? EmailSendState { get; set; }

    public bool? PhoneNeedSend { get; set; }

    public bool? PhoneIsSent { get; set; }

    public int? PhoneFailTimes { get; set; }

    public string? PhoneRet { get; set; }

    public DateTime? PhoneTime { get; set; }

    public string? PhoneSendState { get; set; }

    public string? Content { get; set; }
}