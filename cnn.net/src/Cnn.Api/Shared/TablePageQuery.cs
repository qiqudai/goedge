namespace Cnn.Api.Shared;

public sealed class TablePageQuery
{
    public bool PagingEnabled { get; init; }
    public int Page { get; init; }
    public int PageSize { get; init; }
}
