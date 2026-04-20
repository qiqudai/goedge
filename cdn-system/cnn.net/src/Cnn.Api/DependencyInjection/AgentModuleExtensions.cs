using Cnn.Api.Services.Agent;
using Cnn.Api.Services.Admin;

namespace Cnn.Api.DependencyInjection;

public static class AgentModuleExtensions
{
    public static IServiceCollection AddAgentModule(this IServiceCollection services)
    {
        services.AddSingleton<INodeStatusService, NodeStatusService>();
        services.AddSingleton<IAgentConnectionManager, AgentConnectionManager>();
        services.AddSingleton<IAgentAckWaiter, AgentAckWaiter>();
        services.AddSingleton<INodeRateLimitService, NodeRateLimitService>();
        services.AddScoped<INodeMonitorLogService, NodeMonitorLogService>();
        services.AddScoped<IAgentTaskAckService, AgentTaskAckService>();
        services.AddScoped<IWsDispatchService, WsDispatchService>();
        services.AddScoped<IAgentApiTraceService, AgentApiTraceService>();
        services.AddScoped<IAgentPackageService, AgentPackageService>();
        services.AddScoped<IAgentNodeService, AgentNodeService>();
        services.AddScoped<IAgentLogService, AgentLogService>();
        services.AddScoped<IAgentUpgradeService, AgentUpgradeService>();
        services.AddScoped<IAgentTaskService, AgentTaskService>();
        services.AddScoped<Cnn.Api.Services.Agent.Ws.IAgentWsSessionHandler, Cnn.Api.Services.Agent.Ws.AgentWsSessionHandler>();
        return services;
    }
}
