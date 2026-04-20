using System.Text.RegularExpressions;
using Xunit;

namespace Cnn.Api.Tests;

public sealed class PaginationUiConventionsTests
{
    [Fact]
    public void TablePager_Pages_Should_Use_Consistent_Search_Copy()
    {
        var pagerPages = GetTablePagerPages();
        Assert.True(pagerPages.Count >= 20, $"Expected at least 20 TablePager pages, got {pagerPages.Count}.");

        var queryLabelPattern = new Regex(@">\s*查询\s*<", RegexOptions.Compiled);
        var offenders = pagerPages
            .Where(page => queryLabelPattern.IsMatch(page.Content))
            .Select(page => page.Path)
            .ToList();

        Assert.True(offenders.Count == 0,
            "TablePager pages must use '搜索' instead of '查询'. Offenders:\n" + string.Join('\n', offenders));
    }

    [Fact]
    public void TablePager_Pages_Refresh_Button_Should_Not_Directly_Call_LoadMethods()
    {
        var pagerPages = GetTablePagerPages();
        Assert.True(pagerPages.Count >= 20, $"Expected at least 20 TablePager pages, got {pagerPages.Count}.");

        var refreshLoadPattern = new Regex(
            "<button[^>]*@onclick\\s*=\\s*\"Load[A-Za-z0-9_]*Async\"[^>]*>\\s*刷新\\s*</button>|<button[^>]*@onclick\\s*=\\s*'@\\(\\(\\)\\s*=>\\s*Load[A-Za-z0-9_]*Async\\)'[^>]*>\\s*刷新\\s*</button>",
            RegexOptions.Compiled | RegexOptions.Singleline);

        var offenders = pagerPages
            .Where(page => refreshLoadPattern.IsMatch(page.Content))
            .Select(page => page.Path)
            .ToList();

        Assert.True(offenders.Count == 0,
            "TablePager pages must route refresh actions through RefreshAsync semantics. Offenders:\n" + string.Join('\n', offenders));
    }

    [Fact]
    public void TablePager_StateKey_Should_Follow_Naming_Convention()
    {
        var pagerPages = GetTablePagerPages();
        Assert.True(pagerPages.Count >= 20, $"Expected at least 20 TablePager pages, got {pagerPages.Count}.");

        var stateKeyCapture = new Regex("StateKey\\s*=\\s*\"([^\"]+)\"", RegexOptions.Compiled);
        var stateKeyPattern = new Regex("^[a-z][a-z0-9_]*(?::[a-z0-9_]+)+:table$", RegexOptions.Compiled);
        var offenders = new List<string>();

        foreach (var page in pagerPages)
        {
            foreach (Match match in stateKeyCapture.Matches(page.Content))
            {
                var stateKey = match.Groups[1].Value;
                if (!stateKeyPattern.IsMatch(stateKey))
                {
                    offenders.Add($"{page.Path} => {stateKey}");
                }
            }
        }

        Assert.True(offenders.Count == 0,
            "StateKey naming must follow colon-separated lower_snake style and end with ':table'. Offenders:\n" + string.Join('\n', offenders));
    }

    private static List<(string Path, string Content)> GetTablePagerPages()
    {
        var root = FindRepoRoot();
        var pagesDir = Path.Combine(root, "src", "Cnn.Api", "Pages");
        var files = Directory.GetFiles(pagesDir, "*.razor", SearchOption.AllDirectories);

        return files
            .Select(path => (Path: path, Content: File.ReadAllText(path)))
            .Where(page => page.Content.Contains("<TablePager", StringComparison.Ordinal))
            .ToList();
    }

    private static string FindRepoRoot()
    {
        var current = new DirectoryInfo(AppContext.BaseDirectory);
        while (current != null)
        {
            var marker = Path.Combine(current.FullName, "src", "Cnn.Api");
            if (Directory.Exists(marker))
            {
                return current.FullName;
            }

            current = current.Parent;
        }

        throw new DirectoryNotFoundException("Unable to locate repository root from test base directory.");
    }
}
