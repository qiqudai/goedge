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

namespace Cnn.Api.Endpoints;

public static class HealthEndpointModule
{
    public static void Map(WebApplication app)
    {
        var jsonOptions = new JsonSerializerOptions(JsonSerializerDefaults.Web)
        {
            PropertyNameCaseInsensitive = true,
            DefaultIgnoreCondition = JsonIgnoreCondition.WhenWritingNull
        };
        static object BuildHealthPayload(HttpContext context, IMessageLocalizer localizer)
        {
            var language = LanguageResolver.Resolve(context, localizer.DefaultLanguage);
            return new
            {
                status = localizer.Translate("status.ok", language),
                node = localizer.Translate("server-1", language)
            };
        }
        
        app.MapGet("/health", (HttpContext context, IMessageLocalizer localizer) =>
        {
            return Results.Ok(ApiResponseFactory.Ok(context, localizer, BuildHealthPayload(context, localizer)));
        });
        
        app.MapGet("/api/health", (HttpContext context, IMessageLocalizer localizer) =>
        {
            return Results.Ok(ApiResponseFactory.Ok(context, localizer, BuildHealthPayload(context, localizer)));
        });
        

    }

}
