using Cnn.Api.Services.Stats;
using Xunit;

namespace Cnn.Api.Tests;

public sealed class AccessLogQueryWindowResolverTests
{
    [Fact]
    public void NormalizeSkew_ShouldKeepMeaningfulRealtimeLag()
    {
        var skew = AccessLogQueryWindowResolver.NormalizeSkew(
            maxLogUnixSeconds: 1_800_000_000 - 35 * 60,
            clickHouseNowUnixSeconds: 1_800_000_000);

        Assert.Equal(TimeSpan.FromMinutes(-35), skew);
    }

    [Fact]
    public void NormalizeSkew_ShouldIgnoreTinyOrUnsafeSkew()
    {
        Assert.Equal(TimeSpan.Zero, AccessLogQueryWindowResolver.NormalizeSkew(1_800_000_000 - 30, 1_800_000_000));
        Assert.Equal(TimeSpan.Zero, AccessLogQueryWindowResolver.NormalizeSkew(1_800_000_000 - 15 * 60 * 60, 1_800_000_000));
        Assert.Equal(TimeSpan.Zero, AccessLogQueryWindowResolver.NormalizeSkew(0, 1_800_000_000));
    }

    [Fact]
    public void AdjustForSkew_ShouldShiftRealtimeShortWindowAndReturnDisplayShift()
    {
        var now = new DateTime(2026, 4, 29, 15, 0, 0);
        var start = now.AddMinutes(-10);
        var end = now;

        var window = AccessLogQueryWindowResolver.AdjustForSkew(start, end, now, TimeSpan.FromMinutes(-35));

        Assert.Equal(start.AddMinutes(-35), window.Start);
        Assert.Equal(end.AddMinutes(-35), window.End);
        Assert.Equal(TimeSpan.FromMinutes(35), window.BucketDisplayShift);
    }

    [Fact]
    public void AdjustForSkew_ShouldNotShiftHistoricalOrLongRanges()
    {
        var now = new DateTime(2026, 4, 29, 15, 0, 0);
        var historical = AccessLogQueryWindowResolver.AdjustForSkew(
            new DateTime(2026, 4, 20, 0, 0, 0),
            new DateTime(2026, 4, 20, 23, 59, 59),
            now,
            TimeSpan.FromMinutes(-35));

        Assert.Equal(new DateTime(2026, 4, 20, 0, 0, 0), historical.Start);
        Assert.Equal(new DateTime(2026, 4, 20, 23, 59, 59), historical.End);
        Assert.Equal(TimeSpan.Zero, historical.BucketDisplayShift);

        var longRealtime = AccessLogQueryWindowResolver.AdjustForSkew(
            now.AddHours(-3),
            now,
            now,
            TimeSpan.FromMinutes(-35));

        Assert.Equal(now.AddHours(-3), longRealtime.Start);
        Assert.Equal(now, longRealtime.End);
        Assert.Equal(TimeSpan.Zero, longRealtime.BucketDisplayShift);
    }
}
