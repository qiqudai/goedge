using Bunit;
using Cnn.Api.Shared;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.JSInterop;
using Microsoft.Extensions.Logging;
using Xunit;

namespace Cnn.Api.Tests;

public sealed class TablePagerTests : TestContext
{
    public TablePagerTests()
    {
        JSInterop.Mode = JSRuntimeMode.Loose;
    }

    [Fact]
    public void FirstRender_RestoresState_BeforeFirstQuery()
    {
        JSInterop.Setup<string?>("cnn.storage.get", invocation =>
            invocation.Arguments.Count == 1 &&
            string.Equals(invocation.Arguments[0]?.ToString(), "table:pager:test", StringComparison.Ordinal))
            .SetResult("{\"PagingEnabled\":true,\"Page\":3,\"PageSize\":50}");

        var queries = new List<TablePageQuery>();

        var cut = RenderComponent<TablePager>(parameters => parameters
            .Add(x => x.StateKey, "table:pager:test")
            .Add(x => x.Total, 300)
            .Add(x => x.PageSizes, new[] { 10, 20, 50, 100 })
            .Add(x => x.QueryChanged, (TablePageQuery q) => queries.Add(q)));

        cut.WaitForAssertion(() => Assert.Single(queries));
        Assert.True(queries[0].PagingEnabled);
        Assert.Equal(3, queries[0].Page);
        Assert.Equal(50, queries[0].PageSize);
    }

    [Fact]
    public void PageSizeChanged_ResetsPageToOne_AndEmitsQuery()
    {
        var queries = new List<TablePageQuery>();

        var cut = RenderComponent<TablePager>(parameters => parameters
            .Add(x => x.Total, 200)
            .Add(x => x.PageSizes, new[] { 10, 20, 50, 100 })
            .Add(x => x.QueryChanged, (TablePageQuery q) => queries.Add(q)));

        cut.WaitForAssertion(() => Assert.Single(queries));

        cut.FindAll("button").Single(button => button.TextContent.Contains("下一页", StringComparison.Ordinal)).Click();
        cut.Find("select.form-select-sm").Change("50");

        cut.WaitForAssertion(() => Assert.Equal(3, queries.Count));
        Assert.Equal(2, queries[1].Page);
        Assert.Equal(20, queries[1].PageSize);
        Assert.Equal(1, queries[2].Page);
        Assert.Equal(50, queries[2].PageSize);
    }

    [Fact]
    public void PagingToggleOff_UsesUnpagedLimit()
    {
        var queries = new List<TablePageQuery>();

        var cut = RenderComponent<TablePager>(parameters => parameters
            .Add(x => x.Total, 200)
            .Add(x => x.UnpagedLimit, 1000)
            .Add(x => x.QueryChanged, (TablePageQuery q) => queries.Add(q)));

        cut.WaitForAssertion(() => Assert.Single(queries));

        cut.Find("input.form-check-input[type='checkbox']").Change(false);

        cut.WaitForAssertion(() => Assert.Equal(2, queries.Count));
        Assert.False(queries[1].PagingEnabled);
        Assert.Equal(1, queries[1].Page);
        Assert.Equal(1000, queries[1].PageSize);
    }

    [Fact]
    public async Task RapidTriggers_KeepLatestState()
    {
        JSInterop.Setup<string?>("cnn.storage.get", invocation =>
            invocation.Arguments.Count == 1 &&
            string.Equals(invocation.Arguments[0]?.ToString(), "table:pager:merge", StringComparison.Ordinal))
            .SetResult("{\"PagingEnabled\":true,\"Page\":3,\"PageSize\":20}");

        var queries = new List<TablePageQuery>();
        var gate = new TaskCompletionSource(TaskCreationOptions.RunContinuationsAsynchronously);

        var cut = RenderComponent<TablePager>(parameters => parameters
            .Add(x => x.StateKey, "table:pager:merge")
            .Add(x => x.Total, 200)
            .Add(x => x.QueryChanged, async (TablePageQuery q) =>
            {
                queries.Add(q);
                if (queries.Count == 1)
                {
                    await gate.Task;
                }
            }));

        cut.WaitForAssertion(() => Assert.Single(queries));
        await Task.WhenAll(
            cut.Instance.RefreshAsync(),
            cut.Instance.SearchAsync(),
            cut.Instance.RefreshAsync(),
            cut.Instance.SearchAsync());
        gate.SetResult();

        cut.WaitForAssertion(() => Assert.True(queries.Count >= 2));
        await Task.Delay(150);

        Assert.Equal(1, queries[^1].Page);
        Assert.Equal(20, queries[^1].PageSize);
    }

    [Fact]
    public void EmitQuery_WritesStructuredLogFields()
    {
        var logger = new CapturingLogger<TablePager>();
        Services.AddSingleton<ILogger<TablePager>>(logger);

        var queries = new List<TablePageQuery>();
        var cut = RenderComponent<TablePager>(parameters => parameters
            .Add(x => x.StateKey, "system:tasks:table")
            .Add(x => x.Total, 123)
            .Add(x => x.QueryChanged, (TablePageQuery q) => queries.Add(q)));

        cut.WaitForAssertion(() => Assert.Single(queries));
        cut.WaitForAssertion(() => Assert.NotEmpty(logger.Entries));

        var entry = logger.Entries.Last();
        Assert.Equal(LogLevel.Information, entry.Level);
        Assert.Equal("table_pager_query", entry.State["UiEvent"]?.ToString());
        Assert.Equal("system:tasks:table", entry.State["StateKey"]?.ToString());
        Assert.Equal("True", entry.State["PagingEnabled"]?.ToString());
        Assert.Equal("1", entry.State["Page"]?.ToString());
        Assert.Equal("20", entry.State["PageSize"]?.ToString());
        Assert.Equal("123", entry.State["Total"]?.ToString());
        Assert.Contains("table_pager_query", entry.Message, StringComparison.Ordinal);
    }

    [Fact]
    public void PagingDisabled_EmitsStructuredLogWithUnpagedLimit()
    {
        var logger = new CapturingLogger<TablePager>();
        Services.AddSingleton<ILogger<TablePager>>(logger);

        var queries = new List<TablePageQuery>();
        var cut = RenderComponent<TablePager>(parameters => parameters
            .Add(x => x.StateKey, "website:list:table")
            .Add(x => x.Total, 456)
            .Add(x => x.UnpagedLimit, 1000)
            .Add(x => x.QueryChanged, (TablePageQuery q) => queries.Add(q)));

        cut.WaitForAssertion(() => Assert.Single(queries));
        cut.Find("input.form-check-input[type='checkbox']").Change(false);
        cut.WaitForAssertion(() => Assert.Equal(2, queries.Count));

        Assert.False(queries[^1].PagingEnabled);
        Assert.Equal(1, queries[^1].Page);
        Assert.Equal(1000, queries[^1].PageSize);

        cut.WaitForAssertion(() =>
            Assert.Contains(logger.Entries, e =>
                e.State.TryGetValue("PagingEnabled", out var pagingEnabled) &&
                string.Equals(pagingEnabled?.ToString(), "False", StringComparison.Ordinal) &&
                e.State.TryGetValue("PageSize", out var pageSize) &&
                string.Equals(pageSize?.ToString(), "1000", StringComparison.Ordinal) &&
                e.State.TryGetValue("StateKey", out var stateKey) &&
                string.Equals(stateKey?.ToString(), "website:list:table", StringComparison.Ordinal)));
    }

    private sealed class CapturingLogger<T> : ILogger<T>
    {
        public List<LogEntry> Entries { get; } = new();

        public IDisposable? BeginScope<TState>(TState state) where TState : notnull => null;

        public bool IsEnabled(LogLevel logLevel) => true;

        public void Log<TState>(
            LogLevel logLevel,
            EventId eventId,
            TState state,
            Exception? exception,
            Func<TState, Exception?, string> formatter)
        {
            var values = state as IEnumerable<KeyValuePair<string, object?>> ?? Array.Empty<KeyValuePair<string, object?>>();
            var dict = values.ToDictionary(x => x.Key, x => x.Value, StringComparer.Ordinal);
            Entries.Add(new LogEntry(logLevel, dict, formatter(state, exception)));
        }
    }

    private sealed record LogEntry(LogLevel Level, IReadOnlyDictionary<string, object?> State, string Message);
}
