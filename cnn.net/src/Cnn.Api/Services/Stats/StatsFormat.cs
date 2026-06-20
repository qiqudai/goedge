using System.Globalization;

namespace Cnn.Api.Services.Stats;

public static class StatsFormat
{
    public static double RoundFloat(double value, int precision)
    {
        if (precision < 0)
        {
            return value;
        }

        return Math.Round(value, precision, MidpointRounding.AwayFromZero);
    }

    public static double BytesToMB(ulong bytes)
    {
        return bytes / (1024.0 * 1024.0);
    }

    public static double BytesToMbps(ulong bytes, TimeSpan bucket)
    {
        var seconds = bucket.TotalSeconds;
        if (seconds <= 0)
        {
            return 0;
        }
        return bytes * 8 / seconds / 1_000_000.0;
    }

    public static string FormatBytes(ulong bytes)
    {
        var units = new[] { "B", "KB", "MB", "GB", "TB", "PB" };
        var value = (double)bytes;
        var idx = 0;
        while (value >= 1024 && idx < units.Length - 1)
        {
            value /= 1024;
            idx++;
        }

        if (idx == 0)
        {
            return bytes.ToString(CultureInfo.InvariantCulture) + " B";
        }

        return value.ToString("F2", CultureInfo.InvariantCulture) + " " + units[idx];
    }

    public static string FormatBandwidth(double mbps)
    {
        if (mbps >= 1000)
        {
            return (mbps / 1000).ToString("F2", CultureInfo.InvariantCulture) + " Gbps";
        }

        return mbps.ToString("F2", CultureInfo.InvariantCulture) + " Mbps";
    }

    public static string FormatCount(ulong count)
    {
        return count.ToString(CultureInfo.InvariantCulture);
    }
}
