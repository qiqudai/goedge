using Cnn.Common.Contracts;
using Microsoft.AspNetCore.Components;
using Microsoft.AspNetCore.SignalR.Client;

namespace Cnn.Api.Services;

public sealed class TaskHubClient : IAsyncDisposable
{
    private readonly HubConnection _connection;

    public event Action<TaskUpdateDto>? TaskUpdated;

    public TaskHubClient(NavigationManager navigationManager)
    {
        // Task hub is hosted by the same ASP.NET app serving this UI.
        // Prefer relative resolution from current navigation base URI so local/non-default ports work.
        var baseUrl = navigationManager.BaseUri.TrimEnd('/');
        _connection = new HubConnectionBuilder()
            .WithUrl($"{baseUrl}/ws/tasks")
            .WithAutomaticReconnect()
            .Build();

        _connection.On<TaskUpdateDto>("task_update", update => TaskUpdated?.Invoke(update));
    }

    public Task StartAsync(CancellationToken cancellationToken = default)
    {
        if (_connection.State == HubConnectionState.Disconnected)
        {
            return _connection.StartAsync(cancellationToken);
        }
        return Task.CompletedTask;
    }

    public async ValueTask DisposeAsync()
    {
        await _connection.DisposeAsync();
    }
}
