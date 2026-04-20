namespace Cnn.Agent.Diagnostics;

public sealed class DebugOptions
{
    public bool Enabled { get; set; }
    public bool InternalIpOnly { get; set; } = true;
    public string? Token { get; set; }
    public bool AllowHeaderToken { get; set; } = true;
    public bool AllowQueryFlag { get; set; }
    public double SampleRate { get; set; } = 0.01d;
    public int MaxEventsPerSec { get; set; } = 200;
    public Dictionary<string, bool> Modules { get; set; } = new(StringComparer.OrdinalIgnoreCase);

    public static DebugOptions Disabled => new()
    {
        Enabled = false,
        InternalIpOnly = true,
        AllowHeaderToken = true,
        AllowQueryFlag = false,
        SampleRate = 0.01d,
        MaxEventsPerSec = 200,
        Modules = new Dictionary<string, bool>(StringComparer.OrdinalIgnoreCase)
    };
}
