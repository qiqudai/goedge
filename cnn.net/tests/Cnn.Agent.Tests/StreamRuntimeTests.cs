using Cnn.Agent.Logs;
using Cnn.Agent.Stream;
using Cnn.Common.Contracts.Agent;
using Microsoft.Extensions.Logging.Abstractions;
using Microsoft.Extensions.Options;
using Xunit;

namespace Cnn.Agent.Tests;

public sealed class StreamRuntimeTests
{
    [Fact]
    public void Apply_TracksSkipReason_WhenCompileHasErrors()
    {
        var runtime = CreateRuntime();
        var config = new EdgeConfigDto
        {
            Version = 11,
            Streams = new List<EdgeStreamDto>
            {
                new()
                {
                    Id = 101,
                    ListenPorts = new List<string> { "not-a-port" },
                    Targets = new List<EdgeStreamTargetDto>
                    {
                        new() { Addr = "127.0.0.1:9000", Enable = true }
                    }
                }
            }
        };

        var result = runtime.Apply(config);
        var report = runtime.GetReport();

        Assert.False(result.Success);
        Assert.Equal(1, result.Received);
        Assert.Equal(0, result.Planned);
        Assert.Equal(1, result.Skipped);
        Assert.Contains("compile_errors", result.SkipReasons ?? Array.Empty<string>());
        Assert.Equal(11, report.LastConfigVersion);
        Assert.Equal(1, report.LastReceived);
        Assert.Equal(1, report.LastSkipped);
        Assert.Equal("userspace:empty", report.LastPlanHash);
        Assert.Contains("compile_errors", report.LastSkipReasons);
    }

    [Fact]
    public void Apply_ReportsPlanUnchanged_OnSecondRun()
    {
        var runtime = CreateRuntime();
        var config = new EdgeConfigDto
        {
            Version = 12,
            Streams = new List<EdgeStreamDto>
            {
                new()
                {
                    Id = 102,
                    ListenPorts = new List<string> { "invalid-listen" },
                    Targets = new List<EdgeStreamTargetDto>
                    {
                        new() { Addr = "127.0.0.1:9001", Enable = true }
                    }
                }
            }
        };

        runtime.Apply(config);
        var second = runtime.Apply(config);
        var report = runtime.GetReport();

        Assert.False(second.Success);
        Assert.Contains("plan_unchanged", second.SkipReasons ?? Array.Empty<string>());
        Assert.Equal("userspace:empty", report.LastPlanHash);
        Assert.Contains("plan_unchanged", report.LastSkipReasons);
    }

    private static StreamRuntime CreateRuntime()
    {
        return new StreamRuntime(
            compiler: new StreamRouteCompiler(),
            logWriter: new NoopLogWriter(),
            kernelNatRuntime: new KernelNatRuntime(NullLogger<KernelNatRuntime>.Instance),
            optionsMonitor: new StaticOptionsMonitor<StreamRuntimeOptions>(new StreamRuntimeOptions { Mode = "userspace" }),
            loggerFactory: NullLoggerFactory.Instance,
            logger: NullLogger<StreamRuntime>.Instance);
    }

    private sealed class NoopLogWriter : ILogEventWriter
    {
        public bool TryWrite(LogEvent logEvent)
        {
            return true;
        }
    }

    private sealed class StaticOptionsMonitor<T> : IOptionsMonitor<T> where T : class
    {
        private readonly T _value;

        public StaticOptionsMonitor(T value)
        {
            _value = value;
        }

        public T CurrentValue => _value;

        public T Get(string? name)
        {
            return _value;
        }

        public IDisposable? OnChange(Action<T, string?> listener)
        {
            return null;
        }
    }
}
