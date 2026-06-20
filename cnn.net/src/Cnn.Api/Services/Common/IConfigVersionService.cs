namespace Cnn.Api.Services.Common;

public interface IConfigVersionService
{
    Task<long> BumpAsync(string resource, IReadOnlyList<long> ids, CancellationToken cancellationToken);
}
