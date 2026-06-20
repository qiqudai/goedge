using Cnn.Api.Services.Agent.Ws;

namespace Cnn.Api.Endpoints;

public static class AgentWsEndpointModule
{
    public static void Map(WebApplication app)
    {
        app.MapGet("/ws/agent", async context =>
        {
            if (!context.WebSockets.IsWebSocketRequest)
            {
                context.Response.StatusCode = StatusCodes.Status400BadRequest;
                return;
            }

            using var socket = await context.WebSockets.AcceptWebSocketAsync();
            var handler = context.RequestServices.GetRequiredService<IAgentWsSessionHandler>();
            await handler.HandleAsync(context, socket, context.RequestAborted);
        });
    }
}
