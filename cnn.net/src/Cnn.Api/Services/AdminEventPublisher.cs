using Cnn.Api.Hubs;
using Microsoft.AspNetCore.SignalR;

namespace Cnn.Api.Services;

public static class AdminHubGroups
{
    public const string Admin = "admin";

    public static string User(long userId) => $"user:{userId}";
}

public interface IAdminEventPublisher
{
    Task PublishToAdminsAsync(string eventName, object payload, CancellationToken cancellationToken = default);

    Task PublishToUserAsync(long userId, string eventName, object payload, CancellationToken cancellationToken = default);
}

public sealed class AdminEventPublisher : IAdminEventPublisher
{
    private readonly IHubContext<AdminHub> _hub;

    public AdminEventPublisher(IHubContext<AdminHub> hub)
    {
        _hub = hub;
    }

    public Task PublishToAdminsAsync(string eventName, object payload, CancellationToken cancellationToken = default)
    {
        return _hub.Clients.Group(AdminHubGroups.Admin).SendAsync("event", new { eventName, payload }, cancellationToken);
    }

    public Task PublishToUserAsync(long userId, string eventName, object payload, CancellationToken cancellationToken = default)
    {
        return _hub.Clients.Group(AdminHubGroups.User(userId)).SendAsync("event", new { eventName, payload }, cancellationToken);
    }
}
