using TaskEntity = Cnn.Domain.Entities.Task;
using SiteEntity = Cnn.Domain.Entities.Site;
using SiteConfCacheEntity = Cnn.Domain.Entities.SiteConfCache;
using ConfigEntity = Cnn.Domain.Entities.Config;
using Cnn.Api.Cache;
using Cnn.Common.Contracts;
using Cnn.Infrastructure.Db;
using Cnn.Api.Hubs;
using Cnn.Common.Localization;
using Cnn.Api.Responses;
using Cnn.Api.Data;
using Cnn.Api.Services;
using Cnn.Api.Services.Admin;
using Cnn.Api.Services.Agent;
using Cnn.Api.Services.Common;
using Cnn.Api.Services.Auth;
using Cnn.Api.Services.Authz;
using Cnn.Api.Services.Stats;
using Cnn.Api.Middleware;
using Microsoft.AspNetCore.SignalR;
using System.Net.WebSockets;
using System.Text;
using System.Text.Json;
using System.Text.Json.Serialization;
using SqlSugar;
using MudBlazor.Services;
using Cnn.Api.Extensions;

namespace Cnn.Api.Extensions;

using Cnn.Api.Endpoints;

public static class EndpointExtensions
{
    public static void MapApplicationEndpoints(this WebApplication app)
    {
        HealthEndpointModule.Map(app);
        TaskEndpointModule.Map(app);
        SiteCacheEndpointModule.Map(app);
        AgentWsEndpointModule.Map(app);
    }
}
