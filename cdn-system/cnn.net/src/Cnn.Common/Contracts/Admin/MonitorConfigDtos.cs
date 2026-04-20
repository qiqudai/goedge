using System.Text.Json.Serialization;

namespace Cnn.Common.Contracts.Admin;

public sealed class NodeMonitorConfigDto
{
    [JsonPropertyName("notification_period")]
    public string? NotificationPeriod { get; set; }

    [JsonPropertyName("notify_method")]
    public string? NotifyMethod { get; set; }

    [JsonPropertyName("notify_msg_type")]
    public string? NotifyMsgType { get; set; }

    [JsonPropertyName("email")]
    public string? Email { get; set; }

    [JsonPropertyName("phone")]
    public string? Phone { get; set; }

    [JsonPropertyName("bw_exceed_times")]
    public int BwExceedTimes { get; set; }

    [JsonPropertyName("auto_switch_enable")]
    public bool AutoSwitchEnable { get; set; }

    [JsonPropertyName("auto_switch_threshold")]
    public int AutoSwitchThreshold { get; set; }

    [JsonPropertyName("auto_switch_duration")]
    public int AutoSwitchDuration { get; set; }

    [JsonPropertyName("auto_switch_recover")]
    public int AutoSwitchRecover { get; set; }

    [JsonPropertyName("auto_switch_min_weight")]
    public int AutoSwitchMinWeight { get; set; }

    [JsonPropertyName("monitor_api")]
    public string? MonitorApi { get; set; }

    [JsonPropertyName("interval")]
    public int Interval { get; set; }

    [JsonPropertyName("failed_times")]
    public int FailedTimes { get; set; }

    [JsonPropertyName("failed_rate")]
    public string? FailedRate { get; set; }
}
