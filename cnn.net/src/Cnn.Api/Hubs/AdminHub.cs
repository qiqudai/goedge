using Cnn.Api.Services;
using Microsoft.AspNetCore.SignalR;

namespace Cnn.Api.Hubs;

public sealed class AdminHub : Hub
{
    private readonly IAdminIdentityResolver _identityResolver;

    public AdminHub(IAdminIdentityResolver identityResolver)
    {
        _identityResolver = identityResolver;
    }

    public override async Task OnConnectedAsync()
    {
        var identity = _identityResolver.Resolve(Context.User);
        if (identity == null)
        {
            Context.Abort();
            return;
        }

        await Groups.AddToGroupAsync(Context.ConnectionId, AdminHubGroups.User(identity.UserId));
        if (identity.IsAdmin)
        {
            await Groups.AddToGroupAsync(Context.ConnectionId, AdminHubGroups.Admin);
        }

        await base.OnConnectedAsync();
    }
}
