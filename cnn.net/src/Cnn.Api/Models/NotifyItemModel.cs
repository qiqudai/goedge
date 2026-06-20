namespace Cnn.Api.Models;

public sealed class NotifyItemModel
{
    public int Enable { get; set; }
    public List<string> Methods { get; set; } = new();
    public int ContinuousTimes { get; set; } = 1;
    public int Interval { get; set; } = 24;
    public NotifyTemplate EmailTemplate { get; set; } = new();
    public NotifyTemplate SmsTemplate { get; set; } = new();
    public int? RemainTraffic { get; set; }
    public int? Days { get; set; }
}

public sealed class NotifyTemplate
{
    public string? Title { get; set; }
    public string? Content { get; set; }
}
