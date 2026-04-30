using Cnn.Agent.Acme;
using Cnn.Agent.Cache;
using Cnn.Agent.Config;
using Cnn.Agent.Diagnostics;
using Cnn.Agent.Logs;
using Cnn.Agent.Network;
using Cnn.Agent.Plugin;
using Cnn.Agent.Proxy;
using Cnn.Agent.Security;
using Cnn.Agent.Sync;
using Cnn.Agent.Stream;
using Cnn.Agent.Tasks;
using Cnn.Agent.Ws;
using Microsoft.AspNetCore.OutputCaching;
using Yarp.ReverseProxy;
using Yarp.ReverseProxy.Forwarder;
using Yarp.ReverseProxy.Configuration;
using Yarp.ReverseProxy.Health;
using Yarp.ReverseProxy.LoadBalancing;
using System.Security.Cryptography.X509Certificates;

var builder = WebApplication.CreateBuilder(args);

var runtimePaths = new AgentRuntimePaths(builder.Configuration);
var tlsCertificateStore = new TlsCertificateStore(runtimePaths);
var tlsRuntimePolicyStore = new TlsRuntimePolicyStore();
builder.WebHost.ConfigureKestrel(options =>
{
    options.ConfigureHttpsDefaults(https =>
    {
        https.ServerCertificateSelector = (_, host) => tlsCertificateStore.GetForHost(host);
        https.OnAuthenticate = (_, sslOptions) =>
        {
            var policy = tlsRuntimePolicyStore.GetCurrent();
            sslOptions.EnabledSslProtocols = policy.SslProtocols;
            sslOptions.CertificateRevocationCheckMode = policy.CheckCertificateRevocation
                ? X509RevocationMode.Online
                : X509RevocationMode.NoCheck;
            sslOptions.CipherSuitesPolicy = policy.CipherSuitesPolicy;
        };
    });
});

builder.Services.AddSingleton(runtimePaths);
builder.Services.AddSingleton<ITlsCertificateStore>(tlsCertificateStore);
builder.Services.AddSingleton<ITlsRuntimePolicyStore>(tlsRuntimePolicyStore);
builder.Services.AddSingleton<AgentNodeState>();

builder.Services.Configure<CacheOptions>(builder.Configuration.GetSection("Cache"));
builder.Services.PostConfigure<CacheOptions>(options =>
{
    options.Root = AgentRuntimePaths.ResolveCacheRoot(options.Root, runtimePaths.RuntimeRoot);
});
builder.Services.AddSingleton<CacheRuntimeStore>();
builder.Services.AddSingleton<DynamicCachePolicy>();
builder.Services.AddSingleton<EdgeConfigStore>();
builder.Services.AddSingleton<AcmeTokenStore>();
builder.Services.AddSingleton<ISyncStateStore, SyncStateStore>();
builder.Services.AddSingleton<IConfigVersionTracker, ConfigVersionTracker>();
builder.Services.AddSingleton<ITaskIdempotencyStore, TaskIdempotencyStore>();
builder.Services.AddSingleton<ITaskAckOutbox, TaskAckOutbox>();
builder.Services.Configure<LogPipelineOptions>(builder.Configuration.GetSection("LogPipeline"));
builder.Services.AddSingleton<ILogSink, FileLogSink>();
builder.Services.AddSingleton<LogPipeline>();
builder.Services.AddSingleton<ILogEventWriter>(sp => sp.GetRequiredService<LogPipeline>());
builder.Services.AddSingleton<ILogPipelineStats>(sp => sp.GetRequiredService<LogPipeline>());
builder.Services.AddSingleton<ILogQueryService, FileLogQueryService>();
builder.Services.AddSingleton<ILogRetentionService, LogRetentionService>();
builder.Services.AddHostedService(sp => sp.GetRequiredService<LogPipeline>());
builder.Services.AddHostedService<LogRetentionWorker>();
builder.Services.AddSingleton<IDebugSwitchStore, DebugSwitchStore>();
builder.Services.AddSingleton<IDebugSessionService, DebugSessionService>();
builder.Services.AddSingleton<IDebugAuditLogger, DebugAuditLogger>();
builder.Services.AddSingleton<IManualDebugLogWriter, ManualDebugLogWriter>();
builder.Services.AddSingleton<IEdgeDomainResolver, EdgeDomainResolver>();
builder.Services.AddSingleton<IPackageBandwidthLimiter, LinuxPackageBandwidthLimiter>();
builder.Services.AddSingleton<WafMatcher>();
builder.Services.AddSingleton<CcEngine>();
builder.Services.AddSingleton<PluginHost>();
builder.Services.AddSingleton<IPluginHost>(sp => sp.GetRequiredService<PluginHost>());
builder.Services.AddSingleton<ISecurityDecisionService, SecurityDecisionService>();
builder.Services.Configure<StreamRuntimeOptions>(builder.Configuration.GetSection("Stream"));
builder.Services.AddSingleton<StreamRouteCompiler>();
builder.Services.AddSingleton<KernelNatRuntime>();
builder.Services.AddSingleton<IStreamRuntime, StreamRuntime>();

builder.Services.AddSingleton<EdgeConfigToYarpCompiler>();
builder.Services.AddSingleton<ProxyConfigValidator>();
builder.Services.AddSingleton<DynamicProxyConfigProvider>();
builder.Services.AddSingleton<IProxyConfigProvider>(sp => sp.GetRequiredService<DynamicProxyConfigProvider>());
builder.Services.AddSingleton<IEdgeProxyRuntime, EdgeProxyRuntime>();
builder.Services.AddSingleton<ProxyHealthReportBuilder>();
builder.Services.AddSingleton<ILoadBalancingPolicy, ClientIpHashLoadBalancingPolicy>();
builder.Services.AddSingleton<ILoadBalancingPolicy, WeightedRoundRobinLoadBalancingPolicy>();
builder.Services.AddSingleton<IForwarderHttpClientFactory, EdgeForwarderHttpClientFactory>();
builder.Services.Configure<ConsecutiveFailuresHealthPolicyOptions>(builder.Configuration.GetSection("ReverseProxy:Health:ActivePolicy"));
builder.Services.PostConfigure<ConsecutiveFailuresHealthPolicyOptions>(options =>
{
    if (options.DefaultThreshold <= 0)
    {
        options.DefaultThreshold = 2;
    }
});
builder.Services.Configure<TransportFailureRateHealthPolicyOptions>(builder.Configuration.GetSection("ReverseProxy:Health:PassivePolicy"));
builder.Services.PostConfigure<TransportFailureRateHealthPolicyOptions>(options =>
{
    if (options.DetectionWindowSize <= TimeSpan.Zero)
    {
        options.DetectionWindowSize = TimeSpan.FromMinutes(1);
    }

    if (options.MinimalTotalCountThreshold <= 0)
    {
        options.MinimalTotalCountThreshold = 10;
    }

    if (options.DefaultFailureRateLimit <= 0 || options.DefaultFailureRateLimit >= 1)
    {
        options.DefaultFailureRateLimit = 0.3;
    }
});
builder.Services.AddReverseProxy();

builder.Services.AddOutputCache(options =>
{
    options.AddBasePolicy(policy =>
    {
        policy.AddPolicy<DynamicCachePolicy>();
        policy.SetVaryByHost(true);
    });
});

builder.Services.AddHttpContextAccessor();
builder.Services.AddSingleton<IOutputCacheStore, FileOutputCacheStore>();
builder.Services.AddHttpClient();

builder.Services.AddHostedService<AgentWsClient>();
builder.Services.AddHostedService(sp => sp.GetRequiredService<PluginHost>());
builder.Services.AddHostedService<StreamProxyHost>();

var app = builder.Build();

var activeHealthMonitor = app.Services.GetService<IActiveHealthCheckMonitor>();
if (activeHealthMonitor == null)
{
    app.Logger.LogWarning("YARP active health monitor service is unavailable.");
}

app.Use(async (context, next) =>
{
    var nodeState = context.RequestServices.GetRequiredService<AgentNodeState>();
    if (!nodeState.Enabled)
    {
        context.Response.StatusCode = StatusCodes.Status503ServiceUnavailable;
        await context.Response.WriteAsync("node disabled");
        return;
    }

    await next();
});

app.UseMiddleware<SiteSecurityMiddleware>();

app.UseOutputCache();

app.MapGet("/debug/proxy/health", (
    IEdgeProxyRuntime runtime,
    ProxyHealthReportBuilder reportBuilder,
    IProxyStateLookup? proxyStateLookup,
    IActiveHealthCheckMonitor? healthMonitor) =>
{
    var snapshot = runtime.GetCurrent();
    var clusters = proxyStateLookup?.GetClusters() ?? [];
    var report = reportBuilder.Build(snapshot, clusters, healthMonitor?.InitialProbeCompleted ?? false);
    return Results.Json(report);
});

app.MapGet("/debug/stream/runtime", (IStreamRuntime streamRuntime) =>
{
    return Results.Json(streamRuntime.GetReport());
});

app.MapGet("/.well-known/acme-challenge/{token}", (string token, AcmeTokenStore store) =>
{
    if (string.IsNullOrWhiteSpace(token))
    {
        return Results.NotFound();
    }

    if (store.TryGet(token.Trim(), out var value) && !string.IsNullOrWhiteSpace(value))
    {
        return Results.Text(value.Trim());
    }

    return Results.NotFound();
});

app.Use(async (context, next) =>
{
    var proxyProvider = context.RequestServices.GetRequiredService<DynamicProxyConfigProvider>();
    if (proxyProvider.IsFallbackMode)
    {
        context.Response.StatusCode = StatusCodes.Status503ServiceUnavailable;
        await context.Response.WriteAsync("route config unavailable");
        return;
    }

    await next();
});

app.MapReverseProxy(proxyPipeline =>
{
    proxyPipeline.UseSessionAffinity();
    proxyPipeline.UseLoadBalancing();
    proxyPipeline.UsePassiveHealthChecks();
}).CacheOutput();

app.Run();
